package content

import (
	"fmt"
	"slices"
	"sync"

	"github.com/lucimux/arya/internal/entity"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compiled holds the two validators for one version of a content type.
type compiled struct {
	ct entity.ContentType
	// full enforces "required" (create); partial does not (update).
	full, partial *jsonschema.Schema
}

// schemaCache compiles each content type's JSON Schema once and recompiles it when the definition
// changes, e.g. after the schema file is edited and the watcher re-syncs.
type schemaCache struct {
	mu      sync.Mutex
	entries map[string]compiled
}

func newSchemaCache() *schemaCache {
	return &schemaCache{entries: make(map[string]compiled)}
}

func (c *schemaCache) get(ct entity.ContentType, partial bool) (*jsonschema.Schema, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, ok := c.entries[ct.TableName]
	if !ok || !slices.Equal(cached.ct.Attributes, ct.Attributes) {
		var err error

		cached, err = compile(ct)
		if err != nil {
			return nil, err
		}

		c.entries[ct.TableName] = cached
	}

	if partial {
		return cached.partial, nil
	}

	return cached.full, nil
}

func compile(ct entity.ContentType) (compiled, error) {
	full, err := compileDoc(ct.TableName+".full", schemaDoc(ct, true))
	if err != nil {
		return compiled{}, err
	}

	partial, err := compileDoc(ct.TableName+".partial", schemaDoc(ct, false))
	if err != nil {
		return compiled{}, err
	}

	return compiled{ct: ct, full: full, partial: partial}, nil
}

func compileDoc(name string, doc map[string]any) (*jsonschema.Schema, error) {
	url := "mem://content/" + name + ".json"

	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	// In draft 2020-12 "format" is only an annotation unless asserted; media ids must be UUIDs.
	c.AssertFormat()

	if err := c.AddResource(url, doc); err != nil {
		return nil, fmt.Errorf("jsonschema AddResource %s: %w", name, err)
	}

	sch, err := c.Compile(url)
	if err != nil {
		return nil, fmt.Errorf("jsonschema Compile %s: %w", name, err)
	}

	return sch, nil
}

// schemaDoc is the payload schema for ct — the same document stored in the schemas directory,
// except that optional fields also accept null so they can be cleared. Built as JSON values
// ([]any, map[string]any) because that is what the compiler accepts.
func schemaDoc(ct entity.ContentType, withRequired bool) map[string]any {
	props := make(map[string]any, len(ct.Attributes))
	required := make([]any, 0, len(ct.Attributes))

	for _, a := range ct.Attributes {
		jsonType, prop := string(a.Type), map[string]any{}

		// A media field holds the id of a Media Library file.
		switch a.Type {
		case entity.AttributeTypeString, entity.AttributeTypeInteger, entity.AttributeTypeBoolean:
			// Uses default jsonType.
		case entity.AttributeTypeMedia:
			jsonType = "string"
			prop["format"] = "uuid"
		case entity.AttributeTypeComponent:
			if a.Repeatable {
				jsonType = "array"
				prop["items"] = map[string]any{"type": "object"}
			} else {
				jsonType = "object"
			}
		}

		if a.Required {
			required = append(required, a.Name)
			prop["type"] = jsonType
		} else {
			prop["type"] = []any{jsonType, "null"}
		}

		props[a.Name] = prop
	}

	doc := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}

	if withRequired && len(required) > 0 {
		doc["required"] = required
	}

	return doc
}
