package audit

import (
	"encoding/csv"
	"fmt"
	"gogin/internal/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// getAuditLogs retrieves audit logs with filtering and pagination
// @Summary Get audit logs
// @Description Retrieve audit logs with optional filtering and pagination (Admin only)
// @Tags Admin - Audit Logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id query string false "Filter by user ID"
// @Param actor_type query string false "Filter by actor type (user, admin, api_client, system)"
// @Param action query string false "Filter by action (partial match)"
// @Param resource_type query string false "Filter by resource type"
// @Param status query string false "Filter by status (success, failure, denied)"
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param sort_by query string false "Sort field (occurred_at, recorded_at)" default(occurred_at)
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} response.Response{data=AuditLogsListResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/audit-logs [get]
func (m *AuditModule) getAuditLogs(c *gin.Context) {
	var filters AuditLogFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	logs, err := m.service.GetAuditLogs(&filters)
	if err != nil {
		response.InternalError(c, "Failed to retrieve audit logs: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Audit logs retrieved successfully", logs)
}

// exportAuditLogs exports audit logs to CSV format
// @Summary Export audit logs
// @Description Export audit logs to CSV format with optional filtering (Admin only)
// @Tags Admin - Audit Logs
// @Accept json
// @Produce text/csv
// @Security BearerAuth
// @Param user_id query string false "Filter by user ID"
// @Param actor_type query string false "Filter by actor type (user, admin, api_client, system)"
// @Param action query string false "Filter by action (partial match)"
// @Param resource_type query string false "Filter by resource type"
// @Param status query string false "Filter by status (success, failure, denied)"
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Param sort_by query string false "Sort field (occurred_at, recorded_at)" default(occurred_at)
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {file} file "CSV file"
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/audit-logs/export [get]
func (m *AuditModule) exportAuditLogs(c *gin.Context) {
	var filters AuditLogFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Set response headers for CSV download
	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Create CSV writer
	writer := csv.NewWriter(c.Writer)

	// Export logs to CSV
	if err := m.service.ExportAuditLogsToCSV(&filters, writer); err != nil {
		response.InternalError(c, "Failed to export audit logs: "+err.Error())
		return
	}

	c.Status(http.StatusOK)
}

// getAuditLogDetail retrieves a specific audit log by ID (NEW)
// @Summary Get audit log detail
// @Description Retrieve detailed information about a specific audit log entry (Admin only)
// @Tags Admin - Audit Logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Audit Log ID"
// @Success 200 {object} response.Response{data=AuditLogResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/audit-logs/{id} [get]
func (m *AuditModule) getAuditLogDetail(c *gin.Context) {
	id := c.Param("id")

	log, err := m.service.GetAuditLogByID(id)
	if err != nil {
		if err.Error() == "audit log not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, "Failed to retrieve audit log: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Audit log retrieved successfully", log)
}
