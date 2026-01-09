package users

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"gogin/internal/models"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// PhoneOTPRequest for sending OTP
type PhoneOTPRequest struct {
	Phone string `json:"phone" binding:"required,min=10,max=20"` // E.164 format recommended
}

// VerifyOTPRequest for OTP verification
type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

// PhoneOTPResponse after sending OTP
type PhoneOTPResponse struct {
	Phone     string `json:"phone"`
	ExpiresAt string `json:"expires_at"`
	Message   string `json:"message"`
}

// sendPhoneOTP sends OTP to phone number
// @Summary Send OTP to phone
// @Description Send 6-digit OTP to phone number for login/registration. OTP valid for 5 minutes.
// @Tags Phone Auth
// @Accept json
// @Produce json
// @Param request body PhoneOTPRequest true "Phone number"
// @Success 200 {object} response.Response{data=PhoneOTPResponse}
// @Failure 400 {object} response.Response
// @Failure 429 {object} response.Response
// @Router /auth/phone/send-otp [post]
func (m *UsersModule) sendPhoneOTP(c *gin.Context) {
	var req PhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), "phone"),
		})
		return
	}

	// Check for existing active challenge (rate limiting)
	existing, _ := m.service.GetActiveChallenge(req.Phone)
	if existing != nil {
		// If challenge was created less than 1 minute ago, reject
		if time.Since(existing.CreatedAt) < time.Minute {
			response.TooManyRequests(c, "Please wait before requesting another OTP")
			return
		}
	}

	// Generate 6-digit OTP
	otp := generateOTP()

	// Hash OTP for storage
	otpHash := hashOTP(otp)

	// Store challenge in database (5 minutes expiry)
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	err := m.service.CreateLoginChallenge(req.Phone, otpHash, expiresAt, c.ClientIP())
	if err != nil {
		response.InternalError(c, "Failed to create OTP challenge")
		return
	}

	// SMS sending has been removed (Twilio wrapper deleted)
	// In development mode, always log OTP for testing
	if m.service.config.IsDevelopment() {
		fmt.Printf("📱 OTP for %s: %s (expires at %s)\n", req.Phone, otp, expiresAt.Format(time.RFC3339))
		fmt.Printf("   Note: SMS sending disabled - external SMS service required for production\n")
		response.Success(c, http.StatusOK, "OTP sent successfully", PhoneOTPResponse{
			Phone:     req.Phone,
			ExpiresAt: expiresAt.Format(time.RFC3339),
			Message:   fmt.Sprintf("OTP sent to %s. Valid for 5 minutes. (DEV: Check console logs)", req.Phone),
		})
		return
	}

	// Production mode - SMS not configured
	fmt.Printf("⚠️  SMS sending not configured for %s - OTP: %s\n", req.Phone, otp)
	if m.service.config.IsProduction() {
		response.InternalError(c, "SMS service not configured. Please contact administrator.")
		return
	}

	response.Success(c, http.StatusOK, "OTP sent successfully", PhoneOTPResponse{
		Phone:     req.Phone,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		Message:   fmt.Sprintf("OTP sent to %s. Valid for 5 minutes.", req.Phone),
	})
}

// verifyPhoneOTP verifies OTP and logs in user
// @Summary Verify OTP and login
// @Description Verify OTP and create session. Creates new user if phone not registered.
// @Tags Phone Auth
// @Accept json
// @Produce json
// @Param request body VerifyOTPRequest true "Phone and OTP"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/phone/verify-otp [post]
func (m *UsersModule) verifyPhoneOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		})
		return
	}

	// Get active challenge
	challenge, err := m.service.GetActiveChallenge(req.Phone)
	if err != nil {
		response.Unauthorized(c, "Invalid or expired OTP")
		return
	}

	// Check if expired
	if time.Now().UTC().After(challenge.ExpiresAt) {
		response.Unauthorized(c, "OTP has expired")
		return
	}

	// Check if locked due to too many attempts
	if challenge.LockedUntil.Valid && time.Now().Before(challenge.LockedUntil.Time) {
		response.Unauthorized(c, "Too many failed attempts. Please request a new OTP")
		return
	}

	// Check attempts limit
	if challenge.Attempts >= challenge.MaxAttempts {
		m.service.LockChallenge(challenge.ID)
		response.Unauthorized(c, "Too many failed attempts. Please request a new OTP")
		return
	}

	// Verify OTP
	otpHash := hashOTP(req.OTP)
	if challenge.CodeHash != otpHash {
		m.service.IncrementChallengeAttempts(challenge.ID)
		response.Unauthorized(c, fmt.Sprintf("Invalid OTP. %d attempts remaining", challenge.MaxAttempts-challenge.Attempts-1))
		return
	}

	// Mark challenge as verified
	m.service.MarkChallengeVerified(challenge.ID)

	// Find or create user by phone
	user, userPhone, err := m.service.FindOrCreateUserByPhone(req.Phone)
	if err != nil {
		response.InternalError(c, "Failed to process user")
		return
	}

	// Generate tokens
	accessToken, _, err := m.service.jwtUtil.GenerateAccessToken(
		user.ID,
		"phone",
		"user",
		[]string{"read", "write"},
		m.service.config.OAuth.AccessTokenExpiry,
	)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	refreshToken, refreshTokenID, err := m.service.jwtUtil.GenerateRefreshToken(
		user.ID,
		"phone",
		m.service.config.OAuth.RefreshTokenExpiry,
	)
	if err != nil {
		response.InternalError(c, "Failed to generate refresh token")
		return
	}

	// Store refresh token
	m.service.storeRefreshToken(user.ID, refreshTokenID, m.service.config.OAuth.RefreshTokenExpiry)

	// Create session
	session, _ := m.sessionService.CreateSession(
		user.ID,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		"phone_otp",
	)

	// Build response
	loginResp := &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(m.service.config.OAuth.AccessTokenExpiry.Seconds()),
		SessionID:    "",
		User: &UserResponse{
			ID:            user.ID,
			Email:         "",
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			Phone:         userPhone.Phone,
			Role:          "user",
			Status:        "active",
			EmailVerified: false,
			PhoneVerified: userPhone.IsVerified,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
		},
	}

	if session != nil {
		loginResp.SessionID = session.SessionID
	}

	response.Success(c, http.StatusOK, "Phone verification successful", loginResp)
}

