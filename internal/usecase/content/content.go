package content

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// contentTypes resolves the content types currently being served. usecase.ContentType implements it.
type contentTypes interface {
	Get(ctx context.Context, projectSlug, tableName string) (entity.ContentType, error)
}

// UseCase -.
type UseCase struct {
	types   contentTypes
	repo    repo.ContentRepo
	schemas *schemaCache
}

// New returns a Content usecase instrumented with OpenTelemetry tracing spans.
func New(types contentTypes, r repo.ContentRepo) usecase.Content {
	return newTraced(&UseCase{
		types:   types,
		repo:    r,
		schemas: newSchemaCache(),
	})
}

// List -.
func (uc *UseCase) List(
	ctx context.Context, projectSlug, tableName string, q entity.ContentQuery,
) ([]entity.Entry, entity.Pagination, error) {
	ct, err := uc.types.Get(ctx, projectSlug, tableName)
	if err != nil {
		return nil, entity.Pagination{}, fmt.Errorf("ContentUseCase - List - uc.types.Get: %w", err)
	}

	q, err = ct.ResolveQuery(q)
	if err != nil {
		return nil, entity.Pagination{}, fmt.Errorf("ContentUseCase - List - ct.ResolveQuery: %w", err)
	}

	entries, total, err := uc.repo.List(ctx, projectSlug, ct, q)
	if err != nil {
		return nil, entity.Pagination{}, fmt.Errorf("ContentUseCase - List - uc.repo.List: %w", err)
	}

	return entries, entity.NewPagination(q.Page, q.PageSize, total), nil
}

// Get -.
func (uc *UseCase) Get(ctx context.Context, projectSlug, tableName, id string) (entity.Entry, error) {
	ct, err := uc.entryType(ctx, projectSlug, tableName, id)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Get - uc.entryType: %w", err)
	}

	entry, err := uc.repo.Get(ctx, projectSlug, ct, id)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Get - uc.repo.Get: %w", err)
	}

	return entry, nil
}

// GetSingle returns the single entry of a single type.
func (uc *UseCase) GetSingle(ctx context.Context, projectSlug, tableName string) (entity.Entry, error) {
	ct, err := uc.types.Get(ctx, projectSlug, tableName)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - GetSingle - uc.types.Get: %w", err)
	}

	entry, err := uc.repo.GetSingle(ctx, projectSlug, ct)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - GetSingle - uc.repo.GetSingle: %w", err)
	}

	return entry, nil
}

// Create validates data against the content type's JSON Schema, required fields included.
func (uc *UseCase) Create(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error) {
	entry, err := uc.saveEntry(ctx, projectSlug, tableName, data, false, uc.repo.Create)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Create - uc.saveEntry: %w", err)
	}

	return entry, nil
}

// Update is a partial update, as in Strapi: fields left out keep their value.
func (uc *UseCase) Update(ctx context.Context, projectSlug, tableName, id string, data entity.Entry) (entity.Entry, error) {
	ct, err := uc.entryType(ctx, projectSlug, tableName, id)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Update - uc.entryType: %w", err)
	}

	values, err := uc.validate(ct, data, true)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Update - uc.validate: %w", err)
	}

	entry, err := uc.repo.Update(ctx, projectSlug, ct, id, values)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - Update - uc.repo.Update: %w", err)
	}

	return entry, nil
}

// UpdateSingle updates or creates the single entry of a single type.
func (uc *UseCase) UpdateSingle(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error) {
	entry, err := uc.saveEntry(ctx, projectSlug, tableName, data, true, uc.repo.UpdateSingle)
	if err != nil {
		return nil, fmt.Errorf("ContentUseCase - UpdateSingle - uc.saveEntry: %w", err)
	}

	return entry, nil
}

func (uc *UseCase) saveEntry(
	ctx context.Context,
	projectSlug, tableName string,
	data entity.Entry,
	partial bool,
	save func(context.Context, string, entity.ContentType, entity.Entry) (entity.Entry, error),
) (entity.Entry, error) {
	ct, err := uc.types.Get(ctx, projectSlug, tableName)
	if err != nil {
		return nil, fmt.Errorf("uc.types.Get: %w", err)
	}

	values, err := uc.validate(ct, data, partial)
	if err != nil {
		return nil, err
	}

	entry, err := save(ctx, projectSlug, ct, values)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// Delete -.
func (uc *UseCase) Delete(ctx context.Context, projectSlug, tableName, id string) error {
	ct, err := uc.entryType(ctx, projectSlug, tableName, id)
	if err != nil {
		return fmt.Errorf("ContentUseCase - Delete - uc.entryType: %w", err)
	}

	if err = uc.repo.Delete(ctx, projectSlug, ct, id); err != nil {
		return fmt.Errorf("ContentUseCase - Delete - uc.repo.Delete: %w", err)
	}

	return nil
}

// entryType resolves the content type of an entry addressed by id. An id that is not a UUID
// cannot exist, so it is reported as not found instead of reaching Postgres.
func (uc *UseCase) entryType(ctx context.Context, projectSlug, tableName, id string) (entity.ContentType, error) {
	ct, err := uc.types.Get(ctx, projectSlug, tableName)
	if err != nil {
		return entity.ContentType{}, fmt.Errorf("uc.types.Get: %w", err)
	}

	if uuid.Validate(id) != nil {
		return entity.ContentType{}, entity.ErrEntryNotFound
	}

	return ct, nil
}

// validate checks data against the JSON Schema and converts it to the values stored in the table.
func (uc *UseCase) validate(ct entity.ContentType, data entity.Entry, partial bool) (entity.Entry, error) {
	sch, err := uc.schemas.get(ct, partial)
	if err != nil {
		return nil, err
	}

	if err = sch.Validate(map[string]any(data)); err != nil {
		return nil, fmt.Errorf("%w: %w", entity.ErrInvalidEntry, err)
	}

	values := make(entity.Entry, len(data))

	for field, v := range data {
		// The schema already matched the JSON type; integers still arrive as json.Number.
		if n, ok := v.(json.Number); ok {
			i, err := n.Int64()
			if err != nil {
				return nil, fmt.Errorf("%w: %q: %w", entity.ErrInvalidEntry, field, err)
			}

			v = i
		}

		values[field] = v
	}

	return values, nil
}
