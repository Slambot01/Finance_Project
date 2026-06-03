package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"finance-dashboard/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupIdempotencyTestDB connects to a test PostgreSQL database.
func setupIdempotencyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	_ = godotenv.Load("../.env.test")
	_ = godotenv.Load("../.env")

	host := idempEnv("TEST_DB_HOST", idempEnv("DB_HOST", "127.0.0.1"))
	port := idempEnv("TEST_DB_PORT", idempEnv("DB_PORT", "5432"))
	user := idempEnv("TEST_DB_USER", idempEnv("DB_USER", "postgres"))
	password := idempEnv("TEST_DB_PASSWORD", idempEnv("DB_PASSWORD", ""))
	dbName := idempEnv("TEST_DB_NAME", idempEnv("DB_NAME", "finance_dashboard")+"_test")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.IdempotencyKey{}); err != nil {
		t.Fatalf("failed to migrate idempotency_keys: %v", err)
	}

	// Clean slate.
	db.Exec("TRUNCATE TABLE idempotency_keys CASCADE")

	return db
}

// idempEnv retrieves an environment variable or returns the fallback.
func idempEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// setupIdempotencyRouter builds a minimal Gin router with the idempotency middleware
// and a simple POST handler that echoes the request body.
func setupIdempotencyRouter(db *gorm.DB, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Simulate authenticated user.
		c.Set("userID", userID)
		c.Next()
	})
	r.Use(IdempotencyMiddleware(db))

	r.POST("/test", func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"received": body})
	})

	return r
}

func TestIdempotencyMiddleware_CachesResponse(t *testing.T) {
	db := setupIdempotencyTestDB(t)
	userID := uuid.New().String()
	r := setupIdempotencyRouter(db, userID)

	idempotencyKey := uuid.New().String()
	body := `{"amount": "1000", "currency": "INR"}`

	// First request — should be processed normally.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusCreated, w1.Code)
	var resp1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	// Second request with same key and same body — should return cached response.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusCreated, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestIdempotencyMiddleware_RejectsMismatchedBody(t *testing.T) {
	db := setupIdempotencyTestDB(t)
	userID := uuid.New().String()
	r := setupIdempotencyRouter(db, userID)

	idempotencyKey := uuid.New().String()

	// First request.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"amount": "500"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second request with same key but DIFFERENT body — should be rejected.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"amount": "9999"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnprocessableEntity, w2.Code)
}

func TestIdempotencyMiddleware_SkipsGetRequests(t *testing.T) {
	db := setupIdempotencyTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IdempotencyMiddleware(db))
	callCount := 0
	r.GET("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// GET with an idempotency key — middleware should be a no-op.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Idempotency-Key", uuid.New().String())
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
	assert.Equal(t, 3, callCount) // All 3 requests processed — no caching on GET.
}

func TestIdempotencyMiddleware_NoKeyPassesThrough(t *testing.T) {
	db := setupIdempotencyTestDB(t)
	userID := uuid.New().String()
	r := setupIdempotencyRouter(db, userID)

	// POST without Idempotency-Key — should pass through normally every time.
	body := `{"amount": "100"}`
	var lastStatus int
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		// No Idempotency-Key header
		r.ServeHTTP(w, req)
		lastStatus = w.Code
	}
	assert.Equal(t, http.StatusCreated, lastStatus)
}
