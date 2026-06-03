package handlers

import (
	"net/http"
	"strconv"

	"finance-dashboard/models"
	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LedgerHandler handles double-entry ledger HTTP requests.
type LedgerHandler struct {
	LedgerService  *services.LedgerService
	AccountService *services.AccountService
}

// PostTransaction handles POST /api/ledger/transactions — posts a balanced
// double-entry transaction. The request must contain at least 2 entries
// where sum(debits) == sum(credits).
func (h *LedgerHandler) PostTransaction(c *gin.Context) {
	var req services.PostTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	entries, err := h.LedgerService.PostTransaction(req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "transaction posted successfully", map[string]interface{}{
		"transaction_id": entries[0].TransactionID,
		"entries":        entries,
	})
}

// GetTransactions handles GET /api/ledger/transactions — returns paginated
// ledger entries, optionally filtered by account.
func (h *LedgerHandler) GetTransactions(c *gin.Context) {
	accountID := c.Query("account_id")
	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	entries, total, err := h.LedgerService.GetTransactions(accountID, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "ledger entries retrieved successfully", map[string]interface{}{
		"entries":   entries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetAccounts handles GET /api/ledger/accounts — returns all accounts for
// the authenticated user (viewers see own, analysts/admins see all).
func (h *LedgerHandler) GetAccounts(c *gin.Context) {
	userID := c.GetString("userID")
	role := c.GetString("userRole")

	var queryUserID string
	if role == string(models.RoleViewer) {
		queryUserID = userID
	} else {
		// Allow filtering by user_id query param for analysts/admins.
		if uid := c.Query("user_id"); uid != "" {
			queryUserID = uid
		} else {
			queryUserID = userID
		}
	}

	accounts, err := h.AccountService.GetAccountsByUser(queryUserID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "accounts retrieved successfully", accounts)
}

// CreateAccount handles POST /api/ledger/accounts — creates a custom account.
func (h *LedgerHandler) CreateAccount(c *gin.Context) {
	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	// Set owner from authenticated context.
	userID := c.GetString("userID")
	parsedUID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid user identity in token")
		return
	}
	account.UserID = parsedUID

	created, err := h.AccountService.CreateAccount(&account)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "account created successfully", created)
}

// GetAccountEntries handles GET /api/ledger/accounts/:id/entries — returns
// ledger entries for a specific account.
func (h *LedgerHandler) GetAccountEntries(c *gin.Context) {
	accountID := c.Param("id")

	// Verify the account exists.
	_, err := h.AccountService.GetAccountByID(accountID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	entries, total, err := h.LedgerService.GetTransactions(accountID, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "account entries retrieved successfully", map[string]interface{}{
		"entries":   entries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
