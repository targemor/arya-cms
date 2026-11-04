package upload

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// _hashBytes of randomness make the stored name unguessable: files are served publicly by URL.
const _hashBytes = 16

// UseCase -.
type UseCase struct {
	repo    repo.UploadRepo
	folders repo.FolderRepo
	storage repo.StorageProvider
	maxSize int64
}

// New returns an Upload usecase instrumented with OpenTelemetry tracing spans. maxSize is in bytes.
func New(r repo.UploadRepo, f repo.FolderRepo, s repo.StorageProvider, maxSize int64) usecase.Upload {
	return newTraced(&UseCase{
		repo:    r,
		folders: f,
		storage: s,
		maxSize: maxSize,
	})
}

// Upload stores the bytes first and the metadata second. If the metadata cannot be saved the stored
// object is removed again, so no file is left on disk without a row pointing at it.
func (uc *UseCase) Upload(ctx context.Context, in usecase.UploadInput, content io.Reader) (entity.UploadFile, error) {
	file, err := uc.newFile(in.Name, in.AlternativeText, in.Size)
	if err != nil {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Upload - uc.newFile: %w", err)
	}

	// Check the folder before storing anything, so a bad folder leaves nothing behind.
	if err = uc.requireFolder(ctx, in.FolderID); err != nil {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Upload - uc.requireFolder: %w", err)
	}

	file.FolderID = in.FolderID

	// Peek at the start of the content to detect its type without consuming it.
	br := bufio.NewReaderSize(content, _sniffLen)

	head, err := br.Peek(_sniffLen)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Upload - br.Peek: %w", err)
	}

	file.Mime = detectMime(file.Ext, head)

	if err = uc.storage.Upload(ctx, &file, br); err != nil {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Upload - uc.storage.Upload: %w", err)
	}

	if err = uc.repo.Store(ctx, &file); err != nil {
		return entity.UploadFile{}, errors.Join(
			fmt.Errorf("UploadUseCase - Upload - uc.repo.Store: %w", err),
			uc.storage.Delete(ctx, &file),
		)
	}

	return file, nil
}

// List -.
func (uc *UseCase) List(
	ctx context.Context, folderID *string, page, pageSize int,
) ([]entity.UploadFile, entity.Pagination, error) {
	if page < 1 || pageSize < 1 || pageSize > entity.MaxPageSize {
		return nil, entity.Pagination{}, fmt.Errorf("%w: page %d, page size %d", entity.ErrInvalidQuery, page, pageSize)
	}

	// An unknown folder is "not found", not an empty listing.
	if err := uc.requireFolder(ctx, folderID); err != nil {
		return nil, entity.Pagination{}, fmt.Errorf("UploadUseCase - List - uc.requireFolder: %w", err)
	}

	files, total, err := uc.repo.List(ctx, folderID, page, pageSize)
	if err != nil {
		return nil, entity.Pagination{}, fmt.Errorf("UploadUseCase - List - uc.repo.List: %w", err)
	}

	return files, entity.NewPagination(page, pageSize, total), nil
}

// Move -.
func (uc *UseCase) Move(ctx context.Context, id string, folderID *string) (entity.UploadFile, error) {
	if err := uc.requireFolder(ctx, folderID); err != nil {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Move - uc.requireFolder: %w", err)
	}

	file, err := uc.repo.Move(ctx, id, folderID)
	if err != nil {
		return entity.UploadFile{}, fmt.Errorf("UploadUseCase - Move - uc.repo.Move: %w", err)
	}

	return file, nil
}

// requireFolder checks that a folder exists; nil, the library root, always does.
func (uc *UseCase) requireFolder(ctx context.Context, folderID *string) error {
	if folderID == nil {
		return nil
	}

	if _, err := uc.folders.GetByID(ctx, *folderID); err != nil {
		return fmt.Errorf("uc.folders.GetByID: %w", err)
	}

	return nil
}

// Delete removes the row first and the stored object second: if the second step fails, what is left
// is an orphaned object, never a row pointing at a missing file.
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	file, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("UploadUseCase - Delete - uc.repo.GetByID: %w", err)
	}

	// A file stored by another provider (e.g. before switching to S3) cannot be deleted by this one.
	if file.Provider != uc.storage.Name() {
		return fmt.Errorf("%w: file %s is stored by %q", entity.ErrStorageProviderAbsent, id, file.Provider)
	}

	if file, err = uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("UploadUseCase - Delete - uc.repo.Delete: %w", err)
	}

	if err = uc.storage.Delete(ctx, &file); err != nil {
		return fmt.Errorf("UploadUseCase - Delete - uc.storage.Delete: %w", err)
	}

	return nil
}

func (uc *UseCase) newFile(name string, alternativeText *string, size int64) (entity.UploadFile, error) {
	name = baseName(name)

	if err := uc.validate(name, alternativeText, size); err != nil {
		return entity.UploadFile{}, err
	}

	hash, err := newHash()
	if err != nil {
		return entity.UploadFile{}, err
	}

	if alternativeText != nil && *alternativeText == "" {
		alternativeText = nil
	}

	return entity.UploadFile{
		Name:            name,
		AlternativeText: alternativeText,
		Hash:            hash,
		Ext:             normalizeExt(name),
		Size:            size,
	}, nil
}

func (uc *UseCase) validate(name string, alternativeText *string, size int64) error {
	if name == "" || len(name) > _maxNameLen || !utf8.ValidString(name) {
		return fmt.Errorf("%w: file name %q", entity.ErrInvalidUpload, name)
	}

	// Postgres rejects invalid UTF-8; catch it here as a client error instead of a failed insert.
	if alternativeText != nil && !utf8.ValidString(*alternativeText) {
		return fmt.Errorf("%w: alternative text is not valid UTF-8", entity.ErrInvalidUpload)
	}

	if size <= 0 {
		return fmt.Errorf("%w: empty file", entity.ErrInvalidUpload)
	}

	if size > uc.maxSize {
		return fmt.Errorf("%w: %d bytes, limit %d", entity.ErrUploadTooLarge, size, uc.maxSize)
	}

	return nil
}

func newHash() (string, error) {
	b := make([]byte, _hashBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}

	return hex.EncodeToString(b), nil
}
