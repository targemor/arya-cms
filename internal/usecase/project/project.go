package project

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/internal/usecase"
)

// UseCase -.
type UseCase struct {
	repo repo.ProjectRepo
}

// New returns a Project usecase instrumented with OpenTelemetry tracing spans.
func New(r repo.ProjectRepo) usecase.Project {
	return newTraced(&UseCase{repo: r})
}

// Create -.
func (uc *UseCase) Create(ctx context.Context, name, slug, description, imageURL string) (entity.Project, error) {
	p := entity.Project{
		ID:          uuid.New(),
		Name:        name,
		Slug:        slug,
		Description: description,
		ImageURL:    imageURL,
	}

	if err := p.Validate(); err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - Create - p.Validate: %w", err)
	}

	if err := uc.repo.Create(ctx, &p); err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - Create - uc.repo.Create: %w", err)
	}

	return p, nil
}

// GetBySlug -.
func (uc *UseCase) GetBySlug(ctx context.Context, slug string) (entity.Project, error) {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - GetBySlug - uc.repo.GetBySlug: %w", err)
	}

	return *p, nil
}

// List -.
func (uc *UseCase) List(ctx context.Context) ([]entity.Project, error) {
	projects, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - List - uc.repo.List: %w", err)
	}

	return projects, nil
}

// ListForUser returns projects accessible to the user.
func (uc *UseCase) ListForUser(ctx context.Context, userID string, isSuperAdmin bool) ([]entity.Project, error) {
	if isSuperAdmin {
		return uc.List(ctx)
	}

	projects, err := uc.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - ListForUser - uc.repo.ListForUser: %w", err)
	}

	return projects, nil
}

// ListMembers returns all members assigned to a project.
func (uc *UseCase) ListMembers(ctx context.Context, slug string) ([]entity.ProjectMember, error) {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - ListMembers - uc.repo.GetBySlug: %w", err)
	}

	members, err := uc.repo.ListMembers(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("ProjectUseCase - ListMembers - uc.repo.ListMembers: %w", err)
	}

	return members, nil
}

// AddMember assigns or updates a user's role in a project.
func (uc *UseCase) AddMember(ctx context.Context, slug, userID string, role entity.Role) error {
	if !role.IsValid() {
		return fmt.Errorf("%w: invalid role %s", entity.ErrInvalidRole, role)
	}

	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("ProjectUseCase - AddMember - uc.repo.GetBySlug: %w", err)
	}

	if err = uc.repo.AddOrUpdateMember(ctx, p.ID, userID, role); err != nil {
		return fmt.Errorf("ProjectUseCase - AddMember - uc.repo.AddOrUpdateMember: %w", err)
	}

	return nil
}

// RemoveMember unassigns a user from a project.
func (uc *UseCase) RemoveMember(ctx context.Context, slug, userID string) error {
	p, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("ProjectUseCase - RemoveMember - uc.repo.GetBySlug: %w", err)
	}

	if err = uc.repo.RemoveMember(ctx, p.ID, userID); err != nil {
		return fmt.Errorf("ProjectUseCase - RemoveMember - uc.repo.RemoveMember: %w", err)
	}

	return nil
}

// Update -.
func (uc *UseCase) Update(ctx context.Context, slug, name, description, imageURL string) (entity.Project, error) {
	existing, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - Update - uc.repo.GetBySlug: %w", err)
	}

	existing.Name = name
	existing.Description = description
	existing.ImageURL = imageURL

	if err = existing.Validate(); err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - Update - existing.Validate: %w", err)
	}

	if err = uc.repo.Update(ctx, existing); err != nil {
		return entity.Project{}, fmt.Errorf("ProjectUseCase - Update - uc.repo.Update: %w", err)
	}

	return *existing, nil
}

// Delete -.
func (uc *UseCase) Delete(ctx context.Context, slug string) error {
	existing, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("ProjectUseCase - Delete - uc.repo.GetBySlug: %w", err)
	}

	if existing.Slug == "default" {
		return fmt.Errorf("%w: cannot delete default project", entity.ErrInvalidProject)
	}

	if err = uc.repo.Delete(ctx, existing.ID); err != nil {
		return fmt.Errorf("ProjectUseCase - Delete - uc.repo.Delete: %w", err)
	}

	return nil
}
