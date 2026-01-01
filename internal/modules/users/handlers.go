package users

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gogin/internal/modules/storage"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration details"
// @Success 201 {object} response.Response{data=object{user=UserResponse}}
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 400 {object} response.Response
// @Router /users/register [post]
func (m *UsersModule) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	user, userEmail, roleName, err := m.service.CreateUser(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// login handles user login
// @Summary User login
// @Description Authenticate user with email or phone number and password. Creates a new session for multi-device tracking.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials (provide either email or phone with password)"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 401 {object} response.Response
// @Router /users/login [post]
func (m *UsersModule) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	// Validate that either email or phone is provided
	if req.Email == "" && req.Phone == "" {
		response.BadRequest(c, "Either email or phone is required")
		return
	}

	// Extract device info for session tracking
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Authenticate user (supports both email and phone)
	var loginResp *LoginResponse
	var err error

	if req.Phone != "" {
		loginResp, err = m.service.AuthenticateUserByPhone(req.Phone, req.Password)
	} else {
		loginResp, err = m.service.AuthenticateUser(req.Email, req.Password)
	}

	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// Check if 2FA is enabled
	metadata, has2FA, err := m.twofaService.Get2FAInfo(loginResp.User.ID)
	if err != nil {
		// Log error but continue - 2FA check is not critical for basic auth
		c.Error(fmt.Errorf("failed to check 2FA: %w", err))
	}

	if has2FA && metadata != nil && metadata.Method != "" {
		// 2FA is enabled - require verification before returning tokens
		sessionID := generateSessionID()

		// Store pending login data in Redis
		if err := m.storePendingLogin(sessionID, loginResp.User.ID, loginResp); err != nil {
			c.Error(fmt.Errorf("failed to store pending login: %w", err))
		}

		// Send OTP if using SMS method
		if metadata.Method == "sms" {
			user, _, userPhone, _, _ := m.service.GetUserByID(loginResp.User.ID)
			if userPhone != nil {
				m.twofaService.SendSMSOTP(user.ID, userPhone.Phone)
			}
		}

		// Return 2FA required response
		response.Success(c, http.StatusOK, "2FA verification required", &LoginResponse{
			SessionID:     sessionID,
			TwoFARequired: true,
			TwoFAMethod:   metadata.Method,
		})
		return
	}

	// Create session for multi-device tracking
	session, err := m.sessionService.CreateSession(
		loginResp.User.ID,
		ipAddress,
		userAgent,
		"password", // login method type
	)
	if err != nil {
		// Log error but don't fail login
		// Session tracking is not critical for authentication
		c.Error(fmt.Errorf("failed to create session: %w", err))
	}

	// Add session ID to response metadata (optional)
	if session != nil {
		loginResp.SessionID = session.SessionID
	}

	response.Success(c, http.StatusOK, "Login successful", loginResp)
}

// refreshToken handles token refresh requests
// @Summary Refresh access token
// @Description Refresh an expired access token using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} response.Response{data=RefreshTokenResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /auth/refresh [post]
func (m *UsersModule) refreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	// Validate refresh token and issue new access token
	tokenResp, err := m.service.RefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Token refreshed successfully", tokenResp)
}

// getProfile retrieves the current user's profile
// @Summary Get user profile
// @Description Get the authenticated user's profile information
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{user=UserResponse}}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/me [get]
func (m *UsersModule) getProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, userEmail, _, roleName, err := m.service.GetUserByID(userID.(string))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, http.StatusOK, "Profile retrieved successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// updateProfile updates the current user's profile
// @Summary Update user profile
// @Description Update the authenticated user's profile information
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "Profile update details"
// @Success 200 {object} response.Response{data=object{user=UserResponse}}
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 400 {object} response.Response
// @Router /users/me [put]
func (m *UsersModule) updateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	user, userEmail, _, roleName, err := m.service.UpdateUser(userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// changePassword changes the current user's password
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change details"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 400 {object} response.Response
// @Router /users/me/password [put]
func (m *UsersModule) changePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.ChangePassword(userID.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Password changed successfully", nil)
}

// forgotPassword handles forgot password request
// @Summary Request password reset
// @Description Send password reset email to user's email address
// @Tags Users
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /forgot-password [post]
func (m *UsersModule) forgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.ForgotPassword(req.Email)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Always return success even if email doesn't exist (security best practice)
	response.Success(c, http.StatusOK, "If the email exists, a password reset link has been sent", nil)
}

