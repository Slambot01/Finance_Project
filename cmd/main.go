package main

import (
	"log"

	"finance-dashboard/config"
	"finance-dashboard/models"
	"finance-dashboard/routes"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	// Ensure critical security config is present.
	if cfg.JWTSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable must be set")
	}

	// Connect to PostgreSQL
	db := config.ConnectDB(cfg)

	// Auto-migrate database tables
	err := db.AutoMigrate(&models.User{}, &models.FinancialRecord{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database migration completed successfully")

	// Wire up routes, middleware, and handlers
	router := routes.SetupRoutes(db)

	// Determine port
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
