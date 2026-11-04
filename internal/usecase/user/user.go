package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// _dummyPasswordHash is a bcrypt hash (bcrypt.DefaultCost) of a throwaway password.
// It is not a credential: it only equalizes Login timing for unknown emails.
const _dummyPasswordHash = "$2a$10$D6IdEd1McpkNAuZ56rzAfutPTRBpHjuLJZid6OWl3O3wij5F8LnA2" //nolint:gosec // See above.

// UseCase -.
type UseCase struct {
	repo repo.UserRepo
	jwt  *jwt.Manager
}

// New returns a User usecase instrumented with OpenTelemetry tracing spans.
func New(r repo.UserRepo, j *jwt.Manager) usecase.User {
	return newTraced(&UseCase{
		repo: r,
		jwt:  j,
	})
}

// Register -.
func (uc *UseCase) Register(
	ctx context.Context, username, email, password string, role entity.Role,
) (entity.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - Register - bcrypt.GenerateFromPassword: %w", err)
	}

	now := time.Now().UTC()

	if !role.IsValid() {
		role = entity.RoleAdmin
	}

	user := entity.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = uc.repo.Store(ctx, &user)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - Register - uc.repo.Store: %w", err)
	}

	return user, nil
}

// Login -.
func (uc *UseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := uc.repo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, entity.ErrUserNotFound) {
		return "", fmt.Errorf("UserUseCase - Login - uc.repo.GetByEmail: %w", err)
	}

	userFound := err == nil

	// An unknown email is still checked against a dummy hash, so both paths pay the
	// bcrypt cost and response time does not reveal which emails are registered.
	hash := _dummyPasswordHash
	if userFound {
		hash = user.PasswordHash
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || !userFound {
		return "", entity.ErrInvalidCredentials
	}

	token, err := uc.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return "", fmt.Errorf("UserUseCase - Login - uc.jwt.GenerateToken: %w", err)
	}

	return token, nil
}

// GetUser -.
func (uc *UseCase) GetUser(ctx context.Context, userID string) (entity.User, error) {
	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - GetUser - uc.repo.GetByID: %w", err)
	}

	return user, nil
}

// List returns all users.
func (uc *UseCase) List(ctx context.Context) ([]entity.User, error) {
	users, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - List - uc.repo.List: %w", err)
	}

	return users, nil
}

// UpdateRole updates a user's role.
func (uc *UseCase) UpdateRole(ctx context.Context, id string, role entity.Role) error {
	if !role.IsValid() {
		return entity.ErrInvalidRole
	}

	if err := uc.repo.UpdateRole(ctx, id, role); err != nil {
		return fmt.Errorf("UserUseCase - UpdateRole - uc.repo.UpdateRole: %w", err)
	}

	return nil
}
