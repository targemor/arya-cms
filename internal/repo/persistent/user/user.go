// Package user implements the Postgres-backed User repository.
package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/pkg/postgres"
)

// Repo -.
type Repo struct {
	*postgres.Postgres
}

// New returns a User repository instrumented with OpenTelemetry tracing spans.
func New(pg *postgres.Postgres) repo.UserRepo {
	return newTraced(&Repo{pg})
}

// Store -.
func (r *Repo) Store(ctx context.Context, user *entity.User) error {
	role := string(user.Role)
	if role == "" {
		role = string(entity.RoleAdmin)
	}

	sql, args, err := r.Builder.
		Insert("users").
		Columns("id, username, email, role, password_hash, created_at, updated_at").
		Values(user.ID, user.Username, user.Email, role, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Store - r.Builder: %w", err)
	}

	_, err = r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return entity.ErrUserAlreadyExists
		}

		return fmt.Errorf("UserRepo - Store - r.Pool.Exec: %w", err)
	}

	return nil
}

// GetByID -.
func (r *Repo) GetByID(ctx context.Context, id string) (entity.User, error) {
	return r.getUser(ctx, "id", id)
}

// GetByEmail -.
func (r *Repo) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	return r.getUser(ctx, "email", email)
}

func (r *Repo) getUser(ctx context.Context, column, value string) (entity.User, error) {
	sql, args, err := r.Builder.
		Select("id, username, email, role, password_hash, created_at, updated_at").
		From("users").
		Where(sq.Eq{column: value}).
		ToSql()
	if err != nil {
		return entity.User{}, fmt.Errorf("UserRepo - getUser - r.Builder: %w", err)
	}

	var user entity.User

	err = r.Pool.QueryRow(ctx, sql, args...).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, entity.ErrUserNotFound
		}

		return entity.User{}, fmt.Errorf("UserRepo - getUser - r.Pool.QueryRow: %w", err)
	}

	return user, nil
}

// List returns all users sorted by created_at.
func (r *Repo) List(ctx context.Context) ([]entity.User, error) {
	sql, args, err := r.Builder.
		Select("id, username, email, role, password_hash, created_at, updated_at").
		From("users").
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Builder: %w", err)
	}

	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	var users []entity.User

	for rows.Next() {
		var u entity.User

		err = rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("UserRepo - List - rows.Scan: %w", err)
		}

		users = append(users, u)
	}

	return users, nil
}

// UpdateRole updates a user's role.
func (r *Repo) UpdateRole(ctx context.Context, id string, role entity.Role) error {
	sql, args, err := r.Builder.
		Update("users").
		Set("role", string(role)).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateRole - r.Builder: %w", err)
	}

	res, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateRole - r.Pool.Exec: %w", err)
	}

	if res.RowsAffected() == 0 {
		return entity.ErrUserNotFound
	}

	return nil
}
