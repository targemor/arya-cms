package contenttype

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// UseCase -.
type UseCase struct {
	store    repo.ContentTypeStore
	migrator repo.ContentTypeMigrator

	// mu serializes Create and Sync.
	mu sync.Mutex

	// active holds the content types whose tables are known to be up to date, keyed by "projectSlug/tableName".
	activeMu sync.RWMutex
	active   map[string]entity.ContentType
}

func projectKey(projectSlug, tableName string) string {
	if projectSlug == "" {
		projectSlug = "default"
	}

	return projectSlug + "/" + tableName
}

// New returns a ContentType usecase instrumented with OpenTelemetry tracing spans.
func New(s repo.ContentTypeStore, m repo.ContentTypeMigrator) usecase.ContentType {
	return newTraced(&UseCase{
		store:    s,
		migrator: m,
		active:   make(map[string]entity.ContentType),
	})
}

// Create migrates the table first and saves the definition second.
func (uc *UseCase) Create(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error) {
	if err := ct.Validate(); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Create - ct.Validate: %w", err)
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	exists, err := uc.store.Exists(ctx, projectSlug, ct.TableName)
	if err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Create - uc.store.Exists: %w", err)
	}

	if exists {
		return entity.ContentType{}, entity.ErrContentTypeAlreadyExists
	}

	if err = uc.migrator.Migrate(ctx, projectSlug, ct); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Create - uc.migrator.Migrate: %w", err)
	}

	if err = uc.store.Save(ctx, projectSlug, ct); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Create - uc.store.Save: %w", err)
	}

	uc.activeMu.Lock()
	uc.active[projectKey(projectSlug, ct.TableName)] = ct
	uc.activeMu.Unlock()

	return ct, nil
}

// Update updates an existing content type's definition and migrates its table.
func (uc *UseCase) Update(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error) {
	if err := ct.Validate(); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Update - ct.Validate: %w", err)
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	exists, err := uc.store.Exists(ctx, projectSlug, ct.TableName)
	if err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Update - uc.store.Exists: %w", err)
	}

	if !exists {
		return entity.ContentType{}, entity.ErrContentTypeNotFound
	}

	if err = uc.migrator.Migrate(ctx, projectSlug, ct); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Update - uc.migrator.Migrate: %w", err)
	}

	if err = uc.store.Save(ctx, projectSlug, ct); err != nil {
		return entity.ContentType{}, fmt.Errorf("ContentTypeUseCase - Update - uc.store.Save: %w", err)
	}

	uc.activeMu.Lock()
	uc.active[projectKey(projectSlug, ct.TableName)] = ct
	uc.activeMu.Unlock()

	return ct, nil
}

// List -.
func (uc *UseCase) List(ctx context.Context, projectSlug string) ([]entity.ContentType, error) {
	cts, err := uc.store.List(ctx, projectSlug)
	if err != nil {
		return nil, fmt.Errorf("ContentTypeUseCase - List - uc.store.List: %w", err)
	}

	return cts, nil
}

// Get returns an active content type.
func (uc *UseCase) Get(_ context.Context, projectSlug, tableName string) (entity.ContentType, error) {
	uc.activeMu.RLock()
	defer uc.activeMu.RUnlock()

	ct, ok := uc.active[projectKey(projectSlug, tableName)]
	if !ok {
		return entity.ContentType{}, entity.ErrContentTypeNotFound
	}

	return ct, nil
}

// Sync migrates every stored definition across all projects.
func (uc *UseCase) Sync(ctx context.Context) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	projects, err := uc.store.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("ContentTypeUseCase - Sync - uc.store.ListProjects: %w", err)
	}

	if len(projects) == 0 {
		projects = []string{"default"}
	}

	var syncErrors []error

	active := make(map[string]entity.ContentType)

	for _, p := range projects {
		stored, listErr := uc.store.List(ctx, p)
		if listErr != nil {
			syncErrors = append(syncErrors, fmt.Errorf("project %s: %w", p, listErr))
		}

		for _, ct := range stored {
			if migErr := uc.migrator.Migrate(ctx, p, ct); migErr != nil {
				syncErrors = append(syncErrors, fmt.Errorf("project %s, table %s: %w", p, ct.TableName, migErr))

				continue
			}

			active[projectKey(p, ct.TableName)] = ct
		}
	}

	uc.activeMu.Lock()
	uc.active = active
	uc.activeMu.Unlock()

	return errors.Join(syncErrors...)
}
