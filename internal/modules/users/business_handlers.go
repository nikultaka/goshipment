package users

import (
	"fmt"
	"net/http"

	"gogin/internal/modules/storage"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// getBusiness gets the authenticated user's business profile
// @Summary Get business profile
// @Description Get authenticated user's business profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=BusinessResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response "Business profile not found"
// @Router /users/me/business [get]
func (m *UsersModule) getBusiness(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	business, email, phone, err := m.service.GetBusiness(userID.(string))
	if err != nil {
		response.NotFound(c, "Business profile not found")
		return
	}

	// Get base URL for avatar URLs
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}

	businessResponse := m.service.sanitizeBusiness(business, email, phone, baseURL)

	response.Success(c, http.StatusOK, "Business profile retrieved successfully", gin.H{
		"business": businessResponse,
	})
}

// createBusiness creates a new business profile for the authenticated user
// @Summary Create business profile
// @Description Create a new business profile for the authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateBusinessRequest true "Business details"
// @Success 201 {object} response.Response{data=BusinessResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/business [post]
func (m *UsersModule) createBusiness(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	business, err := m.service.CreateBusiness(userID.(string), &req)
	if err != nil {
		if err.Error() == "user already has a business profile" {
			response.BadRequest(c, "User already has a business profile")
			return
		}
		response.BadRequest(c, fmt.Sprintf("Failed to create business: %v", err))
		return
	}

	// Get base URL for avatar URLs
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}

	businessResponse := m.service.sanitizeBusiness(business, nil, nil, baseURL)

	response.Success(c, http.StatusCreated, "Business profile created successfully", gin.H{
		"business": businessResponse,
	})
}

// updateBusiness updates the authenticated user's business profile
// @Summary Update business profile
// @Description Update the authenticated user's business profile (partial updates supported)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateBusinessRequest true "Business updates"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/me/business [patch]
func (m *UsersModule) updateBusiness(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.UpdateBusiness(userID.(string), &req)
	if err != nil {
		if err.Error() == "business profile not found" {
			response.NotFound(c, "Business profile not found")
			return
		}
		response.BadRequest(c, fmt.Sprintf("Failed to update business: %v", err))
		return
	}

	response.Success(c, http.StatusOK, "Business profile updated successfully", nil)
}

// uploadBusinessAvatar uploads an avatar for the authenticated user's business
// @Summary Upload business avatar
// @Description Upload an avatar image for the authenticated user's business profile
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "Avatar image file (JPEG, PNG, GIF, WebP, max 5MB)"
// @Success 200 {object} response.Response{data=object{avatar_url=string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response "Business profile not found"
// @Failure 413 {object} response.Response "File too large"
// @Router /users/me/business/avatar [post]
func (m *UsersModule) uploadBusinessAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Check if business exists
	_, _, _, err := m.service.GetBusiness(userID.(string))
	if err != nil {
		response.NotFound(c, "Business profile not found. Please create a business profile first.")
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "Avatar file is required")
		return
	}

	// Validate file size (5MB max)
	maxSize := int64(5 * 1024 * 1024) // 5MB
	if file.Size > maxSize {
		response.Error(c, http.StatusRequestEntityTooLarge, "File size exceeds maximum allowed (5MB)", "FILE_TOO_LARGE")
		return
	}

	// Validate file type
	mimeType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedTypes[mimeType] {
		response.BadRequest(c, "Invalid file type. Allowed types: JPEG, PNG, GIF, WebP")
		return
	}

	// Get old avatar for cleanup
	oldAvatarFileID, _ := m.service.DeleteBusinessAvatar(userID.(string))

	// Upload the file using storage service
	uploadReq := &storage.UploadRequest{
		Visibility: "public",
		Metadata:   `{"type":"business_avatar"}`,
	}
	uploadedFile, err := m.storageService.UploadFile(file, uploadReq, userID.(string))
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Failed to upload avatar: %v", err))
		return
	}

	// Update business avatar reference
	err = m.service.UpdateBusinessAvatar(userID.(string), uploadedFile.ID)
	if err != nil {
		// Delete the uploaded file since we couldn't update the reference
		m.storageService.DeleteFile(uploadedFile.ID, userID.(string))
		response.BadRequest(c, "Failed to update business avatar")
		return
	}

	// Delete old avatar file if it exists
	if oldAvatarFileID != "" {
		_ = m.storageService.DeleteFile(oldAvatarFileID, userID.(string))
	}

	// Build avatar URL
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}
	avatarURL := fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, uploadedFile.ID)

	response.Success(c, http.StatusOK, "Business avatar uploaded successfully", gin.H{
		"avatar_url": avatarURL,
	})
}

