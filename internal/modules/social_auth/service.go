package social_auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SocialAuthService handles social authentication business logic
type SocialAuthService struct {
	db               *clients.Database
	jwtUtil          *utils.JWTUtil
	redisHelper      *redishelper.RedisHelper
	config           *config.Config
	stateManager     *StateManager
	googleProvider   *GoogleProvider
	facebookProvider *FacebookProvider
}

// NewSocialAuthService creates a new social auth service
func NewSocialAuthService(db *clients.Database, jwtUtil *utils.JWTUtil, redisHelper *redishelper.RedisHelper, cfg *config.Config) *SocialAuthService {
	return &SocialAuthService{
		db:               db,
		jwtUtil:          jwtUtil,
		redisHelper:      redisHelper,
		config:           cfg,
		stateManager:     NewStateManager(redisHelper),
		googleProvider:   NewGoogleProvider(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_REDIRECT_URL"),
		),
		facebookProvider: NewFacebookProvider(
			os.Getenv("FACEBOOK_APP_ID"),
			os.Getenv("FACEBOOK_APP_SECRET"),
			os.Getenv("FACEBOOK_REDIRECT_URL"),
		),
	}
}

// GetAuthorizationURL generates an OAuth authorization URL with state parameter
func (s *SocialAuthService) GetAuthorizationURL(provider OAuthProvider) (string, error) {
	state, err := s.stateManager.GenerateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := provider.GetAuthURL(state)
	return authURL, nil
}

