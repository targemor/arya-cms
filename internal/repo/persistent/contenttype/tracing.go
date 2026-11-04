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

const _tracerName = "github.com/lucimux/arya/internal/repo/persistent/contenttype"

// tracedRepo wraps a ContentTypeMigrator with OpenTelemetry spans.
type tracedRepo struct {
	next repo.ContentTypeMigrator
}

func newTraced(next repo.ContentTypeMigrator) repo.ContentTypeMigrator {
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

func (r *tracedRepo) Migrate(ctx context.Context, projectSlug string, ct entity.ContentType) error {
	ctx, span := startSpan(ctx, "ContentTypeRepo.Migrate",
		attribute.String("project.slug", projectSlug),
		attribute.String("content_type.table", ct.TableName))

	err := r.next.Migrate(ctx, projectSlug, ct)
	endSpan(span, err)

	return err
}
