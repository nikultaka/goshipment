package dashboard

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// DashboardHandlers handles dashboard HTTP requests
type DashboardHandlers struct {
	service *DashboardService
}

// NewDashboardHandlers creates a new dashboard handlers instance
func NewDashboardHandlers(service *DashboardService) *DashboardHandlers {
	return &DashboardHandlers{
		service: service,
	}
}

// GetFinancialMetrics godoc
// @Summary Get financial metrics for dashboard
// @Description Retrieve financial metrics including gross revenue, carrier costs, gross profit, commissions, and net profit. Supports filtering by time period and organization.
// @Tags Admin - Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period filter" Enums(today, yesterday, last_7_days, last_week, this_month, last_month, this_quarter, this_year, custom) default(today)
// @Param start_date query string false "Start date for custom period (YYYY-MM-DD)" format(date) example(2026-01-01)
// @Param end_date query string false "End date for custom period (YYYY-MM-DD)" format(date) example(2026-01-31)
// @Param organization_id query string false "Filter by organization ID (admin can view any org)" format(uuid)
// @Param currency query string false "Currency code" default(CAD) example(CAD)
// @Param compare query boolean false "Include comparison with previous period" default(false)
// @Success 200 {object} response.Response{data=FinancialMetricsResponse} "Financial metrics retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /admin/dashboard/financial-metrics [get]
func (h *DashboardHandlers) GetFinancialMetrics(c *gin.Context) {
	// Parse request parameters
	var req FinancialMetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	// Validate custom period dates
	if req.Period == "custom" {
		if req.StartDate == "" || req.EndDate == "" {
			response.BadRequest(c, "start_date and end_date are required when period is 'custom'")
			return
		}
	}

	// Get financial metrics
	result, err := h.service.GetFinancialMetrics(req)
	if err != nil {
		response.InternalError(c, "Failed to get financial metrics: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Financial metrics retrieved successfully", result)
}

// GetRevenueChart godoc
// @Summary Get revenue chart data
// @Description Retrieve daily/weekly/monthly revenue data for charting
// @Tags Admin - Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period filter" Enums(last_7_days, last_30_days, this_month, last_month, this_quarter, this_year) default(last_30_days)
// @Param granularity query string false "Data granularity" Enums(daily, weekly, monthly) default(daily)
// @Param organization_id query string false "Filter by organization ID" format(uuid)
// @Success 200 {object} response.Response "Revenue chart data retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /admin/dashboard/revenue-chart [get]
func (h *DashboardHandlers) GetRevenueChart(c *gin.Context) {
	// Placeholder for future chart data endpoint
	response.Success(c, http.StatusOK, "Revenue chart data retrieved successfully", map[string]interface{}{
		"message": "Chart endpoint - coming soon",
		"data":    []interface{}{},
	})
}

// GetTopCarriers godoc
// @Summary Get top carriers by revenue
// @Description Retrieve top performing carriers ranked by revenue
// @Tags Admin - Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period filter" Enums(today, yesterday, last_7_days, last_week, this_month, last_month) default(this_month)
// @Param limit query int false "Number of carriers to return" default(5) minimum(1) maximum(20)
// @Param organization_id query string false "Filter by organization ID" format(uuid)
// @Success 200 {object} response.Response{data=[]CarrierBreakdown} "Top carriers retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /admin/dashboard/top-carriers [get]
func (h *DashboardHandlers) GetTopCarriers(c *gin.Context) {
	period := c.DefaultQuery("period", "this_month")
	organizationID := c.Query("organization_id")

	dateRange, err := h.service.GetDateRange(period, "", "")
	if err != nil {
		response.BadRequest(c, "Invalid period: "+err.Error())
		return
	}

	carriers, err := h.service.getCarrierBreakdown(dateRange, organizationID)
	if err != nil {
		response.InternalError(c, "Failed to get top carriers: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Top carriers retrieved successfully", carriers)
}

// GetShipmentStats godoc
// @Summary Get shipment statistics
// @Description Retrieve shipment count statistics by status
// @Tags Admin - Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Time period filter" Enums(today, yesterday, last_7_days, last_week, this_month, last_month) default(today)
// @Param organization_id query string false "Filter by organization ID" format(uuid)
// @Success 200 {object} response.Response{data=StatusBreakdown} "Shipment stats retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /admin/dashboard/shipment-stats [get]
func (h *DashboardHandlers) GetShipmentStats(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	organizationID := c.Query("organization_id")

	dateRange, err := h.service.GetDateRange(period, "", "")
	if err != nil {
		response.BadRequest(c, "Invalid period: "+err.Error())
		return
	}

	stats, err := h.service.getStatusBreakdown(dateRange, organizationID)
	if err != nil {
		response.InternalError(c, "Failed to get shipment stats: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Shipment stats retrieved successfully", stats)
}
