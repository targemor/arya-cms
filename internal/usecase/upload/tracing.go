package upload

import (
	"context"
	"io"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/upload"

// tracedUseCase wraps an Upload usecase with OpenTelemetry spans.
type tracedUseCase struct {
	next usecase.Upload
}

// newTraced wraps an Upload usecase with tracing spans.
func newTraced(next usecase.Upload) usecase.Upload {
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

func folderAttr(folderID *string) attribute.KeyValue {
	if folderID == nil {
		return attribute.String("folder.id", "root")
	}

	return attribute.String("folder.id", *folderID)
}

func (u *tracedUseCase) Upload(ctx context.Context, in usecase.UploadInput, content io.Reader) (entity.UploadFile, error) {
	ctx, span := startSpan(ctx, "UploadUseCase.Upload", attribute.Int64("upload.size", in.Size), folderAttr(in.FolderID))

	result, err := u.next.Upload(ctx, in, content)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) List(
	ctx context.Context, folderID *string, page, pageSize int,
) ([]entity.UploadFile, entity.Pagination, error) {
	ctx, span := startSpan(ctx, "UploadUseCase.List",
		folderAttr(folderID), attribute.Int("page", page), attribute.Int("page_size", pageSize))

	files, pagination, err := u.next.List(ctx, folderID, page, pageSize)
	endSpan(span, err)

	return files, pagination, err
}

func (u *tracedUseCase) Move(ctx context.Context, id string, folderID *string) (entity.UploadFile, error) {
	ctx, span := startSpan(ctx, "UploadUseCase.Move", attribute.String("upload.id", id), folderAttr(folderID))

	result, err := u.next.Move(ctx, id, folderID)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Delete(ctx context.Context, id string) error {
	ctx, span := startSpan(ctx, "UploadUseCase.Delete", attribute.String("upload.id", id))

	err := u.next.Delete(ctx, id)
	endSpan(span, err)

	return err
}
