package utils

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// GetValidator returns the singleton validator instance
func GetValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New()

		// Use JSON tag names in error messages
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})

		// Register custom validators
		registerCustomValidators(validate)
	})

	return validate
}

// InitValidator initializes the validator with Gin
func InitValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Use JSON tag names in error messages
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})

		// Register custom validators
		registerCustomValidators(v)
	}
}

// registerCustomValidators registers custom validation rules
func registerCustomValidators(v *validator.Validate) {
	// Password validation: at least 8 chars, 1 uppercase, 1 lowercase, 1 number
	v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}
		hasUpper := false
		hasLower := false
		hasNumber := false
		for _, c := range password {
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= '0' && c <= '9':
				hasNumber = true
			}
		}
		return hasUpper && hasLower && hasNumber
	})

	// Phone number validation (basic)
	v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		phone := fl.Field().String()
		if phone == "" {
			return true // Empty is valid (use 'required' for mandatory)
		}
		// Basic phone validation: 10-15 digits, optional + prefix
		if strings.HasPrefix(phone, "+") {
			phone = phone[1:]
		}
		if len(phone) < 10 || len(phone) > 15 {
			return false
		}
		for _, c := range phone {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	})

	// No spaces validation
	v.RegisterValidation("nospaces", func(fl validator.FieldLevel) bool {
		return !strings.Contains(fl.Field().String(), " ")
	})

	// Alphanumeric with underscores
	v.RegisterValidation("alphanum_underscore", func(fl validator.FieldLevel) bool {
		for _, c := range fl.Field().String() {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
		return true
	})
}

// ValidateStruct validates a struct and returns formatted errors
func ValidateStruct(s interface{}) map[string]string {
	v := GetValidator()
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	return FormatValidationErrors(err)
}

// FormatValidationErrors converts validator errors to a map
func FormatValidationErrors(err error) map[string]string {
	if err == nil {
		return nil
	}

	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			errors[field] = getErrorMessage(e)
		}
	}

	return errors
}

// getErrorMessage returns a human-readable error message
func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address"
	case "min":
		if e.Type().Kind() == reflect.String {
			return "Must be at least " + e.Param() + " characters"
		}
		return "Must be at least " + e.Param()
	case "max":
		if e.Type().Kind() == reflect.String {
			return "Must be at most " + e.Param() + " characters"
		}
		return "Must be at most " + e.Param()
	case "len":
		return "Must be exactly " + e.Param() + " characters"
	case "password":
		return "Must be at least 8 characters with uppercase, lowercase, and number"
	case "phone":
		return "Invalid phone number"
	case "e164":
		return "Must be in E.164 format (e.g., +1234567890)"
	case "url":
		return "Invalid URL"
	case "uuid":
		return "Invalid UUID"
	case "oneof":
		return "Must be one of: " + e.Param()
	case "nospaces":
		return "Must not contain spaces"
	case "alphanum":
		return "Must contain only letters and numbers"
	case "alphanum_underscore":
		return "Must contain only letters, numbers, and underscores"
	case "eqfield":
		return "Must match " + e.Param()
	case "gtfield":
		return "Must be greater than " + e.Param()
	case "ltfield":
		return "Must be less than " + e.Param()
	default:
		return "Invalid value"
	}
}

// BindAndValidate binds request body and validates it
func BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		if validationErrors := FormatValidationErrors(err); validationErrors != nil {
			ValidationError(c, validationErrors)
		} else {
			BadRequest(c, "Invalid request body: "+err.Error())
		}
		return false
	}

	if errors := ValidateStruct(obj); errors != nil {
		ValidationError(c, errors)
		return false
	}

	return true
}

// BindQueryAndValidate binds query parameters and validates them
func BindQueryAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		if validationErrors := FormatValidationErrors(err); validationErrors != nil {
			ValidationError(c, validationErrors)
		} else {
			BadRequest(c, "Invalid query parameters: "+err.Error())
		}
		return false
	}

	if errors := ValidateStruct(obj); errors != nil {
		ValidationError(c, errors)
		return false
	}

	return true
}

// BindURIAndValidate binds URI parameters and validates them
func BindURIAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		if validationErrors := FormatValidationErrors(err); validationErrors != nil {
			ValidationError(c, validationErrors)
		} else {
			BadRequest(c, "Invalid URI parameters: "+err.Error())
		}
		return false
	}

	if errors := ValidateStruct(obj); errors != nil {
		ValidationError(c, errors)
		return false
	}

	return true
}
