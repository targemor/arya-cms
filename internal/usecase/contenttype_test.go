package usecase_test

import (
	"context"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/contenttype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newContentTypeUseCase(t *testing.T) (usecase.ContentType, *MockContentTypeStore, *MockContentTypeMigrator) {
	t.Helper()

	ctrl := gomock.NewController(t)

	store := NewMockContentTypeStore(ctrl)
	migrator := NewMockContentTypeMigrator(ctrl)

	return contenttype.New(store, migrator), store, migrator
}

func articlesContentType() entity.ContentType {
	return entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
		{Name: "title", Type: entity.AttributeTypeString, Required: true},
	}}
}

type contentTypeCreateCase struct {
	name    string
	input   entity.ContentType
	mock    func(s *MockContentTypeStore, m *MockContentTypeMigrator)
	wantErr error
}

func contentTypeCreateCases(ct entity.ContentType) []contentTypeCreateCase {
	return []contentTypeCreateCase{
		{
			name:  "migrates then saves",
			input: ct,
			mock: func(s *MockContentTypeStore, m *MockContentTypeMigrator) {
				gomock.InOrder(
					s.EXPECT().Exists(gomock.Any(), "default", "articles").Return(false, nil),
					m.EXPECT().Migrate(gomock.Any(), "default", ct).Return(nil),
					s.EXPECT().Save(gomock.Any(), "default", ct).Return(nil),
				)
			},
		},
		{
			name:    "invalid definition touches nothing",
			input:   entity.ContentType{TableName: "Bad Name"},
			mock:    func(*MockContentTypeStore, *MockContentTypeMigrator) {},
			wantErr: entity.ErrInvalidContentType,
		},
		{
			name:  "already exists",
			input: ct,
			mock: func(s *MockContentTypeStore, _ *MockContentTypeMigrator) {
				s.EXPECT().Exists(gomock.Any(), "default", "articles").Return(true, nil)
			},
			wantErr: entity.ErrContentTypeAlreadyExists,
		},
		{
			name:  "migration failure does not save the file",
			input: ct,
			mock: func(s *MockContentTypeStore, m *MockContentTypeMigrator) {
				s.EXPECT().Exists(gomock.Any(), "default", "articles").Return(false, nil)
				m.EXPECT().Migrate(gomock.Any(), "default", ct).Return(errInternalServErr)
			},
			wantErr: errInternalServErr,
		},
		{
			name:  "save failure",
			input: ct,
			mock: func(s *MockContentTypeStore, m *MockContentTypeMigrator) {
				s.EXPECT().Exists(gomock.Any(), "default", "articles").Return(false, nil)
				m.EXPECT().Migrate(gomock.Any(), "default", ct).Return(nil)
				s.EXPECT().Save(gomock.Any(), "default", ct).Return(errInternalServErr)
			},
			wantErr: errInternalServErr,
		},
	}
}

func TestContentTypeCreate(t *testing.T) {
	t.Parallel()

	tests := contentTypeCreateCases(articlesContentType())

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, store, migrator := newContentTypeUseCase(t)
			tc.mock(store, migrator)

			got, err := uc.Create(context.Background(), "default", tc.input)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.input, got)
		})
	}
}

func TestContentTypeList(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		uc, store, _ := newContentTypeUseCase(t)
		store.EXPECT().List(gomock.Any(), "default").Return([]entity.ContentType{articlesContentType()}, nil)

		got, err := uc.List(context.Background(), "default")

		require.NoError(t, err)
		assert.Equal(t, []entity.ContentType{articlesContentType()}, got)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		uc, store, _ := newContentTypeUseCase(t)
		store.EXPECT().List(gomock.Any(), "default").Return(nil, errInternalServErr)

		_, err := uc.List(context.Background(), "default")

		require.ErrorIs(t, err, errInternalServErr)
	})
}

func TestContentTypeSync(t *testing.T) {
	t.Parallel()

	articles := articlesContentType()
	pages := entity.ContentType{TableName: "pages", Attributes: []entity.Attribute{
		{Name: "slug", Type: entity.AttributeTypeString},
	}}

	t.Run("migrates every definition", func(t *testing.T) {
		t.Parallel()

		uc, store, migrator := newContentTypeUseCase(t)
		store.EXPECT().ListProjects(gomock.Any()).Return([]string{"default"}, nil)
		store.EXPECT().List(gomock.Any(), "default").Return([]entity.ContentType{articles, pages}, nil)
		migrator.EXPECT().Migrate(gomock.Any(), "default", articles).Return(nil)
		migrator.EXPECT().Migrate(gomock.Any(), "default", pages).Return(nil)

		require.NoError(t, uc.Sync(context.Background()))
	})

	t.Run("keeps going past failures and reports them all", func(t *testing.T) {
		t.Parallel()

		uc, store, migrator := newContentTypeUseCase(t)
		store.EXPECT().ListProjects(gomock.Any()).Return([]string{"default"}, nil)
		store.EXPECT().List(gomock.Any(), "default").Return([]entity.ContentType{articles, pages}, entity.ErrInvalidContentType)
		migrator.EXPECT().Migrate(gomock.Any(), "default", articles).Return(errInternalServErr)
		migrator.EXPECT().Migrate(gomock.Any(), "default", pages).Return(nil)

		err := uc.Sync(context.Background())

		require.ErrorIs(t, err, entity.ErrInvalidContentType)
		require.ErrorIs(t, err, errInternalServErr)
		assert.ErrorContains(t, err, "articles")

		_, err = uc.Get(context.Background(), "default", "articles")
		require.ErrorIs(t, err, entity.ErrContentTypeNotFound, "a type whose migration failed is not served")

		got, err := uc.Get(context.Background(), "default", "pages")
		require.NoError(t, err)
		assert.Equal(t, pages, got)
	})
}

func TestContentTypeGet(t *testing.T) {
	t.Parallel()

	uc, store, migrator := newContentTypeUseCase(t)
	ctx := context.Background()

	_, err := uc.Get(ctx, "default", "articles")
	require.ErrorIs(t, err, entity.ErrContentTypeNotFound, "nothing is served before a sync or create")

	store.EXPECT().Exists(gomock.Any(), "default", "articles").Return(false, nil)
	migrator.EXPECT().Migrate(gomock.Any(), "default", articlesContentType()).Return(nil)
	store.EXPECT().Save(gomock.Any(), "default", articlesContentType()).Return(nil)

	_, err = uc.Create(ctx, "default", articlesContentType())
	require.NoError(t, err)

	got, err := uc.Get(ctx, "default", "articles")
	require.NoError(t, err)
	assert.Equal(t, articlesContentType(), got, "a created type is served immediately")

	store.EXPECT().ListProjects(gomock.Any()).Return([]string{"default"}, nil)
	store.EXPECT().List(gomock.Any(), "default").Return(nil, nil)
	require.NoError(t, uc.Sync(ctx))

	_, err = uc.Get(ctx, "default", "articles")
	require.ErrorIs(t, err, entity.ErrContentTypeNotFound, "a type whose file is gone is no longer served")
}
