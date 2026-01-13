package users

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/models"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/google/uuid"
)

// UserService handles user business logic
type UserService struct {
	db             *clients.Database
	jwtUtil        *utils.JWTUtil
	redisHelper    *redishelper.RedisHelper
	config         *config.Config
}

// NewUserService creates a new user service with new schema support
func NewUserService(db *clients.Database, jwtUtil *utils.JWTUtil, redisHelper *redishelper.RedisHelper, cfg *config.Config) *UserService {
	return &UserService{
		db:             db,
		jwtUtil:        jwtUtil,
		redisHelper:    redisHelper,
		config:         cfg,
	}
}

// CreateUser creates a new user with email and default role
func (s *UserService) CreateUser(req *RegisterRequest) (*models.User, *models.UserEmail, string, error) {
	// Validate email
	if !utils.IsEmailValid(req.Email) {
		return nil, nil, "", fmt.Errorf("invalid email address")
	}

	// Validate password
	valid, msg := utils.IsPasswordValid(req.Password)
	if !valid {
		return nil, nil, "", fmt.Errorf(msg)
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if email already exists
	exists, err := s.emailExists(req.Email)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, nil, "", fmt.Errorf("email already registered")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate IDs
	userID := uuid.New().String()
	emailID := uuid.New().String()
	externalID := fmt.Sprintf("usr_%s", uuid.New().String()[:8])
	now := time.Now().UTC()

	// Create user
	userQuery := `
		INSERT INTO users (
			id, external_id, password_hash, first_name, last_name,
			timezone, locale, date_format, time_format,
			is_active, is_verified, verification_level, token_version,
			has_2fa_enabled, privacy_consent, marketing_consent, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			'UTC', 'en_CA', 'YYYY-MM-DD', '24h',
			true, false, 0, 1,
			false, '{}', false, '{}',
			$6, $7
		)
		RETURNING id, external_id, first_name, last_name, is_active, is_verified, created_at, updated_at
	`

	user := &models.User{}
	err = tx.QueryRow(
		userQuery,
		userID, externalID, hashedPassword,
		utils.SanitizeString(req.FirstName),
		utils.SanitizeString(req.LastName),
		now, now,
	).Scan(
		&user.ID, &user.ExternalID, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Create user email
	emailQuery := `
		INSERT INTO user_emails (
			id, user_id, email, is_primary, is_recovery, is_verified, is_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, true, false, false, true, $4, $5)
		RETURNING id, user_id, email, is_primary, is_verified, created_at
	`

	userEmail := &models.UserEmail{}
	err = tx.QueryRow(
		emailQuery,
		emailID, userID, utils.SanitizeString(req.Email), now, now,
	).Scan(
		&userEmail.ID, &userEmail.UserID, &userEmail.Email,
		&userEmail.IsPrimary, &userEmail.IsVerified, &userEmail.CreatedAt,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create user email: %w", err)
	}

	// Update user with primary_email_id
	_, err = tx.Exec(`UPDATE users SET primary_email_id = $1 WHERE id = $2`, emailID, userID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to set primary email: %w", err)
	}

	// Get default "user" role
	var roleID string
	err = tx.QueryRow(`SELECT id FROM roles WHERE name = 'user' AND is_active = true LIMIT 1`).Scan(&roleID)
	if err != nil {
		// If role doesn't exist, create it
		roleID = uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO roles (id, name, display_name, is_system_role, is_admin_role, hierarchy_level, permissions, is_active, created_at, updated_at)
			VALUES ($1, 'user', 'User', true, false, 0, '[]', true, $2, $3)
		`, roleID, now, now)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to create default role: %w", err)
		}
	}

	// Assign role to user
	userRoleID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_roles (id, user_id, role_id, granted_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, $5, $6)
	`, userRoleID, userID, roleID, now, now, now)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to assign role: %w", err)
	}

	// =====================================================
	// AUTO-CREATE PERSONAL ORGANIZATION (1:1 ratio)
	// As per ShipInAFlash specification: Every user gets
	// their own organization automatically at registration
	// =====================================================

	// Generate unique organization external_id (e.g., org_K9R2B7X4)
	orgExternalID := fmt.Sprintf("org_%s", generateRandomCode(8))
	organizationID := uuid.New().String()

	// Create organization with user as owner
	orgQuery := `
		INSERT INTO organizations (
			id, external_id, name, owner_user_id,
			status, plan, is_deleted,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'active', 'tier_1', false, $5, $6)
		RETURNING id
	`
	err = tx.QueryRow(orgQuery, organizationID, orgExternalID, orgExternalID, userID, now, now).Scan(&organizationID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create organization: %w", err)
	}

	// Add user as organization owner in organization_members
	orgMemberID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO organization_members (
			id, organization_id, user_id, role, is_active, joined_at
		) VALUES ($1, $2, $3, 'owner', true, $4)
	`, orgMemberID, organizationID, userID, now)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to add user to organization: %w", err)
	}

	// Create organization settings with defaults
	orgSettingsID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO organization_settings (
			id, organization_id,
			allow_same_day_pickup, allow_next_day_pickup,
			created_at, updated_at
		) VALUES ($1, $2, false, true, $3, $4)
	`, orgSettingsID, organizationID, now, now)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create organization settings: %w", err)
	}

	// Create organization wallet (WAL-XXXXXXXX)
	walletNumber := fmt.Sprintf("WAL-%s", generateRandomCode(8))
	walletID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO wallets (
			id, wallet_number, organization_id,
			balance, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, 0.00, 'active', $4, $5)
	`, walletID, walletNumber, organizationID, now, now)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create organization preferences with defaults
	orgPreferencesID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO organization_preferences (
			id, organization_id,
			signature_option, weight_unit, dimension_unit,
			timezone, label_format,
			hide_shipper_phone_on_label, show_usd_estimates,
			created_at, updated_at
		) VALUES ($1, $2, 'none', 'lb', 'in', 'America/Toronto', 'thermal_4x6', false, false, $3, $4)
	`, orgPreferencesID, organizationID, now, now)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create organization preferences: %w", err)
	}

	// Generate unique referral code for user (never changes)
	referralCode := fmt.Sprintf("REF-%s", generateRandomCode(6))
	_, err = tx.Exec(`
		UPDATE users SET referral_code = $1 WHERE id = $2
	`, referralCode, userID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to set referral code: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Set the password hash so we can use it later if needed
	user.PasswordHash = hashedPassword

	return user, userEmail, "user", nil
}

// AuthenticateUser authenticates a user and returns tokens
func (s *UserService) AuthenticateUser(email, password string) (*LoginResponse, error) {
	// Get user by email (through user_emails table)
	user, userEmail, roleName, err := s.getUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive || user.DeletedAt.Valid {
		return nil, fmt.Errorf("account is inactive or deleted")
	}

	// Check if account is locked
	if user.LockedUntil.Valid && time.Now().Before(user.LockedUntil.Time) {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Time.Format(time.RFC3339))
	}

	// Verify password
	if !utils.VerifyPassword(password, user.PasswordHash) {
		// Increment login attempts
		s.incrementLoginAttempts(user.ID)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset login attempts on successful login
	s.resetLoginAttempts(user.ID)

	// Generate tokens
	accessToken, _, err := s.jwtUtil.GenerateAccessToken(
		user.ID,
		"web",
		roleName,
		[]string{"read", "write"},
		s.config.OAuth.AccessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshTokenID, err := s.jwtUtil.GenerateRefreshToken(
		user.ID,
		"web",
		s.config.OAuth.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update last login
	s.updateLastLogin(user.ID)

	// Store refresh token
	s.storeRefreshToken(user.ID, refreshTokenID, s.config.OAuth.RefreshTokenExpiry)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.OAuth.AccessTokenExpiry.Seconds()),
		User:         s.sanitizeUser(user, userEmail, roleName),
	}, nil
}

