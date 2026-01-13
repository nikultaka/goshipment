package referrals

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// ReferralHandlers handles referral HTTP requests
type ReferralHandlers struct {
	service *ReferralService
}

// NewReferralHandlers creates a new referral handlers instance
func NewReferralHandlers(service *ReferralService) *ReferralHandlers {
	return &ReferralHandlers{
		service: service,
	}
}

// ListReferrals godoc
// @Summary List referral redemptions
// @Description Retrieve a list of referral redemptions. Sales Reps see only their own referrals (referrals they created), Admins/Managers see all referrals. Referral records are immutable and track referred users with commission eligibility snapshots. Includes status counts (active, revoked, expired).
// @Tags Referrals
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(active, revoked, expired)
// @Param from_date query string false "Filter from this date (YYYY-MM-DD)" format(date)
// @Param to_date query string false "Filter until this date (YYYY-MM-DD)" format(date)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListReferralsResponse} "Referrals retrieved successfully"
// @Failure 400 {object} response.Response "Invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized - JWT token required"
// @Failure 403 {object} response.Response "Forbidden - user not in organization or lacks permission"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /referrals [get]
func (h *ReferralHandlers) ListReferrals(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
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
	var req ListReferralsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List referrals
	result, err := h.service.ListReferrals(req, userID.(string), isAdmin)
	if err != nil {
		response.InternalError(c, "Failed to list referrals: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Referrals retrieved successfully", result)
}

// GetReferralByID godoc
// @Summary Get referral redemption by ID
// @Description Retrieve detailed information about a specific referral redemption. Must be owned by the requesting Sales Rep or accessed by an Admin/Manager. Referral records are immutable and cannot be modified. Shows commission snapshot, status, and involved users.
// @Tags Referrals
// @Produce json
// @Security BearerAuth
// @Param id path string true "Referral ID (UUID)" format(uuid)
// @Success 200 {object} response.Response{data=ReferralDetailResponse} "Referral retrieved successfully"
// @Failure 400 {object} response.Response "Invalid referral ID"
// @Failure 401 {object} response.Response "Unauthorized - JWT token required"
// @Failure 403 {object} response.Response "Forbidden - referral not owned by user or user lacks permission"
// @Failure 404 {object} response.Response "Referral not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /referrals/{id} [get]
func (h *ReferralHandlers) GetReferralByID(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get referral ID from path
	referralID := c.Param("id")
	if referralID == "" {
		response.BadRequest(c, "Referral ID is required")
		return
	}

	// Get user role
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	isAdmin := role == "admin" || role == "manager"

	// Get referral
	referral, err := h.service.GetReferralByID(referralID, userID.(string), isAdmin)
	if err != nil {
		if err.Error() == "referral not found or access denied" {
			response.NotFound(c, "Referral not found or access denied")
			return
		}
		response.InternalError(c, "Failed to get referral: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Referral retrieved successfully", referral)
}
