package contenttype

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/filestore/contenttype"

// tracedRepo wraps a ContentTypeStore with OpenTelemetry spans.
type tracedRepo struct {
	next repo.ContentTypeStore
}

func newTraced(next repo.ContentTypeStore) repo.ContentTypeStore {
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

func (r *tracedRepo) Exists(ctx context.Context, projectSlug, tableName string) (bool, error) {
	ctx, span := startSpan(ctx, "ContentTypeStore.Exists",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", tableName))

	result, err := r.next.Exists(ctx, projectSlug, tableName)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Save(ctx context.Context, projectSlug string, ct entity.ContentType) error {
	ctx, span := startSpan(ctx, "ContentTypeStore.Save",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", ct.TableName))

	err := r.next.Save(ctx, projectSlug, ct)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) List(ctx context.Context, projectSlug string) ([]entity.ContentType, error) {
	ctx, span := startSpan(ctx, "ContentTypeStore.List",
		attribute.String("project.slug", projectSlug))

	result, err := r.next.List(ctx, projectSlug)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) ListProjects(ctx context.Context) ([]string, error) {
	ctx, span := startSpan(ctx, "ContentTypeStore.ListProjects")

	result, err := r.next.ListProjects(ctx)
	endSpan(span, err)

	return result, err
}
