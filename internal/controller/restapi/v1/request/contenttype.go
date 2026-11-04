package request

// CreateContentType -.
type CreateContentType struct {
	TableName   string                 `json:"tableName"             validate:"required,max=48"                      example:"articles"`
	DisplayName string                 `json:"displayName,omitempty"                                                  example:"Articles"`
	Kind        string                 `json:"kind,omitempty"        validate:"omitempty,oneof=collectionType singleType" example:"collectionType"`
	Attributes  []ContentTypeAttribute `json:"attributes"            validate:"required,min=1,max=100,dive"`
} // @name v1.CreateContentType

// UpdateContentType -.
type UpdateContentType struct {
	DisplayName string                 `json:"displayName,omitempty"                                                  example:"Articles"`
	Kind        string                 `json:"kind,omitempty"        validate:"omitempty,oneof=collectionType singleType" example:"collectionType"`
	Attributes  []ContentTypeAttribute `json:"attributes"            validate:"required,min=1,max=100,dive"`
} // @name v1.UpdateContentType

// ContentTypeAttribute -.
type ContentTypeAttribute struct {
	Name        string `json:"name"                  validate:"required,max=48"                                      example:"title"`
	DisplayName string `json:"displayName,omitempty" validate:"omitempty,max=100"                                    example:"Title"`
	Type        string `json:"type"                  validate:"required,oneof=string integer boolean media component" example:"string"`
	Required    bool   `json:"required"                                                                              example:"true"`
	Component   string `json:"component,omitempty"   validate:"omitempty,max=96"                                     example:"shared.contact_data"`
	Repeatable  bool   `json:"repeatable,omitempty"                                                                  example:"false"`
} // @name v1.ContentTypeAttribute
