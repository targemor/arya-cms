package entity

import "time"

// Role represents a user's permission level.
type Role string

const (
	RoleSuperAdmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleEditor     Role = "editor"
	RoleViewer     Role = "viewer"
)

// IsValid checks if role is one of the recognized roles.
func (r Role) IsValid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// User -.
type User struct {
	ID           string    `json:"id"           example:"550e8400-e29b-41d4-a716-446655440000"`
	Username     string    `json:"username"     example:"johndoe"`
	Email        string    `json:"email"        example:"john@example.com"`
	Role         Role      `json:"role"         example:"superadmin"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"   example:"2026-01-01T00:00:00Z"`
	UpdatedAt    time.Time `json:"updated_at"   example:"2026-01-01T00:00:00Z"`
} // @name entity.User

// IsSuperAdmin returns true if the user has the superadmin role.
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}
