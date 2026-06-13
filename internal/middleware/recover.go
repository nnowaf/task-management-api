package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gofiber/fiber/v2"
)

// NewRecover converts any panic into a 500 AppError.
func NewRecover() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				Logger(c).Error().
					Interface("panic", r).
					Bytes("stack", debug.Stack()).
					Msg("recovered from panic")
				err = domain.NewInternal(fmt.Errorf("panic: %v", r))
			}
		}()
		return c.Next()
	}
}
