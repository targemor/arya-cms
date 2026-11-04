package jwt_test

import (
	"testing"
	"time"

	"github.com/lucimux/arya/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_GenerateAndParse(t *testing.T) {
	t.Parallel()

	j := jwt.New("test-secret", time.Hour)

	token, err := j.GenerateToken("user-123", "superadmin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := j.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "superadmin", claims.Role)
}

func TestJWT_ParseToken_Invalid(t *testing.T) {
	t.Parallel()

	j := jwt.New("test-secret", time.Hour)

	_, err := j.ParseToken("invalid-token")
	require.Error(t, err)
}

func TestJWT_ParseToken_WrongSecret(t *testing.T) {
	t.Parallel()

	j1 := jwt.New("secret-1", time.Hour)
	j2 := jwt.New("secret-2", time.Hour)

	token, err := j1.GenerateToken("user-123", "admin")
	require.NoError(t, err)

	_, err = j2.ParseToken(token)
	require.Error(t, err)
}

func TestJWT_ParseToken_Expired(t *testing.T) {
	t.Parallel()

	j := jwt.New("test-secret", -time.Hour)

	token, err := j.GenerateToken("user-123", "editor")
	require.NoError(t, err)

	_, err = j.ParseToken(token)
	require.Error(t, err)
}
