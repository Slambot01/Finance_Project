package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey is the Gin context key used to store/retrieve the request ID.
	RequestIDKey = "requestID"
	// RequestIDHeader is the HTTP header name for the request ID.
	RequestIDHeader = "X-Request-ID"
)

// RequestID assigns a unique UUID v4 to every incoming request. If the client
// provides an X-Request-ID header, it is respected; otherwise a new one is
// generated. The ID is stored in the Gin context and set as a response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		// Validate client-supplied IDs to prevent log injection attacks.
		if id != "" {
			if _, err := uuid.Parse(id); err != nil {
				id = "" // reject invalid IDs
			}
		}
		if id == "" {
			id = uuid.New().String()
		}

		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)

		c.Next()
	}
}
