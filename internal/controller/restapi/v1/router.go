package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/middleware"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/pkg/jwt"
	"github.com/lucimux/arya/pkg/logger"
)

// NewRoutes -.
func NewRoutes(
	apiV1Group fiber.Router,
	p usecase.Project, u usecase.User, ct usecase.ContentType, comp usecase.Component, c usecase.Content,
	up usecase.Upload, fo usecase.Folder,
	jwtManager *jwt.Manager, l logger.Interface,
) {
	r := &V1{p: p, u: u, ct: ct, comp: comp, c: c, up: up, fo: fo, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	// Public routes
	authGroup := apiV1Group.Group("/auth")
	{
		authGroup.Post("/register", r.register)
		authGroup.Post("/login", r.login)
	}

	// Protected routes
	protected := apiV1Group.Group("", middleware.Auth(jwtManager, p))
	r.registerProtectedRoutes(protected)
}

func (r *V1) registerProjectRoutes(protected fiber.Router) {
	projectsGroup := protected.Group("/projects")
	{
		projectsGroup.Get("", r.listProjects)
		projectsGroup.Get("/:slug", r.getProject)
		projectsGroup.Post("", middleware.RequireRole("superadmin"), r.createProject)
		projectsGroup.Put("/:slug", middleware.RequireRole("superadmin"), r.updateProject)
		projectsGroup.Delete("/:slug", middleware.RequireRole("superadmin"), r.deleteProject)
		projectsGroup.Get("/:slug/members", middleware.RequireRole("superadmin", "admin"), r.listProjectMembers)
		projectsGroup.Post("/:slug/members", middleware.RequireRole("superadmin", "admin"), r.addProjectMember)
		projectsGroup.Delete("/:slug/members/:userId", middleware.RequireRole("superadmin", "admin"), r.removeProjectMember)
		projectsGroup.Get("/:slug/api-keys", middleware.RequireRole("superadmin", "admin"), r.listAPIKeys)
		projectsGroup.Post("/:slug/api-keys", middleware.RequireRole("superadmin", "admin"), r.createAPIKey)
		projectsGroup.Put("/:slug/api-keys/:id", middleware.RequireRole("superadmin", "admin"), r.updateAPIKey)
		projectsGroup.Delete("/:slug/api-keys/:id", middleware.RequireRole("superadmin", "admin"), r.deleteAPIKey)
	}
}

func (r *V1) registerProtectedRoutes(protected fiber.Router) {
	r.registerProjectRoutes(protected)

	usersGroup := protected.Group("/users", middleware.RequireRole("superadmin"))
	{
		usersGroup.Get("", r.listUsers)
		usersGroup.Put("/:id/role", r.updateUserRole)
	}

	userGroup := protected.Group("/user")
	{
		userGroup.Get("/profile", r.profile)
	}

	schemasGroup := protected.Group("/internal/schemas")
	{
		schemasGroup.Get("", r.listContentTypes)
		schemasGroup.Post("", r.createContentType)
		schemasGroup.Put("/:tableName", r.updateContentType)
	}

	componentsGroup := protected.Group("/internal/components")
	{
		componentsGroup.Get("", r.listComponents)
		componentsGroup.Post("", r.createComponent)
		componentsGroup.Put("/:uid", r.updateComponent)
		componentsGroup.Delete("/:uid", r.deleteComponent)
	}

	// One parametrized group serves every content type: Fiber cannot add routes once it is
	// listening, and content types appear at runtime. The use case resolves :tableName against
	// the content types loaded in memory and answers 404 for unknown ones.
	contentGroup := protected.Group("/content/:tableName")
	{
		contentGroup.Get("", r.listEntries)
		contentGroup.Post("", r.createEntry)
		contentGroup.Get("/single", r.getSingleEntry)
		contentGroup.Put("/single", r.updateSingleEntry)
		contentGroup.Get("/:id", r.getEntry)
		contentGroup.Put("/:id", r.updateEntry)
		contentGroup.Delete("/:id", r.deleteEntry)
	}

	uploadGroup := protected.Group("/upload")
	{
		uploadGroup.Post("", r.uploadFile)
		uploadGroup.Get("/files", r.listUploads)
		uploadGroup.Put("/files/:id", r.moveUpload)
		uploadGroup.Delete("/files/:id", r.deleteUpload)
		uploadGroup.Get("/folders", r.listFolders)
		uploadGroup.Post("/folders", r.createFolder)
		uploadGroup.Put("/folders/:id", r.updateFolder)
		uploadGroup.Delete("/folders/:id", r.deleteFolder)
	}
}
