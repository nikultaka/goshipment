package organizations

import (
	"fmt"
	"net/http"

	"gogin/internal/middleware"
	"gogin/internal/models"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// OrganizationInviteHandlers handles organization invitation HTTP requests
type OrganizationInviteHandlers struct {
	service    *OrganizationInviteService
	orgService *OrganizationService
}

// NewOrganizationInviteHandlers creates a new organization invite handlers instance
func NewOrganizationInviteHandlers(service *OrganizationInviteService, orgService *OrganizationService) *OrganizationInviteHandlers {
	return &OrganizationInviteHandlers{
		service:    service,
		orgService: orgService,
	}
}

// CreateInvite godoc
// @Summary Create organization invitation
// @Description Invite a user to join your organization by email. Sends an email with a secure invite link. (Admin/Owner only)
// @Tags Organization Invites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateInviteRequest true "Invitation details"
// @Success 201 {object} response.Response{data=CreateInviteResponse} "Invitation created successfully"
// @Failure 400 {object} response.Response "Invalid request or user already invited"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - insufficient permissions"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organization/invites [post]
func (h *OrganizationInviteHandlers) CreateInvite(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization from database (admin users can access any org via query param)
	var org *models.Organization
	orgID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && orgID != "" {
		// Admin user with specific org ID - get that org
		var err error
		org, err = h.orgService.GetOrganizationByID(orgID)
		if err != nil {
			response.NotFound(c, "Organization not found")
			return
		}
	} else if !isAdminUser {
		// Non-admin user - must be member of org
		var err error
		org, _, err = h.orgService.GetOrganizationByUserID(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	} else {
		// Admin user without org ID - require org ID
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	}

	// Parse request
	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Build invite link (before creating invite so we can pass it to service)
	baseURL := "http://localhost:8081" // TODO: Get from config
	// Temporary token will be replaced by actual token after creation

	// Create invitation (service will generate token and send email)
	invite, err := h.service.CreateInvite(req, org.ID, userID.(string), baseURL+"/invites/accept")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Build actual invite link with real token
	inviteLink := fmt.Sprintf("%s/invites/accept?token=%s", baseURL, invite.InviteToken)

	// Build response
	resp := CreateInviteResponse{
		Invite: InviteResponse{
			ID:             invite.ID,
			OrganizationID: invite.OrganizationID,
			Email:          invite.Email,
			Role:           invite.Role,
			Status:         string(invite.Status),
			ExpiresAt:      invite.ExpiresAt,
			CreatedAt:      invite.CreatedAt,
			IsExpired:      invite.IsExpired(),
		},
		InviteLink: inviteLink,
		Message:    fmt.Sprintf("Invitation sent successfully to %s", invite.Email),
	}

	response.Success(c, http.StatusCreated, "Invitation created successfully", resp)
}

// AcceptInvite godoc
// @Summary Accept organization invitation
// @Description Accept an invitation to join an organization using the invite token. Creates organization membership.
// @Tags Organization Invites
// @Accept json
// @Produce json
// @Param request body AcceptInviteRequest true "Invite token"
// @Success 200 {object} response.Response{data=AcceptInviteResponse} "Invitation accepted successfully"
// @Failure 400 {object} response.Response "Invalid or expired token"
// @Failure 404 {object} response.Response "Invitation not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organization/invites/accept [post]
func (h *OrganizationInviteHandlers) AcceptInvite(c *gin.Context) {
	// Parse request
	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Accept invitation
	invite, err := h.service.AcceptInvite(req.InviteToken)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get organization info
	orgInfo, err := h.service.GetOrganizationInfo(invite.OrganizationID)
	if err != nil {
		response.InternalError(c, "Failed to retrieve organization info")
		return
	}

	// Build response
	resp := AcceptInviteResponse{
		Message:       "Invitation accepted successfully. You are now a member of the organization.",
		Organization:  *orgInfo,
		YourRole:      invite.Role,
		AccessGranted: true,
	}

	response.Success(c, http.StatusOK, "Invitation accepted successfully", resp)
}

// RejectInvite godoc
// @Summary Reject organization invitation
// @Description Reject an invitation to join an organization using the invite token.
// @Tags Organization Invites
// @Accept json
// @Produce json
// @Param request body RejectInviteRequest true "Invite token"
// @Success 200 {object} response.Response{data=RejectInviteResponse} "Invitation rejected successfully"
// @Failure 400 {object} response.Response "Invalid or expired token"
// @Failure 404 {object} response.Response "Invitation not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organization/invites/reject [post]
func (h *OrganizationInviteHandlers) RejectInvite(c *gin.Context) {
	// Parse request
	var req RejectInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Reject invitation
	invite, err := h.service.RejectInvite(req.InviteToken)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get organization info
	orgInfo, err := h.service.GetOrganizationInfo(invite.OrganizationID)
	if err != nil {
		response.InternalError(c, "Failed to retrieve organization info")
		return
	}

	// Build response
	resp := RejectInviteResponse{
		Message:          "Invitation rejected successfully.",
		OrganizationName: orgInfo.Name,
	}

	response.Success(c, http.StatusOK, "Invitation rejected successfully", resp)
}

// ListInvites godoc
// @Summary List organization invitations
// @Description Get a paginated list of invitations for the current organization (Admin/Owner only)
// @Tags Organization Invites
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (pending, accepted, rejected, expired, revoked)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=ListInvitesResponse} "Invitations retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organization/invites [get]
func (h *OrganizationInviteHandlers) ListInvites(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization from database (admin users can access any org via query param)
	var org *models.Organization
	orgID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && orgID != "" {
		// Admin user with specific org ID - get that org
		var err error
		org, err = h.orgService.GetOrganizationByID(orgID)
		if err != nil {
			response.NotFound(c, "Organization not found")
			return
		}
	} else if !isAdminUser {
		// Non-admin user - must be member of org
		var err error
		org, _, err = h.orgService.GetOrganizationByUserID(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	} else {
		// Admin user without org ID - require org ID
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	}

	// Parse query parameters
	var query ListInvitesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		query.Page = 1
		query.Limit = 20
	}

	// List invitations
	invites, total, err := h.service.ListInvites(org.ID, query)
	if err != nil {
		response.InternalError(c, "Failed to list invitations: "+err.Error())
		return
	}

	// Build pagination
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	page := query.Page
	if page < 1 {
		page = 1
	}

	resp := ListInvitesResponse{
		Invites: invites,
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	}

	response.Success(c, http.StatusOK, "Invitations retrieved successfully", resp)
}

// GetAvailableRoles godoc
// @Summary Get available organization roles
// @Description Returns list of available roles that can be assigned when creating invitations
// @Tags Organization Invites
// @Produce json
// @Success 200 {object} response.Response{data=[]RoleInfo} "Available roles"
// @Router /organization/invites/roles [get]
func (h *OrganizationInviteHandlers) GetAvailableRoles(c *gin.Context) {
	roles := []RoleInfo{
		{
			Name:        "viewer",
			DisplayName: "Viewer",
			Description: "Read-only access to organization resources",
			Level:       10,
		},
		{
			Name:        "operator",
			DisplayName: "Operator",
			Description: "Can create and manage shipments, labels, and pickups",
			Level:       30,
		},
		{
			Name:        "manager",
			DisplayName: "Manager",
			Description: "Can manage team members, shipments, and access reports",
			Level:       50,
		},
		{
			Name:        "administrator",
			DisplayName: "Administrator",
			Description: "Full access to organization settings and management",
			Level:       70,
		},
		{
			Name:        "owner",
			DisplayName: "Owner",
			Description: "Complete control over organization including billing and deletion",
			Level:       100,
		},
	}

	response.Success(c, http.StatusOK, "Available roles retrieved successfully", roles)
}

// RevokeInvite godoc
// @Summary Revoke organization invitation
// @Description Revoke a pending invitation (Admin/Owner only)
// @Tags Organization Invites
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invitation ID (UUID)"
// @Success 200 {object} response.Response "Invitation revoked successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden"
// @Failure 404 {object} response.Response "Invitation not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organization/invites/{id} [delete]
func (h *OrganizationInviteHandlers) RevokeInvite(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization from database (admin users can access any org via query param)
	var org *models.Organization
	orgID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && orgID != "" {
		// Admin user with specific org ID - get that org
		var err error
		org, err = h.orgService.GetOrganizationByID(orgID)
		if err != nil {
			response.NotFound(c, "Organization not found")
			return
		}
	} else if !isAdminUser {
		// Non-admin user - must be member of org
		var err error
		org, _, err = h.orgService.GetOrganizationByUserID(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	} else {
		// Admin user without org ID - require org ID
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	}

	// Get invitation ID from path
	inviteID := c.Param("id")
	if inviteID == "" {
		response.BadRequest(c, "Invitation ID is required")
		return
	}

	// Revoke invitation
	err := h.service.RevokeInvite(inviteID, org.ID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Invitation revoked successfully", nil)
}
