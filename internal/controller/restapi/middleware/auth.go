package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/pkg/jwt"
)

const (
	_bearerParts  = 2
	_apiKeyPrefix = "cms_"
)

// APIKeyValidator validates a raw API key and returns the associated APIKey entity.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (entity.APIKey, error)
}

type errorResponse struct {
	Error string `json:"error"`
}

func extractRawAPIKey(ctx *fiber.Ctx) string {
	if k := strings.TrimSpace(ctx.Get("X-API-Key")); k != "" {
		return k
	}

	header := ctx.Get("Authorization")
	parts := strings.SplitN(header, " ", _bearerParts)

	if len(parts) == _bearerParts && parts[0] == "Bearer" && strings.HasPrefix(parts[1], _apiKeyPrefix) {
		return parts[1]
	}

	return ""
}

func handleAPIKeyAuth(ctx *fiber.Ctx, validator APIKeyValidator, rawKey string) error {
	if validator == nil {
		return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "invalid or expired token"})
	}

	if !strings.Contains(ctx.Path(), "/content/") {
		return ctx.Status(http.StatusForbidden).JSON(errorResponse{Error: "forbidden: api keys can only access content endpoints"})
	}

	apiKey, err := validator.ValidateAPIKey(ctx.UserContext(), rawKey)
	if err != nil {
		return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "invalid or revoked api key"})
	}

	ctx.Locals("apiKey", apiKey)

	return ctx.Next()
}

// Auth returns a JWT / API Key authentication middleware for Fiber.
func Auth(jwtManager *jwt.Manager, validators ...APIKeyValidator) func(*fiber.Ctx) error {
	var validator APIKeyValidator
	if len(validators) > 0 {
		validator = validators[0]
	}

	return func(ctx *fiber.Ctx) error {
		if rawKey := extractRawAPIKey(ctx); rawKey != "" {
			return handleAPIKeyAuth(ctx, validator, rawKey)
		}

		header := ctx.Get("Authorization")
		if header == "" {
			return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "missing authorization header"})
		}

		parts := strings.SplitN(header, " ", _bearerParts)
		if len(parts) != _bearerParts || parts[0] != "Bearer" {
			return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "invalid authorization header format"})
		}

		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "invalid or expired token"})
		}

		ctx.Locals("userID", claims.Subject)
		ctx.Locals("userRole", claims.Role)

		return ctx.Next()
	}
}

// RequireRole verifies that the authenticated user has one of the required roles.
func RequireRole(roles ...string) func(*fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		userRole, ok := ctx.Locals("userRole").(string)
		if !ok || userRole == "" {
			return ctx.Status(http.StatusUnauthorized).JSON(errorResponse{Error: "unauthorized"})
		}

		for _, r := range roles {
			if r == userRole {
				return ctx.Next()
			}
		}

		return ctx.Status(http.StatusForbidden).JSON(errorResponse{Error: "forbidden: insufficient permissions"})
	}
}
