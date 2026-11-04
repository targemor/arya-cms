package request

import "github.com/lucimux/arya/internal/entity"

// CreateProject -.
type CreateProject struct {
	Name        string `json:"name"                  validate:"required,min=1,max=255" example:"My Shop"`
	Slug        string `json:"slug"                  validate:"required,min=1,max=64"  example:"my_shop"`
	Description string `json:"description,omitempty"                                    example:"Ecommerce project"`
	ImageURL    string `json:"imageUrl,omitempty"                                       example:"/uploads/2026/11/logo.png"`
} // @name v1.CreateProject

// UpdateProject -.
type UpdateProject struct {
	Name        string `json:"name"                  validate:"required,min=1,max=255" example:"My Shop Renamed"`
	Description string `json:"description,omitempty"                                    example:"Updated description"`
	ImageURL    string `json:"imageUrl,omitempty"                                       example:"/uploads/2026/11/logo.png"`
} // @name v1.UpdateProject

// AddProjectMember -.
type AddProjectMember struct {
	UserID string      `json:"userId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role   entity.Role `json:"role"   validate:"required"      example:"editor"`
} // @name v1.AddProjectMember

// CreateAPIKey -.
type CreateAPIKey struct {
	Name        string                           `json:"name"                  validate:"required,min=1,max=255" example:"Frontend Web"`
	Description string                           `json:"description,omitempty"                                    example:"Read access for Next.js website"`
	AccessType  entity.APIKeyAccessType          `json:"accessType"                                               example:"custom"`
	Permissions map[string][]entity.APIKeyAction `json:"permissions"           swaggertype:"object"`
} // @name v1.CreateAPIKey

// UpdateAPIKey -.
type UpdateAPIKey struct {
	Name        string                           `json:"name"                  validate:"required,min=1,max=255" example:"Frontend Web Updated"`
	Description string                           `json:"description,omitempty"                                    example:"Updated permissions"`
	AccessType  entity.APIKeyAccessType          `json:"accessType"                                               example:"custom"`
	Permissions map[string][]entity.APIKeyAction `json:"permissions"           swaggertype:"object"`
} // @name v1.UpdateAPIKey