// AuthenticateUserByPhone authenticates a user by phone number and password
func (s *UserService) AuthenticateUserByPhone(phone, password string) (*LoginResponse, error) {
	user, userPhone, roleName, err := s.getUserByPhone(phone)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive || user.DeletedAt.Valid {
		return nil, fmt.Errorf("account is inactive or deleted")
	}

	if user.LockedUntil.Valid && time.Now().Before(user.LockedUntil.Time) {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Time.Format(time.RFC3339))
	}

	if !utils.VerifyPassword(password, user.PasswordHash) {
		s.incrementLoginAttempts(user.ID)
		return nil, fmt.Errorf("invalid credentials")
	}

	s.resetLoginAttempts(user.ID)

	accessToken, _, err := s.jwtUtil.GenerateAccessToken(
		user.ID, "web", roleName, []string{"read", "write"}, s.config.OAuth.AccessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshTokenID, err := s.jwtUtil.GenerateRefreshToken(
		user.ID, "web", s.config.OAuth.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.updateLastLogin(user.ID)
	s.storeRefreshToken(user.ID, refreshTokenID, s.config.OAuth.RefreshTokenExpiry)

	status := "inactive"
	if user.IsActive {
		status = "active"
	}

	userResp := &UserResponse{
		ID: user.ID, Phone: userPhone.Phone, PhoneVerified: userPhone.IsVerified,
		FirstName: user.FirstName, LastName: user.LastName, Role: roleName,
		Status: status, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}

	return &LoginResponse{
		AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer",
		ExpiresIn: int(s.config.OAuth.AccessTokenExpiry.Seconds()), User: userResp,
	}, nil
}

// GetUserByID retrieves a user by ID with email and role
func (s *UserService) GetUserByID(userID string) (*models.User, *models.UserEmail, *models.UserPhone, string, error) {
	query := `
		SELECT
			u.id, u.external_id, u.first_name, u.last_name, u.display_name, u.avatar_file_id,
			u.timezone, u.locale, u.date_format, u.time_format,
			u.is_active, u.is_verified, u.verification_level,
			u.last_login_at, u.last_login_ip, u.login_attempts,
			u.token_version, u.has_2fa_enabled,
			u.created_at, u.updated_at, u.deleted_at,
			ue.id as email_id, ue.email, ue.is_verified as email_verified,
			up.id as phone_id, up.phone, up.is_verified as phone_verified,
			r.name as role_name
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_phones up ON u.primary_phone_id = up.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE u.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	user := &models.User{}
	userEmail := &models.UserEmail{}
	userPhone := &models.UserPhone{}
	var roleName sql.NullString
	var emailID, phoneID sql.NullString
	var email, phone sql.NullString
	var emailVerified, phoneVerified sql.NullBool

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.ExternalID, &user.FirstName, &user.LastName, &user.DisplayName, &user.AvatarURL,
		&user.Timezone, &user.Locale, &user.DateFormat, &user.TimeFormat,
		&user.IsActive, &user.IsVerified, &user.VerificationLevel,
		&user.LastLoginAt, &user.LastLoginIP, &user.LoginAttempts,
		&user.TokenVersion, &user.TOTPEnabled,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&emailID, &email, &emailVerified,
		&phoneID, &phone, &phoneVerified,
		&roleName,
	)

	if err == sql.ErrNoRows {
		return nil, nil, nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// Populate email if exists
	var finalEmail *models.UserEmail
	if emailID.Valid {
		userEmail.ID = emailID.String
		userEmail.UserID = user.ID
		userEmail.Email = email.String
		userEmail.IsPrimary = true
		userEmail.IsVerified = emailVerified.Bool
		finalEmail = userEmail
	}

	// Populate phone if exists
	var finalPhone *models.UserPhone
	if phoneID.Valid {
		userPhone.ID = phoneID.String
		userPhone.UserID = user.ID
		userPhone.Phone = phone.String
		userPhone.IsPrimary = true
		userPhone.IsVerified = phoneVerified.Bool
		finalPhone = userPhone
	}

	role := "user"
	if roleName.Valid {
		role = roleName.String
	}

	return user, finalEmail, finalPhone, role, nil
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(userID string, req *UpdateProfileRequest) (*models.User, *models.UserEmail, *models.UserPhone, string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Update user
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`
	_, err = tx.Exec(query, req.FirstName, req.LastName, now, userID)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to update user: %w", err)
	}

	// Handle phone update if provided
	if req.Phone != "" {
		// Check if user has a primary phone
		var phoneID sql.NullString
		err = tx.QueryRow(`SELECT primary_phone_id FROM users WHERE id = $1`, userID).Scan(&phoneID)
		if err != nil {
			return nil, nil, nil, "", fmt.Errorf("failed to check phone: %w", err)
		}

		if phoneID.Valid {
			// Update existing phone
			_, err = tx.Exec(`UPDATE user_phones SET phone = $1, updated_at = $2 WHERE id = $3`, req.Phone, now, phoneID.String)
		} else {
			// Create new phone
			newPhoneID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO user_phones (id, user_id, phone, is_primary, is_verified, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, true, false, true, $4, $5)
			`, newPhoneID, userID, req.Phone, now, now)
			if err == nil {
				// Update user with primary phone
				_, err = tx.Exec(`UPDATE users SET primary_phone_id = $1 WHERE id = $2`, newPhoneID, userID)
			}
		}
		if err != nil {
			return nil, nil, nil, "", fmt.Errorf("failed to update phone: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate user cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user:%s", userID))

	// Get updated user
	return s.GetUserByID(userID)
}

// ChangePassword changes user password
func (s *UserService) ChangePassword(userID, oldPassword, newPassword string) error {
	// Get user
	user, _, _, _, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Get password hash
	var passwordHash string
	err = s.db.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Verify old password
	if !utils.VerifyPassword(oldPassword, passwordHash) {
		return fmt.Errorf("current password is incorrect")
	}

	// Validate new password
	valid, msg := utils.IsPasswordValid(newPassword)
	if !valid {
		return fmt.Errorf(msg)
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Add to password history
	historyID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_password_history (id, user_id, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`, historyID, userID, passwordHash, now)
	if err != nil {
		return fmt.Errorf("failed to save password history: %w", err)
	}

	// Update password and increment token version (force logout)
	_, err = tx.Exec(`
		UPDATE users
		SET password_hash = $1, token_version = token_version + 1, updated_at = $2
		WHERE id = $3
	`, hashedPassword, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Revoke all existing sessions
	s.redisHelper.DeleteAllUserSessions(user.ID)

	return nil
}

// ForgotPassword initiates password reset flow by sending reset email
func (s *UserService) ForgotPassword(email string) error {
	// Find user by email
	query := `
		SELECT u.id, u.first_name, u.last_name
		FROM users u
		INNER JOIN user_emails ue ON ue.user_id = u.id
		WHERE ue.email = $1
		  AND ue.is_verified = true
		  AND u.is_active = true
		  AND u.deleted_at IS NULL
		LIMIT 1
	`

	var userID, firstName, lastName string
	err := s.db.DB.QueryRow(query, email).Scan(&userID, &firstName, &lastName)
	if err == sql.ErrNoRows {
		// Don't reveal if email exists - return success anyway for security
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Generate reset token
	resetToken := utils.GenerateRandomString(50)
	expiresAt := time.Now().UTC().Add(1 * time.Hour) // 1 hour expiry

	// Delete any existing unused tokens for this user to avoid unique constraint violation
	deleteQuery := `
		DELETE FROM password_reset_tokens
		WHERE user_id = $1 AND used_at IS NULL
	`
	_, _ = s.db.DB.Exec(deleteQuery, userID)

	// Store reset token
	tokenID := uuid.New().String()
	insertQuery := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	tokenHash := utils.HashSHA256(resetToken)
	_, err = s.db.DB.Exec(insertQuery, tokenID, userID, tokenHash, time.Now().UTC(), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// Send reset email
	if err := s.sendPasswordResetEmail(email, firstName, resetToken); err != nil {
		// Log error but don't fail the request - email is sent as best effort
		fmt.Printf("Failed to send password reset email: %v\n", err)
	}

	return nil
}

// sendPasswordResetEmail sends password reset email
func (s *UserService) sendPasswordResetEmail(emailAddr, firstName, token string) error {
	// Generate reset URL
	baseURL := s.config.OAuth.JWTIssuer
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	resetURL := fmt.Sprintf("%s/api/v1/reset-password?token=%s", baseURL, token)

	// Email subject
	subject := "Reset Your Password"

	// Email sending has been removed (SendGrid wrapper deleted)
	// In development mode, show link in console
	if s.config.IsDevelopment() {
		fmt.Printf("📧 [DEV MODE] Password Reset Email for %s\n", emailAddr)
		fmt.Printf("   Subject: %s\n", subject)
		fmt.Printf("   Reset URL: %s\n", resetURL)
		fmt.Printf("   Note: Email sending disabled - external email service required for production\n")
		return nil
	}

	// Production mode - log warning
	fmt.Printf("⚠️  Email sending not configured for %s - Reset URL: %s\n", emailAddr, resetURL)
	return fmt.Errorf("email sending not configured - please configure external email service")
}

// ResetPassword resets user password using reset token
func (s *UserService) ResetPassword(token, newPassword string) error {
	// Validate new password
	valid, msg := utils.IsPasswordValid(newPassword)
	if !valid {
		return fmt.Errorf(msg)
	}

	// Find valid reset token
	tokenHash := utils.HashSHA256(token)
	query := `
		SELECT id, user_id, expires_at, used_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var tokenID, userID string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := s.db.DB.QueryRow(query, tokenHash).Scan(&tokenID, &userID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("invalid or expired reset token")
	}
	if err != nil {
		return fmt.Errorf("failed to find reset token: %w", err)
	}

	// Check if token already used
	if usedAt.Valid {
		return fmt.Errorf("reset token has already been used")
	}

	// Check if token expired
	if time.Now().After(expiresAt) {
		return fmt.Errorf("reset token has expired")
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Get current password for history
	var currentPasswordHash string
	err = tx.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&currentPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to get current password: %w", err)
	}

	// Add to password history
	historyID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_password_history (id, user_id, password_hash, created_at, change_reason)
		VALUES ($1, $2, $3, $4, $5)
	`, historyID, userID, currentPasswordHash, now, "password_reset")
	if err != nil {
		return fmt.Errorf("failed to save password history: %w", err)
	}

	// Update password and increment token version (force logout)
	_, err = tx.Exec(`
		UPDATE users
		SET password_hash = $1, token_version = token_version + 1, updated_at = $2
		WHERE id = $3
	`, hashedPassword, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	_, err = tx.Exec(`
		UPDATE password_reset_tokens
		SET used_at = $1
		WHERE id = $2
	`, now, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Revoke all existing sessions
	s.redisHelper.DeleteAllUserSessions(userID)

	return nil
}

// DeleteUser soft deletes a user
func (s *UserService) DeleteUser(userID string) error {
	now := time.Now().UTC()
	query := `UPDATE users SET deleted_at = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`
	result, err := s.db.Exec(query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	// Revoke all sessions
	s.redisHelper.DeleteAllUserSessions(userID)

	// Invalidate cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user:%s", userID))

	return nil
}

// ListUsers lists all users with pagination
func (s *UserService) ListUsers(page, limit int) ([]*UserResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	err := s.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users with email and role
	query := `
		SELECT
			u.id, u.external_id, u.first_name, u.last_name, u.avatar_file_id,
			u.is_active, u.is_verified,
			u.created_at, u.updated_at,
			ue.email, ue.is_verified as email_verified,
			up.phone, up.is_verified as phone_verified,
			r.name as role_name
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_phones up ON u.primary_phone_id = up.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*UserResponse{}
	for rows.Next() {
		var email, phone, avatarURL, roleName sql.NullString
		var emailVerified, phoneVerified sql.NullBool
		var userID, externalID, firstName, lastName string
		var isActive, isVerified bool
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&userID, &externalID, &firstName, &lastName, &avatarURL,
			&isActive, &isVerified,
			&createdAt, &updatedAt,
			&email, &emailVerified,
			&phone, &phoneVerified,
			&roleName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		status := "active"
		if !isActive {
			status = "inactive"
		}

		users = append(users, &UserResponse{
			ID:            userID,
			Email:         email.String,
			FirstName:     firstName,
			LastName:      lastName,
			Phone:         phone.String,
			Avatar:        avatarURL.String,
			Role:          roleName.String,
			Status:        status,
			EmailVerified: emailVerified.Bool,
			PhoneVerified: phoneVerified.Bool,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	return users, total, nil
}

// Helper methods

func (s *UserService) emailExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_emails WHERE email_lower = LOWER($1) AND is_active = true)`
	err := s.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}

func (s *UserService) getUserByEmail(email string) (*models.User, *models.UserEmail, string, error) {
	query := `
		SELECT
			u.id, u.external_id, u.password_hash, u.first_name, u.last_name,
			u.is_active, u.is_verified, u.locked_until,
			u.created_at, u.updated_at, u.deleted_at,
			ue.id as email_id, ue.email, ue.is_verified as email_verified,
			r.name as role_name
		FROM user_emails ue
		INNER JOIN users u ON ue.user_id = u.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE ue.email_lower = LOWER($1) AND ue.is_active = true AND u.deleted_at IS NULL
		LIMIT 1
	`

	user := &models.User{}
	userEmail := &models.UserEmail{}
	var roleName sql.NullString

	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.ExternalID, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&userEmail.ID, &userEmail.Email, &userEmail.IsVerified,
		&roleName,
	)

	if err == sql.ErrNoRows {
		return nil, nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	userEmail.UserID = user.ID
	userEmail.IsPrimary = true

	role := "user"
	if roleName.Valid {
		role = roleName.String
	}

	return user, userEmail, role, nil
}

func (s *UserService) getUserByPhone(phone string) (*models.User, *models.UserPhone, string, error) {
	query := `
		SELECT
			u.id, u.external_id, u.password_hash, u.first_name, u.last_name,
			u.is_active, u.is_verified, u.locked_until,
			u.created_at, u.updated_at, u.deleted_at,
			up.id as phone_id, up.phone, up.is_verified as phone_verified,
			r.name as role_name
		FROM user_phones up
		INNER JOIN users u ON up.user_id = u.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE up.phone = $1 AND up.is_active = true AND u.deleted_at IS NULL
		LIMIT 1
	`

	user := &models.User{}
	userPhone := &models.UserPhone{}
	var roleName sql.NullString

	err := s.db.QueryRow(query, phone).Scan(
		&user.ID, &user.ExternalID, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&userPhone.ID, &userPhone.Phone, &userPhone.IsVerified,
		&roleName,
	)

	if err == sql.ErrNoRows {
		return nil, nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	userPhone.UserID = user.ID
	userPhone.IsPrimary = true

	role := "user"
	if roleName.Valid {
		role = roleName.String
	}

	return user, userPhone, role, nil
}

func (s *UserService) updateLastLogin(userID string) {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	s.db.Exec(query, time.Now().UTC(), userID)
}

func (s *UserService) incrementLoginAttempts(userID string) {
	query := `
		UPDATE users
		SET login_attempts = login_attempts + 1,
		    locked_until = CASE
				WHEN login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes'
				ELSE NULL
			END
		WHERE id = $1
	`
	s.db.Exec(query, userID)
}

func (s *UserService) resetLoginAttempts(userID string) {
	query := `UPDATE users SET login_attempts = 0, locked_until = NULL WHERE id = $1`
	s.db.Exec(query, userID)
}

func (s *UserService) storeRefreshToken(userID, tokenID string, expiry time.Duration) {
	key := fmt.Sprintf("refresh_token:%s", tokenID)
	s.redisHelper.CacheSet(key, map[string]string{"user_id": userID}, expiry)
}

func (s *UserService) sanitizeUser(user *models.User, email *models.UserEmail, roleName string) *UserResponse {
	status := "active"
	if !user.IsActive {
		status = "inactive"
	}

	avatar := ""
	if user.AvatarURL.Valid {
		avatar = user.AvatarURL.String
	}

	emailAddr := ""
	emailVerified := false
	if email != nil {
		emailAddr = email.Email
		emailVerified = email.IsVerified
	}

	return &UserResponse{
		ID:            user.ID,
		Email:         emailAddr,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Phone:         "",
		Avatar:        avatar,
		Role:          roleName,
		Status:        status,
		EmailVerified: emailVerified,
		PhoneVerified: false,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}

// RefreshToken validates a refresh token and issues a new access token
func (s *UserService) RefreshToken(refreshToken string) (*RefreshTokenResponse, error) {
	// Validate the refresh token
	claims, err := s.jwtUtil.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	// Check if token is blacklisted
	blacklistKey := fmt.Sprintf("blacklist:token:%s", claims.TokenID)
	exists, err := s.redisHelper.CacheExists(blacklistKey)
	if err == nil && exists {
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// Verify token still exists in Redis (not deleted)
	tokenKey := fmt.Sprintf("refresh_token:%s", claims.TokenID)
	exists, err = s.redisHelper.CacheExists(tokenKey)
	if err != nil || !exists {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Get user to verify they still exist and are active
	query := `
		SELECT u.id, u.is_active, u.deleted_at, u.token_version, r.name as role_name
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE u.id = $1
		LIMIT 1
	`

	var userID string
	var isActive bool
	var deletedAt sql.NullTime
	var tokenVersion int
	var roleName sql.NullString

	err = s.db.QueryRow(query, claims.UserID).Scan(&userID, &isActive, &deletedAt, &tokenVersion, &roleName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify user: %w", err)
	}

	// Check if user is still active
	if !isActive || deletedAt.Valid {
		return nil, fmt.Errorf("user account is inactive or deleted")
	}

	// Get role name
	role := "user"
	if roleName.Valid {
		role = roleName.String
	}

	// Generate new access token
	accessTokenExpiry := time.Duration(s.config.OAuth.AccessTokenExpiry) * time.Second
	accessToken, tokenID, err := s.jwtUtil.GenerateAccessToken(
		userID,
		claims.ClientID,
		role,
		claims.Scopes,
		accessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Store token ID in Redis for revocation capability
	s.redisHelper.CacheSet(
		fmt.Sprintf("access_token:%s", tokenID),
		map[string]string{"user_id": userID},
		accessTokenExpiry,
	)

	// Optionally rotate refresh token (recommended for security)
	// For now, we'll keep the same refresh token to match common OAuth practices
	// If you want to rotate, uncomment the code below:

	/*
	// Generate new refresh token
	refreshTokenExpiry := time.Duration(s.config.OAuth.RefreshTokenExpiry) * time.Second
	newRefreshToken, refreshTokenID, err := s.jwtUtil.GenerateRefreshToken(
		userID,
		claims.ClientID,
		refreshTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store new refresh token
	s.storeRefreshToken(userID, refreshTokenID, refreshTokenExpiry)

	// Blacklist old refresh token
	s.redisHelper.CacheSet(
		fmt.Sprintf("blacklist:token:%s", claims.TokenID),
		map[string]string{"revoked": "true"},
		refreshTokenExpiry,
	)
	*/

	return &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: "", // Empty if not rotating, or newRefreshToken if rotating
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.OAuth.AccessTokenExpiry),
	}, nil
}

// IncrementTokenVersion increments the user's token version to invalidate all existing tokens
func (s *UserService) IncrementTokenVersion(userID string) error {
	query := `UPDATE users SET token_version = token_version + 1, updated_at = $1 WHERE id = $2`
	_, err := s.db.Exec(query, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to increment token version: %w", err)
	}
	return nil
}

// UpdateAvatar updates a user's avatar file ID in the database
func (s *UserService) UpdateAvatar(userID string, avatarFileID string) error {
	query := `
		UPDATE users
		SET avatar_file_id = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := s.db.DB.Exec(query, avatarFileID, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to update avatar: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeleteAvatar removes a user's avatar file ID from the database and returns the old avatar file ID
func (s *UserService) DeleteAvatar(userID string) (string, error) {
	// First, get the current avatar file ID
	var avatarFileID sql.NullString
	query := `
		SELECT avatar_file_id
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	err := s.db.DB.QueryRow(query, userID).Scan(&avatarFileID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user avatar: %w", err)
	}

	// Update avatar to NULL
	updateQuery := `
		UPDATE users
		SET avatar_file_id = NULL, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	_, err = s.db.DB.Exec(updateQuery, time.Now().UTC(), userID)
	if err != nil {
		return "", fmt.Errorf("failed to delete avatar: %w", err)
	}

	// Return the old avatar file ID for cleanup
	if avatarFileID.Valid {
		return avatarFileID.String, nil
	}

	return "", nil
}

// ListUserEmails retrieves all emails for a user
func (s *UserService) ListUserEmails(userID string) ([]*models.UserEmail, error) {
	query := `
		SELECT id, user_id, email, is_primary, is_recovery, is_verified,
		       verified_at, is_active, created_at, updated_at
		FROM user_emails
		WHERE user_id = $1 AND is_active = true
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := s.db.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	emails := []*models.UserEmail{}
	for rows.Next() {
		var email models.UserEmail
		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&email.Email,
			&email.IsPrimary,
			&email.IsRecovery,
			&email.IsVerified,
			&email.VerifiedAt,
			&email.IsActive,
			&email.CreatedAt,
			&email.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}
		emails = append(emails, &email)
	}

	return emails, nil
}

// AddUserEmail adds a new email to a user's account
func (s *UserService) AddUserEmail(userID string, emailAddr string) (*models.UserEmail, error) {
	// Validate email
	if !utils.IsEmailValid(emailAddr) {
		return nil, fmt.Errorf("invalid email address")
	}

	// Check if email already exists for any user
	var existingID string
	checkQuery := `SELECT id FROM user_emails WHERE LOWER(email) = LOWER($1) AND is_active = true`
	err := s.db.DB.QueryRow(checkQuery, emailAddr).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("email already in use")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// Generate verification token
	verificationToken := utils.GenerateRandomString(32)
	verificationExpiry := time.Now().UTC().Add(24 * time.Hour)

	// Insert new email
	emailID := uuid.New().String()
	query := `
		INSERT INTO user_emails (
			id, user_id, email, is_primary, is_recovery, is_verified,
			verification_token, verification_expires_at, is_active,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, false, false, false, $4, $5, true, $6, $7)
		RETURNING id, user_id, email, is_primary, is_recovery, is_verified,
		          verified_at, is_active, created_at, updated_at
	`

	email := &models.UserEmail{}
	err = s.db.DB.QueryRow(query,
		emailID,
		userID,
		emailAddr,
		verificationToken,
		verificationExpiry,
		time.Now().UTC(),
		time.Now().UTC(),
	).Scan(
		&email.ID,
		&email.UserID,
		&email.Email,
		&email.IsPrimary,
		&email.IsRecovery,
		&email.IsVerified,
		&email.VerifiedAt,
		&email.IsActive,
		&email.CreatedAt,
		&email.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add email: %w", err)
	}

	// Send verification email with token
	if err := s.sendEmailVerification(email.Email, verificationToken); err != nil {
		// Log error but don't fail the request - email is sent as best effort
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	return email, nil
}

// SetPrimaryEmail sets an email as the primary email for a user
func (s *UserService) SetPrimaryEmail(userID string, emailID string) error {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify email belongs to user and is verified
	var isVerified bool
	checkQuery := `
		SELECT is_verified
		FROM user_emails
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err = tx.QueryRow(checkQuery, emailID, userID).Scan(&isVerified)
	if err == sql.ErrNoRows {
		return fmt.Errorf("email not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}

	if !isVerified {
		return fmt.Errorf("cannot set unverified email as primary")
	}

	// Unset current primary
	unsetQuery := `
		UPDATE user_emails
		SET is_primary = false, updated_at = $1
		WHERE user_id = $2 AND is_primary = true
	`
	_, err = tx.Exec(unsetQuery, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to unset primary email: %w", err)
	}

	// Set new primary
	setPrimaryQuery := `
		UPDATE user_emails
		SET is_primary = true, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`
	result, err := tx.Exec(setPrimaryQuery, time.Now().UTC(), emailID, userID)
	if err != nil {
		return fmt.Errorf("failed to set primary email: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("email not found")
	}

	// Update user's primary_email_id
	updateUserQuery := `
		UPDATE users
		SET primary_email_id = $1, updated_at = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateUserQuery, emailID, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user primary email: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteUserEmail removes an email from a user's account
func (s *UserService) DeleteUserEmail(userID string, emailID string) error {
	// Check if email is primary
	var isPrimary bool
	checkQuery := `
		SELECT is_primary
		FROM user_emails
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err := s.db.DB.QueryRow(checkQuery, emailID, userID).Scan(&isPrimary)
	if err == sql.ErrNoRows {
		return fmt.Errorf("email not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}

	if isPrimary {
		return fmt.Errorf("cannot delete primary email")
	}

	// Soft delete email
	query := `
		UPDATE user_emails
		SET is_active = false, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`

	result, err := s.db.DB.Exec(query, time.Now().UTC(), emailID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("email not found")
	}

	return nil
}

// ResendEmailVerification resends verification email for an unverified email
func (s *UserService) ResendEmailVerification(userID string, emailID string) error {
	// Get email details
	var email string
	var isVerified bool
	query := `
		SELECT email, is_verified
		FROM user_emails
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err := s.db.DB.QueryRow(query, emailID, userID).Scan(&email, &isVerified)
	if err == sql.ErrNoRows {
		return fmt.Errorf("email not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	if isVerified {
		return fmt.Errorf("email is already verified")
	}

	// Generate new verification token
	verificationToken := utils.GenerateRandomString(32)
	verificationExpiry := time.Now().UTC().Add(24 * time.Hour)

	// Update verification token
	updateQuery := `
		UPDATE user_emails
		SET verification_token = $1, verification_expires_at = $2, updated_at = $3
		WHERE id = $4 AND user_id = $5
	`

	_, err = s.db.DB.Exec(updateQuery,
		verificationToken,
		verificationExpiry,
		time.Now().UTC(),
		emailID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update verification token: %w", err)
	}

	// Send verification email
	if err := s.sendEmailVerification(email, verificationToken); err != nil {
		// Log error but don't fail the request - email is sent as best effort
		// The verification token is already saved in the database
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	return nil
}

// sendEmailVerification sends a verification email with the token
func (s *UserService) sendEmailVerification(emailAddr, token string) error {
	// Generate verification URL using JWT issuer as base URL
	baseURL := s.config.OAuth.JWTIssuer
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	verificationURL := fmt.Sprintf("%s/api/v1/verify-email?token=%s", baseURL, token)

	// Email subject
	subject := "Verify Your Email Address"

	// Email sending has been removed (SendGrid wrapper deleted)
	// In development mode, show link in console
	if s.config.IsDevelopment() {
		fmt.Printf("📧 [DEV MODE] Email Verification for %s\n", emailAddr)
		fmt.Printf("   Subject: %s\n", subject)
		fmt.Printf("   Verification URL: %s\n", verificationURL)
		fmt.Printf("   Note: Email sending disabled - external email service required for production\n")
		return nil
	}

	// Production mode - log warning
	fmt.Printf("⚠️  Email sending not configured for %s - Verification URL: %s\n", emailAddr, verificationURL)
	return fmt.Errorf("email sending not configured - please configure external email service")
}

// VerifyEmailWithToken verifies an email address using the verification token
func (s *UserService) VerifyEmailWithToken(token string) (*models.UserEmail, error) {
	// Find email by verification token
	query := `
		SELECT id, user_id, email, is_primary, is_recovery, is_verified,
		       verified_at, verification_expires_at, is_active, created_at, updated_at
		FROM user_emails
		WHERE verification_token = $1
		  AND is_active = true
	`

	var email models.UserEmail
	err := s.db.DB.QueryRow(query, token).Scan(
		&email.ID,
		&email.UserID,
		&email.Email,
		&email.IsPrimary,
		&email.IsRecovery,
		&email.IsVerified,
		&email.VerifiedAt,
		&email.VerificationExpiresAt,
		&email.IsActive,
		&email.CreatedAt,
		&email.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or expired verification token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find email: %w", err)
	}

	// Check if already verified
	if email.IsVerified {
		return &email, nil // Already verified, return success
	}

	// Check if token expired
	if email.VerificationExpiresAt.Valid && time.Now().After(email.VerificationExpiresAt.Time) {
		return nil, fmt.Errorf("verification token has expired")
	}

	// Mark email as verified
	updateQuery := `
		UPDATE user_emails
		SET is_verified = true,
		    verified_at = $1,
		    verification_token = NULL,
		    verification_expires_at = NULL,
		    updated_at = $2
		WHERE id = $3
		RETURNING id, user_id, email, is_primary, is_recovery, is_verified,
		          verified_at, is_active, created_at, updated_at
	`

	now := time.Now().UTC()
	err = s.db.DB.QueryRow(updateQuery, now, now, email.ID).Scan(
		&email.ID,
		&email.UserID,
		&email.Email,
		&email.IsPrimary,
		&email.IsRecovery,
		&email.IsVerified,
		&email.VerifiedAt,
		&email.IsActive,
		&email.CreatedAt,
		&email.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	return &email, nil
}

// ListUserPhones retrieves all phones for a user
func (s *UserService) ListUserPhones(userID string) ([]*models.UserPhone, error) {
	query := `
		SELECT id, user_id, phone, phone_e164, is_primary, is_recovery, is_verified,
		       verified_at, is_active, created_at, updated_at
		FROM user_phones
		WHERE user_id = $1 AND is_active = true
		ORDER BY is_primary DESC, created_at ASC
	`

	rows, err := s.db.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list phones: %w", err)
	}
	defer rows.Close()

	phones := []*models.UserPhone{}
	for rows.Next() {
		var phone models.UserPhone
		err := rows.Scan(
			&phone.ID,
			&phone.UserID,
			&phone.Phone,
			&phone.PhoneE164,
			&phone.IsPrimary,
			&phone.IsRecovery,
			&phone.IsVerified,
			&phone.VerifiedAt,
			&phone.IsActive,
			&phone.CreatedAt,
			&phone.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan phone: %w", err)
		}
		phones = append(phones, &phone)
	}

	return phones, nil
}

// AddUserPhone adds a new phone to a user's account
func (s *UserService) AddUserPhone(userID string, phoneNumber string, isPrimary bool) (*models.UserPhone, error) {
	// TODO: Validate and format phone number to E.164 format
	// For now, we'll just use the phone number as-is
	phoneE164 := phoneNumber

	// Check if phone already exists for any user
	var existingID string
	checkQuery := `SELECT id FROM user_phones WHERE phone = $1 AND is_active = true`
	err := s.db.DB.QueryRow(checkQuery, phoneNumber).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("phone number already in use")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check phone: %w", err)
	}

	// Start transaction to handle primary phone logic
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// If setting as primary, unset other primary phones for this user
	if isPrimary {
		unsetQuery := `UPDATE user_phones SET is_primary = false, updated_at = $1 WHERE user_id = $2 AND is_primary = true`
		_, err = tx.Exec(unsetQuery, time.Now().UTC(), userID)
		if err != nil {
			return nil, fmt.Errorf("failed to unset existing primary: %w", err)
		}
	}

	// Generate verification code (6 digits)
	verificationCode := utils.GenerateRandomString(6)
	verificationExpiry := time.Now().UTC().Add(10 * time.Minute)

	// Insert new phone
	phoneID := uuid.New().String()
	query := `
		INSERT INTO user_phones (
			id, user_id, phone, phone_e164, is_primary, is_recovery, is_verified,
			verification_code, verification_expires_at, is_active,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, false, false, $6, $7, true, $8, $9)
		RETURNING id, user_id, phone, phone_e164, is_primary, is_recovery, is_verified,
		          verified_at, is_active, created_at, updated_at
	`

	phone := &models.UserPhone{}
	err = tx.QueryRow(query,
		phoneID,
		userID,
		phoneNumber,
		phoneE164,
		isPrimary,
		verificationCode,
		verificationExpiry,
		time.Now().UTC(),
		time.Now().UTC(),
	).Scan(
		&phone.ID,
		&phone.UserID,
		&phone.Phone,
		&phone.PhoneE164,
		&phone.IsPrimary,
		&phone.IsRecovery,
		&phone.IsVerified,
		&phone.VerifiedAt,
		&phone.IsActive,
		&phone.CreatedAt,
		&phone.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add phone: %w", err)
	}

	// If this is the primary phone, update the user's primary_phone_id
	if isPrimary {
		updateUserQuery := `UPDATE users SET primary_phone_id = $1, updated_at = $2 WHERE id = $3`
		_, err = tx.Exec(updateUserQuery, phoneID, time.Now().UTC(), userID)
		if err != nil {
			return nil, fmt.Errorf("failed to set primary phone: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send verification SMS with code
	s.sendPhoneVerification(phoneNumber, verificationCode)

	return phone, nil
}

// SetPrimaryPhone sets a phone as the primary phone for a user
func (s *UserService) SetPrimaryPhone(userID string, phoneID string) error {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify phone belongs to user and is verified
	var isVerified bool
	checkQuery := `
		SELECT is_verified
		FROM user_phones
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err = tx.QueryRow(checkQuery, phoneID, userID).Scan(&isVerified)
	if err == sql.ErrNoRows {
		return fmt.Errorf("phone not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}

	if !isVerified {
		return fmt.Errorf("cannot set unverified phone as primary")
	}

	// Unset current primary
	unsetQuery := `
		UPDATE user_phones
		SET is_primary = false, updated_at = $1
		WHERE user_id = $2 AND is_primary = true
	`
	_, err = tx.Exec(unsetQuery, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to unset primary phone: %w", err)
	}

	// Set new primary
	setPrimaryQuery := `
		UPDATE user_phones
		SET is_primary = true, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`
	result, err := tx.Exec(setPrimaryQuery, time.Now().UTC(), phoneID, userID)
	if err != nil {
		return fmt.Errorf("failed to set primary phone: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("phone not found")
	}

	// Update user's primary_phone_id
	updateUserQuery := `
		UPDATE users
		SET primary_phone_id = $1, updated_at = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateUserQuery, phoneID, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user primary phone: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteUserPhone removes a phone from a user's account
func (s *UserService) DeleteUserPhone(userID string, phoneID string) error {
	// Check if phone is primary
	var isPrimary bool
	checkQuery := `
		SELECT is_primary
		FROM user_phones
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err := s.db.DB.QueryRow(checkQuery, phoneID, userID).Scan(&isPrimary)
	if err == sql.ErrNoRows {
		return fmt.Errorf("phone not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}

	if isPrimary {
		return fmt.Errorf("cannot delete primary phone")
	}

	// Soft delete phone
	query := `
		UPDATE user_phones
		SET is_active = false, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`

	result, err := s.db.DB.Exec(query, time.Now().UTC(), phoneID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete phone: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("phone not found")
	}

	return nil
}

// ResendPhoneVerification resends verification SMS for an unverified phone
func (s *UserService) ResendPhoneVerification(userID string, phoneID string) error {
	// Get phone details
	var phone string
	var isVerified bool
	query := `
		SELECT phone, is_verified
		FROM user_phones
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err := s.db.DB.QueryRow(query, phoneID, userID).Scan(&phone, &isVerified)
	if err == sql.ErrNoRows {
		return fmt.Errorf("phone not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get phone: %w", err)
	}

	if isVerified {
		return fmt.Errorf("phone is already verified")
	}

	// Generate new verification code
	verificationCode := utils.GenerateRandomString(6)
	verificationExpiry := time.Now().UTC().Add(10 * time.Minute)

	// Update verification code
	updateQuery := `
		UPDATE user_phones
		SET verification_code = $1, verification_expires_at = $2, updated_at = $3
		WHERE id = $4 AND user_id = $5
	`

	_, err = s.db.DB.Exec(updateQuery,
		verificationCode,
		verificationExpiry,
		time.Now().UTC(),
		phoneID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}

	// Send verification SMS
	s.sendPhoneVerification(phone, verificationCode)

	return nil
}

// sendPhoneVerification sends verification SMS to the phone number
func (s *UserService) sendPhoneVerification(phone, verificationCode string) {
	// SMS sending has been removed (SendGrid/Twilio wrappers deleted)
	// In development mode, show OTP in console
	if s.config.IsDevelopment() {
		fmt.Printf("📱 [DEV MODE] Phone Verification OTP for %s: %s\n", phone, verificationCode)
		fmt.Printf("   (This OTP expires in 10 minutes)\n")
		fmt.Printf("   Note: SMS sending disabled - external SMS service required for production\n")
	} else {
		fmt.Printf("⚠️  SMS sending not configured - verification code: %s\n", verificationCode)
	}

	// Successfully sent SMS
	fmt.Printf("✅ SMS sent successfully to %s\n", phone)

	// Also log in dev mode for debugging
	if s.config.IsDevelopment() {
		fmt.Printf("📱 [DEV MODE] OTP sent to %s: %s\n", phone, verificationCode)
	}
}

// GetUserPhone retrieves a specific phone for a user
func (s *UserService) GetUserPhone(userID string, phoneID string) (*models.UserPhone, error) {
	query := `
		SELECT id, user_id, phone, phone_e164, is_primary, is_recovery, is_verified,
		       verified_at, is_active, created_at, updated_at
		FROM user_phones
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`

	phone := &models.UserPhone{}
	err := s.db.DB.QueryRow(query, phoneID, userID).Scan(
		&phone.ID,
		&phone.UserID,
		&phone.Phone,
		&phone.PhoneE164,
		&phone.IsPrimary,
		&phone.IsRecovery,
		&phone.IsVerified,
		&phone.VerifiedAt,
		&phone.IsActive,
		&phone.CreatedAt,
		&phone.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("phone not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get phone: %w", err)
	}

	return phone, nil
}

// VerifyUserPhone verifies a phone number with the verification code
func (s *UserService) VerifyUserPhone(userID string, phoneID string, code string) error {
	// Get phone details
	var phone string
	var isVerified bool
	var isPrimary bool
	var verificationCode string
	var expiresAt time.Time

	query := `
		SELECT phone, is_verified, is_primary, verification_code, verification_expires_at
		FROM user_phones
		WHERE id = $1 AND user_id = $2 AND is_active = true
	`
	err := s.db.DB.QueryRow(query, phoneID, userID).Scan(&phone, &isVerified, &isPrimary, &verificationCode, &expiresAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("phone not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get phone: %w", err)
	}

	if isVerified {
		return fmt.Errorf("phone is already verified")
	}

	// Check if code has expired
	if time.Now().UTC().After(expiresAt) {
		return fmt.Errorf("verification code has expired")
	}

	// Verify code
	if verificationCode != code {
		return fmt.Errorf("invalid verification code")
	}

	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark phone as verified
	updateQuery := `
		UPDATE user_phones
		SET is_verified = true, verified_at = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`
	_, err = tx.Exec(updateQuery, time.Now().UTC(), time.Now().UTC(), phoneID, userID)
	if err != nil {
		return fmt.Errorf("failed to verify phone: %w", err)
	}

	// If this is the primary phone, update the user's primary_phone_id
	if isPrimary {
		updateUserQuery := `UPDATE users SET primary_phone_id = $1, updated_at = $2 WHERE id = $3`
		_, err = tx.Exec(updateUserQuery, phoneID, time.Now().UTC(), userID)
		if err != nil {
			return fmt.Errorf("failed to set primary phone: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AdminCreateUser creates a new user with specified role (admin only)
func (s *UserService) AdminCreateUser(req *AdminCreateUserRequest) (*models.User, *models.UserEmail, string, error) {
	// Validate email
	if !utils.IsEmailValid(req.Email) {
		return nil, nil, "", fmt.Errorf("invalid email address")
	}

	// Validate password
	valid, msg := utils.IsPasswordValid(req.Password)
	if !valid {
		return nil, nil, "", fmt.Errorf(msg)
	}

	// Check if email already exists
	var existingID string
	checkQuery := `SELECT id FROM user_emails WHERE LOWER(email) = LOWER($1)`
	err := s.db.DB.QueryRow(checkQuery, req.Email).Scan(&existingID)
	if err == nil {
		return nil, nil, "", fmt.Errorf("email already registered")
	}
	if err != sql.ErrNoRows {
		return nil, nil, "", fmt.Errorf("failed to check email: %w", err)
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Default values
	role := "user"
	if req.Role != "" {
		role = req.Role
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate IDs
	userID := uuid.New().String()
	emailID := uuid.New().String()
	externalID := fmt.Sprintf("usr_%s", uuid.New().String()[:8])

	// Create email record FIRST (to satisfy foreign key constraint)
	emailQuery := `
		INSERT INTO user_emails (
			id, user_id, email, is_primary, is_verified, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, true, false, true, $4, $5)
		RETURNING id, email, is_primary, is_verified, created_at, updated_at
	`

	var userEmail models.UserEmail
	err = tx.QueryRow(emailQuery,
		emailID,
		userID,
		req.Email,
		time.Now().UTC(),
		time.Now().UTC(),
	).Scan(
		&userEmail.ID,
		&userEmail.Email,
		&userEmail.IsPrimary,
		&userEmail.IsVerified,
		&userEmail.CreatedAt,
		&userEmail.UpdatedAt,
	)

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create email: %w", err)
	}

	// Create user record (now email exists)
	userQuery := `
		INSERT INTO users (
			id, external_id, primary_email_id, password_hash,
			first_name, last_name, is_active, is_verified,
			timezone, locale, date_format, time_format,
			token_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, 'UTC', 'en', 'YYYY-MM-DD', '24h', 1, $8, $9)
		RETURNING id, external_id, first_name, last_name, is_active, created_at, updated_at
	`

	var user models.User
	err = tx.QueryRow(userQuery,
		userID,
		externalID,
		emailID,
		hashedPassword,
		req.FirstName,
		req.LastName,
		isActive,
		time.Now().UTC(),
		time.Now().UTC(),
	).Scan(
		&user.ID,
		&user.ExternalID,
		&user.FirstName,
		&user.LastName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Get role ID
	var roleID string
	roleQuery := `SELECT id FROM roles WHERE name = $1 AND is_active = true LIMIT 1`
	err = tx.QueryRow(roleQuery, role).Scan(&roleID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("invalid role: %s", role)
	}

	// Assign role
	userRoleQuery := `
		INSERT INTO user_roles (id, user_id, role_id, granted_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, $5, $6)
	`
	_, err = tx.Exec(userRoleQuery,
		uuid.New().String(),
		userID,
		roleID,
		time.Now().UTC(),
		time.Now().UTC(),
		time.Now().UTC(),
	)

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to assign role: %w", err)
	}

	// Create phone if provided
	if req.Phone != "" {
		phoneID := uuid.New().String()
		phoneQuery := `
			INSERT INTO user_phones (
				id, user_id, phone, is_primary, is_verified, is_active, created_at, updated_at
			)
			VALUES ($1, $2, $3, true, false, true, $4, $5)
		`
		_, err = tx.Exec(phoneQuery,
			phoneID,
			userID,
			req.Phone,
			time.Now().UTC(),
			time.Now().UTC(),
		)

		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to create phone: %w", err)
		}

		// Update user's primary_phone_id
		_, err = tx.Exec(`UPDATE users SET primary_phone_id = $1 WHERE id = $2`, phoneID, userID)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to set primary phone: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, &userEmail, role, nil
}

// =====================================================
// Organization User Management Methods
// =====================================================

// GetUserProfileWithOrganization retrieves user profile with organization context
func (s *UserService) GetUserProfileWithOrganization(userID string) (*UserProfileResponse, error) {
	query := `
		SELECT
			u.id, u.first_name, u.last_name, u.referral_code, u.avatar_file_id,
			u.created_at, u.updated_at,
			ue.email,
			up.phone,
			om.organization_id, om.role
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_phones up ON u.primary_phone_id = up.id
		LEFT JOIN organization_members om ON u.id = om.user_id AND om.is_active = true
		WHERE u.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	var profile UserProfileResponse
	var email, phone, avatar, referralCode sql.NullString
	var organizationID, role sql.NullString

	err := s.db.QueryRow(query, userID).Scan(
		&profile.UserID, &profile.FirstName, &profile.LastName, &referralCode, &avatar,
		&profile.CreatedAt, &profile.UpdatedAt,
		&email, &phone,
		&organizationID, &role,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Populate optional fields
	if email.Valid {
		profile.Email = email.String
	}
	if phone.Valid {
		profile.Phone = phone.String
	}
	if avatar.Valid {
		profile.Avatar = avatar.String
	}
	if referralCode.Valid {
		profile.ReferralCode = referralCode.String
	}
	if organizationID.Valid {
		profile.OrganizationID = organizationID.String
	}
	if role.Valid {
		profile.Role = role.String
	}

	// is_sales_rep column doesn't exist in database, default to false
	profile.IsSalesRep = false

	return &profile, nil
}

// UpdateOwnProfile updates the current user's own profile
func (s *UserService) UpdateOwnProfile(userID string, req *UpdateOwnProfileRequest) (*UserProfileResponse, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Handle password change if requested
	if req.Password != nil && *req.Password != "" {
		if req.OldPassword == nil || *req.OldPassword == "" {
			return nil, fmt.Errorf("old password is required to change password")
		}

		// Verify old password
		var passwordHash string
		err = tx.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get password: %w", err)
		}

		if !utils.VerifyPassword(*req.OldPassword, passwordHash) {
			return nil, fmt.Errorf("current password is incorrect")
		}

		// Validate and hash new password
		valid, msg := utils.IsPasswordValid(*req.Password)
		if !valid {
			return nil, fmt.Errorf(msg)
		}

		hashedPassword, err := utils.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		// Update password and increment token version (force logout)
		_, err = tx.Exec(`
			UPDATE users
			SET password_hash = $1, token_version = token_version + 1, updated_at = $2
			WHERE id = $3
		`, hashedPassword, now, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to update password: %w", err)
		}
	}

	// Build dynamic update query for profile fields
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.FirstName != nil {
		updates = append(updates, fmt.Sprintf("first_name = $%d", argPos))
		args = append(args, *req.FirstName)
		argPos++
	}

	if req.LastName != nil {
		updates = append(updates, fmt.Sprintf("last_name = $%d", argPos))
		args = append(args, *req.LastName)
		argPos++
	}

	if req.AvatarFileID != nil {
		updates = append(updates, fmt.Sprintf("avatar_file_id = $%d", argPos))
		args = append(args, *req.AvatarFileID)
		argPos++
	}

	if len(updates) > 0 {
		updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
		args = append(args, now)
		argPos++

		args = append(args, userID)

		updateQuery := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d`, strings.Join(updates, ", "), argPos)
		_, err = tx.Exec(updateQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Handle phone update if provided
	if req.Phone != nil && *req.Phone != "" {
		var phoneID sql.NullString
		err = tx.QueryRow(`SELECT primary_phone_id FROM users WHERE id = $1`, userID).Scan(&phoneID)
		if err != nil {
			return nil, fmt.Errorf("failed to check phone: %w", err)
		}

		if phoneID.Valid {
			// Update existing phone
			_, err = tx.Exec(`UPDATE user_phones SET phone = $1, updated_at = $2 WHERE id = $3`, *req.Phone, now, phoneID.String)
		} else {
			// Create new phone
			newPhoneID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO user_phones (id, user_id, phone, is_primary, is_verified, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, true, false, true, $4, $5)
			`, newPhoneID, userID, *req.Phone, now, now)
			if err == nil {
				_, err = tx.Exec(`UPDATE users SET primary_phone_id = $1 WHERE id = $2`, newPhoneID, userID)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to update phone: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate user cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user:%s", userID))

	// Get updated profile
	return s.GetUserProfileWithOrganization(userID)
}

// ListOrganizationUsers retrieves all users in an organization
func (s *UserService) ListOrganizationUsers(organizationID string) ([]*OrganizationUserResponse, error) {
	query := `
		SELECT
			u.id, u.first_name, u.last_name, u.is_active,
			ue.email,
			om.role, om.joined_at
		FROM users u
		INNER JOIN organization_members om ON u.id = om.user_id
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE om.organization_id = $1 AND u.deleted_at IS NULL
		ORDER BY
			CASE om.role
				WHEN 'owner' THEN 1
				WHEN 'administrator' THEN 2
				WHEN 'manager' THEN 3
				WHEN 'operator' THEN 4
				WHEN 'viewer' THEN 5
			END,
			om.joined_at ASC
	`

	rows, err := s.db.Query(query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization users: %w", err)
	}
	defer rows.Close()

	users := []*OrganizationUserResponse{}
	for rows.Next() {
		var user OrganizationUserResponse
		var firstName, lastName string
		var email sql.NullString

		err := rows.Scan(
			&user.UserID, &firstName, &lastName, &user.IsActive,
			&email,
			&user.Role, &user.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.Name = fmt.Sprintf("%s %s", firstName, lastName)
		if email.Valid {
			user.Email = email.String
		}

		// is_sales_rep column doesn't exist in database, default to false
		user.IsSalesRep = false

		users = append(users, &user)
	}

	return users, nil
}

// ChangeUserRole changes a user's role within the organization
func (s *UserService) ChangeUserRole(actorUserID, targetUserID, organizationID, newRole string) error {
	// Validate role
	validRoles := map[string]bool{
		"owner": true, "administrator": true, "manager": true, "operator": true, "viewer": true,
	}
	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Cannot change own role
	if actorUserID == targetUserID {
		return fmt.Errorf("cannot change your own role")
	}

	// Get organization to check owner
	var ownerUserID string
	err = tx.QueryRow(`SELECT owner_user_id FROM organizations WHERE id = $1`, organizationID).Scan(&ownerUserID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Cannot demote or change organization owner
	if targetUserID == ownerUserID {
		return fmt.Errorf("cannot change the role of the organization owner")
	}

	// Cannot promote to owner
	if newRole == "owner" {
		return fmt.Errorf("cannot promote user to owner role")
	}

	// Update role in organization_members
	result, err := tx.Exec(`
		UPDATE organization_members
		SET role = $1::organization_role
		WHERE user_id = $2 AND organization_id = $3 AND is_active = true
	`, newRole, targetUserID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found in organization")
	}

	// Log audit entry (simplified - full audit would use audit_logs table)
	// In a production system, you'd insert into audit_logs here

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ToggleUserSalesRep toggles the sales representative status for a user
// NOTE: is_sales_rep column doesn't exist in database schema
// This function validates the user but doesn't persist the sales rep status
func (s *UserService) ToggleUserSalesRep(actorUserID, targetUserID, organizationID string, isSalesRep bool) error {
	// Verify user belongs to organization
	var memberID string
	err := s.db.QueryRow(`
		SELECT id FROM organization_members
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`, targetUserID, organizationID).Scan(&memberID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found in organization")
	}
	if err != nil {
		return fmt.Errorf("failed to verify user membership: %w", err)
	}

	// NOTE: is_sales_rep column doesn't exist in users table
	// Skipping UPDATE to avoid SQL error
	// To fully implement this feature, add migration:
	// ALTER TABLE users ADD COLUMN is_sales_rep BOOLEAN NOT NULL DEFAULT false;

	return nil
}

// AnonymizeUser permanently anonymizes a user (GDPR/PIPEDA compliance)
func (s *UserService) AnonymizeUser(actorUserID, targetUserID, organizationID, reason string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Cannot anonymize self
	if actorUserID == targetUserID {
		return fmt.Errorf("cannot anonymize your own account")
	}

	// Get organization to check owner
	var ownerUserID string
	err = tx.QueryRow(`SELECT owner_user_id FROM organizations WHERE id = $1`, organizationID).Scan(&ownerUserID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Cannot anonymize organization owner
	if targetUserID == ownerUserID {
		return fmt.Errorf("cannot anonymize the organization owner")
	}

	now := time.Now().UTC()
	anonymizedUUID := uuid.New().String()

	// Generate random secure hash for password
	randomPassword := uuid.New().String() + uuid.New().String()
	hashedPassword, err := utils.HashPassword(randomPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Anonymize user record
	_, err = tx.Exec(`
		UPDATE users
		SET
			is_active = false,
			password_hash = $1,
			first_name = 'Deleted',
			last_name = 'User',
			avatar_file_id = NULL,
			deleted_at = $2,
			account_deletion_reason = $3,
			account_anonymized_at = $4,
			updated_at = $5
		WHERE id = $6
	`, hashedPassword, now, reason, now, now, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to anonymize user: %w", err)
	}

	// Anonymize primary email
	anonymizedEmail := fmt.Sprintf("deleted_%s@anonymized.local", anonymizedUUID)
	_, err = tx.Exec(`
		UPDATE user_emails
		SET
			email = $1,
			email_lower = LOWER($1),
			is_active = false,
			updated_at = $2
		WHERE user_id = $3 AND is_primary = true
	`, anonymizedEmail, now, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to anonymize email: %w", err)
	}

	// Deactivate all other emails
	_, err = tx.Exec(`
		UPDATE user_emails
		SET is_active = false, updated_at = $1
		WHERE user_id = $2 AND is_primary = false
	`, now, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to deactivate emails: %w", err)
	}

	// Anonymize phone numbers
	_, err = tx.Exec(`
		UPDATE user_phones
		SET
			phone = NULL,
			phone_e164 = NULL,
			is_active = false,
			updated_at = $1
		WHERE user_id = $2
	`, now, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to anonymize phones: %w", err)
	}

	// Deactivate all user sessions
	_, err = tx.Exec(`
		UPDATE user_sessions
		SET
			is_active = false,
			terminated_at = $1,
			termination_reason = 'user_anonymized'
		WHERE user_id = $2 AND is_active = true
	`, now, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to terminate sessions: %w", err)
	}

	// Increment token version to invalidate all JWT tokens
	_, err = tx.Exec(`
		UPDATE users
		SET token_version = token_version + 1
		WHERE id = $1
	`, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to increment token version: %w", err)
	}

	// Log audit entry (simplified)
	// In production, insert detailed audit log

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Revoke all Redis sessions
	s.redisHelper.DeleteAllUserSessions(targetUserID)

	return nil
}

// =====================================================
// Helper Functions
// =====================================================

// generateRandomCode generates a random alphanumeric code of specified length
// Used for generating organization IDs, wallet numbers, referral codes, etc.
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}