// HandleCallback processes the OAuth callback and returns JWT tokens
func (s *SocialAuthService) HandleCallback(ctx context.Context, provider OAuthProvider, code, state string) (*OAuthLoginResponse, error) {
	// 1. Validate state parameter (CSRF protection)
	if !s.stateManager.ValidateState(state) {
		return nil, fmt.Errorf("invalid or expired state parameter")
	}

	// 2. Exchange authorization code for access token
	tokens, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// 3. Fetch user information from provider
	userInfo, err := provider.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from provider: %w", err)
	}

	// Validate email
	if userInfo.Email == "" {
		return nil, fmt.Errorf("email not provided by OAuth provider")
	}

	// 4. Check if OAuth account already exists
	userID, exists, err := s.checkOAuthAccountExists(provider.GetProviderName(), userInfo.ProviderUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check OAuth account: %w", err)
	}

	if exists {
		// 5a. Update existing OAuth account tokens
		if err := s.updateOAuthTokens(userID, provider.GetProviderName(), userInfo.ProviderUserID, tokens); err != nil {
			return nil, fmt.Errorf("failed to update OAuth tokens: %w", err)
		}
	} else {
		// 5b. Check if user exists by email (auto-link)
		existingUserID, err := s.findUserByEmail(userInfo.Email)
		if err == nil && existingUserID != "" {
			// Link OAuth account to existing user
			userID = existingUserID
			if err := s.linkOAuthAccount(userID, provider.GetProviderName(), userInfo, tokens); err != nil {
				return nil, fmt.Errorf("failed to link OAuth account: %w", err)
			}
		} else {
			// Create new user from OAuth data
			newUserID, err := s.createUserFromOAuth(provider.GetProviderName(), userInfo, tokens)
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			userID = newUserID
		}
	}

	// 6. Update last login timestamp
	s.updateLastLogin(userID)

	// 7. Get complete user details
	user, userEmail, roleName, err := s.getUserDetails(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	// 8. Generate JWT tokens
	accessToken, _, err := s.jwtUtil.GenerateAccessToken(
		userID,
		"web",
		roleName,
		[]string{"read", "write"},
		s.config.OAuth.AccessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshTokenID, err := s.jwtUtil.GenerateRefreshToken(
		userID,
		"web",
		s.config.OAuth.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 9. Store refresh token in Redis
	s.storeRefreshToken(userID, refreshTokenID, s.config.OAuth.RefreshTokenExpiry)

	// 10. Build and return response
	return &OAuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.OAuth.AccessTokenExpiry.Seconds()),
		User: UserResponse{
			ID:            userID,
			ExternalID:    user.ExternalID,
			Email:         userEmail,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			Avatar:        user.AvatarURL.String,
			Role:          roleName,
			EmailVerified: true, // OAuth emails are verified
			Provider:      provider.GetProviderName(),
			CreatedAt:     user.CreatedAt,
		},
	}, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// checkOAuthAccountExists checks if an OAuth account exists and returns the user_id
func (s *SocialAuthService) checkOAuthAccountExists(provider, providerUserID string) (string, bool, error) {
	query := `
		SELECT uoa.user_id
		FROM user_oauth_accounts uoa
		INNER JOIN users u ON uoa.user_id = u.id
		WHERE uoa.provider = $1 AND uoa.provider_user_id = $2
		  AND u.deleted_at IS NULL
		LIMIT 1
	`

	var userID string
	err := s.db.QueryRow(query, provider, providerUserID).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return userID, true, nil
}

// findUserByEmail finds a user by email address
func (s *SocialAuthService) findUserByEmail(email string) (string, error) {
	query := `
		SELECT u.id
		FROM users u
		INNER JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE LOWER(ue.email) = LOWER($1) AND u.deleted_at IS NULL
		LIMIT 1
	`

	var userID string
	err := s.db.QueryRow(query, email).Scan(&userID)
	return userID, err
}

// createUserFromOAuth creates a new user from OAuth data
func (s *SocialAuthService) createUserFromOAuth(provider string, userInfo *OAuthUserInfo, tokens *OAuthTokens) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	userID := uuid.New().String()
	emailID := uuid.New().String()
	oauthAccountID := uuid.New().String()
	externalID := fmt.Sprintf("usr_%s", uuid.New().String()[:8])

	// 1. Create user record
	userQuery := `
		INSERT INTO users (
			id, external_id, first_name, last_name,
			timezone, locale, date_format, time_format,
			is_active, is_verified, verification_level, token_version,
			has_2fa_enabled, privacy_consent, marketing_consent, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			'UTC', 'en_CA', 'YYYY-MM-DD', '24h',
			true, true, 1, 1,
			false, '{}', false, '{}',
			$5, $6
		)
	`

	_, err = tx.Exec(
		userQuery,
		userID, externalID,
		utils.SanitizeString(userInfo.FirstName),
		utils.SanitizeString(userInfo.LastName),
		now, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// 2. Create user email record
	emailQuery := `
		INSERT INTO user_emails (
			id, user_id, email, is_primary, is_recovery, is_verified, is_active,
			verified_at, created_at, updated_at
		) VALUES ($1, $2, $3, true, false, true, true, $4, $5, $6)
	`

	_, err = tx.Exec(
		emailQuery,
		emailID, userID, utils.SanitizeString(userInfo.Email),
		now, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create user email: %w", err)
	}

	// 3. Update user with primary_email_id
	_, err = tx.Exec(`UPDATE users SET primary_email_id = $1 WHERE id = $2`, emailID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to set primary email: %w", err)
	}

	// 4. Get or create default "user" role
	var roleID string
	err = tx.QueryRow(`SELECT id FROM roles WHERE name = 'user' AND is_active = true LIMIT 1`).Scan(&roleID)
	if err == sql.ErrNoRows {
		// Create default user role if it doesn't exist
		roleID = uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO roles (id, name, display_name, is_system_role, is_admin_role, hierarchy_level, permissions, is_active, created_at, updated_at)
			VALUES ($1, 'user', 'User', true, false, 0, '[]', true, $2, $3)
		`, roleID, now, now)
		if err != nil {
			return "", fmt.Errorf("failed to create default role: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to get role: %w", err)
	}

	// 5. Assign role to user
	userRoleID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_roles (id, user_id, role_id, granted_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, $5, $6)
	`, userRoleID, userID, roleID, now, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to assign role: %w", err)
	}

	// 6. Create OAuth account link
	encryptedAccessToken, err := s.encryptToken(tokens.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := s.encryptToken(tokens.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	expiresAt := now.Add(time.Duration(tokens.ExpiresIn) * time.Second)

	oauthQuery := `
		INSERT INTO user_oauth_accounts (
			id, user_id, provider, provider_user_id, email, avatar_url,
			access_token_encrypted, refresh_token_encrypted, expires_at,
			is_primary, token_type, scopes, linked_at, last_login_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $11, $12, $13)
	`

	_, err = tx.Exec(
		oauthQuery,
		oauthAccountID, userID, provider, userInfo.ProviderUserID,
		userInfo.Email, userInfo.AvatarURL,
		encryptedAccessToken, encryptedRefreshToken, expiresAt,
		tokens.TokenType, pq.Array([]string{tokens.Scope}), now, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth account: %w", err)
	}

	// 7. Commit transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return userID, nil
}

// linkOAuthAccount links an OAuth account to an existing user
func (s *SocialAuthService) linkOAuthAccount(userID, provider string, userInfo *OAuthUserInfo, tokens *OAuthTokens) error {
	now := time.Now().UTC()
	oauthAccountID := uuid.New().String()

	encryptedAccessToken, err := s.encryptToken(tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := s.encryptToken(tokens.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	expiresAt := now.Add(time.Duration(tokens.ExpiresIn) * time.Second)

	query := `
		INSERT INTO user_oauth_accounts (
			id, user_id, provider, provider_user_id, email, avatar_url,
			access_token_encrypted, refresh_token_encrypted, expires_at,
			is_primary, token_type, scopes, linked_at, last_login_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $11, $12, $13)
	`

	_, err = s.db.Exec(
		query,
		oauthAccountID, userID, provider, userInfo.ProviderUserID,
		userInfo.Email, userInfo.AvatarURL,
		encryptedAccessToken, encryptedRefreshToken, expiresAt,
		tokens.TokenType, pq.Array([]string{tokens.Scope}), now, now,
	)

	return err
}

// updateOAuthTokens updates OAuth tokens for an existing account
func (s *SocialAuthService) updateOAuthTokens(userID, provider, providerUserID string, tokens *OAuthTokens) error {
	now := time.Now().UTC()

	encryptedAccessToken, err := s.encryptToken(tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encryptedRefreshToken, err := s.encryptToken(tokens.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	expiresAt := now.Add(time.Duration(tokens.ExpiresIn) * time.Second)

	query := `
		UPDATE user_oauth_accounts
		SET access_token_encrypted = $1,
			refresh_token_encrypted = $2,
			expires_at = $3,
			token_type = $4,
			last_login_at = $5
		WHERE provider = $6 AND provider_user_id = $7
	`

	_, err = s.db.Exec(
		query,
		encryptedAccessToken, encryptedRefreshToken, expiresAt,
		tokens.TokenType, now, provider, providerUserID,
	)

	return err
}

// getUserDetails retrieves complete user details with email and role
func (s *SocialAuthService) getUserDetails(userID string) (*UserDetails, string, string, error) {
	query := `
		SELECT
			u.id, u.external_id, u.first_name, u.last_name, u.avatar_file_id,
			u.is_active, u.is_verified, u.created_at,
			ue.email,
			r.name as role_name
		FROM users u
		INNER JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE u.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	var user UserDetails
	var email string
	var roleName sql.NullString

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.ExternalID, &user.FirstName, &user.LastName, &user.AvatarURL,
		&user.IsActive, &user.IsVerified, &user.CreatedAt,
		&email,
		&roleName,
	)
	if err != nil {
		return nil, "", "", err
	}

	role := "user"
	if roleName.Valid && roleName.String != "" {
		role = roleName.String
	}

	return &user, email, role, nil
}

// UserDetails holds basic user information
type UserDetails struct {
	ID         string
	ExternalID string
	FirstName  string
	LastName   string
	AvatarURL  sql.NullString
	IsActive   bool
	IsVerified bool
	CreatedAt  time.Time
}

// updateLastLogin updates the last login timestamp
func (s *SocialAuthService) updateLastLogin(userID string) {
	now := time.Now().UTC()
	s.db.Exec(`UPDATE users SET last_login_at = $1 WHERE id = $2`, now, userID)
}

// storeRefreshToken stores refresh token in Redis
func (s *SocialAuthService) storeRefreshToken(userID, tokenID string, expiry time.Duration) {
	key := fmt.Sprintf("refresh_token:%s", tokenID)
	data := map[string]interface{}{
		"user_id":    userID,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	s.redisHelper.CacheSet(key, data, expiry)
}

// encryptToken encrypts an OAuth token using AES-256-GCM
func (s *SocialAuthService) encryptToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}

	// Use JWT secret as encryption key (first 32 bytes for AES-256)
	key := []byte(s.config.OAuth.JWTSecret)
	if len(key) < 32 {
		// Pad key if it's too short
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		key = key[:32]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
