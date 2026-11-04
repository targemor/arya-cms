package entity_test

import (
	"testing"
	"time"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func articlesType() entity.ContentType {
	return entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
		{Name: "title", Type: entity.AttributeTypeString},
		{Name: "views", Type: entity.AttributeTypeInteger},
		{Name: "is_published", Type: entity.AttributeTypeBoolean},
		{Name: "cover", Type: entity.AttributeTypeMedia},
	}}
}

func filterQuery(field string, op entity.FilterOperator, raw ...string) entity.ContentQuery {
	return entity.ContentQuery{
		Page: 1, PageSize: entity.DefaultPageSize,
		Filters: []entity.Filter{{Field: field, Operator: op, Raw: raw}},
	}
}

func TestResolveQueryConvertsValues(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		q    entity.ContentQuery
		want []any
	}{
		{name: "string", q: filterQuery("title", entity.OpContains, "go"), want: []any{"go"}},
		{name: "integer", q: filterQuery("views", entity.OpGte, "10"), want: []any{int64(10)}},
		{name: "boolean", q: filterQuery("is_published", entity.OpEq, "true"), want: []any{true}},
		{name: "datetime", q: filterQuery(entity.FieldCreatedAt, entity.OpLt, created.Format(time.RFC3339)), want: []any{created}},
		{
			name: "uuid list",
			q:    filterQuery(entity.FieldID, entity.OpIn, "550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			want: []any{"550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		},
		{name: "null takes a boolean", q: filterQuery("views", entity.OpNull, "false"), want: []any{false}},
		{name: "media by file id", q: filterQuery("cover", entity.OpEq, "550e8400-e29b-41d4-a716-446655440000"), want: []any{"550e8400-e29b-41d4-a716-446655440000"}},
		{name: "media without a file", q: filterQuery("cover", entity.OpNull, "true"), want: []any{true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := articlesType().ResolveQuery(tc.q)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Filters[0].Values)
		})
	}
}

func TestResolveQueryRejects(t *testing.T) {
	t.Parallel()

	page := func(p, size int) entity.ContentQuery { return entity.ContentQuery{Page: p, PageSize: size} }
	sortBy := func(field string) entity.ContentQuery {
		q := page(1, entity.DefaultPageSize)
		q.Sort = []entity.Sort{{Field: field}}

		return q
	}

	tests := []struct {
		name string
		q    entity.ContentQuery
	}{
		{name: "page zero", q: page(0, 10)},
		{name: "page size zero", q: page(1, 0)},
		{name: "page size above max", q: page(1, entity.MaxPageSize+1)},
		{name: "unknown sort field", q: sortBy("nope")},
		{name: "unknown filter field", q: filterQuery("nope", entity.OpEq, "x")},
		{name: "unknown operator", q: filterQuery("title", "$regex", ".*")},
		{name: "text operator on integer", q: filterQuery("views", entity.OpContains, "1")},
		{name: "ordering on boolean", q: filterQuery("is_published", entity.OpGt, "true")},
		{name: "not an integer", q: filterQuery("views", entity.OpEq, "ten")},
		{name: "not a boolean", q: filterQuery("is_published", entity.OpEq, "yes please")},
		{name: "not a datetime", q: filterQuery(entity.FieldCreatedAt, entity.OpGt, "yesterday")},
		{name: "not a uuid", q: filterQuery(entity.FieldID, entity.OpEq, "1")},
		{name: "in without values", q: filterQuery("views", entity.OpIn)},
		{name: "eq with two values", q: filterQuery("title", entity.OpEq, "a", "b")},
		{name: "null with non-boolean", q: filterQuery("title", entity.OpNull, "maybe")},
		{name: "media not a uuid", q: filterQuery("cover", entity.OpEq, "cover.png")},
		{name: "text operator on media", q: filterQuery("cover", entity.OpContains, "png")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := articlesType().ResolveQuery(tc.q)

			require.ErrorIs(t, err, entity.ErrInvalidQuery)
		})
	}
}

func TestNewPagination(t *testing.T) {
	t.Parallel()

	assert.Equal(t, entity.Pagination{Page: 2, PageSize: 10, PageCount: 9, Total: 87}, entity.NewPagination(2, 10, 87))
	assert.Equal(t, 0, entity.NewPagination(1, 10, 0).PageCount)
	assert.Equal(t, 1, entity.NewPagination(1, 10, 10).PageCount)
}
