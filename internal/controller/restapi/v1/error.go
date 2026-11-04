package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/response"
)

func errorResponse(ctx *fiber.Ctx, code int, msg string) error {
	return ctx.Status(code).JSON(response.Error{Error: msg})
}

// errorStatus maps one use-case sentinel to the status and generic message sent to the client.
type errorStatus struct {
	err    error
	status int
	msg    string
}

// mappedError logs err and answers with the first matching entry of statuses, or 500.
func (r *V1) mappedError(ctx *fiber.Ctx, handler string, err error, statuses []errorStatus) error {
	r.l.Error(err, "restapi - v1 - "+handler)

	for _, s := range statuses {
		if errors.Is(err, s.err) {
			return errorResponse(ctx, s.status, s.msg)
		}
	}

	return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
}
