package restapi

import (
	"net/http"
	"strings"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/lucimux/arya/config"
	_ "github.com/lucimux/arya/docs" // Swagger docs.
	"github.com/lucimux/arya/internal/controller/restapi/middleware"
	v1 "github.com/lucimux/arya/internal/controller/restapi/v1"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/pkg/jwt"
	"github.com/lucimux/arya/pkg/logger"
)

// NewRouter -.
// Swagger spec:
//
//	@title       Go Clean Template API
//	@description Clean architecture template with user management
//	@version     1.0
//	@host        localhost:8080
//	@BasePath    /v1
//	@securityDefinitions.apikey BearerAuth
//	@in header
//	@name Authorization
func NewRouter(
	app *fiber.App, cfg *config.Config,
	p usecase.Project, u usecase.User, ct usecase.ContentType, comp usecase.Component, c usecase.Content,
	up usecase.Upload, fo usecase.Folder,
	jwtManager *jwt.Manager, l logger.Interface,
) {
	// Options
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))

	// Prometheus metrics
	if cfg.Metrics.Enabled {
		prometheus := fiberprometheus.New("my-service-name")
		prometheus.RegisterAt(app, "/metrics")
		app.Use(prometheus.Middleware)
	}

	// Swagger
	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	// Media Library files. Public, like Strapi's: the random hash in the name is what keeps a URL
	// unguessable.
	uploadsPrefix := "/" + strings.Trim(cfg.Uploads.PublicPath, "/")
	uploads := uploadsHandler(uploadsPrefix, cfg.Uploads.Dir)
	app.Get(uploadsPrefix+"/*", uploads)
	app.Head(uploadsPrefix+"/*", uploads)

	// K8s probe
	app.Get("/healthz", func(ctx *fiber.Ctx) error { return ctx.SendStatus(http.StatusOK) })

	// Routers
	apiV1Group := app.Group("/v1")
	{
		if cfg.Tracing.Enabled {
			apiV1Group.Use(otelfiber.Middleware())
		}

		v1.NewRoutes(apiV1Group, p, u, ct, comp, c, up, fo, jwtManager, l)
	}
}
