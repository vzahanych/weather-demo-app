package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	validate.RegisterValidation("latitude", validateLatitude)
	validate.RegisterValidation("longitude", validateLongitude)

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func GetValidator() *validator.Validate {
	return validate
}

func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90.0 && lat <= 90.0
}

func validateLongitude(fl validator.FieldLevel) bool {
	lon := fl.Field().Float()
	return lon >= -180.0 && lon <= 180.0
}

type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

func FormatValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validatorErrs, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validatorErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Value:   err.Value(),
				Tag:     err.Tag(),
				Message: getErrorMessage(err),
			})
		}
	}

	return validationErrors
}

func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "latitude":
		return fmt.Sprintf("%s must be a valid latitude between -90 and 90 degrees", err.Field())
	case "longitude":
		return fmt.Sprintf("%s must be a valid longitude between -180 and 180 degrees", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", err.Field(), err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("%s is invalid", err.Field())
	}
}

func ValidateStruct(s interface{}) []ValidationError {
	err := validate.Struct(s)
	if err != nil {
		return FormatValidationErrors(err)
	}
	return nil
}
