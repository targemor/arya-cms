package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	"github.com/lucimux/arya/internal/controller/restapi/v1/response"
	"github.com/lucimux/arya/internal/entity"
)

func (r *V1) authorizeContent(ctx *fiber.Ctx, action entity.APIKeyAction) (string, error) {
	tableName := ctx.Params("tableName")

	if apiKey, ok := ctx.Locals("apiKey").(entity.APIKey); ok {
		explicitProj := ctx.Get("X-Project")
		if explicitProj == "" {
			explicitProj = ctx.Query("project")
		}

		if explicitProj != "" && explicitProj != apiKey.ProjectSlug {
			return "", errorResponse(ctx, http.StatusForbidden, "forbidden: api key does not belong to this project")
		}

		if !apiKey.CanPerform(tableName, action) {
			return "", errorResponse(ctx, http.StatusForbidden, "forbidden: insufficient api key permissions")
		}

		return apiKey.ProjectSlug, nil
	}

	if _, ok := ctx.Locals("userID").(string); !ok {
		return "", errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	return r.getProjectSlug(ctx), nil
}

// @Summary     List entries
// @Description List entries of a content type. Supports pagination[page], pagination[pageSize], sort=field:asc|desc,... and filters[field][$op]=value with $eq $ne $lt $lte $gt $gte $in $notIn $contains $notContains $containsi $notContainsi $startsWith $endsWith $null $notNull.
// @ID          list-entries
// @Tags        content
// @Produce     json
// @Param       tableName            path     string true  "Content type"
// @Param       pagination[page]     query    int    false "Page, from 1"
// @Param       pagination[pageSize] query    int    false "Page size, 1-100"
// @Param       sort                 query    string false "e.g. createdAt:desc,title:asc"
// @Success     200                  {object} response.ContentEntries
// @Failure     400                  {object} response.Error
// @Failure     401                  {object} response.Error
// @Failure     403                  {object} response.Error
// @Failure     404                  {object} response.Error
// @Failure     500                  {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName} [get]
func (r *V1) listEntries(ctx *fiber.Ctx) error {
	projectSlug, authErr := r.authorizeContent(ctx, entity.APIKeyActionFind)
	if authErr != nil {
		return authErr
	}

	q, err := parseContentQuery(queryParams(ctx))
	if err != nil {
		r.l.Error(err, "restapi - v1 - listEntries")

		return errorResponse(ctx, http.StatusBadRequest, "invalid query")
	}

	entries, pagination, err := r.c.List(ctx.UserContext(), projectSlug, ctx.Params("tableName"), q)
	if err != nil {
		return r.contentError(ctx, "listEntries", err)
	}

	if entries == nil {
		entries = []entity.Entry{}
	}

	return ctx.Status(http.StatusOK).JSON(response.ContentEntries{
		Data: entries,
		Meta: response.ContentMeta{Pagination: pagination},
	})
}

// @Summary     Get entry
// @ID          get-entry
// @Tags        content
// @Produce     json
// @Param       tableName path     string true "Content type"
// @Param       id        path     string true "Entry id"
// @Success     200       {object} response.ContentEntry
// @Failure     401       {object} response.Error
// @Failure     403       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName}/{id} [get]
func (r *V1) getEntry(ctx *fiber.Ctx) error {
	projectSlug, authErr := r.authorizeContent(ctx, entity.APIKeyActionFindOne)
	if authErr != nil {
		return authErr
	}

	entry, err := r.c.Get(ctx.UserContext(), projectSlug, ctx.Params("tableName"), ctx.Params("id"))
	if err != nil {
		return r.contentError(ctx, "getEntry", err)
	}

	return ctx.Status(http.StatusOK).JSON(response.ContentEntry{Data: entry})
}

// @Summary     Get single entry
// @ID          get-single-entry
// @Tags        content
// @Produce     json
// @Param       tableName path     string true "Content type"
// @Success     200       {object} response.ContentEntry
// @Failure     401       {object} response.Error
// @Failure     403       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName}/single [get]
func (r *V1) getSingleEntry(ctx *fiber.Ctx) error {
	projectSlug, authErr := r.authorizeContent(ctx, entity.APIKeyActionFindOne)
	if authErr != nil {
		return authErr
	}

	entry, err := r.c.GetSingle(ctx.UserContext(), projectSlug, ctx.Params("tableName"))
	if err != nil {
		if errors.Is(err, entity.ErrEntryNotFound) {
			return ctx.Status(http.StatusOK).JSON(response.ContentEntry{Data: nil})
		}

		return r.contentError(ctx, "getSingleEntry", err)
	}

	return ctx.Status(http.StatusOK).JSON(response.ContentEntry{Data: entry})
}

func (r *V1) mutateEntry(
	ctx *fiber.Ctx,
	handlerName string,
	permAction entity.APIKeyAction,
	action func(projectSlug, tableName string, data entity.Entry) (entity.Entry, error),
) (entity.Entry, error) {
	projectSlug, authErr := r.authorizeContent(ctx, permAction)
	if authErr != nil {
		return nil, authErr
	}

	body, err := r.decodeEntry(ctx)
	if err != nil {
		r.l.Error(err, "restapi - v1 - "+handlerName)

		return nil, errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	entry, err := action(projectSlug, ctx.Params("tableName"), body.Data)
	if err != nil {
		return nil, r.contentError(ctx, handlerName, err)
	}

	return entry, nil
}

// @Summary     Update single entry
// @ID          update-single-entry
// @Tags        content
// @Accept      json
// @Produce     json
// @Param       tableName path     string               true "Content type"
// @Param       request   body     request.ContentEntry true "Entry"
// @Success     200       {object} response.ContentEntry
// @Failure     400       {object} response.Error
// @Failure     401       {object} response.Error
// @Failure     403       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName}/single [put]
func (r *V1) updateSingleEntry(ctx *fiber.Ctx) error {
	entry, err := r.mutateEntry(ctx, "updateSingleEntry", entity.APIKeyActionUpdate, func(projectSlug, table string, data entity.Entry) (entity.Entry, error) {
		return r.c.UpdateSingle(ctx.UserContext(), projectSlug, table, data)
	})
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusOK).JSON(response.ContentEntry{Data: entry})
}

// @Summary     Create entry
// @Description The data object is validated against the content type's JSON Schema.
// @ID          create-entry
// @Tags        content
// @Accept      json
// @Produce     json
// @Param       tableName path     string               true "Content type"
// @Param       request   body     request.ContentEntry true "Entry"
// @Success     201       {object} response.ContentEntry
// @Failure     400       {object} response.Error
// @Failure     401       {object} response.Error
// @Failure     403       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName} [post]
func (r *V1) createEntry(ctx *fiber.Ctx) error {
	entry, err := r.mutateEntry(ctx, "createEntry", entity.APIKeyActionCreate, func(projectSlug, table string, data entity.Entry) (entity.Entry, error) {
		return r.c.Create(ctx.UserContext(), projectSlug, table, data)
	})
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusCreated).JSON(response.ContentEntry{Data: entry})
}