// generateOTP generates a 6-digit OTP
func generateOTP() string {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n.Int64())
}

// hashOTP hashes OTP using SHA256
func hashOTP(otp string) string {
	hash := sha256.Sum256([]byte(otp))
	return fmt.Sprintf("%x", hash)
}

// CreateLoginChallenge creates OTP challenge in database
func (s *UserService) CreateLoginChallenge(phone, codeHash string, expiresAt time.Time, ipAddress string) error {
	// Invalidate any existing active challenges for this phone
	s.db.DB.Exec(`UPDATE login_challenges SET verified_at = NOW() WHERE destination = $1 AND verified_at IS NULL`, phone)

	query := `
		INSERT INTO login_challenges (
			destination, challenge_type, code_hash,
			created_at, expires_at, attempts, max_attempts, ip_address
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.DB.Exec(query, phone, "phone_otp", codeHash, time.Now().UTC(), expiresAt, 0, 5, ipAddress)
	return err
}

// GetActiveChallenge retrieves active OTP challenge
func (s *UserService) GetActiveChallenge(phone string) (*models.LoginChallenge, error) {
	query := `
		SELECT id, destination, challenge_type, code_hash, created_at, expires_at,
		       verified_at, attempts, max_attempts, locked_until, ip_address
		FROM login_challenges
		WHERE destination = $1
		  AND challenge_type = 'phone_otp'
		  AND verified_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var challenge models.LoginChallenge
	err := s.db.DB.QueryRow(query, phone).Scan(
		&challenge.ID, &challenge.Destination, &challenge.ChallengeType,
		&challenge.CodeHash, &challenge.CreatedAt, &challenge.ExpiresAt,
		&challenge.VerifiedAt, &challenge.Attempts, &challenge.MaxAttempts,
		&challenge.LockedUntil, &challenge.IPAddress,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active challenge found")
	}
	if err != nil {
		return nil, err
	}

	return &challenge, nil
}

// IncrementChallengeAttempts increments failed attempts
func (s *UserService) IncrementChallengeAttempts(challengeID string) {
	s.db.DB.Exec(`UPDATE login_challenges SET attempts = attempts + 1 WHERE id = $1`, challengeID)
}

// LockChallenge locks challenge after too many attempts
func (s *UserService) LockChallenge(challengeID string) {
	lockUntil := time.Now().UTC().Add(15 * time.Minute)
	s.db.DB.Exec(`UPDATE login_challenges SET locked_until = $1 WHERE id = $2`, lockUntil, challengeID)
}

// MarkChallengeVerified marks challenge as verified
func (s *UserService) MarkChallengeVerified(challengeID string) {
	s.db.DB.Exec(`UPDATE login_challenges SET verified_at = NOW() WHERE id = $1`, challengeID)
}

// FindOrCreateUserByPhone finds existing user or creates new one
func (s *UserService) FindOrCreateUserByPhone(phone string) (*models.User, *models.UserPhone, error) {
	// Try to find existing user by phone
	var userID string
	phoneQuery := `SELECT user_id FROM user_phones WHERE phone = $1 OR phone_e164 = $1 LIMIT 1`
	err := s.db.DB.QueryRow(phoneQuery, phone).Scan(&userID)

	if err == sql.ErrNoRows {
		// Create new user
		return s.createUserWithPhone(phone)
	}
	if err != nil {
		return nil, nil, err
	}

	// Get existing user
	user, _, userPhone, _, err := s.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}

	return user, userPhone, nil
}

// createUserWithPhone creates new user with phone number
func (s *UserService) createUserWithPhone(phone string) (*models.User, *models.UserPhone, error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	// Generate external ID
	externalID := fmt.Sprintf("usr_%d", time.Now().UnixNano()%1000000)

	// Create user
	userQuery := `
		INSERT INTO users (
			external_id, first_name, last_name, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var userID string
	err = tx.QueryRow(userQuery, externalID, "User", phone[len(phone)-4:], true, time.Now().UTC(), time.Now().UTC()).Scan(&userID)
	if err != nil {
		return nil, nil, err
	}

	// Create phone record
	phoneQuery := `
		INSERT INTO user_phones (
			user_id, phone, phone_e164, is_primary, is_verified, verified_at, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var phoneID string
	err = tx.QueryRow(
		phoneQuery, userID, phone, phone, true, true, time.Now().UTC(), true, time.Now().UTC(), time.Now().UTC(),
	).Scan(&phoneID)
	if err != nil {
		return nil, nil, err
	}

	// Update user with primary phone
	_, err = tx.Exec(`UPDATE users SET primary_phone_id = $1 WHERE id = $2`, phoneID, userID)
	if err != nil {
		return nil, nil, err
	}

	// Assign default role
	var roleID string
	err = tx.QueryRow(`SELECT id FROM roles WHERE name = 'user' LIMIT 1`).Scan(&roleID)
	if err == nil {
		tx.Exec(`INSERT INTO user_roles (user_id, role_id, granted_at) VALUES ($1, $2, $3)`, userID, roleID, time.Now().UTC())
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}

	// Fetch created user
	user, _, userPhone, _, _ := s.GetUserByID(userID)
	return user, userPhone, nil
}
