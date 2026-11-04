package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/pkg/logger"
)

// V1 -.
type V1 struct {
	p    usecase.Project
	u    usecase.User
	ct   usecase.ContentType
	comp usecase.Component
	c    usecase.Content
	up   usecase.Upload
	fo   usecase.Folder
	l    logger.Interface
	v    *validator.Validate
}

func (r *V1) getProjectSlug(ctx *fiber.Ctx) string {
	if p := ctx.Get("X-Project"); p != "" {
		return p
	}

	if p := ctx.Query("project"); p != "" {
		return p
	}

	return "default"
}
