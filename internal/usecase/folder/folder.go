package folder

import (
	"context"
	"fmt"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// UseCase -.
type UseCase struct {
	folders repo.FolderRepo
	files   repo.UploadRepo
}

// New returns a Folder usecase instrumented with OpenTelemetry tracing spans.
func New(f repo.FolderRepo, u repo.UploadRepo) usecase.Folder {
	return newTraced(&UseCase{
		folders: f,
		files:   u,
	})
}

// List -.
func (uc *UseCase) List(ctx context.Context) ([]entity.Folder, error) {
	folders, err := uc.folders.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("FolderUseCase - List - uc.folders.List: %w", err)
	}

	return folders, nil
}

// Create -.
func (uc *UseCase) Create(ctx context.Context, name string, parentID *string) (entity.Folder, error) {
	name, err := entity.NormalizeFolderName(name)
	if err != nil {
		return entity.Folder{}, fmt.Errorf("FolderUseCase - Create - entity.NormalizeFolderName: %w", err)
	}

	if parentID != nil {
		if _, err = uc.folders.GetByID(ctx, *parentID); err != nil {
			return entity.Folder{}, fmt.Errorf("FolderUseCase - Create - uc.folders.GetByID: %w", err)
		}
	}

	folder := entity.Folder{Name: name, ParentID: parentID}

	if err = uc.folders.Create(ctx, &folder); err != nil {
		return entity.Folder{}, fmt.Errorf("FolderUseCase - Create - uc.folders.Create: %w", err)
	}

	return folder, nil
}

// Update -.
func (uc *UseCase) Update(ctx context.Context, id, name string, parentID *string) (entity.Folder, error) {
	folder, err := uc.folders.GetByID(ctx, id)
	if err != nil {
		return entity.Folder{}, fmt.Errorf("FolderUseCase - Update - uc.folders.GetByID: %w", err)
	}

	if folder.Name, err = entity.NormalizeFolderName(name); err != nil {
		return entity.Folder{}, fmt.Errorf("FolderUseCase - Update - entity.NormalizeFolderName: %w", err)
	}

	if parentID != nil {
		if err = uc.checkNewParent(ctx, id, *parentID); err != nil {
			return entity.Folder{}, fmt.Errorf("FolderUseCase - Update - uc.checkNewParent: %w", err)
		}
	}

	folder.ParentID = parentID

	if err = uc.folders.Update(ctx, &folder); err != nil {
		return entity.Folder{}, fmt.Errorf("FolderUseCase - Update - uc.folders.Update: %w", err)
	}

	return folder, nil
}

// Delete -.
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	if _, err := uc.folders.GetByID(ctx, id); err != nil {
		return fmt.Errorf("FolderUseCase - Delete - uc.folders.GetByID: %w", err)
	}

	folders, err := uc.folders.List(ctx)
	if err != nil {
		return fmt.Errorf("FolderUseCase - Delete - uc.folders.List: %w", err)
	}

	for i := range folders {
		if folders[i].ParentID != nil && *folders[i].ParentID == id {
			return fmt.Errorf("%w: it has subfolders", entity.ErrFolderNotEmpty)
		}
	}

	files, err := uc.files.CountInFolder(ctx, id)
	if err != nil {
		return fmt.Errorf("FolderUseCase - Delete - uc.files.CountInFolder: %w", err)
	}

	if files > 0 {
		return fmt.Errorf("%w: it has %d files", entity.ErrFolderNotEmpty, files)
	}

	if err = uc.folders.Delete(ctx, id); err != nil {
		return fmt.Errorf("FolderUseCase - Delete - uc.folders.Delete: %w", err)
	}

	return nil
}

// checkNewParent rejects a parent that does not exist, and moving folder id into itself or into
// one of its own subfolders, which would detach that branch from the tree.
func (uc *UseCase) checkNewParent(ctx context.Context, id, parentID string) error {
	folders, err := uc.folders.List(ctx)
	if err != nil {
		return fmt.Errorf("uc.folders.List: %w", err)
	}

	found := false
	for i := range folders {
		found = found || folders[i].ID == parentID
	}

	if !found {
		return entity.ErrFolderNotFound
	}

	if entity.IsDescendant(folders, parentID, id) {
		return fmt.Errorf("%w: a folder cannot move into itself or one of its subfolders", entity.ErrInvalidFolder)
	}

	return nil
}
