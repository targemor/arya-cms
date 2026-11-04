package contenttype

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/contenttype"

// tracedUseCase wraps a ContentType usecase with OpenTelemetry spans.
type tracedUseCase struct {
	next usecase.ContentType
}

// newTraced wraps a ContentType usecase with tracing spans.
func newTraced(next usecase.ContentType) usecase.ContentType {
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

func (u *tracedUseCase) Create(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error) {
	ctx, span := startSpan(ctx, "ContentTypeUseCase.Create",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", ct.TableName))

	result, err := u.next.Create(ctx, projectSlug, ct)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Update(ctx context.Context, projectSlug string, ct entity.ContentType) (entity.ContentType, error) {
	ctx, span := startSpan(ctx, "ContentTypeUseCase.Update",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", ct.TableName))

	result, err := u.next.Update(ctx, projectSlug, ct)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) List(ctx context.Context, projectSlug string) ([]entity.ContentType, error) {
	ctx, span := startSpan(ctx, "ContentTypeUseCase.List",
		attribute.String("project.slug", projectSlug))

	result, err := u.next.List(ctx, projectSlug)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Get(ctx context.Context, projectSlug, tableName string) (entity.ContentType, error) {
	ctx, span := startSpan(ctx, "ContentTypeUseCase.Get",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", tableName))

	result, err := u.next.Get(ctx, projectSlug, tableName)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Sync(ctx context.Context) error {
	ctx, span := startSpan(ctx, "ContentTypeUseCase.Sync")

	err := u.next.Sync(ctx)
	endSpan(span, err)

	return err
}
