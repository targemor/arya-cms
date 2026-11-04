package v1

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	"github.com/lucimux/arya/internal/controller/restapi/v1/response"
	"github.com/lucimux/arya/internal/entity"
)

// @Summary     List folders
// @Description Every Media Library folder, flat and sorted by name. parentId links them into a tree; null is the root.
// @ID          list-folders
// @Tags        media-library
// @Produce     json
// @Success     200 {object} response.Folders
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /upload/folders [get]
func (r *V1) listFolders(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	folders, err := r.fo.List(ctx.UserContext())
	if err != nil {
		return r.uploadError(ctx, "listFolders", err)
	}

	if folders == nil {
		folders = []entity.Folder{}
	}

	return ctx.Status(http.StatusOK).JSON(response.Folders{Data: folders})
}

// @Summary     Create folder
// @Description Create a Media Library folder, at the root or inside parentId. Names are unique within a folder.
// @ID          create-folder
// @Tags        media-library
// @Accept      json
// @Produce     json
// @Param       request body     request.Folder true "Folder"
// @Success     201     {object} entity.Folder
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /upload/folders [post]
func (r *V1) createFolder(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	body, err := r.decodeFolder(ctx)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createFolder")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	folder, err := r.fo.Create(ctx.UserContext(), body.Name, body.ParentID)
	if err != nil {
		return r.uploadError(ctx, "createFolder", err)
	}

	return ctx.Status(http.StatusCreated).JSON(folder)
}

// @Summary     Update folder
// @Description Rename and/or move a Media Library folder. Send the full state: name and parentId (null: the root).
// @ID          update-folder
// @Tags        media-library
// @Accept      json
// @Produce     json
// @Param       id      path     string         true "Folder id"
// @Param       request body     request.Folder true "Folder"
// @Success     200     {object} entity.Folder
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /upload/folders/{id} [put]
func (r *V1) updateFolder(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	body, err := r.decodeFolder(ctx)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateFolder")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	folder, err := r.fo.Update(ctx.UserContext(), ctx.Params("id"), body.Name, body.ParentID)
	if err != nil {
		return r.uploadError(ctx, "updateFolder", err)
	}

	return ctx.Status(http.StatusOK).JSON(folder)
}

// @Summary     Delete folder
// @Description Delete an empty Media Library folder. A folder that still holds files or folders is 409.
// @ID          delete-folder
// @Tags        media-library
// @Param       id path string true "Folder id"
// @Success     204
// @Failure     401 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     409 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /upload/folders/{id} [delete]
func (r *V1) deleteFolder(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	if err := r.fo.Delete(ctx.UserContext(), ctx.Params("id")); err != nil {
		return r.uploadError(ctx, "deleteFolder", err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

func (r *V1) decodeFolder(ctx *fiber.Ctx) (request.Folder, error) {
	var body request.Folder

	if err := ctx.BodyParser(&body); err != nil {
		return request.Folder{}, fmt.Errorf("ctx.BodyParser: %w", err)
	}

	if err := r.v.Struct(body); err != nil {
		return request.Folder{}, fmt.Errorf("r.v.Struct: %w", err)
	}

	return body, nil
}

// optionalFolderID reads a folder id from a query or form value: empty is the library root (nil),
// anything else must be a UUID.
func optionalFolderID(raw string) (*string, error) {
	if raw == "" {
		return nil, nil //nolint:nilnil // nil is a valid answer here: the root.
	}

	if uuid.Validate(raw) != nil {
		return nil, fmt.Errorf("%w: folder %q", entity.ErrInvalidFolder, raw)
	}

	return &raw, nil
}
