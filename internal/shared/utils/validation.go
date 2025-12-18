package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
)

var validate *validator.Validate

// init initializes the validator
func init() {
	validate = validator.New()

	// Use JSON tag names for validation errors
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidateStruct validates a struct and returns a user-friendly error
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors := err.(validator.ValidationErrors)
	if len(validationErrors) == 0 {
		return nil
	}

	// Create a detailed error message
	var errorMessages []string
	for _, fieldError := range validationErrors {
		errorMessages = append(errorMessages, getFieldErrorMessage(fieldError))
	}

	return errors.NewValidationError(
		"Validation failed",
		strings.Join(errorMessages, "; "),
	)
}

// getFieldErrorMessage returns a user-friendly error message for a field validation error
func getFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("%s must be at least %s characters long", field, param)
		}
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("%s must be at most %s characters long", field, param)
		}
		return fmt.Sprintf("%s must be at most %s", field, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, param)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a valid number", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uri":
		return fmt.Sprintf("%s must be a valid URI", field)
	default:
		return fmt.Sprintf("%s failed validation for '%s'", field, tag)
	}
}

// ValidateID validates that an ID string is not empty and follows expected format
func ValidateID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.NewValidationError("ID cannot be empty")
	}
	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, pageSize int) error {
	if page < 1 {
		return errors.NewValidationError("Page must be greater than 0")
	}
	if pageSize < 1 {
		return errors.NewValidationError("Page size must be greater than 0")
	}
	if pageSize > constants.MaxPageSize {
		return errors.NewValidationError(fmt.Sprintf("Page size must not exceed %d", constants.MaxPageSize))
	}
	return nil
}
