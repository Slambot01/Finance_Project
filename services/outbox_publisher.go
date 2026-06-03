package services

import (
	"context"
	"log/slog"
	"time"

	"finance-dashboard/models"

	"gorm.io/gorm"
)

// OutboxPublisher is a background worker that polls the outbox table for
// unpublished entries and processes them. Currently, processing means logging
// the event; in production, this would forward to Kafka, webhooks, or an
// event bus.
//
// The outbox pattern ensures that events are only processed AFTER the business
// transaction has committed, preventing phantom events from rolled-back
// transactions.
type OutboxPublisher struct {
	DB       *gorm.DB
	Interval time.Duration
}

// Start begins the polling loop. It runs until the context is cancelled,
// supporting graceful shutdown.
func (p *OutboxPublisher) Start(ctx context.Context) {
	slog.Info("outbox publisher started", slog.Duration("interval", p.Interval))

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox publisher shutting down")
			return
		case <-ticker.C:
			p.processOutbox()
		}
	}
}

// processOutbox fetches and processes unpublished outbox entries in batches.
func (p *OutboxPublisher) processOutbox() {
	var entries []models.OutboxEntry

	// Fetch up to 50 unpublished entries ordered by creation time.
	if err := p.DB.Where("published_at IS NULL").
		Order("created_at ASC").
		Limit(50).
		Find(&entries).Error; err != nil {
		slog.Error("outbox publisher: failed to fetch entries", slog.String("error", err.Error()))
		return
	}

	if len(entries) == 0 {
		return
	}

	slog.Info("outbox publisher: processing entries", slog.Int("count", len(entries)))

	for _, entry := range entries {
		// In production, this would publish to Kafka, call a webhook, or
		// push to an event bus. For now, we log the event for observability.
		slog.Info("outbox event published",
			slog.String("event_id", entry.EventID.String()),
			slog.String("event_type", entry.EventType),
		)

		// Mark as published.
		now := time.Now()
		if err := p.DB.Model(&entry).Update("published_at", &now).Error; err != nil {
			slog.Error("outbox publisher: failed to mark entry as published",
				slog.String("entry_id", entry.ID.String()),
				slog.String("error", err.Error()),
			)
			// Continue processing other entries — one failure shouldn't block the rest.
			continue
		}
	}

	// Periodic cleanup of expired data to prevent unbounded table growth.
	p.cleanupExpiredIdempotencyKeys()
	p.cleanupExpiredRefreshTokens()
}

// cleanupExpiredIdempotencyKeys deletes idempotency keys past their 24h TTL.
func (p *OutboxPublisher) cleanupExpiredIdempotencyKeys() {
	result := p.DB.Where("expires_at < ?", time.Now()).Delete(&models.IdempotencyKey{})
	if result.Error != nil {
		slog.Error("cleanup: failed to delete expired idempotency keys", slog.String("error", result.Error.Error()))
	} else if result.RowsAffected > 0 {
		slog.Info("cleanup: deleted expired idempotency keys", slog.Int64("count", result.RowsAffected))
	}
}

// cleanupExpiredRefreshTokens deletes refresh tokens that are both expired and revoked.
func (p *OutboxPublisher) cleanupExpiredRefreshTokens() {
	result := p.DB.Where("expires_at < ? AND revoked_at IS NOT NULL", time.Now()).Delete(&models.RefreshToken{})
	if result.Error != nil {
		slog.Error("cleanup: failed to delete expired refresh tokens", slog.String("error", result.Error.Error()))
	} else if result.RowsAffected > 0 {
		slog.Info("cleanup: deleted expired refresh tokens", slog.Int64("count", result.RowsAffected))
	}
}
