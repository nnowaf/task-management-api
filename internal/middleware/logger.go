package middleware

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// NewLogger emits one structured JSON log line per request with request_id, method,
// path, status, and latency. Level is INFO for <400, WARN for 4xx, ERROR for 5xx.
func NewLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()
		latency := time.Since(start)

		status := c.Response().StatusCode()
		var errCode, errReason string
		if chainErr != nil {
			if ae, ok := domain.AsAppError(chainErr); ok {
				status = ae.HTTPStatus
				errCode, errReason = ae.Code, ae.Message
			} else if fe, ok := chainErr.(*fiber.Error); ok {
				status, errReason = fe.Code, fe.Message
			} else {
				status, errReason = fiber.StatusInternalServerError, chainErr.Error()
			}
		}

		l := Logger(c)
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = l.Error()
		case status >= 400:
			event = l.Warn()
		default:
			event = l.Info()
		}

		event.
			Str("method", c.Method()).
			Str("path", c.OriginalURL()).
			Int("status", status).
			Int64("latency_ms", latency.Milliseconds()).
			Int("bytes", len(c.Response().Body()))
		if errCode != "" {
			event.Str("error_code", errCode)
		}
		if errReason != "" {
			event.Str("error", errReason)
		}
		event.Msg("request")

		return chainErr
	}
}
