package handlers

import (
	"net/http"
	"strings"

	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user management HTTP requests.
type UserHandler struct {
	Service *services.UserService
}

// GetUsers handles GET /users — returns all users.
func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.Service.GetAllUsers()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "users retrieved successfully", users)
}

// UpdateUser handles PUT /users/:id — applies partial updates to a user.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	user, err := h.Service.UpdateUser(id, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.Error(c, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid role") || strings.Contains(err.Error(), "role must be") {
			utils.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "user updated successfully", user)
}

// DeleteUser handles DELETE /users/:id — permanently removes a user.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	err := h.Service.DeleteUser(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.Error(c, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "user deleted successfully", nil)
}
