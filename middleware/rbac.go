package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a Gin middleware that enforces role-based access control.
// It checks the userRole value set by AuthMiddleware against the provided allowedRoles.
//
// Usage:
//
//	api.GET("/users", middleware.RequireRole("admin"), handlers.GetUsers)
//	api.GET("/records", middleware.RequireRole("viewer", "analyst", "admin"), handlers.GetRecords)
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	// Pre-build the allowed set for O(1) lookups.
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}

	return func(c *gin.Context) {
		roleValue, exists := c.Get("userRole")
		if !exists {
			utils.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		userRole, ok := roleValue.(string)
		if !ok || userRole == "" {
			utils.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		if _, permitted := allowed[strings.ToLower(userRole)]; !permitted {
			msg := fmt.Sprintf("access denied: requires [%s] role", strings.Join(allowedRoles, ", "))
			utils.Error(c, http.StatusForbidden, msg)
			c.Abort()
			return
		}

		c.Next()
	}
}
