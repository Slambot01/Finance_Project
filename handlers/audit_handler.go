package handlers

import (
	"net/http"
	"strconv"

	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// AuditHandler handles audit log HTTP requests. Admin-only access.
type AuditHandler struct {
	Service *services.AuditService
}

// GetAuditLog handles GET /api/audit — returns filtered, paginated audit events.
// Supports filters: entity_type, entity_id, actor_id, action.
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	filters := make(map[string]string)

	if entityType := c.Query("entity_type"); entityType != "" {
		filters["entity_type"] = entityType
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		filters["entity_id"] = entityID
	}
	if actorID := c.Query("actor_id"); actorID != "" {
		filters["actor_id"] = actorID
	}
	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}

	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	events, total, err := h.Service.QueryEvents(filters, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "audit log retrieved successfully", map[string]interface{}{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
