package entity

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// _maxFolderNameLen matches upload_folders.name VARCHAR(100), in characters.
const _maxFolderNameLen = 100

// Folder groups Media Library files. Folders nest; a nil ParentID is the library root.
type Folder struct {
	ID        string    `json:"id"        example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name"      example:"Portadas"`
	ParentID  *string   `json:"parentId"  example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	CreatedAt time.Time `json:"createdAt" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" example:"2026-01-01T00:00:00Z"`
} // @name entity.Folder

// NormalizeFolderName trims the name and checks it: 1 to 100 characters of valid UTF-8, without
// slashes (they would read as a path) or control characters. Errors wrap ErrInvalidFolder.
func NormalizeFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > _maxFolderNameLen {
		return "", fmt.Errorf("%w: name %q", ErrInvalidFolder, name)
	}

	if strings.ContainsAny(name, `/\`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: name %q has a slash or a control character", ErrInvalidFolder, name)
	}

	return name, nil
}

// IsDescendant reports whether folder id sits somewhere below ancestor in folders (the whole
// library). A folder counts as its own descendant, so moving a folder into ancestor would loop
// exactly when IsDescendant(folders, ancestor, folder) is true.
func IsDescendant(folders []Folder, id, ancestor string) bool {
	parentOf := make(map[string]*string, len(folders))
	for i := range folders {
		parentOf[folders[i].ID] = folders[i].ParentID
	}

	// Each step goes one level up; the bound stops a corrupted, cyclic parent chain.
	for current, steps := &id, 0; current != nil && steps <= len(folders); steps++ {
		if *current == ancestor {
			return true
		}

		current = parentOf[*current]
	}

	return false
}
