package handlers

import (
	"net/http"

	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TokenHandler handles token lifecycle HTTP requests.
type TokenHandler struct {
	TokenService *services.TokenService
}

// refreshRequest defines the expected JSON body for token refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh — validates a refresh token, rotates it,
// and returns a new access + refresh token pair.
func (h *TokenHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	if req.RefreshToken == "" {
		utils.Error(c, http.StatusBadRequest, "refresh_token is required")
		return
	}

	accessToken, refreshToken, err := h.TokenService.RefreshTokens(req.RefreshToken)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "tokens refreshed successfully", map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout handles POST /auth/logout — revokes all refresh tokens for the
// authenticated user, effectively logging them out from all devices.
func (h *TokenHandler) Logout(c *gin.Context) {
	userID := c.GetString("userID")
	parsedUID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid user identity in token")
		return
	}

	if err := h.TokenService.RevokeAllUserTokens(parsedUID); err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "logged out successfully — all refresh tokens revoked", nil)
}
