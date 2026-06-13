package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NewRequestID assigns a correlation ID to every request (honoring an inbound
// X-Request-ID), echoes it back in the response header, and seeds a request-scoped
// logger carrying the ID.
func NewRequestID(base zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := c.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Locals(localRequestID, rid)
		c.Set("X-Request-ID", rid)

		l := base.With().Str("request_id", rid).Logger()
		c.Locals(localLogger, &l)
		return c.Next()
	}
}
