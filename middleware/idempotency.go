package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"finance-dashboard/models"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdempotencyMiddleware prevents duplicate processing of mutating requests.
// Clients include an Idempotency-Key header; if the key was seen before and
// the request body matches, the cached response is returned. If the key was
// seen with a different body, a 422 Unprocessable Entity is returned.
//
// Keys expire after 24 hours and are scoped per-user.
func IdempotencyMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to mutating methods.
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			// No idempotency key provided — proceed normally.
			c.Next()
			return
		}

		userID := c.GetString("userID")

		// Read and hash the request body for mismatch detection.
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "failed to read request body")
			c.Abort()
			return
		}
		// Restore the body so downstream handlers can read it.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestHash := hashBody(bodyBytes)

		// Check if this key already exists.
		var existing models.IdempotencyKey
		result := db.Where("\"key\" = ? AND expires_at > ?", key, time.Now()).First(&existing)

		if result.Error == nil {
			// Key exists — check if the request body matches.
			if existing.RequestHash != requestHash {
				utils.Error(c, http.StatusUnprocessableEntity,
					"idempotency key already used with a different request body")
				c.Abort()
				return
			}

			// Return the cached response.
			c.Data(existing.ResponseCode, "application/json; charset=utf-8", []byte(existing.ResponseBody))
			c.Abort()
			return
		}

		// Key not found — capture the response.
		writer := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// Store the response for future replays.
		idempotencyRecord := models.IdempotencyKey{
			Key:          key,
			RequestHash:  requestHash,
			ResponseCode: writer.statusCode,
			ResponseBody: writer.body.String(),
		}

		// Parse userID if present.
		if userID != "" {
			if uid, parseErr := parseUUID(userID); parseErr == nil {
				idempotencyRecord.UserID = uid
			}
		}

		// Best-effort storage — don't fail the request if caching fails.
		_ = db.Create(&idempotencyRecord).Error
	}
}

// responseCapture wraps gin.ResponseWriter to capture the response body and status.
type responseCapture struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func hashBody(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// parseUUID parses a UUID string using the google/uuid library.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
