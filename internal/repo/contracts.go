// Package repo implements application outer layer logic. Each logic group in own file.
package repo

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=../usecase/mocks_repo_test.go -package=usecase_test

type (
	// ProjectRepo stores projects (tenants).
	ProjectRepo interface {
		Create(ctx context.Context, p *entity.Project) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Project, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Project, error)
		List(ctx context.Context) ([]entity.Project, error)
		ListForUser(ctx context.Context, userID string) ([]entity.Project, error)
		Update(ctx context.Context, p *entity.Project) error
		Delete(ctx context.Context, id uuid.UUID) error
		ListMembers(ctx context.Context, projectID uuid.UUID) ([]entity.ProjectMember, error)
		AddOrUpdateMember(ctx context.Context, projectID uuid.UUID, userID string, role entity.Role) error
		RemoveMember(ctx context.Context, projectID uuid.UUID, userID string) error
		GetUserRoleInProject(ctx context.Context, projectID uuid.UUID, userID string) (entity.Role, error)
		CreateAPIKey(ctx context.Context, k *entity.APIKey) error
		ListAPIKeys(ctx context.Context, projectID uuid.UUID) ([]entity.APIKey, error)
		GetAPIKeyByHash(ctx context.Context, keyHash string) (*entity.APIKey, error)
		UpdateAPIKey(ctx context.Context, k *entity.APIKey) error
		DeleteAPIKey(ctx context.Context, projectID, keyID uuid.UUID) error
		TouchAPIKey(ctx context.Context, keyID uuid.UUID) error
	}

	// UserRepo -.
	UserRepo interface {
		Store(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, id string) (entity.User, error)
		GetByEmail(ctx context.Context, email string) (entity.User, error)
		List(ctx context.Context) ([]entity.User, error)
		UpdateRole(ctx context.Context, id string, role entity.Role) error
	}

	// ContentTypeStore persists content-type definitions as JSON Schema files.
	ContentTypeStore interface {
		Exists(ctx context.Context, projectSlug, tableName string) (bool, error)
		Save(ctx context.Context, projectSlug string, ct entity.ContentType) error
		List(ctx context.Context, projectSlug string) ([]entity.ContentType, error)
		ListProjects(ctx context.Context) ([]string, error)
	}

	// ContentTypeMigrator brings the database table for a content type in line with its definition.
	ContentTypeMigrator interface {
		Migrate(ctx context.Context, projectSlug string, ct entity.ContentType) error
	}

	// ComponentStore persists component definitions as JSON files.
	ComponentStore interface {
		Exists(ctx context.Context, projectSlug, uid string) (bool, error)
		Save(ctx context.Context, projectSlug string, c *entity.Component) error
		Get(ctx context.Context, projectSlug, uid string) (*entity.Component, error)
		List(ctx context.Context, projectSlug string) ([]entity.Component, error)
		Delete(ctx context.Context, projectSlug, uid string) error
	}

	// ContentRepo stores the entries of content types. Queries must already be resolved
	// (entity.ContentType.ResolveQuery) and data must already be validated.
	ContentRepo interface {
		List(ctx context.Context, projectSlug string, ct entity.ContentType, q entity.ContentQuery) ([]entity.Entry, int, error)
		Get(ctx context.Context, projectSlug string, ct entity.ContentType, id string) (entity.Entry, error)
		GetSingle(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.Entry, error)
		Create(ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry) (entity.Entry, error)
		Update(ctx context.Context, projectSlug string, ct entity.ContentType, id string, data entity.Entry) (entity.Entry, error)
		UpdateSingle(ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry) (entity.Entry, error)
		Delete(ctx context.Context, projectSlug string, ct entity.ContentType, id string) error
	}

	// StorageProvider stores the bytes of Media Library files. Implementations: local disk
	// (repo/storage/local); an S3 provider would be another package satisfying this interface.
	StorageProvider interface {
		// Name is what gets recorded in upload_files.provider, e.g. "local" or "aws-s3".
		Name() string
		// Upload stores content under file.ObjectName(), never overwriting an existing object,
		// and sets file.URL and file.Provider.
		Upload(ctx context.Context, file *entity.UploadFile, content io.Reader) error
		// Delete removes the stored object. Deleting an object that is already gone is not an error.
		Delete(ctx context.Context, file *entity.UploadFile) error
	}

	// UploadRepo stores Media Library metadata (the upload_files table).
	UploadRepo interface {
		Store(ctx context.Context, file *entity.UploadFile) error
		GetByID(ctx context.Context, id string) (entity.UploadFile, error)
		// List returns one page of the files directly in folderID (nil: the library root), newest
		// first, and how many files that folder holds.
		List(ctx context.Context, folderID *string, page, pageSize int) ([]entity.UploadFile, int, error)
		// CountInFolder counts the files directly in a folder.
		CountInFolder(ctx context.Context, folderID string) (int, error)
		// Move puts a file in folderID (nil: the library root) and returns it updated.
		Move(ctx context.Context, id string, folderID *string) (entity.UploadFile, error)
		// Delete removes the row and returns it, so the caller can delete the stored object.
		Delete(ctx context.Context, id string) (entity.UploadFile, error)
	}

	// FolderRepo stores Media Library folders (the upload_folders table).
	FolderRepo interface {
		// Create fills in the ID and timestamps. A name already used in the same parent is
		// ErrFolderAlreadyExists.
		Create(ctx context.Context, folder *entity.Folder) error
		GetByID(ctx context.Context, id string) (entity.Folder, error)
		// List returns every folder, sorted by name.
		List(ctx context.Context) ([]entity.Folder, error)
		// Update saves the name and parent and fills in the new UpdatedAt.
		Update(ctx context.Context, folder *entity.Folder) error
		// Delete fails with ErrFolderNotEmpty while the folder holds files or folders.
		Delete(ctx context.Context, id string) error
	}
)
