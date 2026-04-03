package services

import (
	"testing"
	"time"

	"finance-dashboard/models"

	"github.com/stretchr/testify/assert"
)

func TestDashboardService_GetSummary(t *testing.T) {
	service := &DashboardService{DB: testDB}

	t.Run("GetSummary with data returns correct totals", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Summary", "summary@example.com", "admin")
		createTestRecord(t, user.ID, 5000.00, models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, 3000.00, models.RecordIncome, "freelance", time.Now())
		createTestRecord(t, user.ID, 1500.00, models.RecordExpense, "rent", time.Now())
		createTestRecord(t, user.ID, 500.00, models.RecordExpense, "food", time.Now())

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, 8000.00, summary["total_income"])
		assert.Equal(t, 2000.00, summary["total_expenses"])
		assert.Equal(t, 6000.00, summary["net_balance"])
		assert.Equal(t, int64(4), summary["total_records"])
	})

	t.Run("GetSummary empty database returns zeros", func(t *testing.T) {
		cleanupTables(testDB)

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, 0.00, summary["total_income"])
		assert.Equal(t, 0.00, summary["total_expenses"])
		assert.Equal(t, 0.00, summary["net_balance"])
		assert.Equal(t, int64(0), summary["total_records"])
	})

	t.Run("GetSummary excludes soft-deleted records", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "SoftDelSum", "softdelsum@example.com", "admin")
		createTestRecord(t, user.ID, 1000.00, models.RecordIncome, "salary", time.Now())
		toDelete := createTestRecord(t, user.ID, 500.00, models.RecordIncome, "bonus", time.Now())
		createTestRecord(t, user.ID, 300.00, models.RecordExpense, "food", time.Now())

		// Soft delete one income record.
		recordService := &RecordService{DB: testDB}
		err := recordService.DeleteRecord(toDelete.ID.String())
		assert.NoError(t, err)

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		assert.Equal(t, 1000.00, summary["total_income"]) // 500 excluded
		assert.Equal(t, 300.00, summary["total_expenses"])
		assert.Equal(t, 700.00, summary["net_balance"])
		assert.Equal(t, int64(2), summary["total_records"]) // 3 - 1 deleted
	})
}

func TestDashboardService_GetTrends(t *testing.T) {
	service := &DashboardService{DB: testDB}

	t.Run("GetTrends with data returns monthly breakdown", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Trends", "trends@example.com", "admin")
		// January records
		createTestRecord(t, user.ID, 5000.00, models.RecordIncome, "salary", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, 1500.00, models.RecordExpense, "rent", time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
		// March records
		createTestRecord(t, user.ID, 6000.00, models.RecordIncome, "salary", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, 2000.00, models.RecordExpense, "rent", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))

		trends, err := service.GetTrends()

		assert.NoError(t, err)
		assert.Len(t, trends, 2) // January and March

		// Trends are ordered ASC by month.
		assert.Equal(t, "2026-01", trends[0]["month"])
		assert.Equal(t, 5000.00, trends[0]["income"])
		assert.Equal(t, 1500.00, trends[0]["expense"])
		assert.Equal(t, 3500.00, trends[0]["net"])

		assert.Equal(t, "2026-03", trends[1]["month"])
		assert.Equal(t, 6000.00, trends[1]["income"])
		assert.Equal(t, 2000.00, trends[1]["expense"])
		assert.Equal(t, 4000.00, trends[1]["net"])
	})

	t.Run("GetTrends empty database returns empty array", func(t *testing.T) {
		cleanupTables(testDB)

		trends, err := service.GetTrends()

		assert.NoError(t, err)
		assert.NotNil(t, trends)
		assert.Len(t, trends, 0)
	})
}

func TestDashboardService_GetCategoryBreakdown(t *testing.T) {
	service := &DashboardService{DB: testDB}

	t.Run("GetCategoryBreakdown with data returns correct per-category totals", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "CatBreak", "catbreak@example.com", "admin")
		createTestRecord(t, user.ID, 5000.00, models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, 2000.00, models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, 1500.00, models.RecordExpense, "rent", time.Now())
		createTestRecord(t, user.ID, 500.00, models.RecordExpense, "food", time.Now())
		createTestRecord(t, user.ID, 300.00, models.RecordExpense, "food", time.Now())

		breakdown, err := service.GetCategoryBreakdown()

		assert.NoError(t, err)
		assert.NotNil(t, breakdown)
		assert.Len(t, breakdown, 3) // salary, rent, food

		// Results ordered by total DESC.
		// salary: 7000 total (all income)
		assert.Equal(t, "salary", breakdown[0]["category"])
		assert.Equal(t, 7000.00, breakdown[0]["total_income"])
		assert.Equal(t, 0.00, breakdown[0]["total_expense"])
		assert.Equal(t, 7000.00, breakdown[0]["total"])
		assert.Equal(t, int64(2), breakdown[0]["count"])

		// rent: 1500 total (all expense)
		assert.Equal(t, "rent", breakdown[1]["category"])
		assert.Equal(t, 0.00, breakdown[1]["total_income"])
		assert.Equal(t, 1500.00, breakdown[1]["total_expense"])
		assert.Equal(t, int64(1), breakdown[1]["count"])

		// food: 800 total (all expense)
		assert.Equal(t, "food", breakdown[2]["category"])
		assert.Equal(t, 800.00, breakdown[2]["total"])
		assert.Equal(t, int64(2), breakdown[2]["count"])
	})

	t.Run("GetCategoryBreakdown empty database returns empty array", func(t *testing.T) {
		cleanupTables(testDB)

		breakdown, err := service.GetCategoryBreakdown()

		assert.NoError(t, err)
		assert.NotNil(t, breakdown)
		assert.Len(t, breakdown, 0)
	})
}
