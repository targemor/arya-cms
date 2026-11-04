package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	"github.com/lucimux/arya/internal/controller/restapi/v1/response"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/usecase"
)

// _uploadFormField is the multipart field carrying the file; NG-ZORRO's nz-upload sends "file".
const _uploadFormField = "file"

// @Summary     Upload file
// @Description Store a file in the Media Library. Send multipart/form-data with the file in "file" and, optionally, "alternativeText" and "folderId" (empty: the root).
// @ID          upload-file
// @Tags        media-library
// @Accept      multipart/form-data
// @Produce     json
// @Param       file            formData file   true  "File"
// @Param       alternativeText formData string false "Alternative text"
// @Param       folderId        formData string false "Folder id; empty or missing is the library root"
// @Success     201             {object} entity.UploadFile
// @Failure     400             {object} response.Error
// @Failure     401             {object} response.Error
// @Failure     404             {object} response.Error
// @Failure     413             {object} response.Error
// @Failure     500             {object} response.Error
// @Security    BearerAuth
// @Router      /upload [post]
func (r *V1) uploadFile(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	header, err := ctx.FormFile(_uploadFormField)
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadFile")

		return errorResponse(ctx, http.StatusBadRequest, "missing file")
	}

	content, err := header.Open()
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadFile")

		return errorResponse(ctx, http.StatusBadRequest, "unreadable file")
	}

	defer func() {
		if closeErr := content.Close(); closeErr != nil {
			r.l.Error(closeErr, "restapi - v1 - uploadFile - content.Close")
		}
	}()

	alternativeText := ctx.FormValue("alternativeText")

	folderID, err := optionalFolderID(ctx.FormValue("folderId"))
	if err != nil {
		return r.uploadError(ctx, "uploadFile", err)
	}

	file, err := r.up.Upload(ctx.UserContext(), usecase.UploadInput{
		Name:            header.Filename,
		AlternativeText: &alternativeText,
		Size:            header.Size,
		FolderID:        folderID,
	}, content)
	if err != nil {
		return r.uploadError(ctx, "uploadFile", err)
	}

	return ctx.Status(http.StatusCreated).JSON(file)
}

// @Summary     List files
// @Description List the Media Library files directly in one folder (the root when folder is missing), newest first.
// @ID          list-files
// @Tags        media-library
// @Produce     json
// @Param       folder               query    string false "Folder id; missing is the library root"
// @Param       pagination[page]     query    int false "Page, from 1"
// @Param       pagination[pageSize] query    int false "Page size, 1-100"
// @Success     200                  {object} response.UploadFiles
// @Failure     400                  {object} response.Error
// @Failure     401                  {object} response.Error
// @Failure     404                  {object} response.Error
// @Failure     500                  {object} response.Error
// @Security    BearerAuth
// @Router      /upload/files [get]
func (r *V1) listUploads(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	// Only the pagination part of the Strapi-style query applies to files.
	q, err := parseContentQuery(queryParams(ctx))
	if err != nil {
		r.l.Error(err, "restapi - v1 - listUploads")

		return errorResponse(ctx, http.StatusBadRequest, "invalid query")
	}

	folderID, err := optionalFolderID(ctx.Query("folder"))
	if err != nil {
		return r.uploadError(ctx, "listUploads", err)
	}

	files, pagination, err := r.up.List(ctx.UserContext(), folderID, q.Page, q.PageSize)
	if err != nil {
		return r.uploadError(ctx, "listUploads", err)
	}

	if files == nil {
		files = []entity.UploadFile{}
	}

	return ctx.Status(http.StatusOK).JSON(response.UploadFiles{
		Data: files,
		Meta: response.ContentMeta{Pagination: pagination},
	})
}

// @Summary     Move file
// @Description Move a Media Library file to another folder; a null folderId is the library root.
// @ID          move-file
// @Tags        media-library
// @Accept      json
// @Produce     json
// @Param       id      path     string           true "File id"
// @Param       request body     request.MoveFile true "Destination"
// @Success     200     {object} entity.UploadFile
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /upload/files/{id} [put]
func (r *V1) moveUpload(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	var body request.MoveFile

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - moveUpload")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - moveUpload")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	file, err := r.up.Move(ctx.UserContext(), ctx.Params("id"), body.FolderID)
	if err != nil {
		return r.uploadError(ctx, "moveUpload", err)
	}

	return ctx.Status(http.StatusOK).JSON(file)
}

// @Summary     Delete file
// @Description Delete a Media Library file: its metadata and the stored object.
// @ID          delete-file
// @Tags        media-library
// @Param       id path string true "File id"
// @Success     204
// @Failure     401 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /upload/files/{id} [delete]
func (r *V1) deleteUpload(ctx *fiber.Ctx) error {
	if _, ok := ctx.Locals("userID").(string); !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	if err := r.up.Delete(ctx.UserContext(), ctx.Params("id")); err != nil {
		return r.uploadError(ctx, "deleteUpload", err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// uploadError maps the Media Library's sentinels (files and folders) to statuses.
func (r *V1) uploadError(ctx *fiber.Ctx, handler string, err error) error {
	return r.mappedError(ctx, handler, err, []errorStatus{
		{entity.ErrUploadNotFound, http.StatusNotFound, "file not found"},
		{entity.ErrUploadTooLarge, http.StatusRequestEntityTooLarge, "file too large"},
		{entity.ErrInvalidUpload, http.StatusBadRequest, "invalid file"},
		{entity.ErrInvalidQuery, http.StatusBadRequest, "invalid query"},
		{entity.ErrFolderNotFound, http.StatusNotFound, "folder not found"},
		{entity.ErrFolderAlreadyExists, http.StatusConflict, "a folder with that name already exists here"},
		{entity.ErrFolderNotEmpty, http.StatusConflict, "folder is not empty"},
		{entity.ErrInvalidFolder, http.StatusBadRequest, "invalid folder"},
	})
}
