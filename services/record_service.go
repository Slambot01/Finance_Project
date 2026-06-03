package services

import (
	"fmt"
	"strings"
	"time"

	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RecordService encapsulates financial record business logic.
type RecordService struct {
	DB           *gorm.DB
	AuditService *AuditService // optional — if nil, audit logging is skipped
}

// validRecordTypes is the canonical set of allowed transaction types.
var validRecordTypes = map[models.RecordType]struct{}{
	models.RecordIncome:  {},
	models.RecordExpense: {},
}

var validCurrencies = map[string]struct{}{
	"INR": {},
	"USD": {},
	"EUR": {},
	"GBP": {},
}

// CreateRecord validates and persists a new financial record within a
// database transaction to ensure atomicity.
func (s *RecordService) CreateRecord(record *models.FinancialRecord) (*models.FinancialRecord, error) {
	if _, ok := validRecordTypes[record.Type]; !ok {
		return nil, apperrors.Validation("invalid type: must be one of income, expense")
	}

	if record.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.Validation("amount must be greater than zero")
	}

	if record.Currency == "" {
		record.Currency = "INR"
	} else if _, ok := validCurrencies[strings.ToUpper(record.Currency)]; !ok {
		return nil, apperrors.Validation("invalid currency")
	}
	record.Currency = strings.ToUpper(record.Currency)

	if strings.TrimSpace(record.Category) == "" {
		return nil, apperrors.Validation("category is required")
	}

	if record.Date.IsZero() {
		return nil, apperrors.Validation("date is required and must be a valid date")
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return apperrors.Internal("failed to create financial record", err)
		}

		// Emit audit event within the same transaction.
		if s.AuditService != nil {
			event := BuildAuditEvent("record", record.ID.String(), models.AuditCreate, record.UserID.String(), "", "", record)
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}

// GetRecords returns a paginated, filtered list of financial records and the
// total count matching the filters. Soft-deleted records are excluded by GORM.
//
// Supported filter keys: type, category, start_date, end_date, user_id.
func (s *RecordService) GetRecords(filters map[string]string, page, pageSize int) ([]models.FinancialRecord, int64, error) {
	// Sanitise pagination defaults.
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := s.DB.Model(&models.FinancialRecord{})

	// Apply optional filters.
	if typ, ok := filters["type"]; ok && typ != "" {
		query = query.Where("type = ?", typ)
	}
	if category, ok := filters["category"]; ok && category != "" {
		query = query.Where("category = ?", category)
	}
	if startDate, ok := filters["start_date"]; ok && startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("date >= ?", t)
		}
	}
	if endDate, ok := filters["end_date"]; ok && endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// Include the entire end day.
			query = query.Where("date <= ?", t.Add(24*time.Hour-time.Nanosecond))
		}
	}
	if userID, ok := filters["user_id"]; ok && userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Total count for pagination metadata.
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to count financial records", err)
	}

	// Fetch the page.
	var records []models.FinancialRecord
	offset := (page - 1) * pageSize
	if err := query.Order("date DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to retrieve financial records", err)
	}

	return records, total, nil
}

// GetRecordByID looks up a single financial record by UUID string.
func (s *RecordService) GetRecordByID(id string) (*models.FinancialRecord, error) {
	var record models.FinancialRecord
	if err := s.DB.Where("id = ?", id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.NotFound("financial record", id)
		}
		return nil, apperrors.Internal("failed to retrieve financial record", err)
	}
	return &record, nil
}

// UpdateRecord applies a partial update to the record identified by id
// within a database transaction. Validates type and amount if present.
func (s *RecordService) UpdateRecord(id string, updates map[string]interface{}) (*models.FinancialRecord, error) {
	record, err := s.GetRecordByID(id)
	if err != nil {
		return nil, err
	}

	// Validate type if being changed.
	if typeVal, ok := updates["type"]; ok {
		typeStr, valid := typeVal.(string)
		if !valid {
			return nil, apperrors.Validation("type must be a string")
		}
		if _, permitted := validRecordTypes[models.RecordType(typeStr)]; !permitted {
			return nil, apperrors.Validation("invalid type: must be one of income, expense")
		}
	}

	// Validate amount if being changed.
	if amountVal, ok := updates["amount"]; ok {
		var amount decimal.Decimal
		switch v := amountVal.(type) {
		case float64:
			amount = decimal.NewFromFloat(v)
		case string:
			var err error
			amount, err = decimal.NewFromString(v)
			if err != nil {
				return nil, apperrors.Validation("amount must be a valid number string")
			}
		default:
			return nil, apperrors.Validation("amount must be a number or numeric string")
		}

		if amount.LessThanOrEqual(decimal.Zero) {
			return nil, apperrors.Validation("amount must be greater than zero")
		}
		updates["amount"] = amount
	}

	// Validate currency if being changed.
	if currencyVal, ok := updates["currency"]; ok {
		currencyStr, valid := currencyVal.(string)
		if !valid {
			return nil, apperrors.Validation("currency must be a string")
		}
		if _, permitted := validCurrencies[strings.ToUpper(currencyStr)]; !permitted {
			return nil, apperrors.Validation("invalid currency")
		}
		updates["currency"] = strings.ToUpper(currencyStr)
	}

	txErr := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(record).Updates(updates).Error; err != nil {
			return apperrors.Internal("failed to update financial record", err)
		}

		// Emit audit event within the same transaction.
		if s.AuditService != nil {
			event := BuildAuditEvent("record", record.ID.String(), models.AuditUpdate, record.UserID.String(), "", "", updates)
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	// Re-fetch to return fresh data (Updates does not refresh all struct fields).
	if err := s.DB.Where("id = ?", id).First(record).Error; err != nil {
		return nil, apperrors.Internal("failed to retrieve updated record", err)
	}

	return record, nil
}

// DeleteRecord performs a soft delete on the record identified by id
// within a database transaction. GORM automatically sets the DeletedAt timestamp.
func (s *RecordService) DeleteRecord(id string) error {
	var record models.FinancialRecord

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&record).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.NotFound("financial record", id)
			}
			return apperrors.Internal("failed to retrieve financial record", err)
		}

		if err := tx.Delete(&record).Error; err != nil {
			return apperrors.Internal(fmt.Sprintf("failed to delete financial record %s", id), err)
		}

		// Emit audit event within the same transaction.
		if s.AuditService != nil {
			event := BuildAuditEvent("record", record.ID.String(), models.AuditDelete, record.UserID.String(), "", "", nil)
			if err := s.AuditService.LogEvent(tx, event); err != nil {
				return err
			}
		}

		return nil
	})
}
