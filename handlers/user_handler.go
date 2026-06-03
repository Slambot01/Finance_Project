package handlers

import (
	"net/http"

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
		handleServiceError(c, err)
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
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "user updated successfully", user)
}

// DeleteUser handles DELETE /users/:id — permanently removes a user.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	err := h.Service.DeleteUser(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "user deleted successfully", nil)
}
