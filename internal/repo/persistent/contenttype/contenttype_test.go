package contenttype

import (
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDDL(t *testing.T) {
	t.Parallel()

	stmts, err := buildDDL("default", entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
		{Name: "title", Type: entity.AttributeTypeString},
		{Name: "views", Type: entity.AttributeTypeInteger},
		{Name: "is_published", Type: entity.AttributeTypeBoolean},
		{Name: "cover", Type: entity.AttributeTypeMedia},
	}})

	require.NoError(t, err)
	assert.Equal(t, []string{
		`CREATE TABLE IF NOT EXISTS "content_default_articles" (id UUID PRIMARY KEY DEFAULT gen_random_uuid(),` +
			` created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`ALTER TABLE "content_default_articles" ADD COLUMN IF NOT EXISTS "title" TEXT,` +
			` ADD COLUMN IF NOT EXISTS "views" BIGINT, ADD COLUMN IF NOT EXISTS "is_published" BOOLEAN,` +
			` ADD COLUMN IF NOT EXISTS "cover" UUID REFERENCES upload_files (id) ON DELETE SET NULL`,
	}, stmts)
}

func TestBuildDDLUnsupportedType(t *testing.T) {
	t.Parallel()

	_, err := buildDDL("default", entity.ContentType{TableName: "articles", Attributes: []entity.Attribute{
		{Name: "title", Type: "date"},
	}})

	require.ErrorIs(t, err, entity.ErrInvalidContentType)
}
