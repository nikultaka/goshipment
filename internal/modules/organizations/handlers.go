package organizations

import (
	"net/http"

	"gogin/internal/models"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// OrganizationHandlers contains organization HTTP handlers
type OrganizationHandlers struct {
	service *OrganizationService
}

// NewOrganizationHandlers creates new organization handlers
func NewOrganizationHandlers(service *OrganizationService) *OrganizationHandlers {
	return &OrganizationHandlers{
		service: service,
	}
}

// GetMyOrganization godoc
// @Summary Get current user's organization
// @Description Retrieve the organization details for the authenticated user
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=GetOrganizationResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/current [get]
func (h *OrganizationHandlers) GetMyOrganization(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	org, member, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	resp := &GetOrganizationResponse{
		ID:          org.ID,
		ExternalID:  org.ExternalID,
		Name:        org.Name,
		OwnerUserID: org.OwnerUserID,
		Status:      org.Status,
		Plan:        org.Plan,
		CreatedAt:   org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		MyRole:      member.Role,
	}

	response.Success(c, http.StatusOK, "Organization retrieved successfully", resp)
}

// GetOrganizationMembers godoc
// @Summary Get organization members
// @Description Retrieve all members of the current user's organization (requires manager role or higher)
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]OrganizationMemberResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/members [get]
func (h *OrganizationHandlers) GetOrganizationMembers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	// Get user's organization
	org, member, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	// Check if user has permission (manager or higher)
	if !member.Role.IsManagerOrHigher() {
		response.Forbidden(c, "Insufficient permissions to view organization members")
		return
	}

	members, err := h.service.GetOrganizationMembers(org.ID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Convert to response DTOs
	memberResponses := make([]OrganizationMemberResponse, len(members))
	for i, m := range members {
		memberResponses[i] = OrganizationMemberResponse{
			ID:             m.ID,
			OrganizationID: m.OrganizationID,
			UserID:         m.UserID,
			Role:           m.Role,
			IsActive:       m.IsActive,
			JoinedAt:       m.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response.Success(c, http.StatusOK, "Organization members retrieved successfully", memberResponses)
}

// GetOrganizationSettings godoc
// @Summary Get organization settings
// @Description Retrieve admin-controlled feature flags for the organization (requires admin role)
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=GetOrganizationSettingsResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/settings [get]
func (h *OrganizationHandlers) GetOrganizationSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	// Get user's organization
	org, member, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	// Check if user has permission (admin or higher)
	if !member.Role.IsAdminOrHigher() {
		response.Forbidden(c, "Insufficient permissions to view organization settings")
		return
	}

	settings, err := h.service.GetOrganizationSettings(org.ID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	resp := &GetOrganizationSettingsResponse{
		AllowSameDayPickup: settings.AllowSameDayPickup,
		AllowNextDayPickup: settings.AllowNextDayPickup,
		UpdatedAt:          settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	response.Success(c, http.StatusOK, "Organization settings retrieved successfully", resp)
}

// UpdateOrganizationSettings godoc
// @Summary Update organization settings
// @Description Update admin-controlled feature flags (requires admin role)
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateOrganizationSettingsRequest true "Settings update payload"
// @Success 200 {object} response.Response{data=GetOrganizationSettingsResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/settings [patch]
func (h *OrganizationHandlers) UpdateOrganizationSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	// Get user's organization
	org, member, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	// Check if user has permission (admin or higher)
	if !member.Role.IsAdminOrHigher() {
		response.Forbidden(c, "Insufficient permissions to update organization settings")
		return
	}

	var req UpdateOrganizationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	settings, err := h.service.UpdateOrganizationSettings(org.ID, req.AllowSameDayPickup, req.AllowNextDayPickup)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	resp := &GetOrganizationSettingsResponse{
		AllowSameDayPickup: settings.AllowSameDayPickup,
		AllowNextDayPickup: settings.AllowNextDayPickup,
		UpdatedAt:          settings.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	response.Success(c, http.StatusOK, "Organization settings updated successfully", resp)
}

// GetOrganizationPreferences godoc
// @Summary Get organization preferences
// @Description Retrieve user-controlled defaults for the organization
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=GetOrganizationPreferencesResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/preferences [get]
func (h *OrganizationHandlers) GetOrganizationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	// Get user's organization
	org, _, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	prefs, err := h.service.GetOrganizationPreferences(org.ID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	resp := ToPreferencesResponse(prefs)
	response.Success(c, http.StatusOK, "Organization preferences retrieved successfully", resp)
}

// UpdateOrganizationPreferences godoc
// @Summary Update organization preferences
// @Description Update user-controlled defaults for the organization (requires manager role or higher)
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateOrganizationPreferencesRequest true "Preferences update payload"
// @Success 200 {object} response.Response{data=GetOrganizationPreferencesResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /organization/preferences [patch]
func (h *OrganizationHandlers) UpdateOrganizationPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	// Get user's organization
	org, member, err := h.service.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	// Check if user has permission (manager or higher)
	if !member.Role.IsManagerOrHigher() {
		response.Forbidden(c, "Insufficient permissions to update organization preferences")
		return
	}

	var req UpdateOrganizationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	// Convert DTO to service input
	input := UpdateOrganizationPreferencesInput{
		DefaultSenderAddressID:    req.DefaultSenderAddressID,
		DefaultRecipientAddressID: req.DefaultRecipientAddressID,
		PackageReference:          req.PackageReference,
		SignatureOption:           req.SignatureOption,
		DeliveryInstructions:      req.DeliveryInstructions,
		WeightUnit:                req.WeightUnit,
		DimensionUnit:             req.DimensionUnit,
		Timezone:                  req.Timezone,
		LabelFormat:               req.LabelFormat,
		HideShipperPhoneOnLabel:   req.HideShipperPhoneOnLabel,
		ShowUSDEstimates:          req.ShowUSDEstimates,
	}

	prefs, err := h.service.UpdateOrganizationPreferences(org.ID, input)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	resp := ToPreferencesResponse(prefs)
	response.Success(c, http.StatusOK, "Organization preferences updated successfully", resp)
}

// CreateOrganization godoc
// @Summary Create a new organization
// @Description Create a new organization with the authenticated user as the owner. User must not already belong to an organization.
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateOrganizationRequest true "Organization details"
// @Success 201 {object} response.Response{data=CreateOrganizationResponse} "Organization created successfully"
// @Failure 400 {object} response.Response "Invalid request or user already belongs to an organization"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /organizations [post]
func (h *OrganizationHandlers) CreateOrganization(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Parse request
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Set default plan if not provided
	plan := models.OrganizationPlanTier1
	if req.Plan != nil {
		plan = *req.Plan
	}

	// Create organization
	org, err := h.service.CreateOrganization(req.Name, plan, userID.(string))
	if err != nil {
		// Check if error is due to user already having an organization
		if err.Error() == "user already belongs to an organization" ||
		   (len(err.Error()) > 30 && err.Error()[:30] == "user already belongs to an org") {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Failed to create organization: "+err.Error())
		return
	}

	// Build response
	resp := CreateOrganizationResponse{
		ID:          org.ID,
		ExternalID:  org.ExternalID,
		Name:        org.Name,
		OwnerUserID: org.OwnerUserID,
		Status:      org.Status,
		Plan:        org.Plan,
		YourRole:    models.OrganizationRoleOwner,
		CreatedAt:   org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Message:     "Organization created successfully. You are now the owner.",
	}

	response.Success(c, http.StatusCreated, "Organization created successfully", resp)
}
