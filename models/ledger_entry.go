package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// EntryType represents whether a ledger entry is a debit or credit.
type EntryType string

const (
	Debit  EntryType = "debit"
	Credit EntryType = "credit"
)

// LedgerEntry represents a single debit or credit line in a double-entry
// transaction. Every transaction must have balanced debits and credits
// (sum of debits == sum of credits) — this invariant is enforced by the
// LedgerService before any entries are persisted.
type LedgerEntry struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID uuid.UUID       `gorm:"type:uuid;not null;index" json:"transaction_id"`
	AccountID     uuid.UUID       `gorm:"type:uuid;not null;index" json:"account_id"`
	Account       Account         `gorm:"foreignKey:AccountID" json:"-"`
	EntryType     EntryType       `gorm:"type:varchar(10);not null" json:"entry_type"`
	Amount        decimal.Decimal `gorm:"type:numeric(19,4);not null" json:"amount"`
	Currency      string    `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	Description   string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// BeforeCreate generates a new UUID before inserting a LedgerEntry.
func (e *LedgerEntry) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
