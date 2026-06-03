package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finance-dashboard/config"
	"finance-dashboard/models"
	"finance-dashboard/routes"
	"finance-dashboard/services"
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

	// Auto-migrate all database tables including new models.
	err := db.AutoMigrate(
		&models.User{},
		&models.FinancialRecord{},
		&models.Account{},
		&models.LedgerEntry{},
		&models.AuditEvent{},
		&models.OutboxEntry{},
		&models.RefreshToken{},
		&models.IdempotencyKey{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database migration completed successfully")

	// Start the outbox publisher background worker.
	ctx, cancel := context.WithCancel(context.Background())
	publisher := &services.OutboxPublisher{
		DB:       db,
		Interval: 5 * time.Second,
	}
	go publisher.Start(ctx)

	// Wire up routes, middleware, and handlers
	router := routes.SetupRoutes(db)

	// Determine port
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	// Use http.Server for graceful shutdown support.
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		log.Printf("Server running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown: wait for SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	// Cancel the outbox publisher.
	cancel()

	// Allow up to 30 seconds for in-flight requests to complete.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection pool.
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	log.Println("Server stopped cleanly")
}
