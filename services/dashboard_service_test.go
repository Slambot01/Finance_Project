package services

import (
	"testing"
	"time"

	"finance-dashboard/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDashboardService_GetSummary(t *testing.T) {
	service := &DashboardService{DB: testDB}

	t.Run("GetSummary with data returns correct totals", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Summary", "summary@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(5000.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(3000.00), models.RecordIncome, "freelance", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(1500.00), models.RecordExpense, "rent", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordExpense, "food", time.Now())

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Contains(t, summary, "INR")
		
		inr := summary["INR"]
		assert.True(t, decimal.NewFromFloat(8000.00).Equal(inr["total_income"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(2000.00).Equal(inr["total_expenses"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(6000.00).Equal(inr["net_balance"].(decimal.Decimal)))
		assert.Equal(t, int64(4), inr["total_records"])
	})

	t.Run("GetSummary empty database returns zeros", func(t *testing.T) {
		cleanupTables(testDB)

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Len(t, summary, 0)
	})

	t.Run("GetSummary excludes soft-deleted records", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "SoftDelSum", "softdelsum@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(1000.00), models.RecordIncome, "salary", time.Now())
		toDelete := createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordIncome, "bonus", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(300.00), models.RecordExpense, "food", time.Now())

		// Soft delete one income record.
		recordService := &RecordService{DB: testDB}
		err := recordService.DeleteRecord(toDelete.ID.String())
		assert.NoError(t, err)

		summary, err := service.GetSummary()

		assert.NoError(t, err)
		inr := summary["INR"]
		assert.True(t, decimal.NewFromFloat(1000.00).Equal(inr["total_income"].(decimal.Decimal))) // 500 excluded
		assert.True(t, decimal.NewFromFloat(300.00).Equal(inr["total_expenses"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(700.00).Equal(inr["net_balance"].(decimal.Decimal)))
		assert.Equal(t, int64(2), inr["total_records"]) // 3 - 1 deleted
	})
}

func TestDashboardService_GetTrends(t *testing.T) {
	service := &DashboardService{DB: testDB}

	t.Run("GetTrends with data returns monthly breakdown", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Trends", "trends@example.com", "admin")
		// January records
		createTestRecord(t, user.ID, decimal.NewFromFloat(5000.00), models.RecordIncome, "salary", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, decimal.NewFromFloat(1500.00), models.RecordExpense, "rent", time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
		// March records
		createTestRecord(t, user.ID, decimal.NewFromFloat(6000.00), models.RecordIncome, "salary", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, decimal.NewFromFloat(2000.00), models.RecordExpense, "rent", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))

		trends, err := service.GetTrends()

		assert.NoError(t, err)
		assert.Len(t, trends, 2) // January and March

		// Trends are ordered ASC by month.
		assert.Equal(t, "INR", trends[0]["currency"])
		assert.Equal(t, "2026-01", trends[0]["month"])
		assert.True(t, decimal.NewFromFloat(5000.00).Equal(trends[0]["income"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(1500.00).Equal(trends[0]["expense"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(3500.00).Equal(trends[0]["net"].(decimal.Decimal)))

		assert.Equal(t, "INR", trends[1]["currency"])
		assert.Equal(t, "2026-03", trends[1]["month"])
		assert.True(t, decimal.NewFromFloat(6000.00).Equal(trends[1]["income"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(2000.00).Equal(trends[1]["expense"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(4000.00).Equal(trends[1]["net"].(decimal.Decimal)))
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
		createTestRecord(t, user.ID, decimal.NewFromFloat(5000.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(2000.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(1500.00), models.RecordExpense, "rent", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordExpense, "food", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(300.00), models.RecordExpense, "food", time.Now())

		breakdown, err := service.GetCategoryBreakdown()

		assert.NoError(t, err)
		assert.NotNil(t, breakdown)
		assert.Len(t, breakdown, 3) // salary, rent, food

		// Results ordered by total DESC.
		// salary: 7000 total (all income)
		assert.Equal(t, "INR", breakdown[0]["currency"])
		assert.Equal(t, "salary", breakdown[0]["category"])
		assert.True(t, decimal.NewFromFloat(7000.00).Equal(breakdown[0]["total_income"].(decimal.Decimal)))
		assert.True(t, decimal.Zero.Equal(breakdown[0]["total_expense"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(7000.00).Equal(breakdown[0]["total"].(decimal.Decimal)))
		assert.Equal(t, int64(2), breakdown[0]["count"])

		// rent: 1500 total (all expense)
		assert.Equal(t, "INR", breakdown[1]["currency"])
		assert.Equal(t, "rent", breakdown[1]["category"])
		assert.True(t, decimal.Zero.Equal(breakdown[1]["total_income"].(decimal.Decimal)))
		assert.True(t, decimal.NewFromFloat(1500.00).Equal(breakdown[1]["total_expense"].(decimal.Decimal)))
		assert.Equal(t, int64(1), breakdown[1]["count"])

		// food: 800 total (all expense)
		assert.Equal(t, "INR", breakdown[2]["currency"])
		assert.Equal(t, "food", breakdown[2]["category"])
		assert.True(t, decimal.NewFromFloat(800.00).Equal(breakdown[2]["total"].(decimal.Decimal)))
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
