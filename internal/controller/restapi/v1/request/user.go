package request

// Register -.
type Register struct {
	Username string `json:"username"       validate:"required,min=3,max=255" example:"johndoe"`
	Email    string `json:"email"          validate:"required,email"         example:"john@example.com"`
	Password string `json:"password"       validate:"required,min=6"         example:"secret123"`
	Role     string `json:"role,omitempty"                                   example:"admin"`
} // @name v1.Register

// Login -.
type Login struct {
	Email    string `json:"email"    validate:"required,email" example:"john@example.com"`
	Password string `json:"password" validate:"required"       example:"secret123"`
} // @name v1.Login

// UpdateUserRole -.
type UpdateUserRole struct {
	Role string `json:"role" validate:"required" example:"superadmin"`
} // @name v1.UpdateUserRole
