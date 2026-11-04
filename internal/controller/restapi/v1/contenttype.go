package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	_ "github.com/lucimux/arya/internal/controller/restapi/v1/response" // Resolves response.Error in the swagger annotations.
	"github.com/lucimux/arya/internal/entity"
)

// @Summary     Create content type
// @Description Define a new content type. Its table is created in Postgres and its JSON Schema is saved to the schemas directory.
// @ID          create-content-type
// @Tags        content-type-builder
// @Accept      json
// @Produce     json
// @Param       request body     request.CreateContentType true "Content type definition"
// @Success     201     {object} entity.ContentType
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /internal/schemas [post]
func (r *V1) createContentType(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	var body request.CreateContentType

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createContentType")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createContentType")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	ct := entity.ContentType{
		TableName:   body.TableName,
		DisplayName: body.DisplayName,
		Kind:        entity.ContentTypeKind(body.Kind),
		Attributes:  toAttributes(body.Attributes),
	}

	projectSlug := r.getProjectSlug(ctx)

	created, err := r.ct.Create(ctx.UserContext(), projectSlug, ct)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createContentType")

		switch {
		case errors.Is(err, entity.ErrInvalidContentType):
			return errorResponse(ctx, http.StatusBadRequest, "invalid content type definition")
		case errors.Is(err, entity.ErrContentTypeAlreadyExists):
			return errorResponse(ctx, http.StatusConflict, "content type already exists")
		default:
			return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
		}
	}

	return ctx.Status(http.StatusCreated).JSON(created)
}

// @Summary     Update content type
// @Description Update an existing content type's schema and migrate its table
// @ID          update-content-type
// @Tags        content-type-builder
// @Accept      json
// @Produce     json
// @Param       tableName path     string                    true "Table name"
// @Param       request   body     request.UpdateContentType true "Content type definition"
// @Success     200       {object} entity.ContentType
// @Failure     400       {object} response.Error
// @Failure     401       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /internal/schemas/{tableName} [put]
func (r *V1) updateContentType(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	tableName := ctx.Params("tableName")

	var body request.UpdateContentType

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateContentType")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateContentType")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	ct := entity.ContentType{
		TableName:   tableName,
		DisplayName: body.DisplayName,
		Kind:        entity.ContentTypeKind(body.Kind),
		Attributes:  toAttributes(body.Attributes),
	}

	projectSlug := r.getProjectSlug(ctx)

	updated, err := r.ct.Update(ctx.UserContext(), projectSlug, ct)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateContentType")

		switch {
		case errors.Is(err, entity.ErrInvalidContentType):
			return errorResponse(ctx, http.StatusBadRequest, "invalid content type definition")
		case errors.Is(err, entity.ErrContentTypeNotFound):
			return errorResponse(ctx, http.StatusNotFound, "content type not found")
		default:
			return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
		}
	}

	return ctx.Status(http.StatusOK).JSON(updated)
}

// @Summary     List content types
// @Description List every content type defined in the schemas directory
// @ID          list-content-types
// @Tags        content-type-builder
// @Produce     json
// @Success     200 {array}  entity.ContentType
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /internal/schemas [get]
func (r *V1) listContentTypes(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	projectSlug := r.getProjectSlug(ctx)

	cts, err := r.ct.List(ctx.UserContext(), projectSlug)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listContentTypes")

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	if cts == nil {
		cts = []entity.ContentType{}
	}

	return ctx.Status(http.StatusOK).JSON(cts)
}
