package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Fiber locals keys for per-request values.
const (
	localRequestID = "request_id"
	localLogger    = "logger"
	localUserID    = "user_id"
	localUsername  = "username"
)

// RequestID returns the current request's correlation ID.
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals(localRequestID).(string)
	return id
}

// Logger returns the request-scoped structured logger, falling back to a no-op.
func Logger(c *fiber.Ctx) *zerolog.Logger {
	if l, ok := c.Locals(localLogger).(*zerolog.Logger); ok && l != nil {
		return l
	}
	nop := zerolog.Nop()
	return &nop
}

// UserID returns the authenticated user's ID (uuid.Nil when unauthenticated).
func UserID(c *fiber.Ctx) uuid.UUID {
	id, _ := c.Locals(localUserID).(uuid.UUID)
	return id
}

// Username returns the authenticated user's username.
func Username(c *fiber.Ctx) string {
	name, _ := c.Locals(localUsername).(string)
	return name
}
