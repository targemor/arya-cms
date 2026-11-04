package request

// Folder creates, renames or moves a Media Library folder. A null or missing parentId is the
// library root.
type Folder struct {
	Name     string  `json:"name"     validate:"required,max=100"   example:"Portadas"`
	ParentID *string `json:"parentId" validate:"omitempty,uuid"     example:"550e8400-e29b-41d4-a716-446655440000"`
} // @name v1.FolderRequest

// MoveFile moves a Media Library file. A null or missing folderId is the library root.
type MoveFile struct {
	FolderID *string `json:"folderId" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
} // @name v1.MoveFileRequest
