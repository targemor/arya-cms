// Package app configures and runs application.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucimux/arya/config"
	"github.com/lucimux/arya/internal/controller/restapi"
	fileComponentRepo "github.com/lucimux/arya/internal/repo/filestore/component"
	fileContentTypeRepo "github.com/lucimux/arya/internal/repo/filestore/contenttype"
	persistContentRepo "github.com/lucimux/arya/internal/repo/persistent/content"
	persistContentTypeRepo "github.com/lucimux/arya/internal/repo/persistent/contenttype"
	persistFolderRepo "github.com/lucimux/arya/internal/repo/persistent/folder"
	persistProjectRepo "github.com/lucimux/arya/internal/repo/persistent/project"
	persistUploadRepo "github.com/lucimux/arya/internal/repo/persistent/upload"
	persistUserRepo "github.com/lucimux/arya/internal/repo/persistent/user"
	localStorage "github.com/lucimux/arya/internal/repo/storage/local"
	"github.com/lucimux/arya/internal/usecase"
	"github.com/lucimux/arya/internal/usecase/component"
	"github.com/lucimux/arya/internal/usecase/content"
	"github.com/lucimux/arya/internal/usecase/contenttype"
	"github.com/lucimux/arya/internal/usecase/folder"
	"github.com/lucimux/arya/internal/usecase/project"
	"github.com/lucimux/arya/internal/usecase/upload"
	"github.com/lucimux/arya/internal/usecase/user"
	"github.com/lucimux/arya/pkg/fswatcher"
	"github.com/lucimux/arya/pkg/httpserver"
	"github.com/lucimux/arya/pkg/jwt"
	"github.com/lucimux/arya/pkg/logger"
	"github.com/lucimux/arya/pkg/postgres"
	"github.com/lucimux/arya/pkg/tracing"
)

const _megabyte = 1 << 20

// _schemaSyncTimeout bounds one pass of applying the schema files to the database.
const _schemaSyncTimeout = 30 * time.Second

type useCases struct {
	project     usecase.Project
	user        usecase.User
	contentType usecase.ContentType
	component   usecase.Component
	content     usecase.Content
	upload      usecase.Upload
	folder      usecase.Folder
}

type servers struct {
	http *httpserver.Server
}

func initUseCases(cfg *config.Config, pg *postgres.Postgres, jwtManager *jwt.Manager) useCases {
	projectRepo := persistProjectRepo.New(pg)
	projectUseCase := project.New(projectRepo)
	userRepo := persistUserRepo.New(pg)
	contentTypeStore := fileContentTypeRepo.New(cfg.Schemas.Dir)
	contentTypeMigrator := persistContentTypeRepo.New(pg)

	contentTypeUseCase := contenttype.New(contentTypeStore, contentTypeMigrator)
	componentStore := fileComponentRepo.New(cfg.Schemas.Dir)
	componentUseCase := component.New(componentStore)
	uploadRepo := persistUploadRepo.New(pg)
	folderRepo := persistFolderRepo.New(pg)

	return useCases{
		project:     projectUseCase,
		user:        user.New(userRepo, jwtManager),
		contentType: contentTypeUseCase,
		component:   componentUseCase,
		content:     content.New(contentTypeUseCase, persistContentRepo.New(pg)),
		upload: upload.New(
			uploadRepo,
			folderRepo,
			localStorage.New(cfg.Uploads.Dir, cfg.Uploads.PublicPath),
			int64(cfg.Uploads.MaxSizeMB)*_megabyte,
		),
		folder: folder.New(folderRepo, uploadRepo),
	}
}

// startSchemaWatcher applies the schema files to the database now and again whenever one
// changes on disk. The watcher starts first so no change between the two is missed.
func startSchemaWatcher(cfg *config.Config, uc *useCases, l logger.Interface) (*fswatcher.Watcher, error) {
	syncSchemas := func() {
		ctx, cancel := context.WithTimeout(context.Background(), _schemaSyncTimeout)
		defer cancel()

		if err := uc.contentType.Sync(ctx); err != nil {
			l.Error(fmt.Errorf("app - Run - uc.contentType.Sync: %w", err))
		}
	}

	w, err := fswatcher.New(cfg.Schemas.Dir, syncSchemas, fswatcher.OnError(func(err error) {
		l.Error(fmt.Errorf("app - Run - fswatcher: %w", err))
	}))
	if err != nil {
		return nil, fmt.Errorf("app - Run - fswatcher.New: %w", err)
	}

	w.Start()
	syncSchemas()

	return w, nil
}

func initServers(cfg *config.Config, uc *useCases, jwtManager *jwt.Manager, l logger.Interface) servers {
	// HTTP Server
	httpServer := httpserver.New(l,
		httpserver.Port(cfg.HTTP.Port),
		httpserver.Prefork(cfg.HTTP.UsePreforkMode),
		// Room for the largest upload plus its multipart envelope.
		httpserver.BodyLimit((cfg.Uploads.MaxSizeMB+1)*_megabyte),
	)
	restapi.NewRouter(httpServer.App, cfg, uc.project, uc.user, uc.contentType, uc.component, uc.content, uc.upload, uc.folder, jwtManager, l)

	return servers{http: httpServer}
}

func (s *servers) startServers() {
	s.http.Start()
}

func (s *servers) waitForShutdown(l logger.Interface) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	var err error

	select {
	case sig := <-interrupt:
		l.Info("app - Run - signal: %s", sig.String())
	case err = <-s.http.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	s.shutdownServers(l)
}

func (s *servers) shutdownServers(l logger.Interface) {
	if err := s.http.Shutdown(); err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}
}

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	l := logger.New(cfg.Log.Level)

	ctx := context.Background()

	// Tracing
	shutdownTracing, err := tracing.New(ctx, tracing.Config{
		Enabled:     cfg.Tracing.Enabled,
		ServiceName: cfg.App.Name,
		Version:     cfg.App.Version,
		Endpoint:    cfg.Tracing.OTLPEndpoint,
		Insecure:    cfg.Tracing.OTLPInsecure,
		SampleRate:  cfg.Tracing.SampleRate,
	})
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - tracing.New: %w", err))
	}
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			l.Error(fmt.Errorf("app - Run - shutdownTracing: %w", err))
		}
	}()

	// Repository
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - postgres.New: %w", err))
	}
	defer pg.Close()

	// JWT
	jwtManager := jwt.New(cfg.JWT.Secret, cfg.JWT.TokenExpiry)

	uc := initUseCases(cfg, pg, jwtManager)

	// Content-Type Builder
	watcher, err := startSchemaWatcher(cfg, &uc, l)
	if err != nil {
		l.Fatal(err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			l.Error(fmt.Errorf("app - Run - watcher.Close: %w", err))
		}
	}()

	s := initServers(cfg, &uc, jwtManager, l)
	s.startServers()
	s.waitForShutdown(l)
}
