package project

import (
	"context"

	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/persistent/project"

type tracedRepo struct {
	next repo.ProjectRepo
}

func newTraced(next repo.ProjectRepo) repo.ProjectRepo {
	return &tracedRepo{next: next}
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

func (r *tracedRepo) Create(ctx context.Context, p *entity.Project) error {
	ctx, span := startSpan(ctx, "ProjectRepo.Create")

	err := r.next.Create(ctx, p)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Project, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.GetByID", attribute.String("project.id", id.String()))

	result, err := r.next.GetByID(ctx, id)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) GetBySlug(ctx context.Context, slug string) (*entity.Project, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.GetBySlug", attribute.String("project.slug", slug))

	result, err := r.next.GetBySlug(ctx, slug)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) List(ctx context.Context) ([]entity.Project, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.List")

	result, err := r.next.List(ctx)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Update(ctx context.Context, p *entity.Project) error {
	ctx, span := startSpan(ctx, "ProjectRepo.Update", attribute.String("project.id", p.ID.String()))

	err := r.next.Update(ctx, p)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := startSpan(ctx, "ProjectRepo.Delete", attribute.String("project.id", id.String()))

	err := r.next.Delete(ctx, id)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) ListForUser(ctx context.Context, userID string) ([]entity.Project, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.ListForUser", attribute.String("user.id", userID))

	result, err := r.next.ListForUser(ctx, userID)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) ListMembers(ctx context.Context, projectID uuid.UUID) ([]entity.ProjectMember, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.ListMembers", attribute.String("project.id", projectID.String()))

	result, err := r.next.ListMembers(ctx, projectID)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) AddOrUpdateMember(ctx context.Context, projectID uuid.UUID, userID string, role entity.Role) error {
	ctx, span := startSpan(ctx, "ProjectRepo.AddOrUpdateMember", attribute.String("project.id", projectID.String()), attribute.String("user.id", userID))

	err := r.next.AddOrUpdateMember(ctx, projectID, userID, role)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) RemoveMember(ctx context.Context, projectID uuid.UUID, userID string) error {
	ctx, span := startSpan(ctx, "ProjectRepo.RemoveMember", attribute.String("project.id", projectID.String()), attribute.String("user.id", userID))

	err := r.next.RemoveMember(ctx, projectID, userID)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) GetUserRoleInProject(ctx context.Context, projectID uuid.UUID, userID string) (entity.Role, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.GetUserRoleInProject", attribute.String("project.id", projectID.String()), attribute.String("user.id", userID))

	role, err := r.next.GetUserRoleInProject(ctx, projectID, userID)
	endSpan(span, err)

	return role, err
}

func (r *tracedRepo) CreateAPIKey(ctx context.Context, k *entity.APIKey) error {
	ctx, span := startSpan(ctx, "ProjectRepo.CreateAPIKey", attribute.String("project.id", k.ProjectID.String()))

	err := r.next.CreateAPIKey(ctx, k)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) ListAPIKeys(ctx context.Context, projectID uuid.UUID) ([]entity.APIKey, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.ListAPIKeys", attribute.String("project.id", projectID.String()))

	keys, err := r.next.ListAPIKeys(ctx, projectID)
	endSpan(span, err)

	return keys, err
}

func (r *tracedRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	ctx, span := startSpan(ctx, "ProjectRepo.GetAPIKeyByHash")

	k, err := r.next.GetAPIKeyByHash(ctx, keyHash)
	endSpan(span, err)

	return k, err
}

func (r *tracedRepo) UpdateAPIKey(ctx context.Context, k *entity.APIKey) error {
	ctx, span := startSpan(ctx, "ProjectRepo.UpdateAPIKey", attribute.String("apikey.id", k.ID.String()))

	err := r.next.UpdateAPIKey(ctx, k)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) DeleteAPIKey(ctx context.Context, projectID, keyID uuid.UUID) error {
	ctx, span := startSpan(ctx, "ProjectRepo.DeleteAPIKey", attribute.String("project.id", projectID.String()), attribute.String("apikey.id", keyID.String()))

	err := r.next.DeleteAPIKey(ctx, projectID, keyID)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) TouchAPIKey(ctx context.Context, keyID uuid.UUID) error {
	ctx, span := startSpan(ctx, "ProjectRepo.TouchAPIKey", attribute.String("apikey.id", keyID.String()))

	err := r.next.TouchAPIKey(ctx, keyID)
	endSpan(span, err)

	return err
}
