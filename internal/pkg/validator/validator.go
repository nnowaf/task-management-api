package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// FieldError is a single human-readable validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate runs struct validation and returns a readable summary plus per-field details.
// ok is false when validation fails.
func Validate(s interface{}) (summary string, details []FieldError, ok bool) {
	err := validate.Struct(s)
	if err == nil {
		return "", nil, true
	}

	var verrs validator.ValidationErrors
	if !asValidationErrors(err, &verrs) {
		return err.Error(), nil, false
	}

	msgs := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		field := strings.ToLower(fe.Field())
		msg := messageFor(fe)
		details = append(details, FieldError{Field: field, Message: msg})
		msgs = append(msgs, msg)
	}
	return strings.Join(msgs, "; "), details, false
}

func asValidationErrors(err error, target *validator.ValidationErrors) bool {
	if v, ok := err.(validator.ValidationErrors); ok {
		*target = v
		return true
	}
	return false
}

func messageFor(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, fe.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
