package content

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/content"

// tracedUseCase wraps a Content usecase with OpenTelemetry spans.
type tracedUseCase struct {
	next usecase.Content
}

// newTraced wraps a Content usecase with tracing spans.
func newTraced(next usecase.Content) usecase.Content {
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

func projectAttr(projectSlug string) attribute.KeyValue {
	return attribute.String("project.slug", projectSlug)
}

func tableAttr(tableName string) attribute.KeyValue {
	return attribute.String("content_type.table", tableName)
}

func (u *tracedUseCase) List(
	ctx context.Context, projectSlug, tableName string, q entity.ContentQuery,
) ([]entity.Entry, entity.Pagination, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.List", projectAttr(projectSlug), tableAttr(tableName))

	entries, pagination, err := u.next.List(ctx, projectSlug, tableName, q)
	endSpan(span, err)

	return entries, pagination, err
}

func (u *tracedUseCase) Get(ctx context.Context, projectSlug, tableName, id string) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.Get", projectAttr(projectSlug), tableAttr(tableName), attribute.String("entry.id", id))

	result, err := u.next.Get(ctx, projectSlug, tableName, id)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) GetSingle(ctx context.Context, projectSlug, tableName string) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.GetSingle", projectAttr(projectSlug), tableAttr(tableName))

	result, err := u.next.GetSingle(ctx, projectSlug, tableName)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Create(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.Create", projectAttr(projectSlug), tableAttr(tableName))

	result, err := u.next.Create(ctx, projectSlug, tableName, data)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Update(
	ctx context.Context, projectSlug, tableName, id string, data entity.Entry,
) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.Update", projectAttr(projectSlug), tableAttr(tableName), attribute.String("entry.id", id))

	result, err := u.next.Update(ctx, projectSlug, tableName, id, data)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) UpdateSingle(ctx context.Context, projectSlug, tableName string, data entity.Entry) (entity.Entry, error) {
	ctx, span := startSpan(ctx, "ContentUseCase.UpdateSingle", projectAttr(projectSlug), tableAttr(tableName))

	result, err := u.next.UpdateSingle(ctx, projectSlug, tableName, data)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Delete(ctx context.Context, projectSlug, tableName, id string) error {
	ctx, span := startSpan(ctx, "ContentUseCase.Delete", projectAttr(projectSlug), tableAttr(tableName), attribute.String("entry.id", id))

	err := u.next.Delete(ctx, projectSlug, tableName, id)
	endSpan(span, err)

	return err
}
