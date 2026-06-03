package services

import (
	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountService manages user accounts in the double-entry bookkeeping system.
type AccountService struct {
	DB *gorm.DB
}

// CreateDefaultAccounts creates the standard set of accounts for a new user:
// Cash (asset), Revenue (revenue), and Expenses (expense). This is called
// during user registration to bootstrap the ledger system.
func (s *AccountService) CreateDefaultAccounts(tx *gorm.DB, userID uuid.UUID) error {
	defaults := []models.Account{
		{UserID: userID, Name: "Cash", Type: models.AccountAsset, Currency: "INR"},
		{UserID: userID, Name: "Revenue", Type: models.AccountRevenue, Currency: "INR"},
		{UserID: userID, Name: "Expenses", Type: models.AccountExpense, Currency: "INR"},
	}

	for i := range defaults {
		if err := tx.Create(&defaults[i]).Error; err != nil {
			return apperrors.Internal("failed to create default account", err)
		}
	}

	return nil
}

// GetAccountsByUser returns all accounts for a given user.
func (s *AccountService) GetAccountsByUser(userID string) ([]models.Account, error) {
	var accounts []models.Account
	if err := s.DB.Where("user_id = ?", userID).Order("name ASC").Find(&accounts).Error; err != nil {
		return nil, apperrors.Internal("failed to retrieve accounts", err)
	}
	return accounts, nil
}

// GetAccountByID returns a single account by its UUID.
func (s *AccountService) GetAccountByID(id string) (*models.Account, error) {
	var account models.Account
	if err := s.DB.Where("id = ?", id).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.NotFound("account", id)
		}
		return nil, apperrors.Internal("failed to retrieve account", err)
	}
	return &account, nil
}

// CreateAccount creates a custom account for a user.
func (s *AccountService) CreateAccount(account *models.Account) (*models.Account, error) {
	validTypes := map[models.AccountType]struct{}{
		models.AccountAsset:     {},
		models.AccountLiability: {},
		models.AccountEquity:    {},
		models.AccountRevenue:   {},
		models.AccountExpense:   {},
	}

	if _, ok := validTypes[account.Type]; !ok {
		return nil, apperrors.Validation("invalid account type: must be one of asset, liability, equity, revenue, expense")
	}

	if account.Name == "" {
		return nil, apperrors.Validation("account name is required")
	}

	if account.Currency == "" {
		account.Currency = "INR"
	}

	if err := s.DB.Create(account).Error; err != nil {
		return nil, apperrors.Internal("failed to create account", err)
	}

	return account, nil
}
