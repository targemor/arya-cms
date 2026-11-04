package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	_maxProjectNameLen = 255
)

// Project is a tenant project within the CMS.
type Project struct {
	ID          uuid.UUID `json:"id"                    example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name"                  example:"My Shop"`
	Slug        string    `json:"slug"                  example:"my_shop"`
	Description string    `json:"description,omitempty" example:"Ecommerce project"`
	ImageURL    string    `json:"imageUrl,omitempty"    example:"/uploads/2026/11/logo.png"`
	CreatedAt   time.Time `json:"createdAt"             example:"2026-11-01T12:00:00Z"`
	UpdatedAt   time.Time `json:"updatedAt"             example:"2026-11-01T12:00:00Z"`
} // @name entity.Project

// Validate ensures name is non-empty and slug is a lowercase snake_case identifier.
func (p *Project) Validate() error {
	if p.Name == "" || len(p.Name) > _maxProjectNameLen {
		return fmt.Errorf("%w: invalid project name", ErrInvalidProject)
	}

	if !isIdentifier(p.Slug) {
		return fmt.Errorf("%w: invalid project slug %q", ErrInvalidProject, p.Slug)
	}

	return nil
}

// ProjectMember associates a user with a project and a specific role within that project.
type ProjectMember struct {
	ProjectID uuid.UUID `json:"projectId" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string    `json:"userId"    example:"550e8400-e29b-41d4-a716-446655440001"`
	Username  string    `json:"username"  example:"johndoe"`
	Email     string    `json:"email"     example:"john@example.com"`
	Role      Role      `json:"role"      example:"editor"`
	CreatedAt time.Time `json:"createdAt" example:"2026-11-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" example:"2026-11-01T12:00:00Z"`
} // @name entity.ProjectMember
