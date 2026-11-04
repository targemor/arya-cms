package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/user"
	"github.com/lucimux/arya/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

var errInternalServErr = errors.New("internal server error")

func newUserUseCase(t *testing.T) (usecase.User, *MockUserRepo) {
	t.Helper()

	ctrl := gomock.NewController(t)

	repo := NewMockUserRepo(ctrl)
	jwtManager := jwt.New("test-secret", time.Hour)
	useCase := user.New(repo, jwtManager)

	return useCase, repo
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("register success", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

		u, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123", entity.RoleAdmin)

		require.NoError(t, err)
		assert.NotEmpty(t, u.ID)
		assert.Equal(t, "testuser", u.Username)
		assert.Equal(t, "test@example.com", u.Email)
		assert.Equal(t, entity.RoleAdmin, u.Role)
	})

	t.Run("register duplicate", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(entity.ErrUserAlreadyExists)

		_, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123", entity.RoleAdmin)

		require.ErrorIs(t, err, entity.ErrUserAlreadyExists)
	})
}

func TestLoginSuccess(t *testing.T) {
	t.Parallel()

	uc, repo := newUserUseCase(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	storedUser := entity.User{
		ID: "user-id-123", Username: "testuser",
		Email: "test@example.com", PasswordHash: string(hash),
		Role: entity.RoleAdmin,
	}
	repo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(storedUser, nil)

	token, err := uc.Login(context.Background(), "test@example.com", "password123")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLoginFailures(t *testing.T) {
	t.Parallel()

	t.Run("login wrong password", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		require.NoError(t, err)

		storedUser := entity.User{
			ID: "user-id-123", Username: "testuser",
			Email: "test@example.com", PasswordHash: string(hash),
			Role: entity.RoleAdmin,
		}
		repo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(storedUser, nil)

		token, err := uc.Login(context.Background(), "test@example.com", "wrongpassword")

		require.ErrorIs(t, err, entity.ErrInvalidCredentials)
		assert.Empty(t, token)
	})

	t.Run("login user not found", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().GetByEmail(gomock.Any(), "notfound@example.com").Return(entity.User{}, entity.ErrUserNotFound)

		token, err := uc.Login(context.Background(), "notfound@example.com", "password123")

		require.ErrorIs(t, err, entity.ErrInvalidCredentials)
		assert.Empty(t, token)
	})

	t.Run("login repo failure is not reported as invalid credentials", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(entity.User{}, errInternalServErr)

		token, err := uc.Login(context.Background(), "test@example.com", "password123")

		require.ErrorIs(t, err, errInternalServErr)
		require.NotErrorIs(t, err, entity.ErrInvalidCredentials)
		assert.Empty(t, token)
	})
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	expectedUser := entity.User{
		ID:       "user-id-123",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleAdmin,
	}

	t.Run("get user success", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().GetByID(gomock.Any(), "user-id-123").Return(expectedUser, nil)

		u, err := uc.GetUser(context.Background(), "user-id-123")

		require.NoError(t, err)
		assert.Equal(t, expectedUser, u)
	})

	t.Run("get user not found", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().GetByID(gomock.Any(), "missing-id").Return(entity.User{}, entity.ErrUserNotFound)

		_, err := uc.GetUser(context.Background(), "missing-id")

		require.ErrorIs(t, err, entity.ErrUserNotFound)
	})
}

func TestUserListAndUpdateRole(t *testing.T) {
	t.Parallel()

	t.Run("list users", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		expected := []entity.User{{ID: "1", Username: "user1", Role: entity.RoleAdmin}}
		repo.EXPECT().List(gomock.Any()).Return(expected, nil)

		users, err := uc.List(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expected, users)
	})

	t.Run("update role valid", func(t *testing.T) {
		t.Parallel()

		uc, repo := newUserUseCase(t)
		repo.EXPECT().UpdateRole(gomock.Any(), "1", entity.RoleSuperAdmin).Return(nil)

		err := uc.UpdateRole(context.Background(), "1", entity.RoleSuperAdmin)
		require.NoError(t, err)
	})

	t.Run("update role invalid", func(t *testing.T) {
		t.Parallel()

		uc, _ := newUserUseCase(t)

		err := uc.UpdateRole(context.Background(), "1", entity.Role("invalid_role"))
		require.ErrorIs(t, err, entity.ErrInvalidRole)
	})
}
