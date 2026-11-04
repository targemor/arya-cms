package v1

import (
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContentQuery(t *testing.T) {
	t.Parallel()

	q, err := parseContentQuery([][2]string{
		{"pagination[page]", "2"},
		{"pagination[pageSize]", "10"},
		{"sort", "createdAt:desc, title"},
		{"sort[1]", "views:ASC"},
		{"filters[title][$contains]", "go"},
		{"filters[id][$in][0]", "a"},
		{"filters[views][$gt]", "5"},
		{"filters[id][$in][1]", "b"},
		{"filters[is_published]", "true"},
		{"populate", "*"}, // ignored
	})

	require.NoError(t, err)
	assert.Equal(t, entity.ContentQuery{
		Page:     2,
		PageSize: 10,
		Sort: []entity.Sort{
			{Field: "createdAt", Desc: true},
			{Field: "title"},
			{Field: "views"},
		},
		Filters: []entity.Filter{
			{Field: "title", Operator: entity.OpContains, Raw: []string{"go"}},
			{Field: "id", Operator: entity.OpIn, Raw: []string{"a", "b"}},
			{Field: "views", Operator: entity.OpGt, Raw: []string{"5"}},
			{Field: "is_published", Operator: entity.OpEq, Raw: []string{"true"}},
		},
	}, q)
}

func TestParseContentQueryDefaults(t *testing.T) {
	t.Parallel()

	q, err := parseContentQuery(nil)

	require.NoError(t, err)
	assert.Equal(t, entity.ContentQuery{Page: 1, PageSize: entity.DefaultPageSize, Filters: []entity.Filter{}}, q)
}

func TestParseContentQueryRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		param [2]string
	}{
		{name: "page not a number", param: [2]string{"pagination[page]", "one"}},
		{name: "unknown pagination key", param: [2]string{"pagination[limit]", "10"}},
		{name: "bad sort direction", param: [2]string{"sort", "title:up"}},
		{name: "empty sort field", param: [2]string{"sort", "title,,views"}},
		{name: "unclosed filter", param: [2]string{"filters[title", "x"}},
		{name: "empty filter field", param: [2]string{"filters[][$eq]", "x"}},
		{name: "non-numeric filter index", param: [2]string{"filters[id][$in][x]", "a"}},
		{name: "filter nested too deep", param: [2]string{"filters[a][$eq][0][1]", "x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseContentQuery([][2]string{tc.param})

			require.ErrorIs(t, err, entity.ErrInvalidQuery)
		})
	}
}
