package main

import (
	"fmt"
	"log"

	"finance-dashboard/config"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	// Connect to PostgreSQL
	_ = config.ConnectDB(cfg)

	log.Println("Server starting...")
	fmt.Printf("Listening on port %s\n", cfg.Port)
}
