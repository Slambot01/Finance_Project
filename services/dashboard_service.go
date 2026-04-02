package services

import (
	"errors"

	"finance-dashboard/models"

	"gorm.io/gorm"
)

// DashboardService encapsulates analytics and reporting business logic.
type DashboardService struct {
	DB *gorm.DB
}

// GetSummary returns aggregate totals across all non-deleted financial records:
// total_income, total_expenses, net_balance, and total_records.
func (s *DashboardService) GetSummary() (map[string]interface{}, error) {
	type summaryResult struct {
		TotalIncome   float64
		TotalExpenses float64
		TotalRecords  int64
	}

	var result summaryResult

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expenses",
			"COUNT(*) AS total_records",
		).
		Scan(&result).Error

	if err != nil {
		return nil, errors.New("failed to retrieve dashboard summary")
	}

	return map[string]interface{}{
		"total_income":   result.TotalIncome,
		"total_expenses": result.TotalExpenses,
		"net_balance":    result.TotalIncome - result.TotalExpenses,
		"total_records":  result.TotalRecords,
	}, nil
}

// GetTrends returns a monthly income vs expense breakdown ordered
// chronologically. Each entry contains month, income, expense, and net.
func (s *DashboardService) GetTrends() ([]map[string]interface{}, error) {
	type trendRow struct {
		Month   string
		Income  float64
		Expense float64
	}

	var rows []trendRow

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"TO_CHAR(date, 'YYYY-MM') AS month",
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS expense",
		).
		Group("TO_CHAR(date, 'YYYY-MM')").
		Order("month ASC").
		Scan(&rows).Error

	if err != nil {
		return nil, errors.New("failed to retrieve trend data")
	}

	trends := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		trends = append(trends, map[string]interface{}{
			"month":   r.Month,
			"income":  r.Income,
			"expense": r.Expense,
			"net":     r.Income - r.Expense,
		})
	}

	return trends, nil
}

// GetCategoryBreakdown returns per-category totals for income, expense,
// overall total, and transaction count, ordered by total descending.
func (s *DashboardService) GetCategoryBreakdown() ([]map[string]interface{}, error) {
	type categoryRow struct {
		Category     string
		TotalIncome  float64
		TotalExpense float64
		Total        float64
		Count        int64
	}

	var rows []categoryRow

	err := s.DB.Model(&models.FinancialRecord{}).
		Select(
			"category",
			"COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income",
			"COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense",
			"COALESCE(SUM(amount), 0) AS total",
			"COUNT(*) AS count",
		).
		Group("category").
		Order("total DESC").
		Scan(&rows).Error

	if err != nil {
		return nil, errors.New("failed to retrieve category breakdown")
	}

	breakdown := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		breakdown = append(breakdown, map[string]interface{}{
			"category":      r.Category,
			"total_income":  r.TotalIncome,
			"total_expense": r.TotalExpense,
			"total":         r.Total,
			"count":         r.Count,
		})
	}

	return breakdown, nil
}
