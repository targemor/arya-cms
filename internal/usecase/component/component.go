package component

import (
	"context"
	"fmt"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// UseCase -.
type UseCase struct {
	store repo.ComponentStore
}

// New returns a Component usecase instrumented with OpenTelemetry tracing spans.
func New(s repo.ComponentStore) usecase.Component {
	return newTraced(&UseCase{store: s})
}

// Create -.
func (uc *UseCase) Create(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Create - c.Validate: %w", err)
	}

	exists, err := uc.store.Exists(ctx, projectSlug, c.UID())
	if err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Create - uc.store.Exists: %w", err)
	}

	if exists {
		return nil, entity.ErrComponentAlreadyExists
	}

	if err = uc.store.Save(ctx, projectSlug, c); err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Create - uc.store.Save: %w", err)
	}

	return c, nil
}

// Update -.
func (uc *UseCase) Update(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Update - c.Validate: %w", err)
	}

	exists, err := uc.store.Exists(ctx, projectSlug, c.UID())
	if err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Update - uc.store.Exists: %w", err)
	}

	if !exists {
		return nil, entity.ErrComponentNotFound
	}

	if err = uc.store.Save(ctx, projectSlug, c); err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Update - uc.store.Save: %w", err)
	}

	return c, nil
}

// List -.
func (uc *UseCase) List(ctx context.Context, projectSlug string) ([]entity.Component, error) {
	comps, err := uc.store.List(ctx, projectSlug)
	if err != nil {
		return nil, fmt.Errorf("ComponentUseCase - List - uc.store.List: %w", err)
	}

	return comps, nil
}

// Get -.
func (uc *UseCase) Get(ctx context.Context, projectSlug, uid string) (*entity.Component, error) {
	c, err := uc.store.Get(ctx, projectSlug, uid)
	if err != nil {
		return nil, fmt.Errorf("ComponentUseCase - Get - uc.store.Get: %w", err)
	}

	return c, nil
}

// Delete -.
func (uc *UseCase) Delete(ctx context.Context, projectSlug, uid string) error {
	if err := uc.store.Delete(ctx, projectSlug, uid); err != nil {
		return fmt.Errorf("ComponentUseCase - Delete - uc.store.Delete: %w", err)
	}

	return nil
}
