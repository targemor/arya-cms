package component

import (
	"context"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const _tracerName = "github.com/lucimux/arya/internal/usecase/component"

type tracedUseCase struct {
	next usecase.Component
}

func newTraced(next usecase.Component) usecase.Component {
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

func (u *tracedUseCase) Create(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentUseCase.Create",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", c.UID()),
	)

	result, err := u.next.Create(ctx, projectSlug, c)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Update(ctx context.Context, projectSlug string, c *entity.Component) (*entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentUseCase.Update",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", c.UID()),
	)

	result, err := u.next.Update(ctx, projectSlug, c)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) List(ctx context.Context, projectSlug string) ([]entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentUseCase.List",
		attribute.String("project.slug", projectSlug),
	)

	result, err := u.next.List(ctx, projectSlug)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Get(ctx context.Context, projectSlug, uid string) (*entity.Component, error) {
	ctx, span := startSpan(ctx, "ComponentUseCase.Get",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", uid),
	)

	result, err := u.next.Get(ctx, projectSlug, uid)
	endSpan(span, err)

	return result, err
}

func (u *tracedUseCase) Delete(ctx context.Context, projectSlug, uid string) error {
	ctx, span := startSpan(ctx, "ComponentUseCase.Delete",
		attribute.String("project.slug", projectSlug),
		attribute.String("component.uid", uid),
	)

	err := u.next.Delete(ctx, projectSlug, uid)
	endSpan(span, err)

	return err
}
