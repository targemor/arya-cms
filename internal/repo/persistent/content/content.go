// Package content implements the Postgres-backed store for the entries of user-defined content
// types. Tables and columns come from the content type, so every identifier is quoted.
package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/repo/persistent/contenttype"
	"github.com/lucimux/arya/pkg/postgres"
)

// _foreignKeyViolation is Postgres' foreign_key_violation: a media field naming a missing file.
const _foreignKeyViolation = "23503"

// Repo -.
type Repo struct {
	*postgres.Postgres
}

// New returns a Content repository instrumented with OpenTelemetry tracing spans.
func New(pg *postgres.Postgres) repo.ContentRepo {
	return newTraced(&Repo{pg})
}

// List -.
func (r *Repo) List(ctx context.Context, projectSlug string, ct entity.ContentType, q entity.ContentQuery) ([]entity.Entry, int, error) {
	table := contenttype.TableName(projectSlug, ct.TableName)

	where, err := whereClause(q.Filters)
	if err != nil {
		return nil, 0, fmt.Errorf("ContentRepo - List - whereClause: %w", err)
	}

	sql, args, err := r.Builder.Select("COUNT(*)").From(table + " AS " + _entryAlias).Where(where).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("ContentRepo - List - count r.Builder: %w", err)
	}

	var total int

	if err = r.Pool.QueryRow(ctx, sql, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ContentRepo - List - count r.Pool.QueryRow: %w", err)
	}

	// LIMIT/OFFSET go in as parameters rather than through Limit/Offset, which take uint64.
	sql, args, err = selectEntries(r.Builder, ct, table+" AS "+_entryAlias).Where(where).
		OrderBy(orderBy(q.Sort)...).
		Suffix("LIMIT ? OFFSET ?", q.PageSize, (q.Page-1)*q.PageSize).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("ContentRepo - List - r.Builder: %w", err)
	}

	entries, err := r.queryEntries(ctx, ct, sql, args)
	if err != nil {
		return nil, 0, fmt.Errorf("ContentRepo - List - r.queryEntries: %w", err)
	}

	return entries, total, nil
}

// Get -.
func (r *Repo) Get(ctx context.Context, projectSlug string, ct entity.ContentType, id string) (entity.Entry, error) {
	sql, args, err := selectEntries(r.Builder, ct, contenttype.TableName(projectSlug, ct.TableName)+" AS "+_entryAlias).
		Where(sq.Eq{qualified(entity.FieldID): id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - Get - r.Builder: %w", err)
	}

	return r.queryOne(ctx, ct, "Get", sql, args)
}

// GetSingle returns the single entry for a single type.
func (r *Repo) GetSingle(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.Entry, error) {
	sql, args, err := selectEntries(r.Builder, ct, contenttype.TableName(projectSlug, ct.TableName)+" AS "+_entryAlias).
		OrderBy(qualified(entity.FieldCreatedAt) + " ASC").
		Suffix("LIMIT 1").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - GetSingle - r.Builder: %w", err)
	}

	return r.queryOne(ctx, ct, "GetSingle", sql, args)
}

// Create -.
func (r *Repo) Create(ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry) (entity.Entry, error) {
	table := contenttype.TableName(projectSlug, ct.TableName)

	var (
		sql  string
		args []any
		err  error
	)

	if len(data) == 0 {
		// Squirrel cannot build an INSERT without columns; every attribute is optional.
		sql = "INSERT INTO " + table + " DEFAULT VALUES RETURNING *"
	} else {
		cols, colErr := columnValues(ct, data)
		if colErr != nil {
			return nil, fmt.Errorf("ContentRepo - Create - columnValues: %w", colErr)
		}

		sql, args, err = r.Builder.Insert(table).SetMap(cols).Suffix("RETURNING *").ToSql()
		if err != nil {
			return nil, fmt.Errorf("ContentRepo - Create - r.Builder: %w", err)
		}
	}

	sql, err = r.readBack(ct, sql)
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - Create - r.readBack: %w", err)
	}

	return r.queryOne(ctx, ct, "Create", sql, args)
}