// resetPassword handles password reset with token
// @Summary Reset password
// @Description Reset user password using reset token from email
// @Tags Users
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /reset-password [post]
func (m *UsersModule) resetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Password reset successfully. Please login with your new password", nil)
}

// uploadAvatar handles user avatar upload
// @Summary Upload user avatar
// @Description Upload or update the authenticated user's avatar image
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Avatar image file (JPEG, PNG, GIF, max 5MB)"
// @Success 200 {object} response.Response{data=object{avatar_url=string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 413 {object} response.Response
// @Router /users/me/avatar [post]
func (m *UsersModule) uploadAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "No file provided")
		return
	}

	// Validate file type (images only)
	mimeType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	if !allowedTypes[mimeType] {
		response.BadRequest(c, "Invalid file type. Only JPEG, PNG, GIF, and WebP images are allowed")
		return
	}

	// Validate file size (max 5MB for avatars)
	maxSize := int64(5 * 1024 * 1024) // 5MB
	if file.Size > maxSize {
		response.Error(c, http.StatusRequestEntityTooLarge, "File too large", "Avatar must be less than 5MB")
		return
	}

	// Upload file using storage service with public visibility (avatars are public)
	req := &storage.UploadRequest{
		Visibility: "public",
		Metadata:   `{"type":"avatar"}`,
	}

	uploadedFile, err := m.storageService.UploadFile(file, req, userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Generate download URL
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://" + c.Request.Host
	}
	avatarURL := fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, uploadedFile.ID)

	// Get old avatar file ID to delete old file
	oldAvatarFileID, _ := m.service.DeleteAvatar(userID.(string))

	// Update user's avatar file ID
	err = m.service.UpdateAvatar(userID.(string), uploadedFile.ID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Delete old avatar file if exists
	if oldAvatarFileID != "" {
		m.storageService.DeleteFile(oldAvatarFileID, userID.(string))
	}

	response.Success(c, http.StatusOK, "Avatar uploaded successfully", gin.H{
		"avatar_url": avatarURL,
		"file_id":    uploadedFile.ID,
	})
}

// deleteAvatar handles user avatar deletion
// @Summary Delete user avatar
// @Description Remove the authenticated user's avatar image
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /users/me/avatar [delete]
func (m *UsersModule) deleteAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get current avatar file ID and remove it from user record
	oldAvatarFileID, err := m.service.DeleteAvatar(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Delete the file from storage if exists
	if oldAvatarFileID != "" {
		// Best-effort deletion, don't fail if file doesn't exist
		m.storageService.DeleteFile(oldAvatarFileID, userID.(string))
	}

	response.Success(c, http.StatusOK, "Avatar deleted successfully", nil)
}

