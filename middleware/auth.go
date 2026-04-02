package middleware

import (
	"net/http"
	"strings"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the JWT from the Authorization header and injects
// the authenticated user's identity into the Gin context for downstream handlers.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "authorization header is required")
			c.Abort()
			return
		}

		// Expect format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.Error(c, http.StatusUnauthorized, "authorization header must be in the format: Bearer <token>")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			utils.Error(c, http.StatusUnauthorized, "token is required")
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "invalid or expired token: "+err.Error())
			c.Abort()
			return
		}

		// Inject authenticated user info into context for downstream handlers.
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}
