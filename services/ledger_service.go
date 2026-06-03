package services

import (
	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LedgerService implements the double-entry bookkeeping engine. Every
// transaction must have balanced debit and credit entries. Account balances
// are updated atomically using row-level locks (SELECT FOR UPDATE) to prevent
// inconsistencies from concurrent operations.
type LedgerService struct {
	DB           *gorm.DB
	AuditService *AuditService // optional — if nil, audit logging is skipped
}

// PostTransactionRequest represents a request to post a double-entry transaction.
type PostTransactionRequest struct {
	Description string                `json:"description"`
	Entries     []LedgerEntryRequest  `json:"entries"`
}

// LedgerEntryRequest represents a single debit or credit line in a transaction request.
type LedgerEntryRequest struct {
	AccountID uuid.UUID       `json:"account_id"`
	EntryType models.EntryType `json:"entry_type"` // "debit" or "credit"
	Amount    decimal.Decimal  `json:"amount"`
	Currency  string           `json:"currency"`
}

// PostTransaction validates that debits equal credits, acquires row-level locks
// on affected accounts, inserts ledger entries, and updates account balances —
// all within a single database transaction.
//
// This is the core invariant of double-entry bookkeeping: for every transaction,
// sum(debits) must exactly equal sum(credits).
func (s *LedgerService) PostTransaction(req PostTransactionRequest) ([]models.LedgerEntry, error) {
	if len(req.Entries) < 2 {
		return nil, apperrors.Validation("a transaction must have at least 2 entries (one debit, one credit)")
	}

	// Validate the fundamental accounting invariant: sum(debits) == sum(credits).
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	
	for _, e := range req.Entries {
		if e.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, apperrors.Validation("all entry amounts must be greater than zero")
		}
		if e.EntryType != models.Debit && e.EntryType != models.Credit {
			return nil, apperrors.Validation("entry_type must be 'debit' or 'credit'")
		}
		if e.EntryType == models.Debit {
			totalDebits = totalDebits.Add(e.Amount)
		} else {
			totalCredits = totalCredits.Add(e.Amount)
		}
	}

	if !totalDebits.Equal(totalCredits) {
		return nil, apperrors.Validation("transaction is unbalanced: sum of debits must equal sum of credits")
	}

	transactionID := uuid.New()
	var createdEntries []models.LedgerEntry

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Collect unique account IDs for row-level locking.
		accountIDs := make([]uuid.UUID, 0, len(req.Entries))
		seen := make(map[uuid.UUID]bool)
		for _, e := range req.Entries {
			if !seen[e.AccountID] {
				accountIDs = append(accountIDs, e.AccountID)
				seen[e.AccountID] = true
			}
		}

		// Acquire row-level locks on all affected accounts (SELECT FOR UPDATE).
		// This prevents concurrent transactions from corrupting balances.
		var accounts []models.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", accountIDs).
			Order("id ASC"). // Consistent lock ordering to prevent deadlocks
			Find(&accounts).Error; err != nil {
			return apperrors.Internal("failed to lock accounts", err)
		}

		if len(accounts) != len(accountIDs) {
			return apperrors.Validation("one or more account IDs are invalid")
		}

		// Build a lookup map for account balances.
		accountMap := make(map[uuid.UUID]*models.Account, len(accounts))
		for i := range accounts {
			accountMap[accounts[i].ID] = &accounts[i]
		}

		// Insert ledger entries and compute balance updates.
		for _, e := range req.Entries {
			currency := e.Currency
			if currency == "" {
				currency = "INR"
			}

			entry := models.LedgerEntry{
				TransactionID: transactionID,
				AccountID:     e.AccountID,
				EntryType:     e.EntryType,
				Amount:        e.Amount,
				Currency:      currency,
				Description:   req.Description,
			}

			if err := tx.Create(&entry).Error; err != nil {
				return apperrors.Internal("failed to create ledger entry", err)
			}
			createdEntries = append(createdEntries, entry)

			// Update account balance.
			// Debits increase asset/expense accounts, decrease liability/revenue/equity.
			// Credits decrease asset/expense accounts, increase liability/revenue/equity.
			account := accountMap[e.AccountID]
			switch account.Type {
			case models.AccountAsset, models.AccountExpense:
				if e.EntryType == models.Debit {
					account.Balance = account.Balance.Add(e.Amount)
				} else {
					account.Balance = account.Balance.Sub(e.Amount)
				}
			case models.AccountLiability, models.AccountRevenue, models.AccountEquity:
				if e.EntryType == models.Credit {
					account.Balance = account.Balance.Add(e.Amount)
				} else {
					account.Balance = account.Balance.Sub(e.Amount)
				}
			}
		}

		// Persist updated account balances.
		for _, account := range accountMap {
			if err := tx.Model(account).Update("balance", account.Balance).Error; err != nil {
				return apperrors.Internal("failed to update account balance", err)
			}
		}

		// Emit audit event inside the same transaction for atomicity.
		if s.AuditService != nil {
			event := BuildAuditEvent("ledger_transaction", transactionID.String(), models.AuditCreate, "", "", "", map[string]interface{}{
				"description":  req.Description,
				"entry_count":  len(req.Entries),
			})
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdEntries, nil
}

// GetTransactions returns ledger entries grouped by transaction ID, optionally
// filtered by account ID.
func (s *LedgerService) GetTransactions(accountID string, page, pageSize int) ([]models.LedgerEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := s.DB.Model(&models.LedgerEntry{})

	if accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to count ledger entries", err)
	}

	var entries []models.LedgerEntry
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&entries).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to retrieve ledger entries", err)
	}

	return entries, total, nil
}
