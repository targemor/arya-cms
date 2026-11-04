package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/content"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const _entryID = "550e8400-e29b-41d4-a716-446655440000"

func newContentUseCase(t *testing.T) (usecase.Content, *MockContentType, *MockContentRepo) {
	t.Helper()

	ctrl := gomock.NewController(t)

	types := NewMockContentType(ctrl)
	repo := NewMockContentRepo(ctrl)

	return content.New(types, repo), types, repo
}

// postsType has one required and two optional attributes.
func postsType() entity.ContentType {
	return entity.ContentType{TableName: "posts", Attributes: []entity.Attribute{
		{Name: "title", Type: entity.AttributeTypeString, Required: true},
		{Name: "views", Type: entity.AttributeTypeInteger},
		{Name: "is_published", Type: entity.AttributeTypeBoolean},
		{Name: "cover", Type: entity.AttributeTypeMedia},
	}}
}

func TestContentCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data entity.Entry
		// stored is what reaches the repo; nil means validation must reject the payload.
		stored entity.Entry
	}{
		{
			name:   "valid, integer converted from json.Number",
			data:   entity.Entry{"title": "Hello", "views": json.Number("9007199254740993"), "is_published": true},
			stored: entity.Entry{"title": "Hello", "views": int64(9007199254740993), "is_published": true},
		},
		{name: "optional field may be null", data: entity.Entry{"title": "Hi", "views": nil}, stored: entity.Entry{"title": "Hi", "views": nil}},
		{name: "missing required field", data: entity.Entry{"views": json.Number("1")}},
		{name: "required field null", data: entity.Entry{"title": nil}},
		{name: "wrong type", data: entity.Entry{"title": "Hi", "views": "many"}},
		{name: "fractional integer", data: entity.Entry{"title": "Hi", "views": json.Number("1.5")}},
		{name: "unknown field", data: entity.Entry{"title": "Hi", "author": "me"}},
		{name: "system field", data: entity.Entry{"title": "Hi", "id": _entryID}},
		{name: "media by file id", data: entity.Entry{"title": "Hi", "cover": _entryID}, stored: entity.Entry{"title": "Hi", "cover": _entryID}},
		{name: "media cleared", data: entity.Entry{"title": "Hi", "cover": nil}, stored: entity.Entry{"title": "Hi", "cover": nil}},
		{name: "media not a uuid", data: entity.Entry{"title": "Hi", "cover": "cover.png"}},
		{name: "media as an object", data: entity.Entry{"title": "Hi", "cover": map[string]any{"id": _entryID}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, types, repo := newContentUseCase(t)
			types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)

			if tc.stored != nil {
				repo.EXPECT().Create(gomock.Any(), "default", postsType(), tc.stored).Return(entity.Entry{"id": _entryID}, nil)
			}

			got, err := uc.Create(context.Background(), "default", "posts", tc.data)

			if tc.stored == nil {
				require.ErrorIs(t, err, entity.ErrInvalidEntry)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, entity.Entry{"id": _entryID}, got)
		})
	}
}

func TestContentUpdateIsPartial(t *testing.T) {
	t.Parallel()

	t.Run("required field may be omitted", func(t *testing.T) {
		t.Parallel()

		uc, types, repo := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)
		repo.EXPECT().Update(gomock.Any(), "default", postsType(), _entryID, entity.Entry{"views": int64(2)}).
			Return(entity.Entry{"id": _entryID}, nil)

		_, err := uc.Update(context.Background(), "default", "posts", _entryID, entity.Entry{"views": json.Number("2")})

		require.NoError(t, err)
	})

	t.Run("required field still cannot be null", func(t *testing.T) {
		t.Parallel()

		uc, types, _ := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)

		_, err := uc.Update(context.Background(), "default", "posts", _entryID, entity.Entry{"title": nil})

		require.ErrorIs(t, err, entity.ErrInvalidEntry)
	})
}

