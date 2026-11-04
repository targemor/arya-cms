package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	_ "github.com/lucimux/arya/internal/controller/restapi/v1/response" // Resolves response.Error in swagger annotations.
	_ "github.com/lucimux/arya/internal/entity"                         // Resolves entity.APIKey in swagger annotations.
)

// @Summary     List project API keys
// @Description Returns all API keys configured for a project
// @ID          list-api-keys
// @Tags        projects
// @Produce     json
// @Param       slug path     string true "Project slug"
// @Success     200  {array}  entity.APIKey
// @Failure     401  {object} response.Error
// @Failure     404  {object} response.Error
// @Failure     500  {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/api-keys [get]
func (r *V1) listAPIKeys(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	keys, err := r.p.ListAPIKeys(ctx.UserContext(), slug)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listAPIKeys")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(keys)
}

// @Summary     Create project API key
// @Description Generates a new API key with granular content-type permissions for a project. The raw token is returned once in the response.
// @ID          create-api-key
// @Tags        projects
// @Accept      json
// @Produce     json
// @Param       slug    path     string               true "Project slug"
// @Param       request body     request.CreateAPIKey true "API key configuration"
// @Success     201     {object} entity.APIKey
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/api-keys [post]
func (r *V1) createAPIKey(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	var body request.CreateAPIKey
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createAPIKey")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createAPIKey")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	k, err := r.p.CreateAPIKey(ctx.UserContext(), slug, body.Name, body.Description, body.AccessType, body.Permissions)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createAPIKey")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusCreated).JSON(k)
}

// @Summary     Update project API key
// @Description Updates name, description, access type, and granular permissions for an API key
// @ID          update-api-key
// @Tags        projects
// @Accept      json
// @Produce     json
// @Param       slug    path     string               true "Project slug"
// @Param       id      path     string               true "API key ID"
// @Param       request body     request.UpdateAPIKey true "API key configuration"
// @Success     200     {object} entity.APIKey
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/api-keys/{id} [put]
func (r *V1) updateAPIKey(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")
	id := ctx.Params("id")

	var body request.UpdateAPIKey
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateAPIKey")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateAPIKey")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	k, err := r.p.UpdateAPIKey(ctx.UserContext(), slug, id, body.Name, body.Description, body.AccessType, body.Permissions)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateAPIKey")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(k)
}

// @Summary     Delete project API key
// @Description Revokes and deletes a project API key
// @ID          delete-api-key
// @Tags        projects
// @Param       slug path string true "Project slug"
// @Param       id   path string true "API key ID"
// @Success     204
// @Failure     400  {object} response.Error
// @Failure     401  {object} response.Error
// @Failure     404  {object} response.Error
// @Failure     500  {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/api-keys/{id} [delete]
func (r *V1) deleteAPIKey(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")
	id := ctx.Params("id")

	if err := r.p.DeleteAPIKey(ctx.UserContext(), slug, id); err != nil {
		r.l.Error(err, "restapi - v1 - deleteAPIKey")

		return r.mapProjectError(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}
