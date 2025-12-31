package users

import (
	"fmt"
	"net/http"
	"time"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// Pending2FASetup stores temporary 2FA setup data
type Pending2FASetup struct {
	Method      string   `json:"method"`
	Secret      string   `json:"secret,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
	Phone       string   `json:"phone,omitempty"`
}

// Helper functions for Redis storage
func (m *UsersModule) storePending2FA(userID string, data *Pending2FASetup) error {
	key := fmt.Sprintf("pending_2fa:%s", userID)
	return m.redisHelper.CacheSet(key, data, 10*time.Minute)
}

func (m *UsersModule) getPending2FA(userID string) (*Pending2FASetup, error) {
	key := fmt.Sprintf("pending_2fa:%s", userID)
	var setup Pending2FASetup
	if err := m.redisHelper.CacheGet(key, &setup); err != nil {
		return nil, fmt.Errorf("no pending setup")
	}
	return &setup, nil
}

func (m *UsersModule) deletePending2FA(userID string) error {
	key := fmt.Sprintf("pending_2fa:%s", userID)
	return m.redisHelper.CacheDelete(key)
}

// Pending login session helpers
type PendingLoginData struct {
	UserID    string         `json:"user_id"`
	LoginResp *LoginResponse `json:"login_resp"`
}

func (m *UsersModule) storePendingLogin(sessionID, userID string, loginResp *LoginResponse) error {
	key := fmt.Sprintf("pending_login:%s", sessionID)
	data := &PendingLoginData{
		UserID:    userID,
		LoginResp: loginResp,
	}
	return m.redisHelper.CacheSet(key, data, 5*time.Minute)
}

func (m *UsersModule) getPendingLogin(sessionID string) (string, *LoginResponse, error) {
	key := fmt.Sprintf("pending_login:%s", sessionID)
	var data PendingLoginData
	if err := m.redisHelper.CacheGet(key, &data); err != nil {
		return "", nil, fmt.Errorf("invalid or expired session")
	}
	return data.UserID, data.LoginResp, nil
}

func (m *UsersModule) deletePendingLogin(sessionID string) error {
	key := fmt.Sprintf("pending_login:%s", sessionID)
	return m.redisHelper.CacheDelete(key)
}

// setup2FA initiates 2FA setup
// @Summary Setup 2FA
// @Description Initiate 2FA setup with either SMS or Authenticator app. Returns QR code for authenticator or sends SMS OTP.
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body Setup2FARequest true "2FA method selection"
// @Success 200 {object} response.Response{data=Setup2FAResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/2fa/setup [post]
func (m *UsersModule) setup2FA(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req Setup2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Check if 2FA is already enabled
	_, enabled, err := m.twofaService.Get2FAInfo(userID)
	if err != nil {
		response.InternalError(c, "Failed to check 2FA status")
		return
	}
	if enabled {
		response.BadRequest(c, "2FA is already enabled. Disable it first to change method.")
		return
	}

	// Get user info for email/phone
	_, userEmail, userPhone, _, err := m.service.GetUserByID(userID)
	if err != nil {
		response.InternalError(c, "Failed to get user info")
		return
	}

	var setupResp Setup2FAResponse

	if req.Method == "authenticator" {
		// Generate TOTP secret and QR code
		email := ""
		if userEmail != nil {
			email = userEmail.Email
		}

		secret, qrURL, err := m.twofaService.GenerateTOTP(userID, email)
		if err != nil {
			response.InternalError(c, "Failed to generate TOTP")
			return
		}

		// Generate backup codes
		backupCodes, err := m.twofaService.GenerateBackupCodes()
		if err != nil {
			response.InternalError(c, "Failed to generate backup codes")
			return
		}

		// Store setup data in Redis for verification
		if err := m.storePending2FA(userID, &Pending2FASetup{
			Method:      "authenticator",
			Secret:      secret,
			BackupCodes: backupCodes,
		}); err != nil {
			response.InternalError(c, "Failed to store setup data")
			return
		}

		setupResp = Setup2FAResponse{
			Method:      "authenticator",
			Secret:      secret,
			QRCodeURL:   qrURL,
			BackupCodes: backupCodes,
			Message:     "Scan the QR code with your authenticator app and enter the 6-digit code to verify",
		}

	} else if req.Method == "sms" {
		// Verify user has a verified phone
		if userPhone == nil || !userPhone.IsVerified {
			response.BadRequest(c, "You must have a verified phone number to use SMS 2FA")
			return
		}

		// Send SMS OTP
		err := m.twofaService.SendSMSOTP(userID, userPhone.Phone)
		if err != nil {
			response.InternalError(c, "Failed to send SMS OTP")
			return
		}

		// Store setup data in Redis
		if err := m.storePending2FA(userID, &Pending2FASetup{
			Method: "sms",
			Phone:  userPhone.Phone,
		}); err != nil {
			response.InternalError(c, "Failed to store setup data")
			return
		}

		setupResp = Setup2FAResponse{
			Method:  "sms",
			Message: "OTP sent to your phone. Enter the 6-digit code to verify and enable 2FA.",
		}
	}

	response.Success(c, http.StatusOK, "2FA setup initiated", setupResp)
}

// verify2FASetup verifies and enables 2FA
// @Summary Verify and Enable 2FA
// @Description Verify the OTP/TOTP code and enable 2FA for the user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body Verify2FASetupRequest true "Verification code"
// @Success 200 {object} response.Response{data=TwoFAStatusResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/2fa/verify [post]
func (m *UsersModule) verify2FASetup(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req Verify2FASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get pending 2FA setup from Redis
	pendingSetup, err := m.getPending2FA(userID)
	if err != nil {
		response.BadRequest(c, "No pending 2FA setup. Please initiate setup first.")
		return
	}

	var verified bool
	var secret string
	var backupCodes []string

	if pendingSetup.Method == "authenticator" {
		// Verify TOTP code
		secret = pendingSetup.Secret
		verified = m.twofaService.VerifyTOTP(secret, req.Code)
		if !verified {
			response.Unauthorized(c, "Invalid verification code")
			return
		}

		backupCodes = pendingSetup.BackupCodes

	} else if pendingSetup.Method == "sms" {
		// Verify SMS OTP
		verified, err = m.twofaService.VerifySMSOTP(pendingSetup.Phone, req.Code)
		if err != nil || !verified {
			response.Unauthorized(c, "Invalid or expired OTP")
			return
		}
	}

	// Enable 2FA
	err = m.twofaService.Enable2FA(userID, TwoFAMethod(pendingSetup.Method), secret, backupCodes)
	if err != nil {
		response.InternalError(c, "Failed to enable 2FA")
		return
	}

	// Clear pending setup from Redis
	m.deletePending2FA(userID)

	response.Success(c, http.StatusOK, "2FA enabled successfully", TwoFAStatusResponse{
		Enabled: true,
		Method:  pendingSetup.Method,
	})
}

// verify2FALogin verifies 2FA code during login
// @Summary Verify 2FA During Login
// @Description Verify 2FA code to complete login process
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body Verify2FALoginRequest true "Session ID and verification code"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/2fa/login/verify [post]
func (m *UsersModule) verify2FALogin(c *gin.Context) {
	var req Verify2FALoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get pending login session from Redis
	userID, loginResp, err := m.getPendingLogin(req.SessionID)
	if err != nil {
		response.Unauthorized(c, "Invalid or expired session")
		return
	}

	// Verify 2FA code
	verified, err := m.twofaService.Verify2FACode(userID, req.Code)
	if err != nil || !verified {
		response.Unauthorized(c, "Invalid 2FA code")
		return
	}

	// Clear 2FA requirement flags
	loginResp.TwoFARequired = false
	loginResp.TwoFAMethod = ""

	// Clear pending login from Redis
	m.deletePendingLogin(req.SessionID)

	response.Success(c, http.StatusOK, "Login successful", loginResp)
}

// disable2FA disables 2FA for the user
// @Summary Disable 2FA
// @Description Disable two-factor authentication for the current user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=TwoFAStatusResponse}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/2fa/disable [post]
func (m *UsersModule) disable2FA(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := m.twofaService.Disable2FA(userID)
	if err != nil {
		response.InternalError(c, "Failed to disable 2FA")
		return
	}

	response.Success(c, http.StatusOK, "2FA disabled successfully", TwoFAStatusResponse{
		Enabled: false,
	})
}

// get2FAStatus returns the current 2FA status
// @Summary Get 2FA Status
// @Description Get the current 2FA configuration status
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=TwoFAStatusResponse}
// @Failure 401 {object} response.Response
// @Router /auth/2fa/status [get]
func (m *UsersModule) get2FAStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	metadata, enabled, err := m.twofaService.Get2FAInfo(userID)
	if err != nil {
		response.InternalError(c, "Failed to get 2FA status")
		return
	}

	response.Success(c, http.StatusOK, "2FA status retrieved", TwoFAStatusResponse{
		Enabled: enabled,
		Method:  metadata.Method,
	})
}

// getBackupCodes retrieves the user's 2FA backup codes
// @Summary Get 2FA backup codes
// @Description Get the current user's 2FA backup codes. Only available for authenticator-based 2FA.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{backup_codes=[]string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/2fa/backup-codes [get]
func (m *UsersModule) getBackupCodes(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	backupCodes, err := m.twofaService.GetBackupCodes(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Backup codes retrieved successfully", gin.H{
		"backup_codes": backupCodes,
	})
}

// regenerateBackupCodes generates new backup codes
// @Summary Regenerate 2FA backup codes
// @Description Generate new backup codes and invalidate the old ones. Only available for authenticator-based 2FA.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{backup_codes=[]string}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/2fa/backup-codes/regenerate [post]
func (m *UsersModule) regenerateBackupCodes(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	newBackupCodes, err := m.twofaService.RegenerateBackupCodes(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Backup codes regenerated successfully", gin.H{
		"backup_codes": newBackupCodes,
	})
}
