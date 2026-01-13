package shipments

import (
	"net/http"
	"strings"

	"gogin/internal/models"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// IShipmentService defines the interface for shipment services
type IShipmentService interface {
	GetOrganizationIDFromUser(userID string) (string, error)
	CreateShipment(req CreateShipmentRequest, organizationID string, userID string) (*models.Shipment, error)
	ListShipments(organizationID string, query ListShipmentsQuery) ([]*models.Shipment, int, error)
	GetShipmentByID(shipmentID string, organizationID string) (*models.Shipment, []*models.ShipmentPackage, []*models.ShipmentEvent, error)
	CancelShipment(shipmentID, organizationID, userID, reason string) error
	VoidShipment(shipmentID, organizationID, userID, reason string) error
	ArchiveShipment(shipmentID, organizationID, userID string) error
}

// ShipmentHandlers handles shipment HTTP requests
type ShipmentHandlers struct {
	service IShipmentService
}

// NewShipmentHandlers creates a new shipment handlers instance
func NewShipmentHandlers(service IShipmentService) *ShipmentHandlers {
	return &ShipmentHandlers{
		service: service,
	}
}

// CreateShipment godoc
// @Summary Create a shipment from an estimate
// @Description Create a shipment by selecting a carrier/service from a shipment estimate. This creates a wallet hold for the shipment cost.
// @Tags Shipments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateShipmentRequest true "Shipment creation request"
// @Success 201 {object} response.Response{data=ShipmentResponse} "Shipment created successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Estimate not found"
// @Failure 409 {object} response.Response "Estimate expired or insufficient wallet balance"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments [post]
func (h *ShipmentHandlers) CreateShipment(c *gin.Context) {
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
	var req CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Create shipment
	shipment, err := h.service.CreateShipment(req, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"), strings.Contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case strings.Contains(errMsg, "expired"), strings.Contains(errMsg, "insufficient"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to create shipment: "+err.Error())
		}
		return
	}

	// Build response
	resp := h.buildShipmentResponse(shipment)

	response.Success(c, http.StatusCreated, "Shipment created successfully", resp)
}

// ListShipments godoc
// @Summary List shipments for organization
// @Description Get a paginated list of shipments for the current organization
// @Tags Shipments
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status (pending, pre_label, label_purchased, in_transit, delivered, cancelled, failed)"
// @Param include_archived query bool false "Include archived shipments" default(false)
// @Success 200 {object} response.Response{data=[]ShipmentListItem} "Shipments retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments [get]
func (h *ShipmentHandlers) ListShipments(c *gin.Context) {
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

	// Parse query parameters
	var query ListShipmentsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		// Set defaults if binding fails
		query.Page = 1
		query.Limit = 20
	}

	// List shipments
	shipments, total, err := h.service.ListShipments(
		organizationID,
		query,
	)
	if err != nil {
		response.InternalError(c, "Failed to list shipments: "+err.Error())
		return
	}

	// Build response with pagination
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	page := query.Page
	if page < 1 {
		page = 1
	}

	result := map[string]interface{}{
		"shipments": shipments,
		"pagination": map[string]interface{}{
			"page":       page,
			"per_page":   limit,
			"total":      total,
			"total_pages": (total + limit - 1) / limit,
		},
	}

	response.Success(c, http.StatusOK, "Shipments retrieved successfully", result)
}

// GetShipment godoc
// @Summary Get shipment by ID
// @Description Get detailed information about a specific shipment
// @Tags Shipments
// @Produce json
// @Security BearerAuth
// @Param id path string true "Shipment ID (UUID)"
// @Success 200 {object} response.Response{data=ShipmentResponse} "Shipment retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Shipment not found or access denied"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments/{id} [get]
func (h *ShipmentHandlers) GetShipment(c *gin.Context) {
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

	// Get shipment ID from path
	shipmentID := c.Param("id")
	if shipmentID == "" {
		response.BadRequest(c, "Shipment ID is required")
		return
	}

	// Get shipment
	shipment, packages, events, err := h.service.GetShipmentByID(shipmentID, organizationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			response.NotFound(c, "Shipment not found or access denied")
			return
		}
		response.InternalError(c, "Failed to get shipment: "+err.Error())
		return
	}

	// Build response
	resp := h.buildShipmentDetailResponse(shipment, packages, events)

	response.Success(c, http.StatusOK, "Shipment retrieved successfully", resp)
}

