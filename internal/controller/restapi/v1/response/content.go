package response

import "github.com/lucimux/arya/internal/entity"

// ContentEntry -.
type ContentEntry struct {
	Data entity.Entry `json:"data"`
} // @name v1.ContentEntry

// ContentEntries -.
type ContentEntries struct {
	Data []entity.Entry `json:"data"`
	Meta ContentMeta    `json:"meta"`
} // @name v1.ContentEntries

// UploadFiles -.
type UploadFiles struct {
	Data []entity.UploadFile `json:"data"`
	Meta ContentMeta         `json:"meta"`
} // @name v1.UploadFiles

// Folders -.
type Folders struct {
	Data []entity.Folder `json:"data"`
} // @name v1.Folders

// ContentMeta -.
type ContentMeta struct {
	Pagination entity.Pagination `json:"pagination"`
} // @name v1.ContentMeta
