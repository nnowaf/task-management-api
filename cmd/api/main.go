// Package main is the entrypoint of the GDCPAY Task Management API.
//
// @title                       GDCPAY Task Management API
// @version                     1.0
// @description                 Multi-user task management API: JWT auth, idempotent task creation, transactional assignment, and structured logging.
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdcpay/task-api/internal/config"
	"github.com/gdcpay/task-api/internal/handler"
	"github.com/gdcpay/task-api/internal/middleware"
	"github.com/gdcpay/task-api/internal/pkg/jwt"
	"github.com/gdcpay/task-api/internal/pkg/logger"
	"github.com/gdcpay/task-api/internal/repository"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	fiberSwagger "github.com/gofiber/swagger"

	_ "github.com/gdcpay/task-api/docs"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	pool, err := repository.Connect(startupCtx, cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()
	if err := repository.Migrate(startupCtx, pool); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	startupCancel()
	log.Info().Msg("database connected and migrated")

	store := repository.NewStore(pool, cfg.IdempotencyTTL)
	jwtMgr := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiry)
	notifier := service.NewLogNotifier(log)

	handlers := handler.Handlers{
		Auth:    handler.NewAuthHandler(service.NewAuthService(store, jwtMgr)),
		Task:    handler.NewTaskHandler(service.NewTaskService(store, notifier)),
		Team:    handler.NewTeamHandler(service.NewTeamService(store)),
		Comment: handler.NewCommentHandler(service.NewCommentService(store)),
	}

	app := fiber.New(fiber.Config{
		AppName:      "gdcpay-task-api",
		ErrorHandler: middleware.NewErrorHandler(cfg.IsProduction()),
	})

	app.Use(helmet.New())
	app.Use(cors.New())
	app.Use(middleware.NewRequestID(log)) // request_id first so it is on every log line
	app.Use(middleware.NewLogger())
	app.Use(middleware.NewRecover())

	app.Get("/swagger/*", fiberSwagger.HandlerDefault)

	handler.RegisterRoutes(app, handlers, middleware.NewAuth(jwtMgr))

	go func() {
		addr := ":" + cfg.AppPort
		log.Info().Str("addr", addr).Msg("server starting")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("server stopped unexpectedly")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
}
