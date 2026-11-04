// Package project implements the Postgres-backed project repository (projects).
package project

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/pkg/postgres"
)

const (
	_table = "projects"
	// _columns is the full row, in the order scanRow reads it.
	_columns = "id, name, slug, description, image_url, created_at, updated_at"

	// Postgres error codes.
	_uniqueViolation = "23505"
)

// Repo -.
type Repo struct {
	*postgres.Postgres
}

// New returns a Project repository instrumented with OpenTelemetry tracing spans.
func New(pg *postgres.Postgres) repo.ProjectRepo {
	return newTraced(&Repo{pg})
}

// Create -.
func (r *Repo) Create(ctx context.Context, p *entity.Project) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	sql, args, err := r.Builder.Insert(_table).
		Columns("id, name, slug, description, image_url").
		Values(p.ID, p.Name, p.Slug, p.Description, p.ImageURL).
		Suffix("RETURNING " + _columns).
		ToSql()
	if err != nil {
		return fmt.Errorf("ProjectRepo - Create - r.Builder: %w", err)
	}

	created, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		return writeError("Create", err)
	}

	*p = created

	return nil
}

// GetByID -.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Project, error) {
	sql, args, err := r.Builder.Select(_columns).
		From(_table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - GetByID - r.Builder: %w", err)
	}

	p, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrProjectNotFound
		}

		return nil, fmt.Errorf("ProjectRepo - GetByID - r.Pool.QueryRow: %w", err)
	}

	return &p, nil
}

// GetBySlug -.
func (r *Repo) GetBySlug(ctx context.Context, slug string) (*entity.Project, error) {
	sql, args, err := r.Builder.Select(_columns).
		From(_table).
		Where(sq.Eq{"slug": slug}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - GetBySlug - r.Builder: %w", err)
	}

	p, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrProjectNotFound
		}

		return nil, fmt.Errorf("ProjectRepo - GetBySlug - r.Pool.QueryRow: %w", err)
	}

	return &p, nil
}

// List returns all projects sorted by name.
func (r *Repo) List(ctx context.Context) ([]entity.Project, error) {
	sql, args, err := r.Builder.Select(_columns).
		From(_table).
		OrderBy("name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - List - r.Builder: %w", err)
	}

	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	var projects []entity.Project

	for rows.Next() {
		p, scanErr := scanRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ProjectRepo - List - scanRow: %w", scanErr)
		}

		projects = append(projects, p)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("ProjectRepo - List - rows.Err: %w", rows.Err())
	}

	return projects, nil
}

// Update updates name, description, and image URL.
func (r *Repo) Update(ctx context.Context, p *entity.Project) error {
	sql, args, err := r.Builder.Update(_table).
		Set("name", p.Name).
		Set("description", p.Description).
		Set("image_url", p.ImageURL).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": p.ID}).
		Suffix("RETURNING " + _columns).
		ToSql()
	if err != nil {
		return fmt.Errorf("ProjectRepo - Update - r.Builder: %w", err)
	}

	updated, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.ErrProjectNotFound
		}

		return writeError("Update", err)
	}

	*p = updated

	return nil
}

// Delete removes the project by ID.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	sql, args, err := r.Builder.Delete(_table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("ProjectRepo - Delete - r.Builder: %w", err)
	}

	tag, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("ProjectRepo - Delete - r.Pool.Exec: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrProjectNotFound
	}

	return nil
}

// ListForUser returns all projects a user belongs to, sorted by name.
func (r *Repo) ListForUser(ctx context.Context, userID string) ([]entity.Project, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - ListForUser - uuid.Parse: %w", err)
	}

	query := `
		SELECT p.id, p.name, p.slug, p.description, p.image_url, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1
		ORDER BY p.name ASC`

	rows, err := r.Pool.Query(ctx, query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - ListForUser - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	var projects []entity.Project

	for rows.Next() {
		p, scanErr := scanRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ProjectRepo - ListForUser - scanRow: %w", scanErr)
		}

		projects = append(projects, p)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("ProjectRepo - ListForUser - rows.Err: %w", rows.Err())
	}

	return projects, nil
}

// ListMembers returns all members of a project with user details.
func (r *Repo) ListMembers(ctx context.Context, projectID uuid.UUID) ([]entity.ProjectMember, error) {
	query := `
		SELECT pm.project_id, pm.user_id, u.username, u.email, pm.role, pm.created_at, pm.updated_at
		FROM project_members pm
		INNER JOIN users u ON pm.user_id = u.id
		WHERE pm.project_id = $1
		ORDER BY u.username ASC`

	rows, err := r.Pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("ProjectRepo - ListMembers - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	var members []entity.ProjectMember

	for rows.Next() {
		var (
			m      entity.ProjectMember
			userID uuid.UUID
		)

		if err := rows.Scan(&m.ProjectID, &userID, &m.Username, &m.Email, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ProjectRepo - ListMembers - rows.Scan: %w", err)
		}

		m.UserID = userID.String()
		members = append(members, m)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("ProjectRepo - ListMembers - rows.Err: %w", rows.Err())
	}

	return members, nil
}

// AddOrUpdateMember adds or updates a member's role in a project.
func (r *Repo) AddOrUpdateMember(ctx context.Context, projectID uuid.UUID, userID string, role entity.Role) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("ProjectRepo - AddOrUpdateMember - uuid.Parse: %w", err)
	}

	query := `
		INSERT INTO project_members (project_id, user_id, role, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (project_id, user_id)
		DO UPDATE SET role = EXCLUDED.role, updated_at = now()`

	if _, err := r.Pool.Exec(ctx, query, projectID, userUUID, string(role)); err != nil {
		return fmt.Errorf("ProjectRepo - AddOrUpdateMember - r.Pool.Exec: %w", err)
	}

	return nil
}

// RemoveMember deletes a member from a project.
func (r *Repo) RemoveMember(ctx context.Context, projectID uuid.UUID, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("ProjectRepo - RemoveMember - uuid.Parse: %w", err)
	}

	query := `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`

	if _, err := r.Pool.Exec(ctx, query, projectID, userUUID); err != nil {
		return fmt.Errorf("ProjectRepo - RemoveMember - r.Pool.Exec: %w", err)
	}

	return nil
}

// GetUserRoleInProject returns the role of a user in a given project.
func (r *Repo) GetUserRoleInProject(ctx context.Context, projectID uuid.UUID, userID string) (entity.Role, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("ProjectRepo - GetUserRoleInProject - uuid.Parse: %w", err)
	}

	query := `SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`

	var role string
	if err := r.Pool.QueryRow(ctx, query, projectID, userUUID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", entity.ErrUserNotFound
		}

		return "", fmt.Errorf("ProjectRepo - GetUserRoleInProject - r.Pool.QueryRow: %w", err)
	}

	return entity.Role(role), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRow(s rowScanner) (entity.Project, error) {
	var (
		p           entity.Project
		description *string
		imageURL    *string
	)

	if err := s.Scan(&p.ID, &p.Name, &p.Slug, &description, &imageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return entity.Project{}, err
	}

	if description != nil {
		p.Description = *description
	}

	if imageURL != nil {
		p.ImageURL = *imageURL
	}

	return p, nil
}

func writeError(op string, err error) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == _uniqueViolation {
		return entity.ErrProjectAlreadyExists
	}

	return fmt.Errorf("ProjectRepo - %s: %w", op, err)
}
