package handlers

import (
	"net/http"

	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
)

// DashboardHandler handles analytics and dashboard HTTP requests.
type DashboardHandler struct {
	Service *services.DashboardService
}

// GetSummary handles GET /dashboard/summary — returns aggregate financial totals.
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	summary, err := h.Service.GetSummary()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "dashboard summary retrieved successfully", summary)
}

// GetTrends handles GET /dashboard/trends — returns monthly income vs expense breakdown.
func (h *DashboardHandler) GetTrends(c *gin.Context) {
	trends, err := h.Service.GetTrends()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "trends retrieved successfully", trends)
}

// GetCategoryBreakdown handles GET /dashboard/categories — returns per-category totals.
func (h *DashboardHandler) GetCategoryBreakdown(c *gin.Context) {
	breakdown, err := h.Service.GetCategoryBreakdown()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	utils.Success(c, http.StatusOK, "category breakdown retrieved successfully", breakdown)
}
