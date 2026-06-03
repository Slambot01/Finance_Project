package services

import (
	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// DashboardService encapsulates analytics and reporting business logic.
type DashboardService struct {
	DB *gorm.DB
}

// GetSummary returns aggregate totals across all non-deleted financial records grouped by currency:
func (s *DashboardService) GetSummary() (map[string]map[string]interface{}, error) {
	type summaryResult struct {
		Currency      string
		TotalIncome   decimal.Decimal
		TotalExpenses decimal.Decimal
		TotalRecords  int64
	}

	var results []summaryResult

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"currency",
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expenses",
			"COUNT(*) AS total_records",
		).
		Group("currency").
		Scan(&results).Error

	if err != nil {
		return nil, apperrors.Internal("failed to retrieve dashboard summary", err)
	}

	summary := make(map[string]map[string]interface{})
	for _, r := range results {
		summary[r.Currency] = map[string]interface{}{
			"total_income":   r.TotalIncome,
			"total_expenses": r.TotalExpenses,
			"net_balance":    r.TotalIncome.Sub(r.TotalExpenses),
			"total_records":  r.TotalRecords,
		}
	}
	return summary, nil
}

// GetTrends returns a monthly income vs expense breakdown ordered
// chronologically, grouped by currency.
func (s *DashboardService) GetTrends() ([]map[string]interface{}, error) {
	type trendRow struct {
		Currency string
		Month    string
		Income   decimal.Decimal
		Expense  decimal.Decimal
	}

	var rows []trendRow

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"currency",
			"TO_CHAR(date, 'YYYY-MM') AS month",
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense",
		).
		Group("currency, TO_CHAR(date, 'YYYY-MM')").
		Order("month ASC").
		Scan(&rows).Error

	if err != nil {
		return nil, apperrors.Internal("failed to retrieve trend data", err)
	}

	trends := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		trends = append(trends, map[string]interface{}{
			"currency": r.Currency,
			"month":    r.Month,
			"income":   r.Income,
			"expense":  r.Expense,
			"net":      r.Income.Sub(r.Expense),
		})
	}

	return trends, nil
}

// GetCategoryBreakdown returns per-category totals for income, expense,
// overall total, and transaction count, grouped by currency and ordered by total descending.
func (s *DashboardService) GetCategoryBreakdown() ([]map[string]interface{}, error) {
	type categoryRow struct {
		Currency     string
		Category     string
		TotalIncome  decimal.Decimal
		TotalExpense decimal.Decimal
		Total        decimal.Decimal
		Count        int64
	}

	var rows []categoryRow

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"currency",
			"category",
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense",
			"COALESCE(SUM(amount), 0) AS total",
			"COUNT(*) AS count",
		).
		Group("currency, category").
		Order("total DESC").
		Scan(&rows).Error

	if err != nil {
		return nil, apperrors.Internal("failed to retrieve category breakdown", err)
	}

	breakdown := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		breakdown = append(breakdown, map[string]interface{}{
			"currency":      r.Currency,
			"category":      r.Category,
			"total_income":  r.TotalIncome,
			"total_expense": r.TotalExpense,
			"total":         r.Total,
			"count":         r.Count,
		})
	}

	return breakdown, nil
}
