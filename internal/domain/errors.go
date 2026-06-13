package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the canonical application error carrying the HTTP status, a stable
// internal error code, a client-safe message, and an optional wrapped cause that
// is logged but never exposed in responses.
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    interface{} // client-safe extra context (e.g. field validation errors)
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// WithCause attaches an internal cause for logging without changing the client message.
func (e *AppError) WithCause(err error) *AppError {
	clone := *e
	clone.Err = err
	return &clone
}

// Internal error codes (stable identifiers consumed by clients).
const (
	CodeValidation      = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeUnprocessable   = "UNPROCESSABLE_ENTITY"
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
	CodeInternal        = "INTERNAL_ERROR"

	CodeIdempotencyKeyRequired = "IDEMPOTENCY_KEY_REQUIRED"
	CodeIdempotencyKeyInvalid  = "IDEMPOTENCY_KEY_INVALID"
	CodeIdempotencyKeyReused   = "IDEMPOTENCY_KEY_REUSED"
	CodeIdempotencyInProgress  = "IDEMPOTENCY_IN_PROGRESS"
)

func newAppError(status int, code, message string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message}
}

func NewValidation(message string) *AppError {
	return newAppError(http.StatusBadRequest, CodeValidation, message)
}

// NewValidationDetails is a 400 carrying per-field validation details.
func NewValidationDetails(message string, details interface{}) *AppError {
	e := newAppError(http.StatusBadRequest, CodeValidation, message)
	e.Details = details
	return e
}

// NewBadRequest is a 400 with an explicit internal code.
func NewBadRequest(code, message string) *AppError {
	return newAppError(http.StatusBadRequest, code, message)
}

func NewUnauthorized(message string) *AppError {
	return newAppError(http.StatusUnauthorized, CodeUnauthorized, message)
}

func NewForbidden(message string) *AppError {
	return newAppError(http.StatusForbidden, CodeForbidden, message)
}

func NewNotFound(message string) *AppError {
	return newAppError(http.StatusNotFound, CodeNotFound, message)
}

func NewConflict(code, message string) *AppError {
	return newAppError(http.StatusConflict, code, message)
}

func NewUnprocessable(message string) *AppError {
	return newAppError(http.StatusUnprocessableEntity, CodeUnprocessable, message)
}

// NewInternal wraps an unexpected error as a generic 500 without leaking detail.
func NewInternal(err error) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       CodeInternal,
		Message:    "An unexpected error occurred",
		Err:        err,
	}
}

// AsAppError extracts an *AppError from an error chain, if present.
func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// CodeForStatus maps an HTTP status to a stable internal error code, used when
// rendering framework-level errors that are not AppErrors.
func CodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeValidation
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusUnprocessableEntity:
		return CodeUnprocessable
	case http.StatusTooManyRequests:
		return CodeTooManyRequests
	default:
		if status >= 500 {
			return CodeInternal
		}
		return "ERROR"
	}
}
