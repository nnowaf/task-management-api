package middleware

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// NewErrorHandler returns the Fiber global error handler. It renders every error
// as the consistent error envelope, distinguishes 4xx from 5xx, logs server errors
// with their internal cause, and hides internal detail in production.
func NewErrorHandler(production bool) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		code := domain.CodeInternal
		message := "An unexpected error occurred"
		var details interface{}

		switch e := err.(type) {
		case *domain.AppError:
			status = e.HTTPStatus
			code = e.Code
			message = e.Message
			details = e.Details
		case *fiber.Error:
			status = e.Code
			code = domain.CodeForStatus(status)
			message = e.Message
		default:
			if ae, ok := domain.AsAppError(err); ok {
				status = ae.HTTPStatus
				code = ae.Code
				message = ae.Message
				details = ae.Details
			}
		}

		if status >= 500 {
			Logger(c).Error().Err(err).Str("code", code).Msg("server error")
			if production {
				message = "An unexpected error occurred"
				details = nil
			}
		}

		return c.Status(status).JSON(response.ErrorBody{
			Status:    "error",
			Code:      code,
			Message:   message,
			Timestamp: time.Now().UTC(),
			Details:   details,
		})
	}
}
