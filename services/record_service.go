package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"finance-dashboard/models"

	"gorm.io/gorm"
)

// RecordService encapsulates financial record business logic.
type RecordService struct {
	DB *gorm.DB
}

// validRecordTypes is the canonical set of allowed transaction types.
var validRecordTypes = map[models.RecordType]struct{}{
	models.RecordIncome:  {},
	models.RecordExpense: {},
}

// CreateRecord validates and persists a new financial record.
func (s *RecordService) CreateRecord(record *models.FinancialRecord) (*models.FinancialRecord, error) {
	if _, ok := validRecordTypes[record.Type]; !ok {
		return nil, errors.New("invalid type: must be one of income, expense")
	}

	if record.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if strings.TrimSpace(record.Category) == "" {
		return nil, errors.New("category is required")
	}

	if record.Date.IsZero() {
		return nil, errors.New("date is required and must be a valid date")
	}

	if err := s.DB.Create(record).Error; err != nil {
		return nil, errors.New("failed to create financial record")
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
		return nil, 0, errors.New("failed to count financial records")
	}

	// Fetch the page.
	var records []models.FinancialRecord
	offset := (page - 1) * pageSize
	if err := query.Order("date DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, errors.New("failed to retrieve financial records")
	}

	return records, total, nil
}

// GetRecordByID looks up a single financial record by UUID string.
func (s *RecordService) GetRecordByID(id string) (*models.FinancialRecord, error) {
	var record models.FinancialRecord
	if err := s.DB.Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("financial record with id %s not found", id)
		}
		return nil, errors.New("failed to retrieve financial record")
	}
	return &record, nil
}

// UpdateRecord applies a partial update to the record identified by id.
// Validates type and amount if present in the updates map.
func (s *RecordService) UpdateRecord(id string, updates map[string]interface{}) (*models.FinancialRecord, error) {
	record, err := s.GetRecordByID(id)
	if err != nil {
		return nil, err
	}

	// Validate type if being changed.
	if typeVal, ok := updates["type"]; ok {
		typeStr, valid := typeVal.(string)
		if !valid {
			return nil, errors.New("type must be a string")
		}
		if _, permitted := validRecordTypes[models.RecordType(typeStr)]; !permitted {
			return nil, errors.New("invalid type: must be one of income, expense")
		}
	}

	// Validate amount if being changed.
	if amountVal, ok := updates["amount"]; ok {
		// JSON numbers are decoded as float64 by default.
		amount, valid := amountVal.(float64)
		if !valid {
			return nil, errors.New("amount must be a number")
		}
		if amount <= 0 {
			return nil, errors.New("amount must be greater than zero")
		}
	}

	if err := s.DB.Model(record).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update financial record")
	}

	return record, nil
}

// DeleteRecord performs a soft delete on the record identified by id.
// GORM automatically sets the DeletedAt timestamp.
func (s *RecordService) DeleteRecord(id string) error {
	var record models.FinancialRecord
	if err := s.DB.Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("financial record with id %s not found", id)
		}
		return errors.New("failed to retrieve financial record")
	}

	if err := s.DB.Delete(&record).Error; err != nil {
		return errors.New("failed to delete financial record")
	}

	return nil
}
