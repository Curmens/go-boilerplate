package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta contains pagination metadata
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// Common error codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeTooManyRequest = "TOO_MANY_REQUESTS"
)

// Success sends a success response with data
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage sends a success response with message and data
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a success response with data and pagination meta
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: "Resource created successfully",
		Data:    data,
	})
}

// NoContent sends a 204 no content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response with the given status code
func Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails sends an error response with additional details
func ErrorWithDetails(c *gin.Context, statusCode int, code, message string, details map[string]string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest sends a 400 bad request response
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, ErrCodeBadRequest, message)
}

// BadRequestWithDetails sends a 400 bad request response with details
func BadRequestWithDetails(c *gin.Context, message string, details map[string]string) {
	ErrorWithDetails(c, http.StatusBadRequest, ErrCodeBadRequest, message, details)
}

// ValidationError sends a 422 validation error response
func ValidationError(c *gin.Context, details map[string]string) {
	ErrorWithDetails(c, http.StatusUnprocessableEntity, ErrCodeValidation, "Validation failed", details)
}

// Unauthorized sends a 401 unauthorized response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	Error(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// Forbidden sends a 403 forbidden response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	Error(c, http.StatusForbidden, ErrCodeForbidden, message)
}

// NotFound sends a 404 not found response
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Resource not found"
	}
	Error(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// Conflict sends a 409 conflict response
func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "Resource already exists"
	}
	Error(c, http.StatusConflict, ErrCodeConflict, message)
}

// TooManyRequests sends a 429 rate limit response
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "Rate limit exceeded"
	}
	Error(c, http.StatusTooManyRequests, ErrCodeTooManyRequest, message)
}

// InternalServerError sends a 500 internal server error response
func InternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "An internal error occurred"
	}
	Error(c, http.StatusInternalServerError, ErrCodeInternal, message)
}

// AbortWithError aborts the request and sends an error response
func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	c.AbortWithStatusJSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// Pagination helper
func NewMeta(page, perPage int, total int64) *Meta {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// GetPaginationParams extracts pagination parameters from query
func GetPaginationParams(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = 10

	if p := c.Query("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pp := c.Query("per_page"); pp != "" {
		if parsed, err := parseInt(pp); err == nil && parsed > 0 && parsed <= 100 {
			perPage = parsed
		}
	}

	return page, perPage
}

// GetOffset calculates the offset for pagination
func GetOffset(page, perPage int) int {
	return (page - 1) * perPage
}

func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
