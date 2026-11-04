package entity

import (
	"fmt"
	"strings"
)

const (
	// _maxIdentifierLen keeps table and column names well under Postgres' 63-byte limit,
	// leaving room for the prefix the repository adds to physical table names.
	_maxIdentifierLen = 48
	_maxAttributes    = 100
	_maxUIDSegments   = 2
)

// ContentTypeKind is the kind of content type: collection type or single type.
type ContentTypeKind string

// Supported content-type kinds.
const (
	ContentTypeKindCollection ContentTypeKind = "collectionType"
	ContentTypeKindSingle     ContentTypeKind = "singleType"
)

// AttributeType is the data type of a content-type attribute. The scalar ones match JSON Schema types.
type AttributeType string

// Supported attribute types.
const (
	AttributeTypeString  AttributeType = "string"
	AttributeTypeInteger AttributeType = "integer"
	AttributeTypeBoolean AttributeType = "boolean"
	// AttributeTypeMedia references a Media Library file (upload_files) by id.
	AttributeTypeMedia AttributeType = "media"
	// AttributeTypeComponent embeds a reusable component structure.
	AttributeTypeComponent AttributeType = "component"
)

// Attribute -.
type Attribute struct {
	Name        string        `json:"name"                  example:"title"`
	DisplayName string        `json:"displayName,omitempty" example:"Title"`
	Type        AttributeType `json:"type"                  example:"string"`
	Required    bool          `json:"required"              example:"true"`
	Component   string        `json:"component,omitempty"   example:"shared.contact_data"`
	Repeatable  bool          `json:"repeatable,omitempty"  example:"false"`
} // @name entity.Attribute

// ContentType is a user-defined collection type or single type, in the style of Strapi's Content-Type Builder.
type ContentType struct {
	TableName   string          `json:"tableName"             example:"articles"`
	DisplayName string          `json:"displayName,omitempty" example:"Articles"`
	Kind        ContentTypeKind `json:"kind,omitempty"        example:"collectionType"`
	Attributes  []Attribute     `json:"attributes"`
} // @name entity.ContentType

// GetKind returns ct.Kind, defaulting to ContentTypeKindCollection when empty.
func (ct ContentType) GetKind() ContentTypeKind {
	if ct.Kind == "" {
		return ContentTypeKindCollection
	}

	return ct.Kind
}

// GetDisplayName returns ct.DisplayName, defaulting to ct.TableName when empty.
func (ct ContentType) GetDisplayName() string {
	if ct.DisplayName == "" {
		return ct.TableName
	}

	return ct.DisplayName
}

// GetDisplayName returns a.DisplayName, defaulting to a.Name when empty.
func (a *Attribute) GetDisplayName() string {
	if a == nil || a.DisplayName == "" {
		if a != nil {
			return a.Name
		}

		return ""
	}

	return a.DisplayName
}

// Attribute returns the attribute with the given name, and false if ct has no such attribute.
func (ct ContentType) Attribute(name string) (Attribute, bool) {
	for _, a := range ct.Attributes {
		if a.Name == name {
			return a, true
		}
	}

	return Attribute{}, false
}

// Validate checks the definition is safe to turn into a table. Every error wraps ErrInvalidContentType.
func (ct ContentType) Validate() error {
	if !isIdentifier(ct.TableName) {
		return fmt.Errorf("%w: table name %q", ErrInvalidContentType, ct.TableName)
	}

	if ct.Kind != "" && ct.Kind != ContentTypeKindCollection && ct.Kind != ContentTypeKindSingle {
		return fmt.Errorf("%w: invalid kind %q", ErrInvalidContentType, ct.Kind)
	}

	if len(ct.Attributes) == 0 || len(ct.Attributes) > _maxAttributes {
		return fmt.Errorf("%w: %d attributes", ErrInvalidContentType, len(ct.Attributes))
	}

	seen := make(map[string]struct{}, len(ct.Attributes))

	for i := range ct.Attributes {
		if err := ct.Attributes[i].validate(); err != nil {
			return err
		}

		if _, dup := seen[ct.Attributes[i].Name]; dup {
			return fmt.Errorf("%w: duplicate attribute %q", ErrInvalidContentType, ct.Attributes[i].Name)
		}

		seen[ct.Attributes[i].Name] = struct{}{}
	}

	return nil
}

func (a *Attribute) validate() error {
	if !isIdentifier(a.Name) {
		return fmt.Errorf("%w: attribute name %q", ErrInvalidContentType, a.Name)
	}

	// Every content table gets these system columns.
	switch a.Name {
	case "id", "created_at", "updated_at":
		return fmt.Errorf("%w: attribute name %q is reserved", ErrInvalidContentType, a.Name)
	}

	switch a.Type {
	case AttributeTypeString, AttributeTypeInteger, AttributeTypeBoolean, AttributeTypeMedia:
		return nil
	case AttributeTypeComponent:
		if !isComponentUID(a.Component) {
			return fmt.Errorf("%w: component attribute %q has invalid component %q", ErrInvalidContentType, a.Name, a.Component)
		}

		return nil
	default:
		return fmt.Errorf("%w: attribute %q has unsupported type %q", ErrInvalidContentType, a.Name, a.Type)
	}
}

// isIdentifier reports whether s is a lowercase snake_case name: a letter followed by letters,
// digits or underscores. Names end up in DDL, so nothing outside this set is accepted.
func isIdentifier(s string) bool {
	if s == "" || len(s) > _maxIdentifierLen || !isLower(rune(s[0])) {
		return false
	}

	for _, c := range s {
		if !isLower(c) && !isDigit(c) && c != '_' {
			return false
		}
	}

	return true
}

func isComponentUID(s string) bool {
	if s == "" || len(s) > _maxIdentifierLen*2 {
		return false
	}

	parts := strings.Split(s, ".")
	if len(parts) > _maxUIDSegments {
		return false
	}

	for _, p := range parts {
		if !isIdentifier(p) {
			return false
		}
	}

	return true
}

func isLower(c rune) bool { return c >= 'a' && c <= 'z' }

func isDigit(c rune) bool { return c >= '0' && c <= '9' }
