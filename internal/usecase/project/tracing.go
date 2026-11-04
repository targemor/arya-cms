package project

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/project"

type tracedUseCase struct {
	next usecase.Project
}

func newTraced(next usecase.Project) usecase.Project {
	return &tracedUseCase{next: next}
}

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return otel.Tracer(_tracerName).Start(ctx, name, trace.WithAttributes(attrs...))
}

func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

func (u *tracedUseCase) Create(ctx context.Context, name, slug, description, imageURL string) (entity.Project, error) {
	ctx, span := startSpan(ctx, "Project.Create", attribute.String("project.slug", slug))

	p, err := u.next.Create(ctx, name, slug, description, imageURL)
	endSpan(span, err)

	return p, err
}

func (u *tracedUseCase) GetBySlug(ctx context.Context, slug string) (entity.Project, error) {
	ctx, span := startSpan(ctx, "Project.GetBySlug", attribute.String("project.slug", slug))

	p, err := u.next.GetBySlug(ctx, slug)
	endSpan(span, err)

	return p, err
}

func (u *tracedUseCase) List(ctx context.Context) ([]entity.Project, error) {
	ctx, span := startSpan(ctx, "Project.List")

	projects, err := u.next.List(ctx)
	endSpan(span, err)

	return projects, err
}

func (u *tracedUseCase) Update(ctx context.Context, slug, name, description, imageURL string) (entity.Project, error) {
	ctx, span := startSpan(ctx, "Project.Update", attribute.String("project.slug", slug))

	p, err := u.next.Update(ctx, slug, name, description, imageURL)
	endSpan(span, err)

	return p, err
}

func (u *tracedUseCase) Delete(ctx context.Context, slug string) error {
	ctx, span := startSpan(ctx, "Project.Delete", attribute.String("project.slug", slug))

	err := u.next.Delete(ctx, slug)
	endSpan(span, err)

	return err
}

func (u *tracedUseCase) ListForUser(ctx context.Context, userID string, isSuperAdmin bool) ([]entity.Project, error) {
	ctx, span := startSpan(ctx, "Project.ListForUser", attribute.String("user.id", userID), attribute.Bool("is_super_admin", isSuperAdmin))

	projects, err := u.next.ListForUser(ctx, userID, isSuperAdmin)
	endSpan(span, err)

	return projects, err
}

func (u *tracedUseCase) ListMembers(ctx context.Context, slug string) ([]entity.ProjectMember, error) {
	ctx, span := startSpan(ctx, "Project.ListMembers", attribute.String("project.slug", slug))

	members, err := u.next.ListMembers(ctx, slug)
	endSpan(span, err)

	return members, err
}

func (u *tracedUseCase) AddMember(ctx context.Context, slug, userID string, role entity.Role) error {
	ctx, span := startSpan(ctx, "Project.AddMember", attribute.String("project.slug", slug), attribute.String("user.id", userID))

	err := u.next.AddMember(ctx, slug, userID, role)
	endSpan(span, err)

	return err
}

func (u *tracedUseCase) RemoveMember(ctx context.Context, slug, userID string) error {
	ctx, span := startSpan(ctx, "Project.RemoveMember", attribute.String("project.slug", slug), attribute.String("user.id", userID))

	err := u.next.RemoveMember(ctx, slug, userID)
	endSpan(span, err)

	return err
}

func (u *tracedUseCase) CreateAPIKey(
	ctx context.Context,
	slug, name, description string,
	accessType entity.APIKeyAccessType,
	permissions map[string][]entity.APIKeyAction,
) (entity.APIKey, error) {
	ctx, span := startSpan(ctx, "Project.CreateAPIKey", attribute.String("project.slug", slug))

	k, err := u.next.CreateAPIKey(ctx, slug, name, description, accessType, permissions)
	endSpan(span, err)

	return k, err
}

func (u *tracedUseCase) ListAPIKeys(ctx context.Context, slug string) ([]entity.APIKey, error) {
	ctx, span := startSpan(ctx, "Project.ListAPIKeys", attribute.String("project.slug", slug))

	keys, err := u.next.ListAPIKeys(ctx, slug)
	endSpan(span, err)

	return keys, err
}

func (u *tracedUseCase) UpdateAPIKey(
	ctx context.Context,
	slug, keyID, name, description string,
	accessType entity.APIKeyAccessType,
	permissions map[string][]entity.APIKeyAction,
) (entity.APIKey, error) {
	ctx, span := startSpan(ctx, "Project.UpdateAPIKey", attribute.String("project.slug", slug), attribute.String("apikey.id", keyID))

	k, err := u.next.UpdateAPIKey(ctx, slug, keyID, name, description, accessType, permissions)
	endSpan(span, err)

	return k, err
}

func (u *tracedUseCase) DeleteAPIKey(ctx context.Context, slug, keyID string) error {
	ctx, span := startSpan(ctx, "Project.DeleteAPIKey", attribute.String("project.slug", slug), attribute.String("apikey.id", keyID))

	err := u.next.DeleteAPIKey(ctx, slug, keyID)
	endSpan(span, err)

	return err
}

func (u *tracedUseCase) ValidateAPIKey(ctx context.Context, rawKey string) (entity.APIKey, error) {
	ctx, span := startSpan(ctx, "Project.ValidateAPIKey")

	k, err := u.next.ValidateAPIKey(ctx, rawKey)
	endSpan(span, err)

	return k, err
}
