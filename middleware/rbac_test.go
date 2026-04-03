package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// buildRBACRouter creates a Gin engine with RequireRole middleware and a simple
// handler that manually sets userRole in the context before RequireRole runs.
// We use a small wrapper middleware to inject the role, simulating AuthMiddleware.
func buildRBACRouter(contextRole interface{}, setRole bool, allowedRoles ...string) *gin.Engine {
	r := gin.New()

	// Inject userRole into context before RequireRole runs (simulates AuthMiddleware).
	injectRole := func(c *gin.Context) {
		if setRole {
			c.Set("userRole", contextRole)
		}
		c.Next()
	}

	r.GET("/resource", injectRole, RequireRole(allowedRoles...), func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "access granted", nil)
	})
	return r
}

func TestRequireRole(t *testing.T) {
	t.Run("User_role_matches_allowed_roles_returns_200", func(t *testing.T) {
		router := buildRBACRouter("admin", true, "admin")

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.True(t, body.Success)
		assert.Equal(t, "access granted", body.Message)
	})

	t.Run("User_role_not_in_allowed_roles_returns_403", func(t *testing.T) {
		router := buildRBACRouter("viewer", true, "admin")

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "access denied")
	})

	t.Run("Multiple_allowed_roles_user_has_one_of_them_passes", func(t *testing.T) {
		router := buildRBACRouter("analyst", true, "viewer", "analyst", "admin")

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.True(t, body.Success)
	})

	t.Run("Missing_userRole_in_context_returns_401", func(t *testing.T) {
		// Don't set userRole at all — simulates request that bypassed AuthMiddleware.
		router := buildRBACRouter(nil, false, "admin")

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "authentication required")
	})

	t.Run("Empty_role_string_in_context_returns_401", func(t *testing.T) {
		router := buildRBACRouter("", true, "admin")

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "authentication required")
	})
}
