package contenttype_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo/filestore/contenttype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func articles() entity.ContentType {
	return entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
		{Name: "title", Type: entity.AttributeTypeString, Required: true},
		{Name: "views", Type: entity.AttributeTypeInteger},
		{Name: "body", Type: entity.AttributeTypeString},
		{Name: "is_published", Type: entity.AttributeTypeBoolean},
		{Name: "cover", Type: entity.AttributeTypeMedia},
	}}
}

func TestSaveListRoundTrip(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "schemas") // Save must create it
	store := contenttype.New(dir)
	ctx := context.Background()

	exists, err := store.Exists(ctx, "default", "articles")
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, store.Save(ctx, "default", articles()))

	exists, err = store.Exists(ctx, "default", "articles")
	require.NoError(t, err)
	assert.True(t, exists)

	got, err := store.List(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, []entity.ContentType{articles()}, got, "attribute order and required flags must survive")

	entries, err := os.ReadDir(filepath.Join(dir, "default"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temp files left behind")
}

func TestSavedFileIsJSONSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, contenttype.New(dir).Save(context.Background(), "default", articles()))

	data, err := os.ReadFile(filepath.Join(dir, "default", "articles.json"))
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))

	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", doc["$schema"])
	assert.Equal(t, "articles", doc["title"])
	assert.Equal(t, "object", doc["type"])
	assert.Equal(t, false, doc["additionalProperties"])
	assert.Equal(t, []any{"title"}, doc["required"])
	props, ok := doc["properties"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"type": "integer"}, props["views"])
	assert.Equal(t, map[string]any{"type": "string", "format": "uuid", "x-cms-type": "media"}, props["cover"],
		"media is stored as a valid JSON Schema type, marked with x-cms-type")
}

func TestListReportsBadFilesAndKeepsGoodOnes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ctx := context.Background()
	store := contenttype.New(dir)

	require.NoError(t, store.Save(ctx, "default", articles()))

	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "default", name), []byte(content), 0o600))
	}

	write("broken.json", "{not json")
	write("renamed.json", `{"title":"other","type":"object","properties":{"a":{"type":"string"}}}`)
	write("bad_type.json", `{"title":"bad_type","type":"object","properties":{"a":{"type":"date"}}}`)
	write("notes.txt", "ignored")

	got, err := store.List(ctx, "default")

	require.ErrorIs(t, err, entity.ErrInvalidContentType)
	assert.ErrorContains(t, err, "broken.json")
	assert.ErrorContains(t, err, "renamed.json")
	assert.ErrorContains(t, err, "bad_type.json")
	assert.Equal(t, []entity.ContentType{articles()}, got)
}

func TestListMissingDir(t *testing.T) {
	t.Parallel()

	got, err := contenttype.New(filepath.Join(t.TempDir(), "missing")).List(context.Background(), "default")

	require.NoError(t, err)
	assert.Empty(t, got)
}
