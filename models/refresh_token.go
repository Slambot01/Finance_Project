package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken stores hashed refresh tokens in the database, enabling token
// rotation and revocation. Each login creates a new "family" — when a token
// is rotated, the replacement inherits the family ID. If a revoked token from
// the same family is ever presented, all tokens in the family are revoked,
// detecting replay attacks.
type RefreshToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_refresh_user_revoked" json:"user_id"`
	User       User       `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	TokenHash  string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"` // SHA-256 hash of the raw token
	FamilyID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"-"`             // Token family for rotation tracking
	ExpiresAt  time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt  *time.Time `gorm:"index:idx_refresh_user_revoked" json:"revoked_at,omitempty"`
	ReplacedBy *uuid.UUID `gorm:"type:uuid" json:"-"` // Points to the rotated successor token
	CreatedAt  time.Time  `json:"created_at"`
}

// BeforeCreate generates a new UUID before inserting a RefreshToken.
func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// IsRevoked returns true if the token has been explicitly revoked.
func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}

// IsExpired returns true if the token has passed its expiration time.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