func TestContentErrors(t *testing.T) {
	t.Parallel()

	t.Run("unknown content type", func(t *testing.T) {
		t.Parallel()

		uc, types, _ := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "nope").Return(entity.ContentType{}, entity.ErrContentTypeNotFound)

		_, err := uc.Create(context.Background(), "default", "nope", entity.Entry{})

		require.ErrorIs(t, err, entity.ErrContentTypeNotFound)
	})

	t.Run("malformed id never reaches the repo", func(t *testing.T) {
		t.Parallel()

		uc, types, _ := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil).Times(3)

		_, err := uc.Get(context.Background(), "default", "posts", "1")
		require.ErrorIs(t, err, entity.ErrEntryNotFound)

		_, err = uc.Update(context.Background(), "default", "posts", "1; DROP TABLE x", entity.Entry{})
		require.ErrorIs(t, err, entity.ErrEntryNotFound)

		require.ErrorIs(t, uc.Delete(context.Background(), "default", "posts", ""), entity.ErrEntryNotFound)
	})

	t.Run("not found from repo", func(t *testing.T) {
		t.Parallel()

		uc, types, repo := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)
		repo.EXPECT().Delete(gomock.Any(), "default", postsType(), _entryID).Return(entity.ErrEntryNotFound)

		require.ErrorIs(t, uc.Delete(context.Background(), "default", "posts", _entryID), entity.ErrEntryNotFound)
	})
}

func TestContentList(t *testing.T) {
	t.Parallel()

	t.Run("resolves the query and computes pagination", func(t *testing.T) {
		t.Parallel()

		uc, types, repo := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)

		q := entity.ContentQuery{Page: 2, PageSize: 10, Filters: []entity.Filter{
			{Field: "views", Operator: entity.OpGt, Raw: []string{"5"}},
		}}
		resolved := q
		resolved.Filters = []entity.Filter{
			{Field: "views", Operator: entity.OpGt, Raw: []string{"5"}, Values: []any{int64(5)}},
		}
		rows := []entity.Entry{{"title": "a"}}
		repo.EXPECT().List(gomock.Any(), "default", postsType(), resolved).Return(rows, 21, nil)

		got, pagination, err := uc.List(context.Background(), "default", "posts", q)

		require.NoError(t, err)
		assert.Equal(t, rows, got)
		assert.Equal(t, entity.Pagination{Page: 2, PageSize: 10, PageCount: 3, Total: 21}, pagination)
	})

	t.Run("invalid query never reaches the repo", func(t *testing.T) {
		t.Parallel()

		uc, types, _ := newContentUseCase(t)
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil)

		_, _, err := uc.List(context.Background(), "default", "posts", entity.ContentQuery{Page: 1, PageSize: 10, Filters: []entity.Filter{
			{Field: "views", Operator: entity.OpContains, Raw: []string{"5"}},
		}})

		require.ErrorIs(t, err, entity.ErrInvalidQuery)
	})
}

func TestContentSchemaFollowsDefinitionChanges(t *testing.T) {
	t.Parallel()

	uc, types, repo := newContentUseCase(t)

	edited := postsType()
	edited.Attributes = append(edited.Attributes, entity.Attribute{Name: "rating", Type: entity.AttributeTypeInteger})

	gomock.InOrder(
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(postsType(), nil),
		types.EXPECT().Get(gomock.Any(), "default", "posts").Return(edited, nil),
	)
	repo.EXPECT().Create(gomock.Any(), "default", edited, gomock.Any()).Return(entity.Entry{}, nil)

	data := entity.Entry{"title": "Hi", "rating": json.Number("5")}

	_, err := uc.Create(context.Background(), "default", "posts", data)
	require.ErrorIs(t, err, entity.ErrInvalidEntry, "rating is unknown before the edit")

	_, err = uc.Create(context.Background(), "default", "posts", data)
	require.NoError(t, err, "the cached schema must be recompiled after the edit")
}
