package pickups

import (
	"net/http"
	"strings"

	"gogin/internal/middleware"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// PickupHandlers handles pickup HTTP requests
type PickupHandlers struct {
	service *PickupService
}

// NewPickupHandlers creates a new pickup handlers instance
func NewPickupHandlers(service *PickupService) *PickupHandlers {
	return &PickupHandlers{
		service: service,
	}
}

// SchedulePickup godoc
// @Summary Schedule a pickup for a shipment
// @Description Schedule a carrier pickup for a shipment. Shipment must be in 'label_purchased' or 'ready_for_pickup' status. Same-day pickups require system and organization settings to allow them. The pickup address is taken from the shipment's from_address and stored as an immutable snapshot.
// @Tags Pickups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SchedulePickupRequest true "Pickup schedule request"
// @Success 200 {object} response.Response{data=PickupResponse} "Pickup scheduled successfully"
// @Failure 400 {object} response.Response "Invalid request - validation errors"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - same-day pickup not allowed or user not in organization"
// @Failure 404 {object} response.Response "Shipment not found or access denied"
// @Failure 409 {object} response.Response "Conflict - shipment not in valid status for pickup"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /pickups [post]
func (h *PickupHandlers) SchedulePickup(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access all)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	// Parse request
	var req SchedulePickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Schedule pickup
	result, err := h.service.SchedulePickup(req, organizationID, userID.(string))
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "same-day pickup is not allowed"):
			response.Forbidden(c, err.Error())
		case contains(errMsg, "must be in"), contains(errMsg, "status"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "invalid"), contains(errMsg, "format"), contains(errMsg, "required"):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to schedule pickup: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Pickup scheduled successfully", result)
}

// CancelPickup godoc
// @Summary Cancel a scheduled pickup
// @Description Cancel a previously scheduled pickup. Cannot cancel pickups that are already completed. Updates shipment status back to 'ready_for_pickup'.
// @Tags Pickups
// @Produce json
// @Security BearerAuth
// @Param id path string true "Pickup ID (UUID)" format(uuid)
// @Success 200 {object} response.Response{data=CancelPickupResponse} "Pickup cancelled successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Pickup not found or access denied"
// @Failure 409 {object} response.Response "Conflict - pickup already cancelled or completed"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /pickups/{id}/cancel [post]
func (h *PickupHandlers) CancelPickup(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access all)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	// Get pickup ID from path
	pickupID := c.Param("id")
	if pickupID == "" {
		response.BadRequest(c, "Pickup ID is required")
		return
	}

	// Cancel pickup
	result, err := h.service.CancelPickup(pickupID, organizationID, userID.(string))
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "already cancelled"), contains(errMsg, "cannot cancel"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to cancel pickup: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Pickup cancelled successfully", result)
}

// ListPickups godoc
// @Summary List pickups for organization
// @Description Retrieve a paginated list of pickups for the authenticated user's organization. Users see only their own shipments' pickups, while managers and admins see all organization pickups. Supports filtering by status, shipment ID, and date range.
// @Tags Pickups
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by pickup status" Enums(draft, scheduled, rescheduled, cancelled, completed, failed)
// @Param shipment_id query string false "Filter by shipment ID (UUID)" format(uuid)
// @Param from_date query string false "Filter pickups from this date (YYYY-MM-DD)" format(date)
// @Param to_date query string false "Filter pickups until this date (YYYY-MM-DD)" format(date)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListPickupsResponse} "Pickups retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request - invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /pickups [get]
func (h *PickupHandlers) ListPickups(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access all)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	// Parse query parameters
	var req ListPickupsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List pickups
	result, err := h.service.ListPickups(req, organizationID, userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to list pickups: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pickups retrieved successfully", result)
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