// listUserEmails retrieves all emails for the authenticated user
// @Summary List user emails
// @Description Get all email addresses associated with the authenticated user's account
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UserEmailsListResponse}
// @Failure 401 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /users/me/emails [get]
func (m *UsersModule) listUserEmails(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	emails, err := m.service.ListUserEmails(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Convert to DTO
	emailResponses := make([]*UserEmailResponse, len(emails))
	for i, email := range emails {
		emailResp := &UserEmailResponse{
			ID:         email.ID,
			Email:      email.Email,
			IsPrimary:  email.IsPrimary,
			IsRecovery: email.IsRecovery,
			IsVerified: email.IsVerified,
			CreatedAt:  email.CreatedAt,
			UpdatedAt:  email.UpdatedAt,
		}
		if email.VerifiedAt.Valid {
			emailResp.VerifiedAt = email.VerifiedAt.Time
		}
		emailResponses[i] = emailResp
	}

	response.Success(c, http.StatusOK, "Emails retrieved successfully", UserEmailsListResponse{
		Emails: emailResponses,
		Total:  len(emailResponses),
	})
}

// addUserEmail adds a new email to the user's account
// @Summary Add user email
// @Description Add a new email address to the authenticated user's account
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddEmailRequest true "Email details"
// @Success 201 {object} response.Response{data=object{email=UserEmailResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /users/me/emails [post]
func (m *UsersModule) addUserEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req AddEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	email, err := m.service.AddUserEmail(userID.(string), req.Email)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	emailResp := &UserEmailResponse{
		ID:         email.ID,
		Email:      email.Email,
		IsPrimary:  email.IsPrimary,
		IsRecovery: email.IsRecovery,
		IsVerified: email.IsVerified,
		CreatedAt:  email.CreatedAt,
		UpdatedAt:  email.UpdatedAt,
	}
	if email.VerifiedAt.Valid {
		emailResp.VerifiedAt = email.VerifiedAt.Time
	}

	response.Success(c, http.StatusCreated, "Email added successfully. Please check your inbox to verify", gin.H{
		"email": emailResp,
	})
}

// setPrimaryEmail sets an email as the primary email
// @Summary Set primary email
// @Description Set a verified email as the primary email for the authenticated user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param email_id path string true "Email ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/emails/{email_id}/primary [put]
func (m *UsersModule) setPrimaryEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	emailID := c.Param("email_id")
	if emailID == "" {
		response.BadRequest(c, "Email ID is required")
		return
	}

	err := m.service.SetPrimaryEmail(userID.(string), emailID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Primary email updated successfully", nil)
}

// deleteUserEmail removes an email from the user's account
// @Summary Delete user email
// @Description Remove an email address from the authenticated user's account
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param email_id path string true "Email ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/emails/{email_id} [delete]
func (m *UsersModule) deleteUserEmail(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	emailID := c.Param("email_id")
	if emailID == "" {
		response.BadRequest(c, "Email ID is required")
		return
	}

	err := m.service.DeleteUserEmail(userID.(string), emailID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Email deleted successfully", nil)
}

// resendEmailVerification resends verification email
// @Summary Resend email verification
// @Description Resend verification email for an unverified email address
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param email_id path string true "Email ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/emails/{email_id}/verify [post]
func (m *UsersModule) resendEmailVerification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	emailID := c.Param("email_id")
	if emailID == "" {
		response.BadRequest(c, "Email ID is required")
		return
	}

	err := m.service.ResendEmailVerification(userID.(string), emailID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Verification email sent successfully", nil)
}

// verifyEmailWithToken handles email verification via token from email link
// @Summary Verify email address
// @Description Verify an email address using the verification token sent to the email
// @Tags Users
// @Produce json
// @Param token query string true "Verification Token"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /verify-email [get]
func (m *UsersModule) verifyEmailWithToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.BadRequest(c, "Verification token is required")
		return
	}

	email, err := m.service.VerifyEmailWithToken(token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Email verified successfully", gin.H{
		"email": email.Email,
		"verified_at": email.VerifiedAt,
	})
}

