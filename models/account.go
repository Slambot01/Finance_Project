package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AccountType represents the classification of an account in double-entry bookkeeping.
type AccountType string

const (
	AccountAsset     AccountType = "asset"
	AccountLiability AccountType = "liability"
	AccountEquity    AccountType = "equity"
	AccountRevenue   AccountType = "revenue"
	AccountExpense   AccountType = "expense"
)

// Account represents a ledger account in the double-entry bookkeeping system.
// Each user gets default accounts (Cash, Revenue, Expenses) on registration.
// Balances are updated atomically within database transactions to prevent
// inconsistencies from concurrent operations.
type Account struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User            `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name      string          `gorm:"type:varchar(255);not null" json:"name"`
	Type      AccountType     `gorm:"type:varchar(20);not null" json:"type"`
	Currency  string          `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	Balance   decimal.Decimal `gorm:"type:numeric(19,4);not null;default:0" json:"balance"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// BeforeCreate generates a new UUID before inserting an Account record.
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
