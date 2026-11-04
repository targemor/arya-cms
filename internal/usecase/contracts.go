// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"
	"io"

	"github.com/lucimux/arya/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=./mocks_usecase_test.go -package=usecase_test

type (
	// Project manages tenant projects.
	Project interface {
		Create(ctx context.Context, name, slug, description, imageURL string) (entity.Project, error)
		GetBySlug(ctx context.Context, slug string) (entity.Project, error)
		List(ctx context.Context) ([]entity.Project, error)
		ListForUser(ctx context.Context, userID string, isSuperAdmin bool) ([]entity.Project, error)
		Update(ctx context.Context, slug, name, description, imageURL string) (entity.Project, error)
		Delete(ctx context.Context, slug string) error
		ListMembers(ctx context.Context, slug string) ([]entity.ProjectMember, error)
		AddMember(ctx context.Context, slug, userID string, role entity.Role) error
		RemoveMember(ctx context.Context, slug, userID string) error
		CreateAPIKey(ctx context.Context, slug, name, description string, accessType entity.APIKeyAccessType, permissions map[string][]entity.APIKeyAction) (entity.APIKey, error)
		ListAPIKeys(ctx context.Context, slug string) ([]entity.APIKey, error)
		UpdateAPIKey(ctx context.Context, slug, keyID, name, description string, accessType entity.APIKeyAccessType, permissions map[string][]entity.APIKeyAction) (entity.APIKey, error)
		DeleteAPIKey(ctx context.Context, slug, keyID string) error
		ValidateAPIKey(ctx context.Context, rawKey string) (entity.APIKey, error)
	}

	// User -.
	User interface {
		Register(ctx context.Context, username, email, password string, role entity.Role) (entity.User, error)
		Login(ctx context.Context, email, password string) (string, error)
		GetUser(ctx context.Context, userID string) (entity.User, error)
		List(ctx context.Context) ([]entity.User, error)
		UpdateRole(ctx context.Context, id string, role entity.Role) error
	}

	// ContentType -.
	ContentType interface {
		Create(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error)
		Update(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error)
		List(ctx context.Context, projectSlug string) ([]entity.ContentType, error)
		// Get returns a content type whose table is ready to serve, from memory.
		Get(ctx context.Context, projectSlug, tableName string) (entity.ContentType, error)
		// Sync applies every stored definition to the database. It keeps going past
		// failures and returns them joined.
		Sync(ctx context.Context) error
	}

	// Component manages reusable component definitions.
	Component interface {
		Create(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error)
		Update(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error)
		List(ctx context.Context, projectSlug string) ([]entity.Component, error)
		Get(ctx context.Context, projectSlug, uid string) (*entity.Component, error)
		Delete(ctx context.Context, projectSlug, uid string) error
	}

	// Upload is the Media Library's files.
	Upload interface {
		// Upload stores content as the file described by in.
		Upload(ctx context.Context, in UploadInput, content io.Reader) (entity.UploadFile, error)
		// List pages through the files directly in folderID (nil: the library root).
		List(ctx context.Context, folderID *string, page, pageSize int) ([]entity.UploadFile, entity.Pagination, error)
		// Move puts a file in folderID (nil: the library root).
		Move(ctx context.Context, id string, folderID *string) (entity.UploadFile, error)
		Delete(ctx context.Context, id string) error
	}

	// UploadInput describes a file being uploaded; the bytes are passed alongside.
	UploadInput struct {
		// Name is the client's original file name.
		Name string
		// AlternativeText may be nil.
		AlternativeText *string
		// Size is the content length in bytes.
		Size int64
		// FolderID is where the file goes; nil is the library root.
		FolderID *string
	}

	// Folder is the Media Library's folders.
	Folder interface {
		// List returns every folder, flat, sorted by name; ParentID links them into a tree.
		List(ctx context.Context) ([]entity.Folder, error)
		Create(ctx context.Context, name string, parentID *string) (entity.Folder, error)
		// Update renames and/or moves a folder. A folder cannot move into itself or below itself.
		Update(ctx context.Context, id, name string, parentID *string) (entity.Folder, error)
		// Delete removes an empty folder; one that still holds files or folders is ErrFolderNotEmpty.
		Delete(ctx context.Context, id string) error
	}

	// Content is the CRUD API over the entries of a content type.
	Content interface {
		List(ctx context.Context, projectSlug, tableName string, q entity.ContentQuery) ([]entity.Entry, entity.Pagination, error)
		Get(ctx context.Context, projectSlug, tableName, id string) (entity.Entry, error)
		GetSingle(ctx context.Context, projectSlug, tableName string) (entity.Entry, error)
		Create(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error)
		Update(ctx context.Context, projectSlug, tableName, id string, data entity.Entry) (entity.Entry, error)
		UpdateSingle(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error)
		Delete(ctx context.Context, projectSlug, tableName, id string) error
	}
)
