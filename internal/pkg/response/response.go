package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Meta carries pagination information for list responses.
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// Success is the consistent envelope for successful responses.
type Success struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
	Meta   *Meta       `json:"meta,omitempty"`
}

// ErrorBody is the consistent envelope for error responses.
type ErrorBody struct {
	Status    string      `json:"status"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
}

// NewMeta computes pagination metadata from total count, page, and limit.
func NewMeta(total int64, page, limit int) *Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &Meta{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
}

func OK(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Success{Status: "success", Data: data})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Success{Status: "success", Data: data})
}

func List(c *fiber.Ctx, data interface{}, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(Success{Status: "success", Data: data, Meta: meta})
}
