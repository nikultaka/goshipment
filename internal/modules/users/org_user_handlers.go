package users

import (
	"gogin/internal/modules/organizations"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Organization User Management Handlers
// These handlers implement the Users & Roles APIs
// =====================================================

// getMeProfile handles GET /users/me
// Returns the current user's profile with organization context
func (m *UsersModule) getMeProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get user profile with organization context
	profile, err := m.service.GetUserProfileWithOrganization(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get user profile")
		return
	}

	response.Success(c, 200, "User profile retrieved successfully", profile)
}

// updateMeProfile handles PATCH /users/me
// Updates the current user's own profile
func (m *UsersModule) updateMeProfile(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Bind request
	var req UpdateOwnProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Update profile
	profile, err := m.service.UpdateOwnProfile(userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 200, "Profile updated successfully", profile)
}

// listOrganizationUsers handles GET /organization/users
// Lists all users in the current user's organization
// Access: Admin only, or Manager with users.manage scope
func (m *UsersModule) listOrganizationUsers(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, member, err := orgService.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Check authorization: Admin or Manager with users.manage scope
	if !member.Role.IsAdminOrHigher() {
		// Check if user is manager with users.manage scope
		if !member.Role.IsManagerOrHigher() {
			response.Forbidden(c, "Access denied: insufficient permissions")
			return
		}

		// For managers, check if they have users.manage scope
		scopes, scopesExist := c.Get("scopes")
		if !scopesExist {
			response.Forbidden(c, "Access denied: scope information missing")
			return
		}

		hasScope := false
		for _, scope := range scopes.([]string) {
			if scope == "users.manage" || scope == "*" {
				hasScope = true
				break
			}
		}

		if !hasScope {
			response.Forbidden(c, "Access denied: users.manage scope required")
			return
		}
	}

	// List organization users
	users, err := m.service.ListOrganizationUsers(org.ID)
	if err != nil {
		response.InternalError(c, "Failed to list organization users")
		return
	}

	resp := &OrganizationUsersListResponse{
		Users: users,
		Total: len(users),
	}

	response.Success(c, 200, "Organization users retrieved successfully", resp)
}

// changeUserRole handles PATCH /users/:id/role
// Changes a user's role within the organization
// Access: Admin only
func (m *UsersModule) changeUserRole(c *gin.Context) {
	// Get user ID from context
	actorUserID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get target user ID from URL
	targetUserID := c.Param("id")
	if targetUserID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	// Bind request
	var req ChangeUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, member, err := orgService.GetOrganizationByUserID(actorUserID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Check authorization: Admin only
	if !member.Role.IsAdminOrHigher() {
		response.Forbidden(c, "Access denied: administrator role required")
		return
	}

	// Verify target user belongs to the same organization
	_, err = orgService.ValidateUserInOrganization(targetUserID, org.ID)
	if err != nil {
		response.NotFound(c, "User not found in organization")
		return
	}

	// Change role
	err = m.service.ChangeUserRole(actorUserID.(string), targetUserID, org.ID, req.Role)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 200, "User role updated successfully", nil)
}

// toggleUserSalesRep handles PATCH /users/:id/sales-rep
// Toggles the sales representative status for a user
// Access: Admin only
func (m *UsersModule) toggleUserSalesRep(c *gin.Context) {
	// Get user ID from context
	actorUserID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get target user ID from URL
	targetUserID := c.Param("id")
	if targetUserID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	// Bind request
	var req ToggleSalesRepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, member, err := orgService.GetOrganizationByUserID(actorUserID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Check authorization: Admin only
	if !member.Role.IsAdminOrHigher() {
		response.Forbidden(c, "Access denied: administrator role required")
		return
	}

	// Verify target user belongs to the same organization
	_, err = orgService.ValidateUserInOrganization(targetUserID, org.ID)
	if err != nil {
		response.NotFound(c, "User not found in organization")
		return
	}

	// Toggle sales rep status
	err = m.service.ToggleUserSalesRep(actorUserID.(string), targetUserID, org.ID, req.IsSalesRep)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 200, "Sales representative status updated successfully", nil)
}

// anonymizeUser handles POST /users/:id/anonymize
// Permanently anonymizes a user (GDPR/PIPEDA compliance)
// Access: Admin only
func (m *UsersModule) anonymizeUser(c *gin.Context) {
	// Get user ID from context
	actorUserID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get target user ID from URL
	targetUserID := c.Param("id")
	if targetUserID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	// Bind request
	var req AnonymizeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, member, err := orgService.GetOrganizationByUserID(actorUserID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Check authorization: Admin only
	if !member.Role.IsAdminOrHigher() {
		response.Forbidden(c, "Access denied: administrator role required")
		return
	}

	// Verify target user belongs to the same organization
	_, err = orgService.ValidateUserInOrganization(targetUserID, org.ID)
	if err != nil {
		response.NotFound(c, "User not found in organization")
		return
	}

	// Anonymize user
	err = m.service.AnonymizeUser(actorUserID.(string), targetUserID, org.ID, req.Reason)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 200, "User anonymized successfully", nil)
}
