package main

import (
	"fmt"
	"log"

	"finance-dashboard/config"
	"finance-dashboard/models"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	// Connect to PostgreSQL
	db := config.ConnectDB(cfg)

	// Auto-migrate database tables
	err := db.AutoMigrate(&models.User{}, &models.FinancialRecord{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database migration completed successfully")

	log.Println("Server starting...")
	fmt.Printf("Listening on port %s\n", cfg.Port)
}