// deleteBusinessAvatar deletes the avatar from the authenticated user's business
// @Summary Delete business avatar
// @Description Remove the avatar from the authenticated user's business profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response "Business profile not found"
// @Router /users/me/business/avatar [delete]
func (m *UsersModule) deleteBusinessAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Check if business exists
	_, _, _, err := m.service.GetBusiness(userID.(string))
	if err != nil {
		response.NotFound(c, "Business profile not found")
		return
	}

	// Delete avatar and get old file ID
	oldFileID, err := m.service.DeleteBusinessAvatar(userID.(string))
	if err != nil {
		if err.Error() == "business profile not found" {
			response.NotFound(c, "Business profile not found")
			return
		}
		response.BadRequest(c, fmt.Sprintf("Failed to delete business avatar: %v", err))
		return
	}

	// Delete the file from storage if it exists
	if oldFileID != "" {
		_ = m.storageService.DeleteFile(oldFileID, userID.(string))
	}

	c.Status(http.StatusNoContent)
}

// ============================================================================
// ADMIN BUSINESS HANDLERS
// ============================================================================

// adminGetUserBusiness gets a user's business profile (admin only)
// @Summary View user business profile
// @Description Get business profile for a specific user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} response.Response{data=BusinessResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response "Business profile not found"
// @Router /admin/users/{user_id}/business [get]
func (m *UsersModule) adminGetUserBusiness(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	business, email, phone, err := m.service.GetBusiness(userID)
	if err != nil {
		response.NotFound(c, "Business profile not found")
		return
	}

	// Get base URL for avatar URLs
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}

	businessResponse := m.service.sanitizeBusiness(business, email, phone, baseURL)

	response.Success(c, http.StatusOK, "Business profile retrieved successfully", gin.H{
		"business": businessResponse,
	})
}

// adminCreateUserBusiness creates a business profile for a user (admin only)
// @Summary Create business for user
// @Description Create a new business profile for a specific user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param request body CreateBusinessRequest true "Business details"
// @Success 201 {object} response.Response{data=BusinessResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /admin/users/{user_id}/business [post]
func (m *UsersModule) adminCreateUserBusiness(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	var req CreateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	business, err := m.service.CreateBusiness(userID, &req)
	if err != nil {
		if err.Error() == "user already has a business profile" {
			response.BadRequest(c, "User already has a business profile")
			return
		}
		response.BadRequest(c, fmt.Sprintf("Failed to create business: %v", err))
		return
	}

	// Get base URL for avatar URLs
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}

	businessResponse := m.service.sanitizeBusiness(business, nil, nil, baseURL)

	response.Success(c, http.StatusCreated, "Business profile created successfully", gin.H{
		"business": businessResponse,
	})
}

// adminUpdateUserBusiness updates a user's business profile (admin only)
// @Summary Update user business
// @Description Update business profile for a specific user (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param request body UpdateBusinessRequest true "Business updates"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/users/{user_id}/business [patch]
func (m *UsersModule) adminUpdateUserBusiness(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	var req UpdateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.UpdateBusiness(userID, &req)
	if err != nil {
		if err.Error() == "business profile not found" {
			response.NotFound(c, "Business profile not found")
			return
		}
		response.BadRequest(c, fmt.Sprintf("Failed to update business: %v", err))
		return
	}

	response.Success(c, http.StatusOK, "Business profile updated successfully", nil)
}

