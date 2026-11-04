package restapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUploadsApp(t *testing.T) (app *fiber.App, dir string) {
	t.Helper()

	root := t.TempDir()
	dir = filepath.Join(root, "uploads")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret"), 0o600))

	app = fiber.New()
	app.Get("/uploads/*", uploadsHandler("/uploads", dir))

	return app, dir
}

func get(t *testing.T, app *fiber.App, path string) (status int, header http.Header, body string) {
	t.Helper()

	resp, err := app.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	require.NoError(t, err)

	defer func() { require.NoError(t, resp.Body.Close()) }()

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, resp.Header, string(b)
}

func TestUploadsHandlerServesWithSafeHeaders(t *testing.T) {
	t.Parallel()

	app, dir := newUploadsApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "abc.html"), []byte("<script>1</script>"), 0o600))

	status, header, body := get(t, app, "/uploads/abc.html")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "<script>1</script>", body)
	assert.Contains(t, header.Get(fiber.HeaderContentSecurityPolicy), "sandbox")
	assert.Equal(t, "nosniff", header.Get(fiber.HeaderXContentTypeOptions))
	assert.Equal(t, "public, max-age=86400", header.Get(fiber.HeaderCacheControl))
}

func TestUploadsHandlerStopsServingDeletedFiles(t *testing.T) {
	t.Parallel()

	app, dir := newUploadsApp(t)
	path := filepath.Join(dir, "abc.png")
	require.NoError(t, os.WriteFile(path, []byte("png"), 0o600))

	status, _, _ := get(t, app, "/uploads/abc.png")
	require.Equal(t, http.StatusOK, status)

	// With fasthttp's default file cache the handle stays open: the remove fails on Windows and
	// the file keeps being served elsewhere.
	require.NoError(t, os.Remove(path))

	status, header, _ := get(t, app, "/uploads/abc.png")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Empty(t, header.Get(fiber.HeaderCacheControl), "a 404 must not be cached")
}

func TestUploadsHandlerStaysInsideDir(t *testing.T) {
	t.Parallel()

	app, _ := newUploadsApp(t)

	for _, path := range []string{"/uploads/../secret.txt", "/uploads/..%2fsecret.txt", "/uploads/%2e%2e/secret.txt"} {
		_, _, body := get(t, app, path)
		assert.NotContains(t, body, "secret", path)
	}
}
