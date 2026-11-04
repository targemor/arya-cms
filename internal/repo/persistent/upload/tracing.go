package upload

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/persistent/upload"

// tracedRepo wraps an UploadRepo with OpenTelemetry spans.
type tracedRepo struct {
	next repo.UploadRepo
}

func newTraced(next repo.UploadRepo) repo.UploadRepo {
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

func (r *tracedRepo) Store(ctx context.Context, file *entity.UploadFile) error {
	ctx, span := startSpan(ctx, "UploadRepo.Store", attribute.String("upload.hash", file.Hash))

	err := r.next.Store(ctx, file)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) GetByID(ctx context.Context, id string) (entity.UploadFile, error) {
	ctx, span := startSpan(ctx, "UploadRepo.GetByID", attribute.String("upload.id", id))

	result, err := r.next.GetByID(ctx, id)
	endSpan(span, err)

	return result, err
}

func folderAttr(folderID *string) attribute.KeyValue {
	if folderID == nil {
		return attribute.String("folder.id", "root")
	}

	return attribute.String("folder.id", *folderID)
}

func (r *tracedRepo) List(
	ctx context.Context, folderID *string, page, pageSize int,
) ([]entity.UploadFile, int, error) {
	ctx, span := startSpan(ctx, "UploadRepo.List",
		folderAttr(folderID), attribute.Int("page", page), attribute.Int("page_size", pageSize))

	files, total, err := r.next.List(ctx, folderID, page, pageSize)
	endSpan(span, err)

	return files, total, err
}

func (r *tracedRepo) CountInFolder(ctx context.Context, folderID string) (int, error) {
	ctx, span := startSpan(ctx, "UploadRepo.CountInFolder", attribute.String("folder.id", folderID))

	n, err := r.next.CountInFolder(ctx, folderID)
	endSpan(span, err)

	return n, err
}

func (r *tracedRepo) Move(ctx context.Context, id string, folderID *string) (entity.UploadFile, error) {
	ctx, span := startSpan(ctx, "UploadRepo.Move", attribute.String("upload.id", id), folderAttr(folderID))

	result, err := r.next.Move(ctx, id, folderID)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Delete(ctx context.Context, id string) (entity.UploadFile, error) {
	ctx, span := startSpan(ctx, "UploadRepo.Delete", attribute.String("upload.id", id))

	result, err := r.next.Delete(ctx, id)
	endSpan(span, err)

	return result, err
}
