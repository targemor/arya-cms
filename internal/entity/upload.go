package entity

import (
	"fmt"
	"time"
)

const (
	_maxHashLen = 64
	_maxExtLen  = 16
)

// UploadFile is a Media Library file: its metadata, and where its storage provider put the bytes.
type UploadFile struct {
	ID              string  `json:"id"              example:"550e8400-e29b-41d4-a716-446655440000"`
	Name            string  `json:"name"            example:"cover.png"`
	AlternativeText *string `json:"alternativeText" example:"Book cover"`
	Hash            string  `json:"hash"            example:"9f86d081884c7d659a2feaa0c55ad015"`
	Ext             string  `json:"ext"             example:".png"`
	Mime            string  `json:"mime"            example:"image/png"`
	Size            int64   `json:"size"            example:"52341"`
	URL             string  `json:"url"             example:"/uploads/9f86d081884c7d659a2feaa0c55ad015.png"`
	Provider        string  `json:"provider"        example:"local"`
	// FolderID is the Media Library folder holding the file; nil is the library root.
	FolderID  *string   `json:"folderId"        example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt time.Time `json:"createdAt"       example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt"       example:"2026-01-01T00:00:00Z"`
} // @name entity.UploadFile

// ObjectName is the name the file is stored under: its hash plus its extension. Providers use it
// as a file name or object key, so it is checked to be a single safe path segment: the hash is
// letters, digits, '-' and '_'; the extension is empty or a dot followed by lowercase letters and digits.
func (f *UploadFile) ObjectName() (string, error) {
	if !isSafeHash(f.Hash) {
		return "", fmt.Errorf("%w: hash %q", ErrInvalidUpload, f.Hash)
	}

	if !isSafeExt(f.Ext) {
		return "", fmt.Errorf("%w: extension %q", ErrInvalidUpload, f.Ext)
	}

	return f.Hash + f.Ext, nil
}

func isSafeHash(s string) bool {
	if s == "" || len(s) > _maxHashLen {
		return false
	}

	for _, c := range s {
		if !isLower(c) && !isDigit(c) && (c < 'A' || c > 'Z') && c != '-' && c != '_' {
			return false
		}
	}

	return true
}

func isSafeExt(s string) bool {
	if s == "" {
		return true
	}

	if s[0] != '.' || len(s) < 2 || len(s) > _maxExtLen {
		return false
	}

	for _, c := range s[1:] {
		if !isLower(c) && !isDigit(c) {
			return false
		}
	}

	return true
}
