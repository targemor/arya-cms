package folder

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/folder"

// tracedUseCase wraps a Folder usecase with OpenTelemetry spans.
type tracedUseCase struct {
	next usecase.Folder
}

// newTraced wraps a Folder usecase with tracing spans.
func newTraced(next usecase.Folder) usecase.Folder {
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

func (u *tracedUseCase) List(ctx context.Context) ([]entity.Folder, error) {
	ctx, span := startSpan(ctx, "FolderUseCase.List")

	result, err := u.next.List(ctx)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Create(ctx context.Context, name string, parentID *string) (entity.Folder, error) {
	ctx, span := startSpan(ctx, "FolderUseCase.Create")

	result, err := u.next.Create(ctx, name, parentID)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Update(ctx context.Context, id, name string, parentID *string) (entity.Folder, error) {
	ctx, span := startSpan(ctx, "FolderUseCase.Update", attribute.String("folder.id", id))

	result, err := u.next.Update(ctx, id, name, parentID)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Delete(ctx context.Context, id string) error {
	ctx, span := startSpan(ctx, "FolderUseCase.Delete", attribute.String("folder.id", id))

	err := u.next.Delete(ctx, id)
	endSpan(span, err)

	return err
}
