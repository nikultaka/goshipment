package commissions

import (
	"net/http"
	"strings"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// CommissionHandlers handles commission HTTP requests
type CommissionHandlers struct {
	service *CommissionService
}

// NewCommissionHandlers creates a new commission handlers instance
func NewCommissionHandlers(service *CommissionService) *CommissionHandlers {
	return &CommissionHandlers{
		service: service,
	}
}

// ListCommissions godoc
// @Summary List commission transactions
// @Description Retrieve a list of commission transactions. Sales Reps see only their own commissions, Admins see all commissions for the organization. Includes balance summaries (total earnings, eligible balance, paid balance).
// @Tags Commissions
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(pending, eligible, processing, paid, voided)
// @Param shipment_id query string false "Filter by shipment ID (UUID)" format(uuid)
// @Param from_date query string false "Filter from this date (YYYY-MM-DD)" format(date)
// @Param to_date query string false "Filter until this date (YYYY-MM-DD)" format(date)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListCommissionsResponse} "Commissions retrieved successfully"
// @Failure 400 {object} response.Response "Invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions [get]
func (h *CommissionHandlers) ListCommissions(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get user role
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	isAdmin := role == "admin" || role == "manager"

	// Parse query parameters
	var req ListCommissionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List commissions
	result, err := h.service.ListCommissions(req, organizationID, userID.(string), isAdmin)
	if err != nil {
		response.InternalError(c, "Failed to list commissions: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Commissions retrieved successfully", result)
}

// ListPayouts godoc
// @Summary List commission payouts
// @Description Retrieve a list of commission payout records. Sales Reps see only their own payouts, Admins see all payouts for the organization.
// @Tags Commissions
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by payout status" Enums(pending, processing, paid, rejected, failed, cancelled)
// @Param from_date query string false "Filter from this date (YYYY-MM-DD)" format(date)
// @Param to_date query string false "Filter until this date (YYYY-MM-DD)" format(date)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListPayoutsResponse} "Payouts retrieved successfully"
// @Failure 400 {object} response.Response "Invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/payouts [get]
func (h *CommissionHandlers) ListPayouts(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get user role
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	isAdmin := role == "admin" || role == "manager"

	// Parse query parameters
	var req ListPayoutsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List payouts
	result, err := h.service.ListPayouts(req, organizationID, userID.(string), isAdmin)
	if err != nil {
		response.InternalError(c, "Failed to list payouts: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Payouts retrieved successfully", result)
}

// RequestPayout godoc
// @Summary Request commission payout (Sales Rep) OR Process payout (Admin)
// @Description Sales Rep: Request a payout for eligible commission transactions. The request will be pending admin review. Admin: Process (approve/reject) a payout request. Approval creates a payout record and marks transactions as paid.
// @Tags Commissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RequestPayoutRequest true "Payout request details (for Sales Rep) - Provide commission_transaction_ids and payout_method"
// @Success 200 {object} response.Response{data=RequestPayoutResponse} "Payout request submitted successfully (Sales Rep)"
// @Failure 400 {object} response.Response "Invalid request - no eligible commissions, validation errors, or currency mismatch"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization or not owner of commissions"
// @Failure 409 {object} response.Response "Conflict - commission already paid or not eligible"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/payout [post]
func (h *CommissionHandlers) RequestPayout(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse request
	var req RequestPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Request payout
	result, err := h.service.RequestPayout(req, organizationID, userID.(string))
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "does not belong"), contains(errMsg, "not owner"):
			response.Forbidden(c, err.Error())
		case contains(errMsg, "not eligible"), contains(errMsg, "already paid"), contains(errMsg, "currency"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "no eligible"):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to request payout: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Payout request submitted successfully", result)
}

// ProcessPayout godoc
// @Summary Process payout request (Admin Only)
// @Description Admin-only endpoint to approve or reject a pending payout request. Approval creates a payout record and marks commission transactions as paid. Rejection updates the request status with a review note.
// @Tags Commissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProcessPayoutRequest true "Payout processing details - action must be 'approve' or 'reject'"
// @Success 200 {object} response.Response{data=ProcessPayoutResponse} "Payout processed successfully"
// @Failure 400 {object} response.Response "Invalid request or payout request not in pending status"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 404 {object} response.Response "Payout request not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/payout/process [post]
func (h *CommissionHandlers) ProcessPayout(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can process payouts
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "admin" && role != "manager" {
		response.Forbidden(c, "Only admins and managers can process payout requests")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse request
	var req ProcessPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Process payout
	result, err := h.service.ProcessPayout(req, organizationID, userID.(string))
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "not pending"):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to process payout: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Payout processed successfully", result)
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
