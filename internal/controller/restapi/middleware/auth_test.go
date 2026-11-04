package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/middleware"
	"github.com/lucimux/arya/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) (*fiber.App, *jwt.Manager) {
	t.Helper()

	jwtManager := jwt.New("test-secret", time.Hour)

	app := fiber.New()
	app.Use(middleware.Auth(jwtManager))
	app.Get("/test", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(string)
		if !ok {
			return c.SendStatus(http.StatusUnauthorized)
		}

		return c.SendString(userID)
	})
	app.Get("/superadmin-only", middleware.RequireRole("superadmin"), func(c *fiber.Ctx) error {
		return c.SendString("superadmin-content")
	})

	return app, jwtManager
}

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	app, jwtManager := newTestApp(t)

	validToken, err := jwtManager.GenerateToken("user-id-123", "admin")
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid format",
			authHeader:     "Basic xxx",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedBody:   "user-id-123",
		},
	}

	for _, tc := range tests {
		localTc := tc

		t.Run(localTc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
			if localTc.authHeader != "" {
				req.Header.Set("Authorization", localTc.authHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, localTc.expectedStatus, resp.StatusCode)

			if localTc.expectedBody != "" {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.Equal(t, localTc.expectedBody, string(body))
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	app, jwtManager := newTestApp(t)

	adminToken, err := jwtManager.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	superAdminToken, err := jwtManager.GenerateToken("user-2", "superadmin")
	require.NoError(t, err)

	t.Run("admin forbidden from superadmin route", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/superadmin-only", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("superadmin allowed", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/superadmin-only", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "superadmin-content", string(body))
	})
}
