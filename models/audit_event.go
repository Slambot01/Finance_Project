package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction represents the type of state change captured.
type AuditAction string

const (
	AuditCreate AuditAction = "create"
	AuditUpdate AuditAction = "update"
	AuditDelete AuditAction = "delete"
	AuditLogin  AuditAction = "login"
)

// AuditEvent is an immutable record of a state-changing operation. Events are
// written atomically with the business data using the transactional outbox
// pattern, ensuring the audit trail is always consistent with the actual state.
//
// This satisfies SOC 2, PCI-DSS, and SOX audit trail requirements for
// regulated fintech environments.
type AuditEvent struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EntityType string      `gorm:"type:varchar(50);not null;index" json:"entity_type"` // "user", "record", "account", "ledger_entry"
	EntityID   string      `gorm:"type:varchar(36);not null;index" json:"entity_id"`
	Action     AuditAction `gorm:"type:varchar(20);not null" json:"action"`
	ActorID    string      `gorm:"type:varchar(36);not null;index" json:"actor_id"` // user who performed the action
	Changes    string      `gorm:"type:text" json:"changes,omitempty"`              // JSON diff of before/after state
	RequestID  string      `gorm:"type:varchar(36);index" json:"request_id"`        // correlation with request ID middleware
	IPAddress  string      `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	CreatedAt  time.Time   `gorm:"not null;index" json:"created_at"`
}

// BeforeCreate generates a new UUID before inserting an AuditEvent.
func (a *AuditEvent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
