package usecase_test

import (
	"context"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/folder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Folder ids for a small tree:  photos/ ─ covers/ ─ 2026/   and   docs/
const (
	_photosID = "11111111-1111-4111-8111-111111111111"
	_coversID = "22222222-2222-4222-8222-222222222222"
	_yearID   = "33333333-3333-4333-8333-333333333333"
	_docsID   = "44444444-4444-4444-8444-444444444444"
)

func folderTree() []entity.Folder {
	return []entity.Folder{
		{ID: _photosID, Name: "photos"},
		{ID: _coversID, Name: "covers", ParentID: new(_photosID)},
		{ID: _yearID, Name: "2026", ParentID: new(_coversID)},
		{ID: _docsID, Name: "docs"},
	}
}

func folderByID(id string) entity.Folder {
	for _, f := range folderTree() {
		if f.ID == id {
			return f
		}
	}

	return entity.Folder{}
}

func newFolderUseCase(t *testing.T) (usecase.Folder, *MockFolderRepo, *MockUploadRepo) {
	t.Helper()

	ctrl := gomock.NewController(t)
	folders := NewMockFolderRepo(ctrl)
	files := NewMockUploadRepo(ctrl)

	return folder.New(folders, files), folders, files
}

func TestFolderCreate(t *testing.T) {
	t.Parallel()

	t.Run("trims the name and stores it under the parent", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _photosID).Return(folderByID(_photosID), nil)
		folders.EXPECT().Create(gomock.Any(), &entity.Folder{Name: "Portadas", ParentID: new(_photosID)}).
			DoAndReturn(func(_ context.Context, f *entity.Folder) error {
				f.ID = _coversID

				return nil
			})

		got, err := uc.Create(context.Background(), "  Portadas ", new(_photosID))

		require.NoError(t, err)
		assert.Equal(t, _coversID, got.ID)
		assert.Equal(t, "Portadas", got.Name)
	})

	t.Run("invalid name touches nothing", func(t *testing.T) {
		t.Parallel()

		uc, _, _ := newFolderUseCase(t)

		_, err := uc.Create(context.Background(), "a/b", nil)

		require.ErrorIs(t, err, entity.ErrInvalidFolder)
	})

	t.Run("unknown parent", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _docsID).Return(entity.Folder{}, entity.ErrFolderNotFound)

		_, err := uc.Create(context.Background(), "x", new(_docsID))

		require.ErrorIs(t, err, entity.ErrFolderNotFound)
	})

	t.Run("duplicate name in the same parent", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entity.ErrFolderAlreadyExists)

		_, err := uc.Create(context.Background(), "docs", nil)

		require.ErrorIs(t, err, entity.ErrFolderAlreadyExists)
	})
}

func TestFolderUpdate(t *testing.T) {
	t.Parallel()

	t.Run("moves a folder under another branch", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _coversID).Return(folderByID(_coversID), nil)
		folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)
		folders.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		got, err := uc.Update(context.Background(), _coversID, "covers", new(_docsID))

		require.NoError(t, err)
		assert.Equal(t, new(_docsID), got.ParentID)
	})

	t.Run("renames at the root", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _docsID).Return(folderByID(_docsID), nil)
		folders.EXPECT().Update(gomock.Any(), &entity.Folder{ID: _docsID, Name: "Documentos"}).Return(nil)

		_, err := uc.Update(context.Background(), _docsID, "Documentos", nil)

		require.NoError(t, err)
	})

	cycles := []struct{ name, id, parent string }{
		{"into itself", _photosID, _photosID},
		{"into its child", _photosID, _coversID},
		{"into its grandchild", _photosID, _yearID},
	}

	for _, tc := range cycles {
		t.Run("rejects moving "+tc.name, func(t *testing.T) {
			t.Parallel()

			uc, folders, _ := newFolderUseCase(t) // no Update expected
			folders.EXPECT().GetByID(gomock.Any(), tc.id).Return(folderByID(tc.id), nil)
			folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)

			_, err := uc.Update(context.Background(), tc.id, "photos", new(tc.parent))

			require.ErrorIs(t, err, entity.ErrInvalidFolder)
		})
	}

	t.Run("unknown parent", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _docsID).Return(folderByID(_docsID), nil)
		folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)

		_, err := uc.Update(context.Background(), _docsID, "docs", new("55555555-5555-4555-8555-555555555555"))

		require.ErrorIs(t, err, entity.ErrFolderNotFound)
	})
}

func TestFolderDelete(t *testing.T) {
	t.Parallel()

	t.Run("deletes an empty folder", func(t *testing.T) {
		t.Parallel()

		uc, folders, files := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _docsID).Return(folderByID(_docsID), nil)
		folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)
		files.EXPECT().CountInFolder(gomock.Any(), _docsID).Return(0, nil)
		folders.EXPECT().Delete(gomock.Any(), _docsID).Return(nil)

		require.NoError(t, uc.Delete(context.Background(), _docsID))
	})

	t.Run("refuses a folder with subfolders", func(t *testing.T) {
		t.Parallel()

		uc, folders, _ := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _photosID).Return(folderByID(_photosID), nil)
		folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)

		require.ErrorIs(t, uc.Delete(context.Background(), _photosID), entity.ErrFolderNotEmpty)
	})

	t.Run("refuses a folder with files", func(t *testing.T) {
		t.Parallel()

		uc, folders, files := newFolderUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), _yearID).Return(folderByID(_yearID), nil)
		folders.EXPECT().List(gomock.Any()).Return(folderTree(), nil)
		files.EXPECT().CountInFolder(gomock.Any(), _yearID).Return(3, nil)

		require.ErrorIs(t, uc.Delete(context.Background(), _yearID), entity.ErrFolderNotEmpty)
	})
}
