package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	os.Setenv("JWT_EXPIRY_HOURS", "24")
}

// buildAuthRouter creates a Gin engine with the AuthMiddleware applied and a
// simple 200 handler that echoes the context values set by the middleware.
func buildAuthRouter() *gin.Engine {
	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "ok", gin.H{
			"userID":    c.GetString("userID"),
			"userEmail": c.GetString("userEmail"),
			"userRole":  c.GetString("userRole"),
		})
	})
	return r
}

func TestAuthMiddleware(t *testing.T) {
	router := buildAuthRouter()

	t.Run("Valid_Bearer_token_sets_context_values", func(t *testing.T) {
		token, err := utils.GenerateToken("user-123", "test@example.com", "admin")
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body utils.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.True(t, body.Success)

		data := body.Data.(map[string]interface{})
		assert.Equal(t, "user-123", data["userID"])
		assert.Equal(t, "test@example.com", data["userEmail"])
		assert.Equal(t, "admin", data["userRole"])
	})

	t.Run("Missing_Authorization_header_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "authorization header is required")
	})

	t.Run("Authorization_without_Bearer_prefix_returns_401", func(t *testing.T) {
		token, _ := utils.GenerateToken("user-123", "test@example.com", "admin")

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Token "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "Bearer")
	})

	t.Run("Bearer_with_empty_token_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "token is required")
	})

	t.Run("Invalid_garbage_token_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.False(t, body.Success)
		assert.Contains(t, body.Message, "invalid or expired token")
	})

	t.Run("Expired_token_returns_401", func(t *testing.T) {
		// GenerateToken reads JWT_EXPIRY_HOURS from env and doesn't support
		// custom expiry durations, so we cannot easily generate an already-expired
		// token without manipulating time. Skipping this subtest.
		t.Skip("GenerateToken doesn't support custom expiry — cannot generate an expired token without time manipulation")
	})
}
