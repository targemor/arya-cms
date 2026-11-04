// Package folder implements the Postgres-backed Media Library folder repository (upload_folders).
package folder

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
	_table = "upload_folders"
	// _columns is the full row, in the order scanRow reads it.
	_columns = "id, name, parent_id, created_at, updated_at"

	// Postgres error codes.
	_uniqueViolation     = "23505" // same name in the same parent
	_foreignKeyViolation = "23503" // missing parent, or deleting a folder that is still referenced
)

// Repo -.
type Repo struct {
	*postgres.Postgres
}

// New returns a Folder repository instrumented with OpenTelemetry tracing spans.
func New(pg *postgres.Postgres) repo.FolderRepo {
	return newTraced(&Repo{pg})
}

// Create -.
func (r *Repo) Create(ctx context.Context, folder *entity.Folder) error {
	sql, args, err := r.Builder.Insert(_table).
		Columns("name, parent_id").
		Values(folder.Name, folder.ParentID).
		Suffix("RETURNING " + _columns).
		ToSql()
	if err != nil {
		return fmt.Errorf("FolderRepo - Create - r.Builder: %w", err)
	}

	created, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		return writeError("Create", err)
	}

	*folder = created

	return nil
}

// GetByID -.
func (r *Repo) GetByID(ctx context.Context, id string) (entity.Folder, error) {
	if uuid.Validate(id) != nil {
		return entity.Folder{}, entity.ErrFolderNotFound
	}

	sql, args, err := r.Builder.Select(_columns).From(_table).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return entity.Folder{}, fmt.Errorf("FolderRepo - GetByID - r.Builder: %w", err)
	}

	folder, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Folder{}, entity.ErrFolderNotFound
		}

		return entity.Folder{}, fmt.Errorf("FolderRepo - GetByID - r.Pool.QueryRow: %w", err)
	}

	return folder, nil
}

// List -.
func (r *Repo) List(ctx context.Context) ([]entity.Folder, error) {
	sql, args, err := r.Builder.Select(_columns).From(_table).OrderBy("name", "id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("FolderRepo - List - r.Builder: %w", err)
	}

	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("FolderRepo - List - r.Pool.Query: %w", err)
	}

	folders, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Folder, error) {
		return scanRow(row)
	})
	if err != nil {
		return nil, fmt.Errorf("FolderRepo - List - pgx.CollectRows: %w", err)
	}

	return folders, nil
}

// Update -.
func (r *Repo) Update(ctx context.Context, folder *entity.Folder) error {
	sql, args, err := r.Builder.Update(_table).
		Set("name", folder.Name).
		Set("parent_id", folder.ParentID).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": folder.ID}).
		Suffix("RETURNING " + _columns).
		ToSql()
	if err != nil {
		return fmt.Errorf("FolderRepo - Update - r.Builder: %w", err)
	}

	updated, err := scanRow(r.Pool.QueryRow(ctx, sql, args...))
	if err != nil {
		return writeError("Update", err)
	}

	*folder = updated

	return nil
}

// Delete -.
func (r *Repo) Delete(ctx context.Context, id string) error {
	if uuid.Validate(id) != nil {
		return entity.ErrFolderNotFound
	}

	sql, args, err := r.Builder.Delete(_table).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("FolderRepo - Delete - r.Builder: %w", err)
	}

	tag, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		// ON DELETE RESTRICT: something was added to the folder after the use case checked it.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == _foreignKeyViolation {
			return entity.ErrFolderNotEmpty
		}

		return fmt.Errorf("FolderRepo - Delete - r.Pool.Exec: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrFolderNotFound
	}

	return nil
}

// writeError maps the errors an insert or update can hit to sentinels.
func writeError(method string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrFolderNotFound
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case _uniqueViolation:
			return entity.ErrFolderAlreadyExists
		case _foreignKeyViolation:
			// The parent was deleted between the use case's check and this write.
			return entity.ErrFolderNotFound
		}
	}

	return fmt.Errorf("FolderRepo - %s - r.Pool.QueryRow: %w", method, err)
}

// scanRow reads one row of _columns. Errors are returned as is so callers can match pgx.ErrNoRows.
func scanRow(row pgx.Row) (entity.Folder, error) {
	var f entity.Folder

	err := row.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedAt, &f.UpdatedAt)

	return f, err
}
