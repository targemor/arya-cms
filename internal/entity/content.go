package entity

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Entry is one row of a content type, keyed by field name.
type Entry map[string]any

// System fields present on every entry. Attributes are lowercase snake_case, so these camelCase
// names can never collide with one.
const (
	FieldID        = "id"
	FieldCreatedAt = "createdAt"
	FieldUpdatedAt = "updatedAt"
)

// FieldType is the type of any queryable field: an attribute or a system field.
type FieldType string

// Field types.
const (
	FieldTypeString   FieldType = "string"
	FieldTypeInteger  FieldType = "integer"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeUUID     FieldType = "uuid"
	FieldTypeDatetime FieldType = "datetime"
	// FieldTypeMedia is the id of a Media Library file, or null.
	FieldTypeMedia FieldType = "media"
)

// FieldType returns the type of a system field or attribute, and false if ct has no such field.
func (ct ContentType) FieldType(name string) (FieldType, bool) {
	switch name {
	case FieldID:
		return FieldTypeUUID, true
	case FieldCreatedAt, FieldUpdatedAt:
		return FieldTypeDatetime, true
	}

	for _, a := range ct.Attributes {
		if a.Name == name {
			return a.Type.fieldType(), true
		}
	}

	return "", false
}

func (t AttributeType) fieldType() FieldType {
	switch t {
	case AttributeTypeInteger:
		return FieldTypeInteger
	case AttributeTypeBoolean:
		return FieldTypeBoolean
	case AttributeTypeMedia:
		return FieldTypeMedia
	case AttributeTypeComponent:
		return FieldTypeString
	case AttributeTypeString:
		return FieldTypeString
	default:
		return FieldTypeString
	}
}

// parseValue converts a raw query-string value to the Go type the field is stored as.
func (t FieldType) parseValue(raw string) (any, error) {
	switch t {
	case FieldTypeString:
		return raw, nil
	case FieldTypeUUID, FieldTypeMedia:
		return parseUUID(raw)
	case FieldTypeInteger:
		return parseInteger(raw)
	case FieldTypeBoolean:
		return parseBoolean(raw)
	case FieldTypeDatetime:
		return parseDatetime(raw)
	default:
		return nil, fmt.Errorf("%w: unsupported field type %q", ErrInvalidQuery, t)
	}
}

func parseUUID(raw string) (any, error) {
	if err := uuid.Validate(raw); err != nil {
		return nil, fmt.Errorf("%w: %q is not a UUID", ErrInvalidQuery, raw)
	}

	return raw, nil
}

func parseInteger(raw string) (any, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not an integer", ErrInvalidQuery, raw)
	}

	return v, nil
}

func parseBoolean(raw string) (any, error) {
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not a boolean", ErrInvalidQuery, raw)
	}

	return v, nil
}

func parseDatetime(raw string) (any, error) {
	v, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not an RFC 3339 datetime", ErrInvalidQuery, raw)
	}

	return v, nil
}
