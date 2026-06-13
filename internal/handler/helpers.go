package handler

import (
	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/middleware"
	"github.com/gdcpay/task-api/internal/pkg/validator"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func actorFrom(c *fiber.Ctx) service.Actor {
	return service.Actor{ID: middleware.UserID(c), Username: middleware.Username(c)}
}

// bindAndValidate parses the JSON body into dst and runs struct validation,
// returning a structured 400 on failure.
func bindAndValidate(c *fiber.Ctx, dst interface{}) error {
	if err := c.BodyParser(dst); err != nil {
		return domain.NewValidation("invalid request body")
	}
	if summary, details, ok := validator.Validate(dst); !ok {
		return domain.NewValidationDetails(summary, details)
	}
	return nil
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, domain.NewValidation(name + " must be a valid UUID")
	}
	return id, nil
}

func parseUUIDQueryErr(field string) error {
	return domain.NewValidation(field + " must be a valid UUID")
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// idempotencyKey reads and validates the required Idempotency-Key header.
func idempotencyKey(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Get("Idempotency-Key")
	if raw == "" {
		return uuid.Nil, domain.NewBadRequest(domain.CodeIdempotencyKeyRequired, "Idempotency-Key header is required")
	}
	key, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.NewBadRequest(domain.CodeIdempotencyKeyInvalid, "Idempotency-Key must be a valid UUID")
	}
	return key, nil
}
