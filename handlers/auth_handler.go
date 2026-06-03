package handlers

import (
	"net/http"
	"net/mail"
	"strings"

	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	Service *services.AuthService
}

// registerRequest defines the expected JSON body for user registration.
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// loginRequest defines the expected JSON body for user login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /auth/register — creates a new user account.
// Admin role cannot be self-assigned during registration. Users must register
// as viewer or analyst, then be promoted by an existing admin.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	// Validate required fields.
	missing := make([]string, 0, 3)
	if strings.TrimSpace(req.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(req.Email) == "" {
		missing = append(missing, "email")
	}
	if strings.TrimSpace(req.Password) == "" {
		missing = append(missing, "password")
	}
	if len(missing) > 0 {
		utils.ValidationError(c, "missing required fields", map[string]interface{}{
			"required": missing,
		})
		return
	}

	// Validate email format.
	if _, err := mail.ParseAddress(req.Email); err != nil {
		utils.ValidationError(c, "invalid email format", nil)
		return
	}

	// Enforce minimum password strength for financial security.
	if len(req.Password) < 8 {
		utils.ValidationError(c, "password must be at least 8 characters", nil)
		return
	}

	user, err := h.Service.Register(req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "user registered successfully", user)
}

// Login handles POST /auth/login — authenticates a user and returns a JWT
// access token, a refresh token, and the user record.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	// Validate required fields.
	missing := make([]string, 0, 2)
	if strings.TrimSpace(req.Email) == "" {
		missing = append(missing, "email")
	}
	if strings.TrimSpace(req.Password) == "" {
		missing = append(missing, "password")
	}
	if len(missing) > 0 {
		utils.ValidationError(c, "missing required fields", map[string]interface{}{
			"required": missing,
		})
		return
	}

	// Extract request metadata for audit logging.
	requestID := c.GetString("requestID")
	ipAddress := c.ClientIP()

	accessToken, refreshToken, user, err := h.Service.Login(c.Request.Context(), req.Email, req.Password, requestID, ipAddress)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	responseData := map[string]interface{}{
		"access_token": accessToken,
		"user":         user,
	}
	// Only include refresh_token if one was issued (requires TokenService to be wired).
	if refreshToken != "" {
		responseData["refresh_token"] = refreshToken
	}

	utils.Success(c, http.StatusOK, "login successful", responseData)
}
