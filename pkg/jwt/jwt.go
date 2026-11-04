package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// ErrUnexpectedSigningMethod is returned when the JWT signing method is not expected.
var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidClaims           = errors.New("invalid claims")
)

// Claims represents the JWT payload including user ID and role.
type Claims struct {
	jwtlib.RegisteredClaims
	Role string `json:"role"`
}

// Manager handles JWT token generation and parsing.
type Manager struct {
	secret   string
	duration time.Duration
}

// New -.
func New(secret string, duration time.Duration) *Manager {
	return &Manager{
		secret:   secret,
		duration: duration,
	}
}

// GenerateToken creates a new JWT token for the given user ID and role.
func (m *Manager) GenerateToken(userID, role string) (string, error) {
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(m.duration)),
		},
		Role: role,
	})

	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("jwt - GenerateToken - token.SignedString: %w", err)
	}

	return tokenString, nil
}

// ParseToken validates a JWT token and returns the parsed Claims.
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(token *jwtlib.Token) (any, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
		}

		return []byte(m.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt - ParseToken - jwtlib.ParseWithClaims: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt - ParseToken: %w", ErrInvalidClaims)
	}

	return claims, nil
}
