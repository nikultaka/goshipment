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

	isAdmin := role == "owner" || role == "administrator" || role == "manager"

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

	isAdmin := role == "owner" || role == "administrator" || role == "manager"

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

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can process payout requests")
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
// COMMISSION PLANS HANDLERS
// =====================================================

// ListPlans godoc
// @Summary List commission plans
// @Description Retrieve a list of commission plans for the organization. Only accessible by admins/managers.
// @Tags Commissions
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by plan status" Enums(active, paused)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListPlansResponse} "Plans retrieved successfully"
// @Failure 400 {object} response.Response "Invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/plans [get]
func (h *CommissionHandlers) ListPlans(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can view plans
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can view commission plans")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse query parameters
	var req ListPlansRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List plans
	result, err := h.service.ListPlans(req, organizationID)
	if err != nil {
		response.InternalError(c, "Failed to list plans: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Commission plans retrieved successfully", result)
}

// CreatePlan godoc
// @Summary Create a commission plan
// @Description Create a new commission plan for the organization. Only accessible by admins/managers.
// @Tags Commissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreatePlanRequest true "Commission plan details"
// @Success 201 {object} response.Response{data=CommissionPlanResponse} "Plan created successfully"
// @Failure 400 {object} response.Response "Invalid request or validation error"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/plans [post]
func (h *CommissionHandlers) CreatePlan(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can create plans
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can create commission plans")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse request body
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Create plan
	result, err := h.service.CreatePlan(req, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "must use") || contains(errMsg, "must be between") {
			response.BadRequest(c, err.Error())
		} else {
			response.InternalError(c, "Failed to create plan: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Commission plan created successfully", result)
}

// GetPlan godoc
// @Summary Get a commission plan
// @Description Retrieve a specific commission plan by ID. Only accessible by admins/managers.
// @Tags Commissions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Plan ID" format(uuid)
// @Success 200 {object} response.Response{data=CommissionPlanResponse} "Plan retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 404 {object} response.Response "Plan not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/plans/{id} [get]
func (h *CommissionHandlers) GetPlan(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can view plans
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can view commission plans")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get plan ID from path
	planID := c.Param("id")
	if planID == "" {
		response.BadRequest(c, "Plan ID is required")
		return
	}

	// Get plan
	result, err := h.service.GetPlan(planID, organizationID)
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Commission plan not found")
		} else {
			response.InternalError(c, "Failed to get plan: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Commission plan retrieved successfully", result)
}

// UpdatePlan godoc
// @Summary Update a commission plan
// @Description Update an existing commission plan. Only accessible by admins/managers.
// @Tags Commissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Plan ID" format(uuid)
// @Param request body UpdatePlanRequest true "Fields to update"
// @Success 200 {object} response.Response{data=CommissionPlanResponse} "Plan updated successfully"
// @Failure 400 {object} response.Response "Invalid request or validation error"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 404 {object} response.Response "Plan not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/plans/{id} [patch]
func (h *CommissionHandlers) UpdatePlan(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can update plans
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can update commission plans")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get plan ID from path
	planID := c.Param("id")
	if planID == "" {
		response.BadRequest(c, "Plan ID is required")
		return
	}

	// Parse request body
	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Update plan
	result, err := h.service.UpdatePlan(planID, req, organizationID)
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "not found") {
			response.NotFound(c, "Commission plan not found")
		} else if contains(errMsg, "must use") || contains(errMsg, "must be between") {
			response.BadRequest(c, err.Error())
		} else {
			response.InternalError(c, "Failed to update plan: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Commission plan updated successfully", result)
}

// DeletePlan godoc
// @Summary Delete a commission plan
// @Description Delete a commission plan. Plan cannot be deleted if it has active referrals or transactions. Only accessible by admins/managers.
// @Tags Commissions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Plan ID" format(uuid)
// @Success 200 {object} response.Response "Plan deleted successfully"
// @Failure 400 {object} response.Response "Cannot delete - plan is in use"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 404 {object} response.Response "Plan not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/plans/{id} [delete]
func (h *CommissionHandlers) DeletePlan(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can delete plans
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can delete commission plans")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get plan ID from path
	planID := c.Param("id")
	if planID == "" {
		response.BadRequest(c, "Plan ID is required")
		return
	}

	// Delete plan
	err = h.service.DeletePlan(planID, organizationID)
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "not found") {
			response.NotFound(c, "Commission plan not found")
		} else if contains(errMsg, "cannot delete") {
			response.BadRequest(c, err.Error())
		} else {
			response.InternalError(c, "Failed to delete plan: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Commission plan deleted successfully", nil)
}

// =====================================================
// MANUAL COMMISSION TRANSACTION HANDLERS
// =====================================================

// CreateCommissionTransaction godoc
// @Summary Create a commission transaction manually
// @Description Admin endpoint to manually create a commission transaction using an existing commission plan. This allows admins to credit commissions for bonuses, adjustments, or special cases without requiring a shipment or referral.
// @Tags Commissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCommissionTransactionRequest true "Commission transaction details"
// @Success 201 {object} response.Response{data=CreateCommissionTransactionResponse} "Commission transaction created successfully"
// @Failure 400 {object} response.Response "Invalid request or validation error"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin role"
// @Failure 404 {object} response.Response "Commission plan or beneficiary user not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /commissions/transactions [post]
func (h *CommissionHandlers) CreateCommissionTransaction(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can create manual transactions
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "owner" && role != "administrator" && role != "manager" {
		response.Forbidden(c, "Only owners, administrators and managers can create commission transactions")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse request body
	var req CreateCommissionTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Create commission transaction
	result, err := h.service.CreateCommissionTransaction(req, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "not active"), contains(errMsg, "must be"):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to create commission transaction: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Commission transaction created successfully", result)
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
