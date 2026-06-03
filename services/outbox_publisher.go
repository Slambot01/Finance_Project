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
}
