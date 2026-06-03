package services

import (
	"testing"
	"time"

	"finance-dashboard/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// createTestRecord is a helper that creates a financial record with the given fields.
func createTestRecord(t *testing.T, userID uuid.UUID, amount decimal.Decimal, recordType models.RecordType, category string, date time.Time) *models.FinancialRecord {
	t.Helper()
	service := &RecordService{DB: testDB}
	record := &models.FinancialRecord{
		UserID:   userID,
		Amount:   amount,
		Type:     recordType,
		Category: category,
		Date:     date,
		Notes:    "test record",
	}
	created, err := service.CreateRecord(record)
	assert.NoError(t, err)
	return created
}

func TestRecordService_CreateRecord(t *testing.T) {
	service := &RecordService{DB: testDB}

	t.Run("CreateRecord success", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "RecordOwner", "owner@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.NewFromFloat(1500.50),
			Type:     models.RecordIncome,
			Category: "salary",
			Date:     time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Notes:    "April salary",
		}
		created, err := service.CreateRecord(record)

		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, user.ID, created.UserID)
		assert.True(t, decimal.NewFromFloat(1500.50).Equal(created.Amount))
		assert.Equal(t, models.RecordIncome, created.Type)
		assert.Equal(t, "salary", created.Category)
	})

	t.Run("CreateRecord invalid type", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "BadType", "badtype@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.NewFromFloat(100.00),
			Type:     "refund",
			Category: "misc",
			Date:     time.Now(),
		}
		created, err := service.CreateRecord(record)

		assert.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("CreateRecord negative amount", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "NegAmount", "negamount@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.NewFromFloat(-50.00),
			Type:     models.RecordExpense,
			Category: "food",
			Date:     time.Now(),
		}
		created, err := service.CreateRecord(record)

		assert.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "amount must be greater than zero")
	})

	t.Run("CreateRecord zero amount", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "ZeroAmount", "zeroamount@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.Zero,
			Type:     models.RecordExpense,
			Category: "food",
			Date:     time.Now(),
		}
		created, err := service.CreateRecord(record)

		assert.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "amount must be greater than zero")
	})

	t.Run("CreateRecord empty category", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "NoCat", "nocat@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.NewFromFloat(100.00),
			Type:     models.RecordIncome,
			Category: "",
			Date:     time.Now(),
		}
		created, err := service.CreateRecord(record)

		assert.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "category is required")
	})

	t.Run("CreateRecord zero date", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "NoDate", "nodate@example.com", "admin")

		record := &models.FinancialRecord{
			UserID:   user.ID,
			Amount:   decimal.NewFromFloat(100.00),
			Type:     models.RecordIncome,
			Category: "salary",
			Date:     time.Time{}, // zero value
		}
		created, err := service.CreateRecord(record)

		assert.Error(t, err)
		assert.Nil(t, created)
		assert.Contains(t, err.Error(), "date is required")
	})
}

func TestRecordService_GetRecords(t *testing.T) {
	service := &RecordService{DB: testDB}

	t.Run("GetRecords no filters returns all records paginated", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Paginator", "paginator@example.com", "admin")
		for i := 0; i < 15; i++ {
			createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "salary", time.Now())
		}

		records, total, err := service.GetRecords(map[string]string{}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(15), total)
		assert.Len(t, records, 10) // page size = 10
	})

	t.Run("GetRecords filter by type", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "TypeFilter", "typefilter@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(200.00), models.RecordExpense, "food", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(300.00), models.RecordIncome, "bonus", time.Now())

		records, total, err := service.GetRecords(map[string]string{"type": "income"}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, records, 2)
		for _, r := range records {
			assert.Equal(t, models.RecordIncome, r.Type)
		}
	})

	t.Run("GetRecords filter by category", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "CatFilter", "catfilter@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(200.00), models.RecordExpense, "food", time.Now())
		createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordExpense, "food", time.Now())

		records, total, err := service.GetRecords(map[string]string{"category": "food"}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, records, 2)
		for _, r := range records {
			assert.Equal(t, "food", r.Category)
		}
	})

	t.Run("GetRecords filter by date range", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "DateFilter", "datefilter@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "a", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, decimal.NewFromFloat(200.00), models.RecordIncome, "b", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
		createTestRecord(t, user.ID, decimal.NewFromFloat(300.00), models.RecordIncome, "c", time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC))

		filters := map[string]string{
			"start_date": "2026-02-01",
			"end_date":   "2026-04-30",
		}
		records, total, err := service.GetRecords(filters, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, records, 1)
		assert.True(t, decimal.NewFromFloat(200.00).Equal(records[0].Amount))
	})

	t.Run("GetRecords filter by user_id", func(t *testing.T) {
		cleanupTables(testDB)
		user1 := createTestUser(t, "User1", "user1@example.com", "admin")
		user2 := createTestUser(t, "User2", "user2@example.com", "viewer")
		createTestRecord(t, user1.ID, decimal.NewFromFloat(500.00), models.RecordIncome, "salary", time.Now())
		createTestRecord(t, user1.ID, decimal.NewFromFloat(300.00), models.RecordExpense, "rent", time.Now())
		createTestRecord(t, user2.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "freelance", time.Now())

		records, total, err := service.GetRecords(map[string]string{"user_id": user2.ID.String()}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, records, 1)
		assert.Equal(t, user2.ID, records[0].UserID)
	})

	t.Run("GetRecords pagination returns correct page", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "PaginTest", "pagintest@example.com", "admin")
		for i := 0; i < 25; i++ {
			createTestRecord(t, user.ID, decimal.NewFromFloat(float64(i+1)*10), models.RecordIncome, "test", time.Now())
		}

		records, total, err := service.GetRecords(map[string]string{}, 2, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(25), total)
		assert.Len(t, records, 10) // page 2 of 25 records with page_size 10
	})

	t.Run("GetRecords page 0 or negative defaults to page 1", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "PageZero", "pagezero@example.com", "admin")
		createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "test", time.Now())

		records, total, err := service.GetRecords(map[string]string{}, 0, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, records, 1)

		// Negative page also defaults to 1.
		records2, total2, err2 := service.GetRecords(map[string]string{}, -5, 10)

		assert.NoError(t, err2)
		assert.Equal(t, int64(1), total2)
		assert.Len(t, records2, 1)
	})
}

