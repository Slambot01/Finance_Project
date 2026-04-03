package middleware

import (
	"net/http"
	"sync"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var limiters sync.Map

func getLimiter(ip string) *rate.Limiter {
	if v, ok := limiters.Load(ip); ok {
		return v.(*rate.Limiter)
	}
	// 100 requests per minute ≈ ~1.67 requests/second, burst of 100.
	limiter := rate.NewLimiter(rate.Limit(100.0/60.0), 100)
	limiters.Store(ip, limiter)
	return limiter
}

// RateLimiter restricts each client IP to 100 requests per minute.
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !getLimiter(c.ClientIP()).Allow() {
			utils.Error(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}
		c.Next()
	}
}