// @Summary     Update entry
// @Description Partial update: only the fields present in data change. Validated against the content type's JSON Schema.
// @ID          update-entry
// @Tags        content
// @Accept      json
// @Produce     json
// @Param       tableName path     string               true "Content type"
// @Param       id        path     string               true "Entry id"
// @Param       request   body     request.ContentEntry true "Fields to change"
// @Success     200       {object} response.ContentEntry
// @Failure     400       {object} response.Error
// @Failure     401       {object} response.Error
// @Failure     403       {object} response.Error
// @Failure     404       {object} response.Error
// @Failure     500       {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName}/{id} [put]
func (r *V1) updateEntry(ctx *fiber.Ctx) error {
	projectSlug, authErr := r.authorizeContent(ctx, entity.APIKeyActionUpdate)
	if authErr != nil {
		return authErr
	}

	body, err := r.decodeEntry(ctx)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateEntry")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	entry, err := r.c.Update(ctx.UserContext(), projectSlug, ctx.Params("tableName"), ctx.Params("id"), body.Data)
	if err != nil {
		return r.contentError(ctx, "updateEntry", err)
	}

	return ctx.Status(http.StatusOK).JSON(response.ContentEntry{Data: entry})
}

// @Summary     Delete entry
// @ID          delete-entry
// @Tags        content
// @Param       tableName path string true "Content type"
// @Param       id        path string true "Entry id"
// @Success     204
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /content/{tableName}/{id} [delete]
func (r *V1) deleteEntry(ctx *fiber.Ctx) error {
	projectSlug, authErr := r.authorizeContent(ctx, entity.APIKeyActionDelete)
	if authErr != nil {
		return authErr
	}

	if err := r.c.Delete(ctx.UserContext(), projectSlug, ctx.Params("tableName"), ctx.Params("id")); err != nil {
		return r.contentError(ctx, "deleteEntry", err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// decodeEntry decodes with UseNumber instead of BodyParser, whose decoder turns every number into
// a float64 and would round integers above 2^53.
func (r *V1) decodeEntry(ctx *fiber.Ctx) (request.ContentEntry, error) {
	var body request.ContentEntry

	dec := json.NewDecoder(bytes.NewReader(ctx.Body()))
	dec.UseNumber()

	if err := dec.Decode(&body); err != nil {
		return request.ContentEntry{}, err
	}

	if err := r.v.Struct(body); err != nil {
		return request.ContentEntry{}, err
	}

	return body, nil
}

// contentError maps the content use case's sentinels to statuses for every content handler.
func (r *V1) contentError(ctx *fiber.Ctx, handler string, err error) error {
	return r.mappedError(ctx, handler, err, []errorStatus{
		{entity.ErrContentTypeNotFound, http.StatusNotFound, "content type not found"},
		{entity.ErrEntryNotFound, http.StatusNotFound, "entry not found"},
		{entity.ErrInvalidQuery, http.StatusBadRequest, "invalid query"},
		{entity.ErrInvalidEntry, http.StatusBadRequest, "invalid entry"},
	})
}
