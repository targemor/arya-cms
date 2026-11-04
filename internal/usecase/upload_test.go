package usecase_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/upload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	_maxUploadSize = 1024
	_uploadID      = "550e8400-e29b-41d4-a716-446655440000"
)

// pngHeader is enough of a PNG for content sniffing.
const pngHeader = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

func newUploadUseCase(t *testing.T) (usecase.Upload, *MockUploadRepo, *MockFolderRepo, *MockStorageProvider) {
	t.Helper()

	ctrl := gomock.NewController(t)

	repo := NewMockUploadRepo(ctrl)
	folders := NewMockFolderRepo(ctrl)
	storage := NewMockStorageProvider(ctrl)

	return upload.New(repo, folders, storage, _maxUploadSize), repo, folders, storage
}

// newUploadUseCaseNoCalls is for tests that expect no repository or storage call at all.
func newUploadUseCaseNoCalls(t *testing.T) usecase.Upload {
	t.Helper()

	ctrl := gomock.NewController(t)

	return upload.New(NewMockUploadRepo(ctrl), NewMockFolderRepo(ctrl), NewMockStorageProvider(ctrl), _maxUploadSize)
}

func uploadInput(name string, alt *string, size int64) usecase.UploadInput {
	return usecase.UploadInput{Name: name, AlternativeText: alt, Size: size}
}

// storeAs makes the storage mock record the content it receives and set the URL, like a real provider.
func storeAs(t *testing.T, storage *MockStorageProvider, gotContent *string) {
	t.Helper()

	storage.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f *entity.UploadFile, r io.Reader) error {
			b, err := io.ReadAll(r)
			require.NoError(t, err)

			*gotContent = string(b)
			f.URL = "/uploads/" + f.Hash + f.Ext
			f.Provider = "local"

			return nil
		})
}

func TestUploadStoresFileAndMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, fileName, content, wantName, wantExt, wantMime string
	}{
		{"png by extension", "cover.PNG", pngHeader, "cover.PNG", ".png", "image/png"},
		{"browser fake path", `C:\fakepath\report.pdf`, "%PDF-1.7", "report.pdf", ".pdf", "application/pdf"},
		{"no extension, sniffed", "notes", "plain text", "notes", "", "text/plain"},
		{"odd extension dropped, sniffed", "image.p~g", pngHeader, "image.p~g", "", "image/png"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, repo, _, storage := newUploadUseCase(t)

			var stored string

			storeAs(t, storage, &stored)
			repo.EXPECT().Store(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f *entity.UploadFile) error {
				f.ID = _uploadID

				return nil
			})

			alt := "Alt"
			file, err := uc.Upload(context.Background(), uploadInput(tc.fileName, &alt, int64(len(tc.content))), strings.NewReader(tc.content))

			require.NoError(t, err)
			assert.Equal(t, tc.content, stored, "the sniffed bytes must still reach storage")
			assert.Equal(t, _uploadID, file.ID)
			assert.Equal(t, tc.wantName, file.Name)
			assert.Equal(t, tc.wantExt, file.Ext)
			assert.Equal(t, tc.wantMime, file.Mime)
			assert.Equal(t, int64(len(tc.content)), file.Size)
			assert.Regexp(t, `^[0-9a-f]{32}$`, file.Hash)
			assert.Equal(t, "/uploads/"+file.Hash+tc.wantExt, file.URL)
			assert.Equal(t, &alt, file.AlternativeText)
		})
	}
}

func TestUploadRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileName string
		size     int64
		want     error
	}{
		{"empty file", "a.png", 0, entity.ErrInvalidUpload},
		{"too large", "a.png", _maxUploadSize + 1, entity.ErrUploadTooLarge},
		{"no name", "", 10, entity.ErrInvalidUpload},
		{"name is only a separator", `\`, 10, entity.ErrInvalidUpload},
		{"name not UTF-8", "p\xedxel.png", 10, entity.ErrInvalidUpload},
		{"name too long", strings.Repeat("a", 256), 10, entity.ErrInvalidUpload},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc := newUploadUseCaseNoCalls(t) // no storage or repo call expected

			_, err := uc.Upload(context.Background(), uploadInput(tc.fileName, nil, tc.size), strings.NewReader("x"))

			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestUploadRejectsInvalidUTF8AlternativeText(t *testing.T) {
	t.Parallel()

	uc := newUploadUseCaseNoCalls(t)
	alt := "p\xedxel"

	_, err := uc.Upload(context.Background(), uploadInput("a.png", &alt, 5), strings.NewReader("hello"))

	require.ErrorIs(t, err, entity.ErrInvalidUpload)
}

func TestUploadRemovesStoredObjectWhenMetadataFails(t *testing.T) {
	t.Parallel()

	uc, repo, _, storage := newUploadUseCase(t)

	var stored string

	storeAs(t, storage, &stored)
	repo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(errInternalServErr)
	storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

	_, err := uc.Upload(context.Background(), uploadInput("a.txt", nil, 5), strings.NewReader("hello"))

	require.ErrorIs(t, err, errInternalServErr)
}

func TestUploadEmptyAlternativeTextIsNil(t *testing.T) {
	t.Parallel()

	uc, repo, _, storage := newUploadUseCase(t)

	var stored string

	storeAs(t, storage, &stored)
	repo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

	empty := ""
	file, err := uc.Upload(context.Background(), uploadInput("a.txt", &empty, 5), strings.NewReader("hello"))

	require.NoError(t, err)
	assert.Nil(t, file.AlternativeText)
}

func TestUploadList(t *testing.T) {
	t.Parallel()

	t.Run("pages", func(t *testing.T) {
		t.Parallel()

		uc, repo, _, _ := newUploadUseCase(t)
		files := []entity.UploadFile{{ID: _uploadID}}
		repo.EXPECT().List(gomock.Any(), (*string)(nil), 2, 10).Return(files, 11, nil)

		got, pagination, err := uc.List(context.Background(), nil, 2, 10)

		require.NoError(t, err)
		assert.Equal(t, files, got)
		assert.Equal(t, entity.Pagination{Page: 2, PageSize: 10, PageCount: 2, Total: 11}, pagination)
	})

	t.Run("bad page size", func(t *testing.T) {
		t.Parallel()

		uc := newUploadUseCaseNoCalls(t)

		_, _, err := uc.List(context.Background(), nil, 1, entity.MaxPageSize+1)

		require.ErrorIs(t, err, entity.ErrInvalidQuery)
	})
}

func TestUploadDelete(t *testing.T) {
	t.Parallel()

	file := entity.UploadFile{ID: _uploadID, Hash: "abc", Ext: ".png", Provider: "local"}

	t.Run("deletes the row, then the object", func(t *testing.T) {
		t.Parallel()

		uc, repo, _, storage := newUploadUseCase(t)
		storage.EXPECT().Name().Return("local")
		gomock.InOrder(
			repo.EXPECT().GetByID(gomock.Any(), _uploadID).Return(file, nil),
			repo.EXPECT().Delete(gomock.Any(), _uploadID).Return(file, nil),
			storage.EXPECT().Delete(gomock.Any(), &file).Return(nil),
		)

		require.NoError(t, uc.Delete(context.Background(), _uploadID))
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		uc, repo, _, _ := newUploadUseCase(t)
		repo.EXPECT().GetByID(gomock.Any(), _uploadID).Return(entity.UploadFile{}, entity.ErrUploadNotFound)

		require.ErrorIs(t, uc.Delete(context.Background(), _uploadID), entity.ErrUploadNotFound)
	})

	t.Run("file stored by another provider is left alone", func(t *testing.T) {
		t.Parallel()

		uc, repo, _, storage := newUploadUseCase(t)
		s3File := file
		s3File.Provider = "aws-s3"
		repo.EXPECT().GetByID(gomock.Any(), _uploadID).Return(s3File, nil)
		storage.EXPECT().Name().Return("local").AnyTimes()

		require.ErrorIs(t, uc.Delete(context.Background(), _uploadID), entity.ErrStorageProviderAbsent)
	})
}

func TestUploadIntoFolder(t *testing.T) {
	t.Parallel()

	folderID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	t.Run("stores the folder on the file", func(t *testing.T) {
		t.Parallel()

		uc, repo, folders, storage := newUploadUseCase(t)

		var stored string

		folders.EXPECT().GetByID(gomock.Any(), folderID).Return(entity.Folder{ID: folderID}, nil)
		storeAs(t, storage, &stored)
		repo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		in := uploadInput("a.txt", nil, 5)
		in.FolderID = &folderID
		file, err := uc.Upload(context.Background(), in, strings.NewReader("hello"))

		require.NoError(t, err)
		assert.Equal(t, &folderID, file.FolderID)
	})

	t.Run("unknown folder stores nothing", func(t *testing.T) {
		t.Parallel()

		uc, _, folders, _ := newUploadUseCase(t) // no storage or repo call expected
		folders.EXPECT().GetByID(gomock.Any(), folderID).Return(entity.Folder{}, entity.ErrFolderNotFound)

		in := uploadInput("a.txt", nil, 5)
		in.FolderID = &folderID
		_, err := uc.Upload(context.Background(), in, strings.NewReader("hello"))

		require.ErrorIs(t, err, entity.ErrFolderNotFound)
	})

	t.Run("listing an unknown folder is not found", func(t *testing.T) {
		t.Parallel()

		uc, _, folders, _ := newUploadUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), folderID).Return(entity.Folder{}, entity.ErrFolderNotFound)

		_, _, err := uc.List(context.Background(), &folderID, 1, 10)

		require.ErrorIs(t, err, entity.ErrFolderNotFound)
	})
}

func TestUploadMove(t *testing.T) {
	t.Parallel()

	folderID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	t.Run("into a folder", func(t *testing.T) {
		t.Parallel()

		uc, repo, folders, _ := newUploadUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), folderID).Return(entity.Folder{ID: folderID}, nil)
		repo.EXPECT().Move(gomock.Any(), _uploadID, &folderID).Return(entity.UploadFile{ID: _uploadID, FolderID: &folderID}, nil)

		file, err := uc.Move(context.Background(), _uploadID, &folderID)

		require.NoError(t, err)
		assert.Equal(t, &folderID, file.FolderID)
	})

	t.Run("to the root needs no folder lookup", func(t *testing.T) {
		t.Parallel()

		uc, repo, _, _ := newUploadUseCase(t)
		repo.EXPECT().Move(gomock.Any(), _uploadID, (*string)(nil)).Return(entity.UploadFile{ID: _uploadID}, nil)

		_, err := uc.Move(context.Background(), _uploadID, nil)

		require.NoError(t, err)
	})

	t.Run("into an unknown folder", func(t *testing.T) {
		t.Parallel()

		uc, _, folders, _ := newUploadUseCase(t)
		folders.EXPECT().GetByID(gomock.Any(), folderID).Return(entity.Folder{}, entity.ErrFolderNotFound)

		_, err := uc.Move(context.Background(), _uploadID, &folderID)

		require.ErrorIs(t, err, entity.ErrFolderNotFound)
	})
}
