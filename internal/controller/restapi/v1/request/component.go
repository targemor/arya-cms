package request

// CreateComponent -.
type CreateComponent struct {
	Name        string                 `json:"name"                  validate:"required,max=48"               example:"contact_data"`
	Category    string                 `json:"category,omitempty"    validate:"omitempty,max=48"              example:"shared"`
	DisplayName string                 `json:"displayName,omitempty"                                           example:"Contact Data"`
	Icon        string                 `json:"icon,omitempty"                                                  example:"phone"`
	Attributes  []ContentTypeAttribute `json:"attributes"            validate:"required,min=1,max=100,dive"`
} // @name v1.CreateComponent

// UpdateComponent -.
type UpdateComponent struct {
	DisplayName string                 `json:"displayName,omitempty"                            example:"Contact Data"`
	Icon        string                 `json:"icon,omitempty"                                   example:"phone"`
	Attributes  []ContentTypeAttribute `json:"attributes"            validate:"required,min=1,max=100,dive"`
} // @name v1.UpdateComponent
