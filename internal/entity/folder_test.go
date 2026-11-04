package entity_test

import (
	"strings"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFolderName(t *testing.T) {
	t.Parallel()

	valid := map[string]string{
		"Portadas":               "Portadas",
		"  Fotos 2026  ":         "Fotos 2026",
		"Año nuevo · ñ":          "Año nuevo · ñ",
		strings.Repeat("é", 100): strings.Repeat("é", 100),
	}

	for in, want := range valid {
		got, err := entity.NormalizeFolderName(in)

		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}

	invalid := []string{"", "   ", "a/b", `a\b`, "tab\there", strings.Repeat("a", 101), "p\xedxel"}

	for _, in := range invalid {
		_, err := entity.NormalizeFolderName(in)

		require.ErrorIs(t, err, entity.ErrInvalidFolder, "%q", in)
	}
}

func TestIsDescendant(t *testing.T) {
	t.Parallel()

	p := func(s string) *string { return &s }
	folders := []entity.Folder{
		{ID: "a"},
		{ID: "b", ParentID: p("a")},
		{ID: "c", ParentID: p("b")},
		{ID: "d"},
	}

	assert.True(t, entity.IsDescendant(folders, "a", "a"), "a folder is its own descendant")
	assert.True(t, entity.IsDescendant(folders, "c", "a"))
	assert.True(t, entity.IsDescendant(folders, "b", "a"))
	assert.False(t, entity.IsDescendant(folders, "a", "c"))
	assert.False(t, entity.IsDescendant(folders, "d", "a"))

	// A corrupted, cyclic chain must not loop forever.
	cyclic := []entity.Folder{{ID: "x", ParentID: p("y")}, {ID: "y", ParentID: p("x")}}
	assert.False(t, entity.IsDescendant(cyclic, "x", "z"))
}
