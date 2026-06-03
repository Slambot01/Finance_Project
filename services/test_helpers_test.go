package services

import (
	"fmt"
	"log"
	"os"
	"testing"

	"finance-dashboard/models"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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

	host := getTestEnv("TEST_DB_HOST", getTestEnv("DB_HOST", "127.0.0.1"))
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

	log.Println("Connecting to test database with DSN:", dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	log.Println("Connected to test database.")

	log.Println("Running AutoMigrate...")
	// Auto-migrate all models.
	err = db.AutoMigrate(
		&models.User{},
		&models.FinancialRecord{},
		&models.Account{},
		&models.LedgerEntry{},
		&models.AuditEvent{},
		&models.OutboxEntry{},
		&models.IdempotencyKey{},
		&models.RefreshToken{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate test database: %v", err)
	}
	log.Println("AutoMigrate completed.")

	log.Printf("Connected to test database: %s", dbName)
	return db
}

// cleanupTables truncates all tables to ensure test isolation.
func cleanupTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE financial_records CASCADE")
	db.Exec("TRUNCATE TABLE users CASCADE")
}

// createTestUser is a helper that creates a user with the given role directly
// in the database, bypassing the registration service. This is necessary because
// admin self-registration is blocked — admins can only be created by other admins
// in production, or directly in the DB for tests.
func createTestUser(t *testing.T, name, email, role string) *models.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     models.RoleType(role),
		IsActive: true,
	}

	if err := testDB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return &user
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