func TestRecordService_GetRecordByID(t *testing.T) {
	service := &RecordService{DB: testDB}

	t.Run("GetRecordByID success", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "GetByID", "getbyid@example.com", "admin")
		created := createTestRecord(t, user.ID, decimal.NewFromFloat(750.00), models.RecordExpense, "rent", time.Now())

		record, err := service.GetRecordByID(created.ID.String())

		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, created.ID, record.ID)
		assert.True(t, decimal.NewFromFloat(750.00).Equal(record.Amount))
	})

	t.Run("GetRecordByID not found", func(t *testing.T) {
		cleanupTables(testDB)

		record, err := service.GetRecordByID("00000000-0000-0000-0000-000000000000")

		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRecordService_UpdateRecord(t *testing.T) {
	service := &RecordService{DB: testDB}

	t.Run("UpdateRecord success", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Updater", "updater@example.com", "admin")
		created := createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "salary", time.Now())

		updated, err := service.UpdateRecord(created.ID.String(), map[string]interface{}{
			"amount":   200.00, // JSON decoding would give a float64
			"category": "bonus",
		})

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.True(t, decimal.NewFromFloat(200.00).Equal(updated.Amount))
		assert.Equal(t, "bonus", updated.Category)
	})

	t.Run("UpdateRecord invalid type", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "BadUpdate", "badupdate@example.com", "admin")
		created := createTestRecord(t, user.ID, decimal.NewFromFloat(100.00), models.RecordIncome, "salary", time.Now())

		updated, err := service.UpdateRecord(created.ID.String(), map[string]interface{}{
			"type": "refund",
		})

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "invalid type")
	})

	t.Run("UpdateRecord not found", func(t *testing.T) {
		cleanupTables(testDB)

		updated, err := service.UpdateRecord("00000000-0000-0000-0000-000000000000", map[string]interface{}{
			"amount": 999.00,
		})

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRecordService_DeleteRecord(t *testing.T) {
	service := &RecordService{DB: testDB}

	t.Run("DeleteRecord success with soft delete", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "SoftDel", "softdel@example.com", "admin")
		created := createTestRecord(t, user.ID, decimal.NewFromFloat(500.00), models.RecordExpense, "rent", time.Now())

		err := service.DeleteRecord(created.ID.String())
		assert.NoError(t, err)

		// Normal query should NOT find it (soft deleted).
		record, err := service.GetRecordByID(created.ID.String())
		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteRecord soft delete verify — record still exists in DB with Unscoped", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "Verifier", "verifier@example.com", "admin")
		created := createTestRecord(t, user.ID, decimal.NewFromFloat(250.00), models.RecordIncome, "freelance", time.Now())

		err := service.DeleteRecord(created.ID.String())
		assert.NoError(t, err)

		// Unscoped query SHOULD find it — proves soft delete, not hard delete.
		var record models.FinancialRecord
		result := testDB.Unscoped().Where("id = ?", created.ID).First(&record)
		assert.NoError(t, result.Error)
		assert.Equal(t, created.ID, record.ID)
		assert.True(t, record.DeletedAt.Valid) // DeletedAt should be set.

		// But normal GetRecords should exclude it.
		records, total, err := service.GetRecords(map[string]string{}, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, records, 0)
	})

	t.Run("DeleteRecord not found", func(t *testing.T) {
		cleanupTables(testDB)

		err := service.DeleteRecord("00000000-0000-0000-0000-000000000000")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