// Update sets the given fields and leaves the others untouched.
func (r *Repo) Update(ctx context.Context, projectSlug string, ct entity.ContentType, id string, data entity.Entry) (entity.Entry, error) {
	cols, err := columnValues(ct, data)
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - Update - columnValues: %w", err)
	}

	sql, args, err := r.Builder.Update(contenttype.TableName(projectSlug, ct.TableName)).
		SetMap(cols).
		Set(column(entity.FieldUpdatedAt), sq.Expr("now()")).
		Where(sq.Eq{column(entity.FieldID): id}).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - Update - r.Builder: %w", err)
	}

	sql, err = r.readBack(ct, sql)
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - Update - r.readBack: %w", err)
	}

	return r.queryOne(ctx, ct, "Update", sql, args)
}

// UpdateSingle updates the single entry if it exists, or creates it if none exists yet.
func (r *Repo) UpdateSingle(ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry) (entity.Entry, error) {
	existing, err := r.GetSingle(ctx, projectSlug, ct)
	if err != nil {
		if errors.Is(err, entity.ErrEntryNotFound) {
			return r.Create(ctx, projectSlug, ct, data)
		}

		return nil, fmt.Errorf("ContentRepo - UpdateSingle - r.GetSingle: %w", err)
	}

	id, ok := existing[entity.FieldID].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing id in existing entry", entity.ErrEntryNotFound)
	}

	return r.Update(ctx, projectSlug, ct, id, data)
}

// Delete -.
func (r *Repo) Delete(ctx context.Context, projectSlug string, ct entity.ContentType, id string) error {
	sql, args, err := r.Builder.Delete(contenttype.TableName(projectSlug, ct.TableName)).
		Where(sq.Eq{column(entity.FieldID): id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("ContentRepo - Delete - r.Builder: %w", err)
	}

	tag, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("ContentRepo - Delete - r.Pool.Exec: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return entity.ErrEntryNotFound
	}

	return nil
}

// readBack wraps a write ending in RETURNING * so the row comes back in the same shape as a read,
// media fields joined: WITH "e" AS (<write>) SELECT ... FROM "e" LEFT JOIN upload_files ...
// The SELECT has no parameters, so the write's placeholders keep their numbers.
func (r *Repo) readBack(ct entity.ContentType, write string) (string, error) {
	sel, _, err := selectEntries(r.Builder, ct, _entryAlias).ToSql()
	if err != nil {
		return "", fmt.Errorf("selectEntries: %w", err)
	}

	return "WITH " + _entryAlias + " AS (" + write + ") " + sel, nil
}

func (r *Repo) queryEntries(ctx context.Context, ct entity.ContentType, sql string, args []any) ([]entity.Entry, error) {
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("r.Pool.Query: %w", err)
	}

	maps, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("pgx.CollectRows: %w", err)
	}

	entries := make([]entity.Entry, 0, len(maps))
	for _, m := range maps {
		unmarshalComponents(ct, m)
		entries = append(entries, m)
	}

	return entries, nil
}

func (r *Repo) queryOne(ctx context.Context, ct entity.ContentType, method, sql string, args []any) (entity.Entry, error) {
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("ContentRepo - %s - r.Pool.Query: %w", method, err)
	}

	m, err := pgx.CollectExactlyOneRow(rows, pgx.RowToMap)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrEntryNotFound
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == _foreignKeyViolation {
			return nil, fmt.Errorf("%w: media file does not exist", entity.ErrInvalidEntry)
		}

		return nil, fmt.Errorf("ContentRepo - %s - pgx.CollectExactlyOneRow: %w", method, err)
	}

	unmarshalComponents(ct, m)

	return m, nil
}

func unmarshalComponents(ct entity.ContentType, m map[string]any) {
	for _, a := range ct.Attributes {
		if a.Type != entity.AttributeTypeComponent {
			continue
		}

		b, ok := m[a.Name].([]byte)
		if !ok {
			continue
		}

		var val any
		if json.Unmarshal(b, &val) == nil {
			m[a.Name] = val
		}
	}
}
