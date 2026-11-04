package folder

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/persistent/folder"

// tracedRepo wraps a FolderRepo with OpenTelemetry spans.
type tracedRepo struct {
	next repo.FolderRepo
}

func newTraced(next repo.FolderRepo) repo.FolderRepo {
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

func (r *tracedRepo) Create(ctx context.Context, folder *entity.Folder) error {
	ctx, span := startSpan(ctx, "FolderRepo.Create")

	err := r.next.Create(ctx, folder)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) GetByID(ctx context.Context, id string) (entity.Folder, error) {
	ctx, span := startSpan(ctx, "FolderRepo.GetByID", attribute.String("folder.id", id))

	result, err := r.next.GetByID(ctx, id)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) List(ctx context.Context) ([]entity.Folder, error) {
	ctx, span := startSpan(ctx, "FolderRepo.List")

	result, err := r.next.List(ctx)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Update(ctx context.Context, folder *entity.Folder) error {
	ctx, span := startSpan(ctx, "FolderRepo.Update", attribute.String("folder.id", folder.ID))

	err := r.next.Update(ctx, folder)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) Delete(ctx context.Context, id string) error {
	ctx, span := startSpan(ctx, "FolderRepo.Delete", attribute.String("folder.id", id))

	err := r.next.Delete(ctx, id)
	endSpan(span, err)

	return err
}
