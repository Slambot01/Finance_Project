package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard JSON envelope returned by every endpoint.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success sends a JSON success response with the given status code, message, and data payload.
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends a JSON error response with the given status code and message. Data is always null.
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
		Data:    nil,
	})
}

// ValidationError sends a 422 response with field-level validation error details.
func ValidationError(c *gin.Context, message string, details interface{}) {
	c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Message: message,
		Data:    details,
	})
}
