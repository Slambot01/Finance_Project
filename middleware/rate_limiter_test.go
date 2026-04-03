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

// buildRateLimitRouter creates a Gin engine with the RateLimiter middleware.
func buildRateLimitRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api", RateLimiter(), func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "ok", nil)
	})
	return r
}

// clearLimiters resets the package-level limiters map to ensure test isolation.
func clearLimiters() {
	limiters.Range(func(key, value interface{}) bool {
		limiters.Delete(key)
		return true
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("Single_request_passes_through", func(t *testing.T) {
		clearLimiters()
		router := buildRateLimitRouter()

		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body utils.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.True(t, body.Success)
	})

	t.Run("Multiple_requests_under_limit_all_pass", func(t *testing.T) {
		clearLimiters()
		router := buildRateLimitRouter()

		for i := 0; i < 50; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			req.RemoteAddr = "10.0.0.2:12345"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
		}
	})

	t.Run("Exceeding_100_requests_same_IP_returns_429", func(t *testing.T) {
		clearLimiters()
		router := buildRateLimitRouter()

		// The rate limiter is configured with burst=100, so the first 100 requests
		// consume the burst bucket instantly. Request 101 should be rejected.
		got429 := false
		for i := 0; i < 110; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			req.RemoteAddr = "10.0.0.3:12345"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				got429 = true

				var body utils.APIResponse
				err := json.Unmarshal(w.Body.Bytes(), &body)
				assert.NoError(t, err)
				assert.False(t, body.Success)
				assert.Contains(t, body.Message, "rate limit exceeded")
				break
			}
		}
		assert.True(t, got429, "expected at least one 429 response after exceeding burst limit")
	})

	t.Run("Different_IPs_do_not_affect_each_other", func(t *testing.T) {
		clearLimiters()
		router := buildRateLimitRouter()

		// Exhaust the burst for IP-A.
		for i := 0; i < 101; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			req.RemoteAddr = "10.0.0.4:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		// IP-B should still be able to make requests without hitting the limit.
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.RemoteAddr = "10.0.0.5:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "different IP should not be rate-limited")
	})
}
