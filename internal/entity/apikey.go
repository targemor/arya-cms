package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	_maxAPIKeyNameLen = 255
)

// APIKeyAction represents a granular CRUD operation on a content type.
type APIKeyAction string

const (
	APIKeyActionFind    APIKeyAction = "find"
	APIKeyActionFindOne APIKeyAction = "findOne"
	APIKeyActionCreate  APIKeyAction = "create"
	APIKeyActionUpdate  APIKeyAction = "update"
	APIKeyActionDelete  APIKeyAction = "delete"
)

// IsValid reports whether action is a recognized API key action.
func (a APIKeyAction) IsValid() bool {
	switch a {
	case APIKeyActionFind, APIKeyActionFindOne, APIKeyActionCreate, APIKeyActionUpdate, APIKeyActionDelete:
		return true
	default:
		return false
	}
}

// APIKeyAccessType represents the preset or custom mode of an API key.
type APIKeyAccessType string

const (
	APIKeyAccessCustom   APIKeyAccessType = "custom"
	APIKeyAccessReadOnly APIKeyAccessType = "read-only"
	APIKeyAccessFull     APIKeyAccessType = "full-access"
)

// IsValid reports whether accessType is a recognized access type.
func (t APIKeyAccessType) IsValid() bool {
	switch t {
	case APIKeyAccessCustom, APIKeyAccessReadOnly, APIKeyAccessFull:
		return true
	default:
		return false
	}
}

// APIKey represents a project-scoped API key with granular content-type permissions.
type APIKey struct {
	ID          uuid.UUID                 `json:"id"                    example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID   uuid.UUID                 `json:"projectId"             example:"550e8400-e29b-41d4-a716-446655440001"`
	ProjectSlug string                    `json:"projectSlug,omitempty" example:"default"`
	Name        string                    `json:"name"                  example:"Frontend Web"`
	Description string                    `json:"description,omitempty" example:"Read access for Next.js website"`
	KeyPrefix   string                    `json:"keyPrefix"             example:"cms_8f9d2a1b"`
	KeyHash     string                    `json:"-"`
	Token       string                    `json:"token,omitempty"       example:"cms_8f9d2a1b3c4d5e6f..."`
	AccessType  APIKeyAccessType          `json:"accessType"            example:"custom"`
	Permissions map[string][]APIKeyAction `json:"permissions"           swaggertype:"object"`
	LastUsedAt  *time.Time                `json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time                 `json:"createdAt"             example:"2026-11-01T12:00:00Z"`
	UpdatedAt   time.Time                 `json:"updatedAt"             example:"2026-11-01T12:00:00Z"`
} // @name entity.APIKey

// Validate checks that name, accessType, and all granular permissions are valid.
func (k *APIKey) Validate() error {
	if k.Name == "" || len(k.Name) > _maxAPIKeyNameLen {
		return fmt.Errorf("%w: invalid api key name", ErrInvalidAPIKey)
	}

	if k.AccessType == "" {
		k.AccessType = APIKeyAccessCustom
	}

	if !k.AccessType.IsValid() {
		return fmt.Errorf("%w: invalid access type %q", ErrInvalidAPIKey, k.AccessType)
	}

	if k.Permissions == nil {
		k.Permissions = make(map[string][]APIKeyAction)
	}

	for tableName, actions := range k.Permissions {
		if !isIdentifier(tableName) {
			return fmt.Errorf("%w: invalid content type %q in permissions", ErrInvalidAPIKey, tableName)
		}

		for _, action := range actions {
			if !action.IsValid() {
				return fmt.Errorf("%w: invalid action %q for %q", ErrInvalidAPIKey, action, tableName)
			}
		}
	}

	return nil
}

// CanPerform checks if the API key grants the specified action on the given content type table.
func (k *APIKey) CanPerform(tableName string, action APIKeyAction) bool {
	switch k.AccessType {
	case APIKeyAccessFull:
		return true
	case APIKeyAccessReadOnly:
		return action == APIKeyActionFind || action == APIKeyActionFindOne
	case APIKeyAccessCustom:
		if k.Permissions == nil {
			return false
		}

		for _, allowed := range k.Permissions[tableName] {
			if allowed == action {
				return true
			}
		}

		return false
	default:
		return false
	}
}
