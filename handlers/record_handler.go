package handlers

import (
	"net/http"
	"strconv"
	"time"

	"finance-dashboard/models"
	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecordHandler handles financial record HTTP requests.
type RecordHandler struct {
	Service *services.RecordService
}

// CreateRecord handles POST /records — creates a new financial record.
func (h *RecordHandler) CreateRecord(c *gin.Context) {
	var record models.FinancialRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	// Prevent mass-assignment of server-controlled fields.
	record.ID = uuid.Nil
	record.CreatedAt = time.Time{}
	record.UpdatedAt = time.Time{}
	record.DeletedAt = gorm.DeletedAt{}

	// Set the owning user from the authenticated context.
	userID := c.GetString("userID")
	parsedUID, err := uuid.Parse(userID)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "invalid user identity in token")
		return
	}
	record.UserID = parsedUID

	created, err := h.Service.CreateRecord(&record)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusCreated, "financial record created successfully", created)
}

// GetRecords handles GET /records — returns paginated, filtered records.
// Viewers are scoped to their own records; analysts and admins see everything.
func (h *RecordHandler) GetRecords(c *gin.Context) {
	// Build filters from query params.
	filters := make(map[string]string)

	if typ := c.Query("type"); typ != "" {
		filters["type"] = typ
	}
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	// Viewers can only see their own records.
	role := c.GetString("userRole")
	if role == string(models.RoleViewer) {
		filters["user_id"] = c.GetString("userID")
	}

	// Parse pagination params.
	page := 1
	pageSize := 10

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

	records, total, err := h.Service.GetRecords(filters, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "financial records retrieved successfully", map[string]interface{}{
		"records":   records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRecordByID handles GET /records/:id — returns a single record.
func (h *RecordHandler) GetRecordByID(c *gin.Context) {
	id := c.Param("id")

	record, err := h.Service.GetRecordByID(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "financial record retrieved successfully", record)
}

// UpdateRecord handles PUT /records/:id — applies partial updates to a record.
func (h *RecordHandler) UpdateRecord(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.ValidationError(c, "invalid request body", err.Error())
		return
	}

	record, err := h.Service.UpdateRecord(id, updates)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "financial record updated successfully", record)
}

// DeleteRecord handles DELETE /records/:id — soft-deletes a record.
func (h *RecordHandler) DeleteRecord(c *gin.Context) {
	id := c.Param("id")

	err := h.Service.DeleteRecord(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "financial record deleted successfully", nil)
}
