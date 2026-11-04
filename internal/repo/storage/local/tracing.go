package local

import (
	"context"
	"io"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/storage/local"

// tracedProvider wraps a StorageProvider with OpenTelemetry spans.
type tracedProvider struct {
	next repo.StorageProvider
}

func newTraced(next repo.StorageProvider) repo.StorageProvider {
	return &tracedProvider{next: next}
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

func (p *tracedProvider) Name() string {
	return p.next.Name()
}

func (p *tracedProvider) Upload(ctx context.Context, file *entity.UploadFile, content io.Reader) error {
	ctx, span := startSpan(ctx, "LocalStorage.Upload", attribute.String("upload.hash", file.Hash))

	err := p.next.Upload(ctx, file, content)
	endSpan(span, err)

	return err
}

func (p *tracedProvider) Delete(ctx context.Context, file *entity.UploadFile) error {
	ctx, span := startSpan(ctx, "LocalStorage.Delete", attribute.String("upload.hash", file.Hash))

	err := p.next.Delete(ctx, file)
	endSpan(span, err)

	return err
}
