package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	_ "github.com/lucimux/arya/internal/controller/restapi/v1/response" // Resolves response.Error in swagger annotations.
	"github.com/lucimux/arya/internal/entity"
)

// @Summary     Create component
// @Description Define a new reusable component
// @ID          create-component
// @Tags        content-type-builder
// @Accept      json
// @Produce     json
// @Param       request body     request.CreateComponent true "Component definition"
// @Success     201     {object} entity.Component
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /internal/components [post]
func (r *V1) createComponent(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	var body request.CreateComponent

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createComponent")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createComponent")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	c := &entity.Component{
		Name:        body.Name,
		Category:    body.Category,
		DisplayName: body.DisplayName,
		Icon:        body.Icon,
		Attributes:  toAttributes(body.Attributes),
	}

	projectSlug := r.getProjectSlug(ctx)

	created, err := r.comp.Create(ctx.UserContext(), projectSlug, c)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createComponent")

		return r.mapComponentError(ctx, err)
	}

	return ctx.Status(http.StatusCreated).JSON(created)
}

// @Summary     Update component
// @Description Update an existing component
// @ID          update-component
// @Tags        content-type-builder
// @Accept      json
// @Produce     json
// @Param       uid     path     string                  true "Component UID"
// @Param       request body     request.UpdateComponent true "Component definition"
// @Success     200     {object} entity.Component
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /internal/components/{uid} [put]
func (r *V1) updateComponent(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	projectSlug := r.getProjectSlug(ctx)

	existing, err := r.comp.Get(ctx.UserContext(), projectSlug, ctx.Params("uid"))
	if err != nil {
		return r.handleComponentGetErr(ctx, err)
	}

	var body request.UpdateComponent

	if err = ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateComponent")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err = r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateComponent")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	c := &entity.Component{
		Name:        existing.Name,
		Category:    existing.Category,
		DisplayName: body.DisplayName,
		Icon:        body.Icon,
		Attributes:  toAttributes(body.Attributes),
	}

	updated, err := r.comp.Update(ctx.UserContext(), projectSlug, c)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateComponent")

		return r.mapComponentError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(updated)
}

func (r *V1) handleComponentGetErr(ctx *fiber.Ctx, err error) error {
	if errors.Is(err, entity.ErrComponentNotFound) {
		return errorResponse(ctx, http.StatusNotFound, "component not found")
	}

	r.l.Error(err, "restapi - v1 - handleComponentGetErr")

	return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
}

func (r *V1) mapComponentError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, entity.ErrInvalidComponent):
		return errorResponse(ctx, http.StatusBadRequest, "invalid component definition")
	case errors.Is(err, entity.ErrComponentAlreadyExists):
		return errorResponse(ctx, http.StatusConflict, "component already exists")
	case errors.Is(err, entity.ErrComponentNotFound):
		return errorResponse(ctx, http.StatusNotFound, "component not found")
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}
}

func toAttributes(raw []request.ContentTypeAttribute) []entity.Attribute {
	attrs := make([]entity.Attribute, 0, len(raw))
	for _, a := range raw {
		attrs = append(attrs, entity.Attribute{
			Name:        a.Name,
			DisplayName: a.DisplayName,
			Type:        entity.AttributeType(a.Type),
			Required:    a.Required,
			Component:   a.Component,
			Repeatable:  a.Repeatable,
		})
	}

	return attrs
}

// @Summary     List components
// @Description List every component defined
// @ID          list-components
// @Tags        content-type-builder
// @Produce     json
// @Success     200 {array}  entity.Component
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /internal/components [get]
func (r *V1) listComponents(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	projectSlug := r.getProjectSlug(ctx)

	comps, err := r.comp.List(ctx.UserContext(), projectSlug)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listComponents")

		return errorResponse(ctx, http.StatusInternalServerError, "failed to list components")
	}

	if len(comps) == 0 {
		return ctx.Status(http.StatusOK).JSON([]entity.Component{})
	}

	return ctx.Status(http.StatusOK).JSON(comps)
}

// @Summary     Delete component
// @Description Delete a component by UID
// @ID          delete-component
// @Tags        content-type-builder
// @Produce     json
// @Param       uid path string true "Component UID"
// @Success     204 "No Content"
// @Failure     401 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /internal/components/{uid} [delete]
func (r *V1) deleteComponent(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	uid := ctx.Params("uid")
	projectSlug := r.getProjectSlug(ctx)

	if err := r.comp.Delete(ctx.UserContext(), projectSlug, uid); err != nil {
		r.l.Error(err, "restapi - v1 - deleteComponent")

		if errors.Is(err, entity.ErrComponentNotFound) {
			return errorResponse(ctx, http.StatusNotFound, "component not found")
		}

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.SendStatus(http.StatusNoContent)
}