// adminUploadUserBusinessAvatar uploads an avatar for a user's business (admin only)
// @Summary Upload user business avatar
// @Description Upload an avatar image for a user's business profile (admin only)
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param avatar formData file true "Avatar image file (JPEG, PNG, GIF, WebP, max 5MB)"
// @Success 200 {object} response.Response{data=object{avatar_url=string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response "Business profile not found"
// @Failure 413 {object} response.Response "File too large"
// @Router /admin/users/{user_id}/business/avatar [post]
func (m *UsersModule) adminUploadUserBusinessAvatar(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.BadRequest(c, "User ID is required")
		return
	}

	// Check if business exists
	_, _, _, err := m.service.GetBusiness(userID)
	if err != nil {
		response.NotFound(c, "Business profile not found. Please create a business profile first.")
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "Avatar file is required")
		return
	}

	// Validate file size (5MB max)
	maxSize := int64(5 * 1024 * 1024) // 5MB
	if file.Size > maxSize {
		response.Error(c, http.StatusRequestEntityTooLarge, "File size exceeds maximum allowed (5MB)", "FILE_TOO_LARGE")
		return
	}

	// Validate file type
	mimeType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedTypes[mimeType] {
		response.BadRequest(c, "Invalid file type. Allowed types: JPEG, PNG, GIF, WebP")
		return
	}

	// Get admin ID for audit trail
	adminID, _ := c.Get("user_id")
	adminIDStr := ""
	if adminID != nil {
		adminIDStr = adminID.(string)
	}

	// Get old avatar for cleanup
	oldAvatarFileID, _ := m.service.DeleteBusinessAvatar(userID)

	// Upload the file using storage service
	uploadReq := &storage.UploadRequest{
		Visibility: "public",
		Metadata:   fmt.Sprintf(`{"type":"business_avatar","uploaded_by_admin":"%s"}`, adminIDStr),
	}
	uploadedFile, err := m.storageService.UploadFile(file, uploadReq, userID)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Failed to upload avatar: %v", err))
		return
	}

	// Update business avatar reference
	err = m.service.UpdateBusinessAvatar(userID, uploadedFile.ID)
	if err != nil {
		// Delete the uploaded file since we couldn't update the reference
		m.storageService.DeleteFile(uploadedFile.ID, userID)
		response.BadRequest(c, "Failed to update business avatar")
		return
	}

	// Delete old avatar file if it exists
	if oldAvatarFileID != "" {
		_ = m.storageService.DeleteFile(oldAvatarFileID, userID)
	}

	// Build avatar URL
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://localhost:8080"
	}
	avatarURL := fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, uploadedFile.ID)

	response.Success(c, http.StatusOK, "Business avatar uploaded successfully", gin.H{
		"avatar_url": avatarURL,
	})
}

// ============================================================================
// ADMIN BUSINESS UTILITIES
// ============================================================================

// adminGetBusinessTypes returns the list of available business types for dropdowns
// @Summary List business types
// @Description Get list of available business types for dropdown selection. Accessible to all authenticated users.
// @Tags Business
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{business_types=[]string}}
// @Failure 401 {object} response.Response
// @Router /admin/business-types [get]
func (m *UsersModule) adminGetBusinessTypes(c *gin.Context) {
	businessTypes := []map[string]string{
		{
			"value": "individual",
			"label": "Individual",
		},
		{
			"value": "business",
			"label": "Business",
		},
		{
			"value": "nonprofit",
			"label": "Non-Profit Organization",
		},
		{
			"value": "government",
			"label": "Government Entity",
		},
		{
			"value": "corporation",
			"label": "Corporation",
		},
		{
			"value": "llc",
			"label": "LLC",
		},
		{
			"value": "sole_proprietor",
			"label": "Sole Proprietor",
		},
	}

	response.Success(c, http.StatusOK, "Business types retrieved successfully", gin.H{
		"business_types": businessTypes,
	})
}
