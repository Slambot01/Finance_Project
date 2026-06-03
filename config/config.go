package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config holds all configuration values loaded from environment variables.
type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecret    string
	JWTExpiryHrs int
	Port         string

	// Rate limiting.
	RateLimitPerMinute int

	// Token lifecycle.
	AccessTokenExpiryMinutes int
	RefreshTokenExpiryDays   int
}

// Load reads the .env file and returns a populated Config struct.
func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, falling back to system environment variables")
	}

	jwtExpiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		log.Fatalf("Invalid JWT_EXPIRY_HOURS value: %v", err)
	}

	rateLimitPerMin, _ := strconv.Atoi(getEnv("RATE_LIMIT_PER_MINUTE", "100"))
	accessTokenExpiry, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_EXPIRY_MINUTES", "15"))
	refreshTokenExpiry, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRY_DAYS", "7"))

	return &Config{
		DBHost:                   getEnv("DB_HOST", "localhost"),
		DBPort:                   getEnv("DB_PORT", "5432"),
		DBUser:                   getEnv("DB_USER", "postgres"),
		DBPassword:               getEnv("DB_PASSWORD", ""),
		DBName:                   getEnv("DB_NAME", "finance_dashboard"),
		JWTSecret:                getEnv("JWT_SECRET", ""),
		JWTExpiryHrs:             jwtExpiry,
		Port:                     getEnv("PORT", "8080"),
		RateLimitPerMinute:       rateLimitPerMin,
		AccessTokenExpiryMinutes: accessTokenExpiry,
		RefreshTokenExpiryDays:   refreshTokenExpiry,
	}
}

// ConnectDB establishes a connection to PostgreSQL using GORM, configures
// the connection pool for production use, and returns the DB instance.
func ConnectDB(cfg *Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool for production workloads.
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Database connection established successfully")
	return db
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
