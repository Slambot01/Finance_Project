package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RecordType represents the type of financial transaction.
type RecordType string

const (
	RecordIncome  RecordType = "income"
	RecordExpense RecordType = "expense"
)

// FinancialRecord represents a single financial transaction tied to a user.
type FinancialRecord struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User            `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Amount    decimal.Decimal `gorm:"type:numeric(19,4);not null" json:"amount"`
	Currency  string          `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	Type      RecordType     `gorm:"type:varchar(20);not null" json:"type"`
	Category  string         `gorm:"type:varchar(100);not null" json:"category"`
	Date      time.Time      `gorm:"not null" json:"date"`
	Notes     string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate generates a new UUID before inserting a FinancialRecord.
func (f *FinancialRecord) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
