package services

import (
	"testing"

	"finance-dashboard/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestLedgerService_PostTransaction(t *testing.T) {
	service := &LedgerService{DB: testDB}
	accountService := &AccountService{DB: testDB}

	t.Run("Balanced transaction posts successfully", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "LedgerUser", "ledger@example.com", "admin")

		// Create accounts.
		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		revenue := &models.Account{UserID: user.ID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"}
		testDB.Create(cash)
		testDB.Create(revenue)

		req := PostTransactionRequest{
			Description: "Salary income",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(50000), Currency: "INR"},
				{AccountID: revenue.ID, EntryType: models.Credit, Amount: decimal.NewFromFloat(50000), Currency: "INR"},
			},
		}

		entries, err := service.PostTransaction(req)

		assert.NoError(t, err)
		assert.Len(t, entries, 2)
		assert.Equal(t, entries[0].TransactionID, entries[1].TransactionID)

		// Verify account balances updated.
		var updatedCash, updatedRevenue models.Account
		testDB.First(&updatedCash, "id = ?", cash.ID)
		testDB.First(&updatedRevenue, "id = ?", revenue.ID)

		assert.True(t, decimal.NewFromFloat(50000).Equal(updatedCash.Balance))   // Asset debited = increased
		assert.True(t, decimal.NewFromFloat(50000).Equal(updatedRevenue.Balance)) // Revenue credited = increased
	})

	t.Run("Unbalanced transaction rejected", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "Unbalanced", "unbalanced@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		revenue := &models.Account{UserID: user.ID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"}
		testDB.Create(cash)
		testDB.Create(revenue)

		req := PostTransactionRequest{
			Description: "Unbalanced entry",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(5000), Currency: "INR"},
				{AccountID: revenue.ID, EntryType: models.Credit, Amount: decimal.NewFromFloat(3000), Currency: "INR"},
			},
		}

		entries, err := service.PostTransaction(req)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "unbalanced")
	})

	t.Run("Transaction with fewer than 2 entries rejected", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "TooFew", "toofew@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		testDB.Create(cash)

		req := PostTransactionRequest{
			Description: "Single entry",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(1000), Currency: "INR"},
			},
		}

		entries, err := service.PostTransaction(req)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "at least 2 entries")
	})

	t.Run("Transaction with zero amount rejected", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "ZeroAmt", "zeroamt@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		revenue := &models.Account{UserID: user.ID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"}
		testDB.Create(cash)
		testDB.Create(revenue)

		req := PostTransactionRequest{
			Description: "Zero amount",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.Zero, Currency: "INR"},
				{AccountID: revenue.ID, EntryType: models.Credit, Amount: decimal.Zero, Currency: "INR"},
			},
		}

		entries, err := service.PostTransaction(req)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "greater than zero")
	})

	t.Run("Transaction with invalid account ID rejected", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "BadAcct", "badacct@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		testDB.Create(cash)

		req := PostTransactionRequest{
			Description: "Bad account",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(1000), Currency: "INR"},
				{AccountID: uuid.New(), EntryType: models.Credit, Amount: decimal.NewFromFloat(1000), Currency: "INR"},
			},
		}

		entries, err := service.PostTransaction(req)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("Multiple transactions accumulate balances correctly", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "Accumulator", "accum@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		expenses := &models.Account{UserID: user.ID, Name: "Expenses", Type: models.AccountExpense, Currency: "INR"}
		revenue := &models.Account{UserID: user.ID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"}
		testDB.Create(cash)
		testDB.Create(expenses)
		testDB.Create(revenue)

		// Income: Debit Cash, Credit Revenue
		_, err := service.PostTransaction(PostTransactionRequest{
			Description: "Salary",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(100000), Currency: "INR"},
				{AccountID: revenue.ID, EntryType: models.Credit, Amount: decimal.NewFromFloat(100000), Currency: "INR"},
			},
		})
		assert.NoError(t, err)

		// Expense: Debit Expenses, Credit Cash
		_, err = service.PostTransaction(PostTransactionRequest{
			Description: "Rent payment",
			Entries: []LedgerEntryRequest{
				{AccountID: expenses.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(25000), Currency: "INR"},
				{AccountID: cash.ID, EntryType: models.Credit, Amount: decimal.NewFromFloat(25000), Currency: "INR"},
			},
		})
		assert.NoError(t, err)

		// Verify final balances.
		var updatedCash, updatedExpenses, updatedRevenue models.Account
		testDB.First(&updatedCash, "id = ?", cash.ID)
		testDB.First(&updatedExpenses, "id = ?", expenses.ID)
		testDB.First(&updatedRevenue, "id = ?", revenue.ID)

		assert.True(t, decimal.NewFromFloat(75000).Equal(updatedCash.Balance))     // 100000 - 25000
		assert.True(t, decimal.NewFromFloat(25000).Equal(updatedExpenses.Balance))  // debited
		assert.True(t, decimal.NewFromFloat(100000).Equal(updatedRevenue.Balance))  // credited
	})

	_ = accountService // suppress unused warning
}

func TestAccountService_CreateDefaultAccounts(t *testing.T) {
	service := &AccountService{DB: testDB}

	t.Run("CreateDefaultAccounts creates 3 accounts", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "DefaultAccts", "defaults@example.com", "viewer")

		err := service.CreateDefaultAccounts(testDB, user.ID)
		assert.NoError(t, err)

		accounts, err := service.GetAccountsByUser(user.ID.String())
		assert.NoError(t, err)
		assert.Len(t, accounts, 3) // Cash, Revenue, Expenses

		// Verify account types.
		types := make(map[string]bool)
		for _, a := range accounts {
			types[string(a.Type)] = true
		}
		assert.True(t, types["asset"])
		assert.True(t, types["revenue"])
		assert.True(t, types["expense"])
	})
}

func TestLedgerService_GetTransactions(t *testing.T) {
	service := &LedgerService{DB: testDB}

	t.Run("GetTransactions returns entries for account", func(t *testing.T) {
		cleanupAllTables(testDB)
		user := createTestUser(t, "TxList", "txlist@example.com", "admin")

		cash := &models.Account{UserID: user.ID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"}
		revenue := &models.Account{UserID: user.ID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"}
		testDB.Create(cash)
		testDB.Create(revenue)

		_, _ = service.PostTransaction(PostTransactionRequest{
			Description: "Test tx",
			Entries: []LedgerEntryRequest{
				{AccountID: cash.ID, EntryType: models.Debit, Amount: decimal.NewFromFloat(1000), Currency: "INR"},
				{AccountID: revenue.ID, EntryType: models.Credit, Amount: decimal.NewFromFloat(1000), Currency: "INR"},
			},
		})

		entries, total, err := service.GetTransactions(cash.ID.String(), 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, entries, 1)
		assert.Equal(t, cash.ID, entries[0].AccountID)
	})
}

// cleanupAllTables truncates all tables including new models.
func cleanupAllTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE outbox_entries CASCADE")
	db.Exec("TRUNCATE TABLE audit_events CASCADE")
	db.Exec("TRUNCATE TABLE ledger_entries CASCADE")
	db.Exec("TRUNCATE TABLE accounts CASCADE")
	db.Exec("TRUNCATE TABLE idempotency_keys CASCADE")
	db.Exec("TRUNCATE TABLE refresh_tokens CASCADE")
	db.Exec("TRUNCATE TABLE financial_records CASCADE")
	db.Exec("TRUNCATE TABLE users CASCADE")
}

// Ensure gorm import used for cleanupAllTables.
var _ = (*gorm.DB)(nil)
