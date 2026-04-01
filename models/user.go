package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleType represents the user's role within the system.
type RoleType string

const (
	RoleViewer  RoleType = "viewer"
	RoleAnalyst RoleType = "analyst"
	RoleAdmin   RoleType = "admin"
)

// User represents an authenticated user of the finance dashboard.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	Role      RoleType  `gorm:"type:varchar(20);not null;default:'viewer'" json:"role"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate generates a new UUID before inserting a User record.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
