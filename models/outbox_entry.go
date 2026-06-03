package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboxEntry implements the transactional outbox pattern. Each outbox entry
// is written in the same database transaction as the corresponding audit event,
// guaranteeing consistency between business data and the event stream.
//
// A background publisher goroutine polls for unpublished entries and processes
// them (currently logging; future: webhook/Kafka delivery). This solves the
// "dual write" problem without requiring distributed transactions.
type OutboxEntry struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EventID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"event_id"`
	AuditEvent  AuditEvent `gorm:"foreignKey:EventID" json:"-"`
	EventType   string     `gorm:"type:varchar(50);not null" json:"event_type"` // e.g. "user.registered", "record.created"
	Payload     string     `gorm:"type:text;not null" json:"payload"`           // JSON serialized event data
	PublishedAt *time.Time `gorm:"index" json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// BeforeCreate generates a new UUID before inserting an OutboxEntry.
func (o *OutboxEntry) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
