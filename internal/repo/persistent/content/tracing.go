package content

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/persistent/content"

// tracedRepo wraps a ContentRepo with OpenTelemetry spans.
type tracedRepo struct {
	next repo.ContentRepo
}

func newTraced(next repo.ContentRepo) repo.ContentRepo {
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

func tableAttr(projectSlug string, ct entity.ContentType) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", ct.TableName),
	}
}

func (r *tracedRepo) List(
	ctx context.Context, projectSlug string, ct entity.ContentType, q entity.ContentQuery,
) ([]entity.Entry, int, error) {
	ctx, span := startSpan(ctx, "ContentRepo.List", tableAttr(projectSlug, ct)...)

	entries, total, err := r.next.List(ctx, projectSlug, ct, q)
	endSpan(span, err)

	return entries, total, err
}

func (r *tracedRepo) Get(ctx context.Context, projectSlug string, ct entity.ContentType, id string) (entity.Entry, error) {
	attrs := append(tableAttr(projectSlug, ct), attribute.String("entry.id", id))
	ctx, span := startSpan(ctx, "ContentRepo.Get", attrs...)

	result, err := r.next.Get(ctx, projectSlug, ct, id)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) GetSingle(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentRepo.GetSingle", tableAttr(projectSlug, ct)...)

	result, err := r.next.GetSingle(ctx, projectSlug, ct)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Create(
	ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry,
) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentRepo.Create", tableAttr(projectSlug, ct)...)

	entry, err := r.next.Create(ctx, projectSlug, ct, data)
	endSpan(span, err)

	return entry, err
}

func (r *tracedRepo) Update(
	ctx context.Context, projectSlug string, ct entity.ContentType, id string, data entity.Entry,
) (entity.Entry, error) {
	attrs := append(tableAttr(projectSlug, ct), attribute.String("entry.id", id))
	ctx, span := startSpan(ctx, "ContentRepo.Update", attrs...)

	entry, err := r.next.Update(ctx, projectSlug, ct, id, data)
	endSpan(span, err)

	return entry, err
}

func (r *tracedRepo) UpdateSingle(
	ctx context.Context, projectSlug string, ct entity.ContentType, data entity.Entry,
) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentRepo.UpdateSingle", tableAttr(projectSlug, ct)...)

	entry, err := r.next.UpdateSingle(ctx, projectSlug, ct, data)
	endSpan(span, err)

	return entry, err
}

func (r *tracedRepo) Delete(ctx context.Context, projectSlug string, ct entity.ContentType, id string) error {
	attrs := append(tableAttr(projectSlug, ct), attribute.String("entry.id", id))
	ctx, span := startSpan(ctx, "ContentRepo.Delete", attrs...)

	err := r.next.Delete(ctx, projectSlug, ct, id)
	endSpan(span, err)

	return err
}
