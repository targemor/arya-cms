package local_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBrokenReader = errors.New("connection reset")

// brokenReader yields some bytes and then fails, like a client that disconnects mid-upload.
type brokenReader struct{ sent bool }

func (r *brokenReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, errBrokenReader
	}

	r.sent = true

	return copy(p, "partial"), nil
}

func TestUploadAndDelete(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "public", "uploads") // Upload must create it
	provider := local.New(dir, "/uploads/")
	ctx := context.Background()
	file := entity.UploadFile{Hash: "abc123", Ext: ".txt"}

	require.NoError(t, provider.Upload(ctx, &file, strings.NewReader("hello")))

	assert.Equal(t, "/uploads/abc123.txt", file.URL)
	assert.Equal(t, "local", file.Provider)
	assert.Equal(t, "local", provider.Name())

	content, err := os.ReadFile(filepath.Join(dir, "abc123.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	require.NoError(t, provider.Delete(ctx, &file))
	assert.NoFileExists(t, filepath.Join(dir, "abc123.txt"))

	require.NoError(t, provider.Delete(ctx, &file), "deleting a file that is already gone is not an error")
}

func TestUploadNeverOverwrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	provider := local.New(dir, "/uploads")
	ctx := context.Background()

	first := entity.UploadFile{Hash: "same", Ext: ".txt"}
	require.NoError(t, provider.Upload(ctx, &first, strings.NewReader("first")))

	second := entity.UploadFile{Hash: "same", Ext: ".txt"}
	require.Error(t, provider.Upload(ctx, &second, strings.NewReader("second")))

	content, err := os.ReadFile(filepath.Join(dir, "same.txt"))
	require.NoError(t, err)
	assert.Equal(t, "first", string(content))
}

func TestUploadLeavesNoPartialFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := entity.UploadFile{Hash: "partial", Ext: ".bin"}

	err := local.New(dir, "/uploads").Upload(context.Background(), &file, &brokenReader{})

	require.ErrorIs(t, err, errBrokenReader)
	assert.NoFileExists(t, filepath.Join(dir, "partial.bin"))
	assert.Empty(t, file.URL)
}

func TestUnsafeNamesAreRejected(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "uploads")
	provider := local.New(dir, "/uploads")
	ctx := context.Background()

	victim := filepath.Join(root, "victim.txt")
	require.NoError(t, os.WriteFile(victim, []byte("keep"), 0o600))

	evil := entity.UploadFile{Hash: "../victim", Ext: ".txt"}

	require.ErrorIs(t, provider.Upload(ctx, &evil, strings.NewReader("pwned")), entity.ErrInvalidUpload)
	require.ErrorIs(t, provider.Delete(ctx, &evil), entity.ErrInvalidUpload)
	assert.FileExists(t, victim)
}
