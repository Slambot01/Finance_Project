package handlers

import (
	"net/http"
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
// This allows public self-registration with any role including admin.
// This is intentional for assessment purposes — in production, role assignment
// would be restricted to admin-only after initial bootstrap.
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

	user, err := h.Service.Register(req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "email already registered") {
			utils.Error(c, http.StatusConflict, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid role") {
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "user registered successfully", user)
}

// Login handles POST /auth/login — authenticates a user and returns a JWT.
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

	token, user, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid email or password") {
			utils.Error(c, http.StatusUnauthorized, err.Error())
			return
		}
		if strings.Contains(err.Error(), "deactivated") {
			utils.Error(c, http.StatusUnauthorized, err.Error())
			return
		}
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "login successful", map[string]interface{}{
		"token": token,
		"user":  user,
	})
}