// listUserPhones retrieves all phones for the authenticated user
// @Summary List user phones
// @Description Get all phone numbers associated with the authenticated user's account
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UserPhonesListResponse}
// @Failure 401 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /users/me/phones [get]
func (m *UsersModule) listUserPhones(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	phones, err := m.service.ListUserPhones(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Convert to DTO
	phoneResponses := make([]*UserPhoneResponse, len(phones))
	for i, phone := range phones {
		phoneResp := &UserPhoneResponse{
			ID:         phone.ID,
			Phone:      phone.Phone,
			IsPrimary:  phone.IsPrimary,
			IsRecovery: phone.IsRecovery,
			IsVerified: phone.IsVerified,
			CreatedAt:  phone.CreatedAt,
			UpdatedAt:  phone.UpdatedAt,
		}
		if phone.PhoneE164.Valid {
			phoneResp.PhoneE164 = phone.PhoneE164.String
		}
		if phone.VerifiedAt.Valid {
			phoneResp.VerifiedAt = phone.VerifiedAt.Time
		}
		phoneResponses[i] = phoneResp
	}

	response.Success(c, http.StatusOK, "Phones retrieved successfully", UserPhonesListResponse{
		Phones: phoneResponses,
		Total:  len(phoneResponses),
	})
}

// addUserPhone adds a new phone to the user's account
// @Summary Add user phone
// @Description Add a new phone number to the authenticated user's account
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddPhoneRequest true "Phone details"
// @Success 201 {object} response.Response{data=object{phone=UserPhoneResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /users/me/phones [post]
func (m *UsersModule) addUserPhone(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req AddPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	phone, err := m.service.AddUserPhone(userID.(string), req.Phone)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	phoneResp := &UserPhoneResponse{
		ID:         phone.ID,
		Phone:      phone.Phone,
		IsPrimary:  phone.IsPrimary,
		IsRecovery: phone.IsRecovery,
		IsVerified: phone.IsVerified,
		CreatedAt:  phone.CreatedAt,
		UpdatedAt:  phone.UpdatedAt,
	}
	if phone.PhoneE164.Valid {
		phoneResp.PhoneE164 = phone.PhoneE164.String
	}
	if phone.VerifiedAt.Valid {
		phoneResp.VerifiedAt = phone.VerifiedAt.Time
	}

	response.Success(c, http.StatusCreated, "Phone added successfully. Please check your SMS to verify", gin.H{
		"phone": phoneResp,
	})
}

// setPrimaryPhone sets a phone as the primary phone
// @Summary Set primary phone
// @Description Set a verified phone as the primary phone for the authenticated user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param phone_id path string true "Phone ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/phones/{phone_id}/primary [put]
func (m *UsersModule) setPrimaryPhone(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	phoneID := c.Param("phone_id")
	if phoneID == "" {
		response.BadRequest(c, "Phone ID is required")
		return
	}

	err := m.service.SetPrimaryPhone(userID.(string), phoneID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Primary phone updated successfully", nil)
}

// deleteUserPhone removes a phone from the user's account
// @Summary Delete user phone
// @Description Remove a phone number from the authenticated user's account
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param phone_id path string true "Phone ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/phones/{phone_id} [delete]
func (m *UsersModule) deleteUserPhone(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	phoneID := c.Param("phone_id")
	if phoneID == "" {
		response.BadRequest(c, "Phone ID is required")
		return
	}

	err := m.service.DeleteUserPhone(userID.(string), phoneID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Phone deleted successfully", nil)
}

// resendPhoneVerification resends verification SMS
// @Summary Resend phone verification
// @Description Resend verification SMS for an unverified phone number
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param phone_id path string true "Phone ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/me/phones/{phone_id}/verify [post]
func (m *UsersModule) resendPhoneVerification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	phoneID := c.Param("phone_id")
	if phoneID == "" {
		response.BadRequest(c, "Phone ID is required")
		return
	}

	err := m.service.ResendPhoneVerification(userID.(string), phoneID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Verification SMS sent successfully", nil)
}

// logout handles user logout
// @Summary User logout
// @Description Logout the authenticated user and invalidate their session
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/logout [post]
func (m *UsersModule) logout(c *gin.Context) {
	tokenIDInterface, exists := c.Get("token_id")
	if !exists {
		response.Unauthorized(c, "Invalid token")
		return
	}

	userID, _ := c.Get("user_id")
	tokenID := tokenIDInterface.(string)

	// Extract token from header to get expiry time
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			tokenString := parts[1]
			claims, err := m.service.jwtUtil.ValidateToken(tokenString)
			if err == nil && claims.ExpiresAt != nil {
				// Revoke the access token
				m.service.redisHelper.RevokeToken(tokenID, claims.ExpiresAt.Time)
			}
		}
	}

	// Terminate sessions
	if userID != nil {
		m.service.redisHelper.DeleteAllUserSessions(userID.(string))
	}

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// deleteAccount handles account deletion
// @Summary Delete account
// @Description Delete the authenticated user's account
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /users/me [delete]
func (m *UsersModule) deleteAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := m.service.DeleteUser(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Account deleted successfully", nil)
}

// Admin handlers

// listUsers lists all users (admin only)
// @Summary List all users
// @Description Get a paginated list of all users (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=object{users=[]UserResponse,total=int,page=int,limit=int,total_pages=int}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users [get]
func (m *UsersModule) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	users, total, err := m.service.ListUsers(page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list users")
		return
	}

	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "Users retrieved successfully", gin.H{
		"users":       users,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// getUserByID retrieves a user by ID (admin only)
// @Summary Get user by ID
// @Description Get a specific user by their ID (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response{data=object{user=UserResponse}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/{id} [get]
func (m *UsersModule) getUserByID(c *gin.Context) {
	userID := c.Param("id")

	user, userEmail, _, roleName, err := m.service.GetUserByID(userID)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// updateUser updates a user (admin only)
// @Summary Update user
// @Description Update a user's profile information (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body UpdateProfileRequest true "Profile update details"
// @Success 200 {object} response.Response{data=object{user=UserResponse}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 400 {object} response.Response
// @Router /users/{id} [put]
func (m *UsersModule) updateUser(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	user, userEmail, _, roleName, err := m.service.UpdateUser(userID, &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User updated successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// adminDeleteUser deletes a user (admin only)
// @Summary Delete user
// @Description Delete a user account (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /users/{id} [delete]
func (m *UsersModule) adminDeleteUser(c *gin.Context) {
	userID := c.Param("id")

	err := m.service.DeleteUser(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User deleted successfully", nil)
}

// updateUserStatus updates a user's status (admin only)
// @Summary Update user status
// @Description Update a user's status (active, inactive, or suspended) (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body object{status=string} true "Status update details (status must be: active, inactive, or suspended)"
// @Success 200 {object} response.Response{data=object{status=string}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id}/status [put]
func (m *UsersModule) updateUserStatus(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required,oneof=active inactive suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	// Update user status in database (map old status to is_active flag)
	isActive := req.Status == "active"
	query := `UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
	result, err := m.service.db.Exec(query, isActive, userID)
	if err != nil {
		response.InternalError(c, "Failed to update user status")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(c, "User not found")
		return
	}

	// If not active, revoke all sessions
	if req.Status == "suspended" || req.Status == "inactive" {
		m.service.redisHelper.DeleteAllUserSessions(userID)
	}

	response.Success(c, http.StatusOK, "User status updated successfully", gin.H{
		"status": req.Status,
	})
}

// adminCreateUser creates a new user (admin only)
// @Summary Create user (Admin)
// @Description Create a new user with specified role and status (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AdminCreateUserRequest true "User creation details"
// @Success 201 {object} response.Response{data=object{user=UserResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /users [post]
func (m *UsersModule) adminCreateUser(c *gin.Context) {
	var req AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	user, userEmail, roleName, err := m.service.AdminCreateUser(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "User created successfully", gin.H{
		"user": m.service.sanitizeUser(user, userEmail, roleName),
	})
}

// adminGetUserSessions lists all sessions for a specific user (admin only)
// @Summary Get user sessions (Admin)
// @Description Get all active sessions for a specific user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response{data=object{sessions=[]object}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /users/{id}/sessions [get]
func (m *UsersModule) adminGetUserSessions(c *gin.Context) {
	userID := c.Param("id")
	sessions, err := m.sessionService.GetUserSessions(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Sessions retrieved successfully", gin.H{"sessions": sessions})
}

// adminTerminateUserSession terminates a specific session for a user (admin only)
// @Summary Terminate user session (Admin)
// @Description Terminate a specific session for a user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param session_id path string true "Session ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /users/{id}/sessions/{session_id} [delete]
func (m *UsersModule) adminTerminateUserSession(c *gin.Context) {
	userID := c.Param("id")
	sessionID := c.Param("session_id")
	err := m.sessionService.TerminateSession(sessionID, userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Session terminated successfully", nil)
}

// adminTerminateAllUserSessions terminates all sessions for a user (admin only)
// @Summary Terminate all user sessions (Admin)
// @Description Terminate all active sessions for a user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /users/{id}/sessions [delete]
func (m *UsersModule) adminTerminateAllUserSessions(c *gin.Context) {
	userID := c.Param("id")
	err := m.sessionService.LogoutAllDevices(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "All sessions terminated successfully", nil)
}

// adminGetUserSettings gets user settings (admin only)
// @Summary Get user settings (Admin)
// @Description Get user preference settings (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} response.Response{data=object}
// @Router /users/{id}/settings [get]
func (m *UsersModule) adminGetUserSettings(c *gin.Context) {
	userID := c.Param("id")
	var settings struct {
		Timezone   string `json:"timezone"`
		Locale     string `json:"locale"`
		DateFormat string `json:"date_format"`
		TimeFormat string `json:"time_format"`
	}
	query := `SELECT timezone, locale, date_format, time_format FROM users WHERE id = $1`
	err := m.service.db.QueryRow(query, userID).Scan(&settings.Timezone, &settings.Locale, &settings.DateFormat, &settings.TimeFormat)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}
	response.Success(c, http.StatusOK, "Settings retrieved", settings)
}

// adminUpdateUserSettings updates user settings (admin only)
// @Summary Update user settings (Admin)
// @Description Update user preference settings (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param request body object{timezone=string,locale=string,date_format=string,time_format=string} true "Settings"
// @Success 200 {object} response.Response
// @Router /users/{id}/settings [put]
func (m *UsersModule) adminUpdateUserSettings(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Timezone   string `json:"timezone"`
		Locale     string `json:"locale"`
		DateFormat string `json:"date_format"`
		TimeFormat string `json:"time_format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	query := `UPDATE users SET timezone=$1, locale=$2, date_format=$3, time_format=$4, updated_at=NOW() WHERE id=$5`
	_, err := m.service.db.Exec(query, req.Timezone, req.Locale, req.DateFormat, req.TimeFormat, userID)
	if err != nil {
		response.BadRequest(c, "Failed to update settings")
		return
	}
	response.Success(c, http.StatusOK, "Settings updated", nil)
}

// ========== New Admin Endpoints ==========

// getAdminProfile gets the admin's own profile
// @Summary Get admin profile
// @Description Get the authenticated admin's profile information
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/me [get]
func (m *UsersModule) getAdminProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	user, userEmail, _, roleName, err := m.service.GetUserByID(userID.(string))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, http.StatusOK, "Admin profile retrieved", m.service.sanitizeUser(user, userEmail, roleName))
}

// getAdminSessions lists all sessions for the authenticated admin
// @Summary List admin sessions
// @Description Get all active sessions for the authenticated admin
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]SessionResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/me/sessions [get]
func (m *UsersModule) getAdminSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	sessions, err := m.sessionService.GetUserSessions(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Sessions retrieved", gin.H{"sessions": sessions})
}

// deleteAdminSession terminates a specific admin session
// @Summary Logout specific admin session
// @Description Terminate a specific session for the authenticated admin
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/me/sessions/{session_id} [delete]
func (m *UsersModule) deleteAdminSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	sessionID := c.Param("session_id")
	err := m.sessionService.TerminateSession(sessionID, userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Session terminated successfully", nil)
}

// deleteAllAdminSessions terminates all admin sessions
// @Summary Logout all admin sessions
// @Description Terminate all active sessions for the authenticated admin
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/me/sessions [delete]
func (m *UsersModule) deleteAllAdminSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	err := m.sessionService.LogoutAllDevices(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "All sessions terminated successfully", nil)
}

// adminUploadUserAvatar uploads avatar for a user (admin only)
// @Summary Set user avatar (Admin)
// @Description Upload and set avatar for a specific user (admin only)
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} response.Response{data=object{avatar_url=string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/avatar [post]
func (m *UsersModule) adminUploadUserAvatar(c *gin.Context) {
	userID := c.Param("id")

	// Check if user exists
	_, _, _, _, err := m.service.GetUserByID(userID)
	if err != nil {
		response.NotFound(c, "User not found")
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

	// Get old avatar file ID for cleanup
	oldAvatarFileID, _ := m.service.DeleteAvatar(userID)

	// Upload the file using storage service
	uploadReq := &storage.UploadRequest{
		Visibility: "public",
		Metadata:   `{"type":"avatar"}`,
	}
	uploadedFile, err := m.storageService.UploadFile(file, uploadReq, userID)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Failed to upload avatar file: %v", err))
		return
	}

	// Update user's avatar file reference
	err = m.service.UpdateAvatar(userID, uploadedFile.ID)
	if err != nil {
		// Delete the uploaded file since we couldn't update the reference
		m.storageService.DeleteFile(uploadedFile.ID, userID)
		response.BadRequest(c, fmt.Sprintf("Failed to link avatar to user profile: %v", err))
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

	response.Success(c, http.StatusOK, "Avatar uploaded and replaced successfully", gin.H{
		"avatar_url": avatarURL,
	})
}

// adminGetUserEmails gets all emails for a user (admin only)
// @Summary View user emails (Admin)
// @Description Get all email addresses associated with a user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response{data=[]object}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/emails [get]
func (m *UsersModule) adminGetUserEmails(c *gin.Context) {
	userID := c.Param("id")

	emails, err := m.service.ListUserEmails(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User emails retrieved", gin.H{"emails": emails})
}

// adminAddUserEmail adds a new email to a user (admin only)
// @Summary Add email to user (Admin)
// @Description Add a new email address to a user's account (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body AddEmailRequest true "Email details"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/emails [post]
func (m *UsersModule) adminAddUserEmail(c *gin.Context) {
	userID := c.Param("id")

	var req AddEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	email, err := m.service.AddUserEmail(userID, req.Email)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// If IsPrimary is true, set this email as primary
	if req.IsPrimary {
		setPrimaryErr := m.service.SetPrimaryEmail(userID, email.ID)
		if setPrimaryErr != nil {
			// Email was added but couldn't be set as primary - not critical
			response.Success(c, http.StatusCreated, "Email added but couldn't be set as primary: "+setPrimaryErr.Error(), nil)
			return
		}
	}

	response.Success(c, http.StatusCreated, "Email added successfully", nil)
}

// adminGetUserPhones gets all phones for a user (admin only)
// @Summary View user phones (Admin)
// @Description Get all phone numbers associated with a user (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response{data=[]object}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/phones [get]
func (m *UsersModule) adminGetUserPhones(c *gin.Context) {
	userID := c.Param("id")

	phones, err := m.service.ListUserPhones(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User phones retrieved", gin.H{"phones": phones})
}

// adminAddUserPhone adds a new phone to a user (admin only)
// @Summary Add phone to user (Admin)
// @Description Add a new phone number to a user's account (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body AddPhoneRequest true "Phone details"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/phones [post]
func (m *UsersModule) adminAddUserPhone(c *gin.Context) {
	userID := c.Param("id")

	var req AddPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	phone, err := m.service.AddUserPhone(userID, req.Phone)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// If IsPrimary is true, set this phone as primary
	if req.IsPrimary {
		setPrimaryErr := m.service.SetPrimaryPhone(userID, phone.ID)
		if setPrimaryErr != nil {
			// Phone was added but couldn't be set as primary - not critical
			response.Success(c, http.StatusCreated, "Phone added but couldn't be set as primary: "+setPrimaryErr.Error(), nil)
			return
		}
	}

	response.Success(c, http.StatusCreated, "Phone added successfully", nil)
}
