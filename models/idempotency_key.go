package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdempotencyKey stores the result of a previous mutating request so that
// retried requests with the same key return the cached response instead of
// creating duplicates. Keys expire after 24 hours.
type IdempotencyKey struct {
	Key          string    `gorm:"type:varchar(255);primaryKey" json:"key"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	RequestHash  string    `gorm:"type:varchar(64);not null" json:"-"`        // SHA-256 of request body to detect mismatches
	ResponseCode int       `gorm:"not null" json:"-"`
	ResponseBody string    `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `gorm:"not null;index" json:"expires_at"`
}

// BeforeCreate sets the expiry to 24 hours from now if not already set.
func (k *IdempotencyKey) BeforeCreate(tx *gorm.DB) error {
	if k.ExpiresAt.IsZero() {
		k.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	return nil
}
