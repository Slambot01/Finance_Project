package services

import (
	"encoding/json"
	"log/slog"

	apperrors "finance-dashboard/errors"
	"finance-dashboard/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditService writes immutable audit events and their corresponding outbox
// entries within existing database transactions. This ensures the audit trail
// is always consistent with the business data (no events without data changes,
// no data changes without events).
type AuditService struct {
	DB *gorm.DB
}

// LogEvent writes an audit event and its outbox entry within the given
// transaction. This method should be called from other services within their
// existing DB transactions to ensure atomicity.
func (s *AuditService) LogEvent(tx *gorm.DB, event models.AuditEvent) error {
	if err := tx.Create(&event).Error; err != nil {
		return apperrors.Internal("failed to create audit event", err)
	}

	// Build the outbox payload.
	payload, err := json.Marshal(event)
	if err != nil {
		return apperrors.Internal("failed to marshal audit event for outbox", err)
	}

	outboxEntry := models.OutboxEntry{
		EventID:   event.ID,
		EventType: string(event.Action) + "." + event.EntityType,
		Payload:   string(payload),
	}

	if err := tx.Create(&outboxEntry).Error; err != nil {
		return apperrors.Internal("failed to create outbox entry", err)
	}

	return nil
}

// QueryEvents returns audit events matching the given filters, with pagination.
func (s *AuditService) QueryEvents(filters map[string]string, page, pageSize int) ([]models.AuditEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.DB.Model(&models.AuditEvent{})

	if entityType, ok := filters["entity_type"]; ok && entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if entityID, ok := filters["entity_id"]; ok && entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}
	if actorID, ok := filters["actor_id"]; ok && actorID != "" {
		query = query.Where("actor_id = ?", actorID)
	}
	if action, ok := filters["action"]; ok && action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to count audit events", err)
	}

	var events []models.AuditEvent
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, apperrors.Internal("failed to retrieve audit events", err)
	}

	return events, total, nil
}

// BuildAuditEvent is a convenience constructor for creating audit events.
func BuildAuditEvent(entityType, entityID string, action models.AuditAction, actorID, requestID, ipAddress string, changes interface{}) models.AuditEvent {
	var changesJSON string
	if changes != nil {
		if data, err := json.Marshal(changes); err == nil {
			changesJSON = string(data)
		}
	}

	event := models.AuditEvent{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		Changes:    changesJSON,
		RequestID:  requestID,
		IPAddress:  ipAddress,
	}

	slog.Info("audit event created",
		slog.String("entity_type", entityType),
		slog.String("entity_id", entityID),
		slog.String("action", string(action)),
		slog.String("actor_id", actorID),
	)

	return event
}
