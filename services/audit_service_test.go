package services

import (
	"testing"

	"finance-dashboard/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAuditService_LogEvent(t *testing.T) {
	service := &AuditService{DB: testDB}

	t.Run("LogEvent creates audit event and outbox entry atomically", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "AuditUser", "audit@example.com", "admin")

		event := BuildAuditEvent("user", user.ID.String(), models.AuditCreate, "system", "req-123", "127.0.0.1", map[string]string{"name": "AuditUser"})

		err := testDB.Transaction(func(tx *gorm.DB) error {
			return service.LogEvent(tx, event)
		})

		assert.NoError(t, err)

		// Verify audit event exists.
		var auditEvent models.AuditEvent
		result := testDB.First(&auditEvent, "id = ?", event.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, "user", auditEvent.EntityType)
		assert.Equal(t, models.AuditCreate, auditEvent.Action)
		assert.Equal(t, "req-123", auditEvent.RequestID)

		// Verify outbox entry exists for this event.
		var outbox models.OutboxEntry
		result = testDB.First(&outbox, "event_id = ?", event.ID)
		assert.NoError(t, result.Error)
		assert.Nil(t, outbox.PublishedAt) // Not yet published.
		assert.Contains(t, outbox.EventType, "create")
	})

	t.Run("LogEvent captures changes as JSON", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "ChangeUser", "change@example.com", "viewer")

		changes := map[string]interface{}{
			"before": map[string]string{"role": "viewer"},
			"after":  map[string]string{"role": "analyst"},
		}

		event := BuildAuditEvent("user", user.ID.String(), models.AuditUpdate, "admin-id", "", "", changes)

		err := testDB.Transaction(func(tx *gorm.DB) error {
			return service.LogEvent(tx, event)
		})

		assert.NoError(t, err)

		var auditEvent models.AuditEvent
		testDB.First(&auditEvent, "id = ?", event.ID)
		assert.Contains(t, auditEvent.Changes, "viewer")
		assert.Contains(t, auditEvent.Changes, "analyst")
	})
}

func TestAuditService_QueryEvents(t *testing.T) {
	service := &AuditService{DB: testDB}

	t.Run("QueryEvents filters by entity_type", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "QueryUser", "query@example.com", "admin")

		// Create events for different entity types.
		event1 := BuildAuditEvent("user", user.ID.String(), models.AuditCreate, "system", "", "", nil)
		event2 := BuildAuditEvent("record", "some-record-id", models.AuditCreate, user.ID.String(), "", "", nil)

		testDB.Transaction(func(tx *gorm.DB) error {
			service.LogEvent(tx, event1)
			service.LogEvent(tx, event2)
			return nil
		})

		events, total, err := service.QueryEvents(map[string]string{"entity_type": "user"}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, events, 1)
		assert.Equal(t, "user", events[0].EntityType)
	})

	t.Run("QueryEvents returns paginated results", func(t *testing.T) {
		cleanupTables(testDB)
		user := createTestUser(t, "PaginAudit", "paginaudit@example.com", "admin")

		// Create 15 events.
		for i := 0; i < 15; i++ {
			event := BuildAuditEvent("record", "record-id", models.AuditCreate, user.ID.String(), "", "", nil)
			testDB.Transaction(func(tx *gorm.DB) error {
				return service.LogEvent(tx, event)
			})
		}

		events, total, err := service.QueryEvents(map[string]string{}, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, int64(15), total)
		assert.Len(t, events, 10) // page size = 10
	})
}
