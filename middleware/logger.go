package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogger logs each request with structured fields using Go's slog
// package. Fields include: method, path, status code, latency, client IP,
// request ID (from RequestID middleware), and authenticated user ID if present.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process the request.
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.ClientIP()),
		}

		// Attach request ID if available.
		if reqID, exists := c.Get(RequestIDKey); exists {
			attrs = append(attrs, slog.String("request_id", reqID.(string)))
		}

		// Attach authenticated user ID if available.
		if userID := c.GetString("userID"); userID != "" {
			attrs = append(attrs, slog.String("user_id", userID))
		}

		// Log at appropriate level based on status code.
		args := make([]any, len(attrs))
		for i, a := range attrs {
			args[i] = a
		}

		switch {
		case status >= 500:
			slog.Error("request completed", args...)
		case status >= 400:
			slog.Warn("request completed", args...)
		default:
			slog.Info("request completed", args...)
		}
	}
}
