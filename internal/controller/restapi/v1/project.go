package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	_ "github.com/lucimux/arya/internal/controller/restapi/v1/response" // Resolves response.Error in swagger annotations.
	"github.com/lucimux/arya/internal/entity"
)

// @Summary     List projects
// @Description Returns all CMS tenant projects
// @ID          list-projects
// @Tags        projects
// @Produce     json
// @Success     200 {array}  entity.Project
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /projects [get]
func (r *V1) listProjects(ctx *fiber.Ctx) error {
	var (
		userID       string
		isSuperAdmin bool
	)

	if uid, ok := ctx.Locals("userID").(string); ok {
		userID = uid
	}

	if role, ok := ctx.Locals("userRole").(string); ok && role == string(entity.RoleSuperAdmin) {
		isSuperAdmin = true
	}

	projects, err := r.p.ListForUser(ctx.UserContext(), userID, isSuperAdmin)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listProjects")

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusOK).JSON(projects)
}

// @Summary     Get project
// @Description Returns a CMS tenant project by its slug
// @ID          get-project
// @Tags        projects
// @Produce     json
// @Param       slug path     string true "Project slug"
// @Success     200  {object} entity.Project
// @Failure     401  {object} response.Error
// @Failure     404  {object} response.Error
// @Failure     500  {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug} [get]
func (r *V1) getProject(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	proj, err := r.p.GetBySlug(ctx.UserContext(), slug)
	if err != nil {
		r.l.Error(err, "restapi - v1 - getProject")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(proj)
}

// @Summary     Create project
// @Description Creates a new CMS tenant project
// @ID          create-project
// @Tags        projects
// @Accept      json
// @Produce     json
// @Param       request body     request.CreateProject true "Project details"
// @Success     201     {object} entity.Project
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /projects [post]
func (r *V1) createProject(ctx *fiber.Ctx) error {
	var body request.CreateProject

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createProject")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createProject")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	proj, err := r.p.Create(ctx.UserContext(), body.Name, body.Slug, body.Description, body.ImageURL)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createProject")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusCreated).JSON(proj)
}

// @Summary     Update project
// @Description Updates an existing CMS tenant project
// @ID          update-project
// @Tags        projects
// @Accept      json
// @Produce     json
// @Param       slug    path     string                true "Project slug"
// @Param       request body     request.UpdateProject true "Project details"
// @Success     200     {object} entity.Project
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug} [put]
func (r *V1) updateProject(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	var body request.UpdateProject

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateProject")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateProject")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	proj, err := r.p.Update(ctx.UserContext(), slug, body.Name, body.Description, body.ImageURL)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateProject")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(proj)
}

// @Summary     Delete project
// @Description Deletes an existing CMS tenant project
// @ID          delete-project
// @Tags        projects
// @Param       slug path string true "Project slug"
// @Success     204
// @Failure     400  {object} response.Error
// @Failure     401  {object} response.Error
// @Failure     404  {object} response.Error
// @Failure     500  {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug} [delete]
func (r *V1) deleteProject(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	if slug == "default" {
		return errorResponse(ctx, http.StatusBadRequest, "cannot delete default project")
	}

	if err := r.p.Delete(ctx.UserContext(), slug); err != nil {
		r.l.Error(err, "restapi - v1 - deleteProject")

		return r.mapProjectError(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     List project members
// @Description Returns all members and their roles for a project
// @ID          list-project-members
// @Tags        projects
// @Produce     json
// @Param       slug path     string true "Project slug"
// @Success     200  {array}  entity.ProjectMember
// @Failure     401  {object} response.Error
// @Failure     404  {object} response.Error
// @Failure     500  {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/members [get]
func (r *V1) listProjectMembers(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	members, err := r.p.ListMembers(ctx.UserContext(), slug)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listProjectMembers")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(members)
}

// @Summary     Add or update project member
// @Description Assigns a user to a project with a specific role
// @ID          add-project-member
// @Tags        projects
// @Accept      json
// @Produce     json
// @Param       slug    path     string                  true "Project slug"
// @Param       request body     request.AddProjectMember true "Member details"
// @Success     200     {object} map[string]string
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/members [post]
func (r *V1) addProjectMember(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")

	var body request.AddProjectMember
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - addProjectMember")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - addProjectMember")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.p.AddMember(ctx.UserContext(), slug, body.UserID, body.Role); err != nil {
		r.l.Error(err, "restapi - v1 - addProjectMember")

		return r.mapProjectError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"message": "member assigned successfully"})
}

// @Summary     Remove project member
// @Description Removes a user from a project
// @ID          remove-project-member
// @Tags        projects
// @Param       slug   path string true "Project slug"
// @Param       userId path string true "User ID"
// @Success     204
// @Failure     400    {object} response.Error
// @Failure     401    {object} response.Error
// @Failure     404    {object} response.Error
// @Failure     500    {object} response.Error
// @Security    BearerAuth
// @Router      /projects/{slug}/members/{userId} [delete]
func (r *V1) removeProjectMember(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")
	userID := ctx.Params("userId")

	if err := r.p.RemoveMember(ctx.UserContext(), slug, userID); err != nil {
		r.l.Error(err, "restapi - v1 - removeProjectMember")

		return r.mapProjectError(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

func (r *V1) mapProjectError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, entity.ErrInvalidProject), errors.Is(err, entity.ErrInvalidAPIKey), errors.Is(err, entity.ErrInvalidRole):
		return errorResponse(ctx, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrProjectNotFound):
		return errorResponse(ctx, http.StatusNotFound, "project not found")
	case errors.Is(err, entity.ErrAPIKeyNotFound):
		return errorResponse(ctx, http.StatusNotFound, "api key not found")
	case errors.Is(err, entity.ErrProjectAlreadyExists):
		return errorResponse(ctx, http.StatusConflict, "project already exists")
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}
}
