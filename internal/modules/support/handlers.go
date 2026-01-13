package support

import (
	"net/http"
	"strings"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// SupportHandlers handles support ticket HTTP requests
type SupportHandlers struct {
	service *SupportService
}

// NewSupportHandlers creates a new support handlers instance
func NewSupportHandlers(service *SupportService) *SupportHandlers {
	return &SupportHandlers{
		service: service,
	}
}

// CreateSupportTicket godoc
// @Summary Create a new support ticket
// @Description Create a new support ticket for the authenticated user's organization. Initial message is required. Optional links to shipments, invoices, or pickups can be included.
// @Tags Support
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateSupportTicketRequest true "Support ticket details"
// @Success 201 {object} response.Response{data=SupportTicketDetailResponse} "Ticket created successfully"
// @Failure 400 {object} response.Response "Invalid request - validation errors"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /support [post]
func (h *SupportHandlers) CreateSupportTicket(c *gin.Context) {
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
	var req CreateSupportTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Create support ticket
	result, err := h.service.CreateSupportTicket(req, organizationID, userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to create support ticket: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Support ticket created successfully", result)
}

// ReplySupportTicket godoc
// @Summary Reply to a support ticket
// @Description Add a reply/message to an existing support ticket. Users can reply to their own tickets, managers/admins can reply to any ticket in the organization.
// @Tags Support
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID (UUID)" format(uuid)
// @Param request body ReplySupportTicketRequest true "Reply message"
// @Success 200 {object} response.Response{data=SupportTicketMessageResponse} "Reply added successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Ticket not found or access denied"
// @Failure 409 {object} response.Response "Conflict - ticket is closed"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /support/{id}/reply [post]
func (h *SupportHandlers) ReplySupportTicket(c *gin.Context) {
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

	isStaff := role == "admin" || role == "manager"

	// Get ticket ID from path
	ticketID := c.Param("id")
	if ticketID == "" {
		response.BadRequest(c, "Ticket ID is required")
		return
	}

	// Parse request
	var req ReplySupportTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Reply to ticket
	result, err := h.service.ReplySupportTicket(ticketID, organizationID, userID.(string), req, isStaff)
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "closed"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to reply to ticket: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Reply added successfully", result)
}

// ManageSupportTicket godoc
// @Summary Manage a support ticket (Admin/Manager only)
// @Description Perform administrative actions on a support ticket. Available actions: assign, escalate, close, reopen. Requires manager or admin role.
// @Tags Support
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID (UUID)" format(uuid)
// @Param request body ManageSupportTicketRequest true "Management action details"
// @Success 200 {object} response.Response{data=ManageSupportTicketResponse} "Action performed successfully"
// @Failure 400 {object} response.Response "Invalid request or action"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires admin/manager role"
// @Failure 404 {object} response.Response "Ticket not found or access denied"
// @Failure 409 {object} response.Response "Conflict - invalid state for requested action"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /support/{id}/manage [post]
func (h *SupportHandlers) ManageSupportTicket(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Check role - only admin/manager can manage tickets
	role, err := h.service.GetUserRole(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user role: "+err.Error())
		return
	}

	if role != "admin" && role != "manager" {
		response.Forbidden(c, "Only admins and managers can manage tickets")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get ticket ID from path
	ticketID := c.Param("id")
	if ticketID == "" {
		response.BadRequest(c, "Ticket ID is required")
		return
	}

	// Parse request
	var req ManageSupportTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Manage ticket
	result, err := h.service.ManageSupportTicket(ticketID, organizationID, userID.(string), req)
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "required"), contains(errMsg, "invalid"):
			response.BadRequest(c, err.Error())
		case contains(errMsg, "not closed"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to manage ticket: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket "+req.Action+" successfully", result)
}

// GetSupportTicket godoc
// @Summary Get support ticket details
// @Description Retrieve a specific support ticket with all its messages. Users can view their own tickets, managers/admins can view all organization tickets. Admin notes are only visible to admins.
// @Tags Support
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID (UUID)" format(uuid)
// @Success 200 {object} response.Response{data=SupportTicketDetailResponse} "Ticket retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - access denied or user not in organization"
// @Failure 404 {object} response.Response "Ticket not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /support/{id} [get]
func (h *SupportHandlers) GetSupportTicket(c *gin.Context) {
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

	// Get ticket ID from path
	ticketID := c.Param("id")
	if ticketID == "" {
		response.BadRequest(c, "Ticket ID is required")
		return
	}

	// Get ticket detail
	result, err := h.service.GetSupportTicketDetail(ticketID, organizationID, userID.(string), isAdmin)
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		default:
			response.InternalError(c, "Failed to get ticket: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket retrieved successfully", result)
}

// ListSupportTickets godoc
// @Summary List support tickets
// @Description Retrieve a paginated list of support tickets. Users see only their own tickets, managers/admins see all organization tickets. Supports filtering by status, category, priority, and assigned user.
// @Tags Support
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(open, resolved, closed)
// @Param category query string false "Filter by category" Enums(shipment, wallet, pickup, invoice, billing, account, technical, other)
// @Param priority query string false "Filter by priority" Enums(critical, high, normal, low)
// @Param assigned_to query string false "Filter by assigned user ID (UUID)" format(uuid)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} response.Response{data=ListSupportTicketsResponse} "Tickets retrieved successfully"
// @Failure 400 {object} response.Response "Invalid query parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /support [get]
func (h *SupportHandlers) ListSupportTickets(c *gin.Context) {
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
	var req ListSupportTicketsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// List tickets
	result, err := h.service.ListSupportTickets(req, organizationID, userID.(string), isAdmin)
	if err != nil {
		response.InternalError(c, "Failed to list tickets: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Tickets retrieved successfully", result)
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
