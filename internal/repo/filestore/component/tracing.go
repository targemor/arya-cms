package component

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/repo/filestore/component"

type tracedRepo struct {
	next repo.ComponentStore
}

func newTraced(next repo.ComponentStore) repo.ComponentStore {
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

func (r *tracedRepo) Exists(ctx context.Context, projectSlug, uid string) (bool, error) {
	ctx, span := startSpan(ctx, "ComponentStore.Exists",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", uid))

	result, err := r.next.Exists(ctx, projectSlug, uid)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Save(ctx context.Context, projectSlug string, c *entity.Component) error {
	ctx, span := startSpan(ctx, "ComponentStore.Save",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", c.UID()))

	err := r.next.Save(ctx, projectSlug, c)
	endSpan(span, err)

	return err
}

func (r *tracedRepo) Get(ctx context.Context, projectSlug, uid string) (*entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentStore.Get",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", uid))

	result, err := r.next.Get(ctx, projectSlug, uid)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) List(ctx context.Context, projectSlug string) ([]entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentStore.List",
		attribute.String("project.slug", projectSlug))

	result, err := r.next.List(ctx, projectSlug)
	endSpan(span, err)

	return result, err
}

func (r *tracedRepo) Delete(ctx context.Context, projectSlug, uid string) error {
	ctx, span := startSpan(ctx, "ComponentStore.Delete",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", uid))

	err := r.next.Delete(ctx, projectSlug, uid)
	endSpan(span, err)

	return err
}
