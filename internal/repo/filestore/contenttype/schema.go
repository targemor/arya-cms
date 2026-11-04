package contenttype

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/lucimux/arya/internal/entity"
)

const _jsonSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

// schemaDocument is the on-disk form of a content type: a JSON Schema that validates one entry's
// payload. The table name is carried in "title".
type schemaDocument struct {
	Schema               string                 `json:"$schema"`
	Title                string                 `json:"title"`
	DisplayName          string                 `json:"x-display-name,omitempty"`
	Kind                 entity.ContentTypeKind `json:"x-cms-kind,omitempty"`
	Type                 string                 `json:"type"`
	Properties           properties             `json:"properties"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

// property is one attribute's JSON Schema. Scalar attributes use their type directly. A media
// attribute is stored as a UUID string ("type": "string", "format": "uuid") marked with the
// "x-cms-type" extension keyword, so the file stays a valid JSON Schema.
type property struct {
	Title      string               `json:"title,omitempty"`
	Type       entity.AttributeType `json:"type"`
	Format     string               `json:"format,omitempty"`
	CMSType    entity.AttributeType `json:"x-cms-type,omitempty"`
	Component  string               `json:"x-component,omitempty"`
	Repeatable bool                 `json:"x-repeatable,omitempty"`
	Items      map[string]any       `json:"items,omitempty"`
}

func newProperty(a *entity.Attribute) property {
	p := property{Title: a.DisplayName}

	switch a.Type {
	case entity.AttributeTypeString, entity.AttributeTypeInteger, entity.AttributeTypeBoolean:
		p.Type = a.Type

		return p
	case entity.AttributeTypeMedia:
		p.Type = entity.AttributeTypeString
		p.Format = "uuid"
		p.CMSType = entity.AttributeTypeMedia

		return p
	case entity.AttributeTypeComponent:
		p.CMSType = entity.AttributeTypeComponent
		p.Component = a.Component
		p.Repeatable = a.Repeatable

		if a.Repeatable {
			p.Type = "array"
			p.Items = map[string]any{"type": "object"}
		} else {
			p.Type = "object"
		}

		return p
	default:
		p.Type = a.Type

		return p
	}
}

// attributeType reverses newProperty.
func (p *property) attributeType() entity.AttributeType {
	if p.CMSType != "" {
		return p.CMSType
	}

	return p.Type
}

type namedProperty struct {
	name string
	property
}

// properties is a JSON object whose key order is kept, so attribute order survives a round trip.
type properties []namedProperty

func marshalSchema(ct entity.ContentType) ([]byte, error) {
	doc := schemaDocument{
		Schema:      _jsonSchemaDialect,
		Title:       ct.TableName,
		DisplayName: ct.DisplayName,
		Kind:        ct.Kind,
		Type:        "object",
		Properties:  make(properties, 0, len(ct.Attributes)),
	}

	for i := range ct.Attributes {
		doc.Properties = append(doc.Properties, namedProperty{name: ct.Attributes[i].Name, property: newProperty(&ct.Attributes[i])})

		if ct.Attributes[i].Required {
			doc.Required = append(doc.Required, ct.Attributes[i].Name)
		}
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("json.MarshalIndent: %w", err)
	}

	return append(data, '\n'), nil
}

func unmarshalSchema(data []byte) (entity.ContentType, error) {
	var doc schemaDocument

	if err := json.Unmarshal(data, &doc); err != nil {
		return entity.ContentType{}, fmt.Errorf("%w: %w", entity.ErrInvalidContentType, err)
	}

	required := make(map[string]bool, len(doc.Required))
	for _, name := range doc.Required {
		required[name] = true
	}

	ct := entity.ContentType{
		TableName:   doc.Title,
		DisplayName: doc.DisplayName,
		Kind:        doc.Kind,
		Attributes:  make([]entity.Attribute, 0, len(doc.Properties)),
	}

	for _, p := range doc.Properties {
		ct.Attributes = append(ct.Attributes, entity.Attribute{
			Name:        p.name,
			DisplayName: p.Title,
			Type:        p.attributeType(),
			Required:    required[p.name],
			Component:   p.Component,
			Repeatable:  p.Repeatable,
		})
	}

	return ct, nil
}

// MarshalJSON -.
func (ps properties) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteByte('{')

	for i, p := range ps {
		if i > 0 {
			buf.WriteByte(',')
		}

		key, err := json.Marshal(p.name)
		if err != nil {
			return nil, fmt.Errorf("properties - MarshalJSON - key: %w", err)
		}

		value, err := json.Marshal(p.property)
		if err != nil {
			return nil, fmt.Errorf("properties - MarshalJSON - value: %w", err)
		}

		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// UnmarshalJSON -.
func (ps *properties) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("%w: properties must be an object", entity.ErrInvalidContentType)
	}

	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("properties - UnmarshalJSON - key: %w", err)
		}

		name, ok := tok.(string)
		if !ok {
			return fmt.Errorf("%w: unexpected token %v in properties", entity.ErrInvalidContentType, tok)
		}

		var p property
		if err = dec.Decode(&p); err != nil {
			return fmt.Errorf("properties - UnmarshalJSON - %q: %w", name, err)
		}

		*ps = append(*ps, namedProperty{name: name, property: p})
	}

	return nil
}
