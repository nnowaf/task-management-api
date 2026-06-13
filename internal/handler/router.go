package handler

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Handlers bundles the HTTP handlers for route registration.
type Handlers struct {
	Auth    *AuthHandler
	Task    *TaskHandler
	Team    *TeamHandler
	Comment *CommentHandler
}

// RegisterRoutes wires every API route under /api/v1. authMW guards the protected
// groups; a small rate limiter protects the auth endpoints from brute force.
func RegisterRoutes(app *fiber.App, h Handlers, authMW fiber.Handler) {
	app.Get("/health", health)

	api := app.Group("/api/v1")
	api.Get("/health", health)

	authLimiter := limiter.New(limiter.Config{Max: 20, Expiration: time.Minute})
	auth := api.Group("/auth")
	auth.Post("/register", authLimiter, h.Auth.Register)
	auth.Post("/login", authLimiter, h.Auth.Login)

	tasks := api.Group("/tasks", authMW)
	tasks.Post("/", h.Task.Create)
	tasks.Get("/", h.Task.List)
	tasks.Get("/:id", h.Task.Get)
	tasks.Put("/:id", h.Task.Update)
	tasks.Delete("/:id", h.Task.Delete)
	tasks.Post("/:id/assign", h.Task.Assign)
	tasks.Get("/:id/logs", h.Task.Logs)
	tasks.Post("/:id/comments", h.Comment.Create)
	tasks.Get("/:id/comments", h.Comment.List)

	comments := api.Group("/comments", authMW)
	comments.Put("/:id", h.Comment.Update)
	comments.Delete("/:id", h.Comment.Delete)

	teams := api.Group("/teams", authMW)
	teams.Post("/", h.Team.Create)
	teams.Get("/", h.Team.List)
	teams.Get("/:id", h.Team.Get)
	teams.Post("/:id/members", h.Team.AddMember)
	teams.Get("/:id/members", h.Team.ListMembers)
	teams.Put("/:id/members/:userId", h.Team.UpdateMember)
	teams.Delete("/:id/members/:userId", h.Team.RemoveMember)

	app.Use(func(c *fiber.Ctx) error {
		return domain.NewNotFound("route not found")
	})
}

func health(c *fiber.Ctx) error {
	return response.OK(c, fiber.Map{"status": "ok", "time": time.Now().UTC()})
}