// CancelShipment godoc
// @Summary Cancel a shipment
// @Description Cancel a shipment that is in pending or pre_label state. Releases wallet hold.
// @Tags Shipments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Shipment ID (UUID)"
// @Param request body CancelShipmentRequest true "Cancel request"
// @Success 200 {object} response.Response{data=ShipmentResponse} "Shipment cancelled successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Shipment not found"
// @Failure 409 {object} response.Response "Shipment cannot be cancelled in current state"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments/{id}/cancel [post]
func (h *ShipmentHandlers) CancelShipment(c *gin.Context) {
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

	// Get shipment ID from path
	shipmentID := c.Param("id")
	if shipmentID == "" {
		response.BadRequest(c, "Shipment ID is required")
		return
	}

	// Parse request
	var req CancelShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Cancel shipment
	err = h.service.CancelShipment(shipmentID, organizationID, userID.(string), req.Reason)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"), strings.Contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case strings.Contains(errMsg, "cannot be cancelled"), strings.Contains(errMsg, "invalid state"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to cancel shipment: "+err.Error())
		}
		return
	}

	// Get updated shipment
	shipment, packages, events, err := h.service.GetShipmentByID(shipmentID, organizationID)
	if err != nil {
		response.InternalError(c, "Shipment cancelled but failed to retrieve updated data: "+err.Error())
		return
	}

	// Build response
	resp := h.buildShipmentDetailResponse(shipment, packages, events)

	response.Success(c, http.StatusOK, "Shipment cancelled successfully", resp)
}

// VoidShipment godoc
// @Summary Void a shipment label
// @Description Void a shipment that has purchased labels but not yet shipped. Refunds to wallet.
// @Tags Shipments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Shipment ID (UUID)"
// @Param request body VoidShipmentRequest true "Void request"
// @Success 200 {object} response.Response{data=ShipmentResponse} "Shipment voided successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Shipment not found"
// @Failure 409 {object} response.Response "Shipment cannot be voided in current state"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments/{id}/void [post]
func (h *ShipmentHandlers) VoidShipment(c *gin.Context) {
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

	// Get shipment ID from path
	shipmentID := c.Param("id")
	if shipmentID == "" {
		response.BadRequest(c, "Shipment ID is required")
		return
	}

	// Parse request
	var req VoidShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Void shipment
	err = h.service.VoidShipment(shipmentID, organizationID, userID.(string), req.Reason)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"), strings.Contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case strings.Contains(errMsg, "cannot be voided"), strings.Contains(errMsg, "invalid state"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to void shipment: "+err.Error())
		}
		return
	}

	// Get updated shipment
	shipment, packages, events, err := h.service.GetShipmentByID(shipmentID, organizationID)
	if err != nil {
		response.InternalError(c, "Shipment voided but failed to retrieve updated data: "+err.Error())
		return
	}

	// Build response
	resp := h.buildShipmentDetailResponse(shipment, packages, events)

	response.Success(c, http.StatusOK, "Shipment voided successfully", resp)
}

// ArchiveShipment godoc
// @Summary Archive/unarchive a shipment
// @Description Toggle the archive status of a shipment
// @Tags Shipments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Shipment ID (UUID)"
// @Success 200 {object} response.Response{data=ShipmentResponse} "Shipment archive status updated successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Shipment not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipments/{id}/archive [post]
func (h *ShipmentHandlers) ArchiveShipment(c *gin.Context) {
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

	// Get shipment ID from path
	shipmentID := c.Param("id")
	if shipmentID == "" {
		response.BadRequest(c, "Shipment ID is required")
		return
	}

	// Archive shipment
	err = h.service.ArchiveShipment(shipmentID, organizationID, userID.(string))
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			response.NotFound(c, "Shipment not found or access denied")
			return
		}
		response.InternalError(c, "Failed to archive shipment: "+err.Error())
		return
	}

	// Get updated shipment
	shipment, packages, events, err := h.service.GetShipmentByID(shipmentID, organizationID)
	if err != nil {
		response.InternalError(c, "Shipment archived but failed to retrieve updated data: "+err.Error())
		return
	}

	// Build response
	resp := h.buildShipmentDetailResponse(shipment, packages, events)

	msg := "Shipment archived successfully"
	if !shipment.IsArchived {
		msg = "Shipment unarchived successfully"
	}

	response.Success(c, http.StatusOK, msg, resp)
}

// =====================================================
// HELPER METHODS
// =====================================================

// buildShipmentResponse builds a basic shipment response
func (h *ShipmentHandlers) buildShipmentResponse(shipment interface{}) interface{} {
	// For now, return the shipment as-is
	// In a production app, you'd build a proper DTO response
	return shipment
}

// buildShipmentDetailResponse builds a detailed shipment response with packages and events
func (h *ShipmentHandlers) buildShipmentDetailResponse(shipment interface{}, packages interface{}, events interface{}) map[string]interface{} {
	// For now, return a simple map
	// In a production app, you'd build a proper DTO response
	return map[string]interface{}{
		"shipment": shipment,
		"packages": packages,
		"timeline": events,
	}
}
