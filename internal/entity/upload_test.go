package entity_test

import (
	"strings"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadFileObjectName(t *testing.T) {
	t.Parallel()

	valid := []struct{ hash, ext, want string }{
		{"9f86d081884c7d65", ".png", "9f86d081884c7d65.png"},
		{"cover_2026-ab", ".pdf", "cover_2026-ab.pdf"},
		{"README", "", "README"},
	}

	for _, tc := range valid {
		t.Run("valid "+tc.want, func(t *testing.T) {
			t.Parallel()

			f := entity.UploadFile{Hash: tc.hash, Ext: tc.ext}
			got, err := f.ObjectName()

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	invalid := []struct{ name, hash, ext string }{
		{"empty hash", "", ".png"},
		{"traversal in hash", "../etc/passwd", ""},
		{"separator in hash", `a\b`, ".png"},
		{"dot in hash", "a.b", ".png"},
		{"hash too long", strings.Repeat("a", 65), ".png"},
		{"ext without dot", "abc", "png"},
		{"bare dot", "abc", "."},
		{"traversal in ext", "abc", "./../x"},
		{"uppercase ext", "abc", ".PNG"},
		{"ext too long", "abc", "." + strings.Repeat("a", 16)},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := entity.UploadFile{Hash: tc.hash, Ext: tc.ext}
			_, err := f.ObjectName()

			require.ErrorIs(t, err, entity.ErrInvalidUpload)
		})
	}
}
