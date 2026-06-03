package handlers

import (
	"errors"
	"net/http"

	apperrors "finance-dashboard/errors"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// handleServiceError extracts the HTTP status code from a typed AppError and
// sends the appropriate JSON response. If the error is not an AppError, it
// falls back to 500 Internal Server Error without leaking implementation details.
func handleServiceError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		utils.Error(c, appErr.StatusCode, appErr.Message)
		return
	}
	utils.Error(c, http.StatusInternalServerError, "internal server error")
}
