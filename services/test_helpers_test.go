package services

import (
	"fmt"
	"log"
	"os"
	"testing"

	"finance-dashboard/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testDB *gorm.DB

// setupTestDB connects to a test PostgreSQL database using TEST_ prefixed env vars.
// Falls back to regular env vars with _test appended to the DB name if TEST_ vars are missing.
func setupTestDB() *gorm.DB {
	// Try loading .env.test first, then fall back to .env
	_ = godotenv.Load("../.env.test")
	_ = godotenv.Load("../.env")

	host := getTestEnv("TEST_DB_HOST", getTestEnv("DB_HOST", "localhost"))
	port := getTestEnv("TEST_DB_PORT", getTestEnv("DB_PORT", "5432"))
	user := getTestEnv("TEST_DB_USER", getTestEnv("DB_USER", "postgres"))
	password := getTestEnv("TEST_DB_PASSWORD", getTestEnv("DB_PASSWORD", ""))
	dbName := getTestEnv("TEST_DB_NAME", "")

	// Fall back to regular DB name with _test suffix.
	if dbName == "" {
		regularName := getTestEnv("DB_NAME", "finance_dashboard")
		dbName = regularName + "_test"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate all models.
	if err := db.AutoMigrate(&models.User{}, &models.FinancialRecord{}); err != nil {
		log.Fatalf("Failed to auto-migrate test database: %v", err)
	}

	log.Printf("Connected to test database: %s", dbName)
	return db
}

// cleanupTables truncates all tables to ensure test isolation.
func cleanupTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE financial_records CASCADE")
	db.Exec("TRUNCATE TABLE users CASCADE")
}

// getTestEnv retrieves an environment variable or returns the fallback.
func getTestEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// TestMain sets up the test database once, runs all tests, then cleans up.
func TestMain(m *testing.M) {
	testDB = setupTestDB()

	code := m.Run()

	// Final cleanup after all tests.
	cleanupTables(testDB)

	os.Exit(code)
}
