package oauth2

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/models"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OAuth2Service handles OAuth 2.1 + OIDC business logic
type OAuth2Service struct {
	db          *clients.Database
	redisHelper *redishelper.RedisHelper
	jwtUtil     *utils.JWTUtil
	config      *config.Config
}

// NewOAuth2Service creates a new OAuth2 service
func NewOAuth2Service(db *clients.Database, redisHelper *redishelper.RedisHelper, jwtUtil *utils.JWTUtil, cfg *config.Config) *OAuth2Service {
	return &OAuth2Service{
		db:          db,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
		config:      cfg,
	}
}

// CreateAuthorizationCode creates an authorization code (OAuth 2.1 with required PKCE)
func (s *OAuth2Service) CreateAuthorizationCode(userID string, req *AuthorizeRequest) (*models.OAuthAuthorizationCode, error) {
	// Verify client
	client, err := s.GetClientByClientID(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client")
	}

	if !client.IsActive {
		return nil, fmt.Errorf("client is inactive")
	}

	// OAuth 2.1: PKCE is REQUIRED
	if req.CodeChallenge == "" {
		return nil, fmt.Errorf("code_challenge is required (OAuth 2.1 PKCE)")
	}

	// Verify redirect URI
	if !s.validateRedirectURI(client, req.RedirectURI) {
		return nil, fmt.Errorf("invalid redirect URI")
	}

	// Parse scopes
	scopes := strings.Split(req.Scope, " ")
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	// Generate authorization code
	code := uuid.New().String()
	expiresAt := time.Now().UTC().Add(10 * time.Minute) // 10 minute expiry

	authCode := &models.OAuthAuthorizationCode{
		ID:                  uuid.New().String(),
		Code:                code,
		ClientID:            client.ID,
		UserID:              userID,
		RedirectURI:         req.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           expiresAt,
		CreatedAt:           time.Now().UTC(),
	}

	if req.State != "" {
		authCode.State = sql.NullString{String: req.State, Valid: true}
	}

	query := `
		INSERT INTO oauth_authorization_codes
		(id, code, client_id, user_id, redirect_uri, scopes, state, code_challenge, code_challenge_method, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = s.db.Exec(query,
		authCode.ID,
		authCode.Code,
		authCode.ClientID,
		authCode.UserID,
		authCode.RedirectURI,
		pq.Array(authCode.Scopes),
		authCode.State,
		authCode.CodeChallenge,
		authCode.CodeChallengeMethod,
		authCode.ExpiresAt,
		authCode.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create authorization code: %w", err)
	}

	return authCode, nil
}

// ExchangeAuthorizationCode exchanges auth code for tokens (with PKCE verification)
func (s *OAuth2Service) ExchangeAuthorizationCode(req *TokenRequest) (*TokenResponse, error) {
	// Get authorization code
	authCode, err := s.getAuthorizationCode(req.Code)
	if err != nil {
		return nil, fmt.Errorf("invalid authorization code")
	}

	// Check if already used
	if authCode.UsedAt.Valid {
		return nil, fmt.Errorf("authorization code already used")
	}

	// Check if expired
	if authCode.IsExpired() {
		return nil, fmt.Errorf("authorization code expired")
	}

	// Verify client (Resolve string ID to UUID first)
	client, err := s.GetClientByClientID(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client")
	}

	if authCode.ClientID != client.ID {
		return nil, fmt.Errorf("client mismatch")
	}

	// Verify redirect URI
	if authCode.RedirectURI != req.RedirectURI {
		return nil, fmt.Errorf("redirect URI mismatch")
	}

	// OAuth 2.1: Verify PKCE
	if err := s.verifyPKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, req.CodeVerifier); err != nil {
		return nil, fmt.Errorf("PKCE verification failed: %w", err)
	}

	// Mark code as used
	s.markAuthorizationCodeUsed(authCode.Code)

	// Get user info for OIDC claims
	userInfo, err := s.getUserInfo(authCode.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Get client for token lifetimes
	// Get client for token lifetimes (client already fetched above)
	// client is available from the validation step
	accessTokenExpiry := time.Duration(client.AccessTokenLifetime) * time.Second
	refreshTokenExpiry := time.Duration(client.RefreshTokenLifetime) * time.Second

	// Get user role
	role := userInfo["role"].(string)

	// Generate OIDC access token
	accessToken, accessTokenID, err := s.jwtUtil.GenerateOIDCAccessToken(
		authCode.UserID,
		req.ClientID,
		role,
		authCode.Scopes,
		userInfo,
		accessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Store access token metadata (Use client.ID for DB UUID)
	err = s.storeAccessTokenMetadata(accessTokenID, client.ID, authCode.UserID, authCode.Scopes, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	// Generate refresh token with family tracking (Use client.ID for DB UUID)
	refreshToken, _, err := s.generateRefreshToken(authCode.UserID, client.ID, authCode.Scopes, refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Generate ID token (OIDC)
	idToken, err := s.jwtUtil.GenerateIDToken(authCode.UserID, req.ClientID, "", userInfo, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenExpiry.Seconds()),
		RefreshToken: refreshToken,
		Scope:        strings.Join(authCode.Scopes, " "),
		IDToken:      idToken, // OIDC ID token
	}, nil
}

// RefreshAccessToken refreshes an access token using refresh token
func (s *OAuth2Service) RefreshAccessToken(req *TokenRequest) (*TokenResponse, error) {
	// Validate refresh token
	rtData, err := s.validateRefreshToken(req.RefreshToken)
	if err != nil {
		// Refresh token rotation: if token was already replaced, revoke entire family
		if strings.Contains(err.Error(), "already replaced") {
			s.revokeRefreshTokenFamily(rtData["family_id"])
		}
		return nil, fmt.Errorf("invalid refresh token")
	}

	userID := rtData["user_id"]
	clientID := rtData["client_id"]
	scopesStr := rtData["scopes"]
	scopes := strings.Split(scopesStr, " ")

	// Verify client
	if clientID != req.ClientID {
		return nil, fmt.Errorf("client mismatch")
	}

	// Get user info
	userInfo, err := s.getUserInfo(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Get client for token lifetimes
	client, _ := s.GetClientByClientID(req.ClientID)
	accessTokenExpiry := time.Duration(client.AccessTokenLifetime) * time.Second
	refreshTokenExpiry := time.Duration(client.RefreshTokenLifetime) * time.Second

	role := userInfo["role"].(string)

	// Generate new access token
	accessToken, accessTokenID, err := s.jwtUtil.GenerateOIDCAccessToken(
		userID,
		clientID,
		role,
		scopes,
		userInfo,
		accessTokenExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Store access token metadata
	err = s.storeAccessTokenMetadata(accessTokenID, clientID, userID, scopes, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	// Generate new refresh token (rotation)
	newRefreshToken, _, err := s.rotateRefreshToken(req.RefreshToken, userID, clientID, scopes, refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	// Generate new ID token
	idToken, err := s.jwtUtil.GenerateIDToken(userID, clientID, "", userInfo, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenExpiry.Seconds()),
		RefreshToken: newRefreshToken,
		Scope:        strings.Join(scopes, " "),
		IDToken:      idToken,
	}, nil
}

// ClientCredentials handles client_credentials grant
func (s *OAuth2Service) ClientCredentials(req *TokenRequest) (*TokenResponse, error) {
	// Verify client credentials
	client, err := s.validateClientCredentials(req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Parse scopes
	scopes := []string{}
	if req.Scope != "" {
		scopes = strings.Split(req.Scope, " ")
	}

	// Validate scopes
	if !s.validateClientScopes(client, scopes) {
		return nil, fmt.Errorf("invalid scopes")
	}

	accessTokenExpiry := time.Duration(client.AccessTokenLifetime) * time.Second

	// Generate client token (no user)
	accessToken, accessTokenID, err := s.jwtUtil.GenerateClientToken(req.ClientID, scopes, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client token: %w", err)
	}

	// Store access token metadata
	err = s.storeAccessTokenMetadata(accessTokenID, req.ClientID, "", scopes, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(accessTokenExpiry.Seconds()),
		Scope:       strings.Join(scopes, " "),
	}, nil
}

// IntrospectToken introspects an access token
func (s *OAuth2Service) IntrospectToken(token string) (*IntrospectResponse, error) {
	// Validate JWT
	claims, err := s.jwtUtil.ValidateToken(token)
	if err != nil {
		return &IntrospectResponse{Active: false}, nil
	}

	// Check if revoked
	revoked, err := s.isTokenRevoked(claims.TokenID)
	if err != nil || revoked {
		return &IntrospectResponse{Active: false}, nil
	}

	return &IntrospectResponse{
		Active:    true,
		ClientID:  claims.ClientID,
		UserID:    claims.UserID,
		Scope:     strings.Join(claims.Scopes, " "),
		TokenType: "Bearer",
		Exp:       claims.ExpiresAt.Unix(),
		Iat:       claims.IssuedAt.Unix(),
	}, nil
}

// RevokeToken revokes a token (access or refresh)
func (s *OAuth2Service) RevokeToken(token string, tokenTypeHint string) error {
	if tokenTypeHint == "refresh_token" {
		return s.revokeRefreshToken(token)
	}

	// Revoke access token
	claims, err := s.jwtUtil.ValidateToken(token)
	if err != nil {
		return nil // Per RFC, return success even if invalid
	}

	return s.revokeAccessToken(claims.TokenID)
}

// Helper methods

func (s *OAuth2Service) GetClientByClientID(clientID string) (*models.OAuthClient, error) {
	query := `
		SELECT id, client_id, client_secret_hash, name, description, redirect_uris, client_type,
		       allowed_grant_types, require_pkce, pkce_challenge_method,
		       access_token_lifetime, refresh_token_lifetime, is_active, created_at, updated_at
		FROM oauth_clients
		WHERE client_id = $1 AND is_active = true
	`

	client := &models.OAuthClient{}
	err := s.db.QueryRow(query, clientID).Scan(
		&client.ID,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.Name,
		&client.Description,
		pq.Array(&client.RedirectURIs),
		&client.ClientType,
		pq.Array(&client.AllowedGrantTypes),
		&client.RequirePKCE,
		&client.PKCEChallengeMethod,
		&client.AccessTokenLifetime,
		&client.RefreshTokenLifetime,
		&client.IsActive,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return client, nil
}

func (s *OAuth2Service) validateRedirectURI(client *models.OAuthClient, redirectURI string) bool {
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

func (s *OAuth2Service) verifyPKCE(codeChallenge, method, verifier string) error {
	if verifier == "" {
		return fmt.Errorf("code_verifier required")
	}

	var computed string
	if method == "S256" {
		hash := sha256.Sum256([]byte(verifier))
		computed = base64.RawURLEncoding.EncodeToString(hash[:])
	} else {
		computed = verifier
	}

	if computed != codeChallenge {
		return fmt.Errorf("code_verifier does not match code_challenge")
	}

	return nil
}

func (s *OAuth2Service) getAuthorizationCode(code string) (*models.OAuthAuthorizationCode, error) {
	query := `
		SELECT id, code, client_id, user_id, redirect_uri, scopes, state,
		       code_challenge, code_challenge_method, expires_at, used_at, created_at
		FROM oauth_authorization_codes
		WHERE code = $1
	`

	authCode := &models.OAuthAuthorizationCode{}
	err := s.db.QueryRow(query, code).Scan(
		&authCode.ID,
		&authCode.Code,
		&authCode.ClientID,
		&authCode.UserID,
		&authCode.RedirectURI,
		pq.Array(&authCode.Scopes),
		&authCode.State,
		&authCode.CodeChallenge,
		&authCode.CodeChallengeMethod,
		&authCode.ExpiresAt,
		&authCode.UsedAt,
		&authCode.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return authCode, nil
}

func (s *OAuth2Service) markAuthorizationCodeUsed(code string) error {
	query := `UPDATE oauth_authorization_codes SET used_at = $1 WHERE code = $2`
	_, err := s.db.Exec(query, time.Now().UTC(), code)
	return err
}

func (s *OAuth2Service) getUserInfo(userID string) (map[string]interface{}, error) {
	query := `
		SELECT u.id, u.first_name, u.last_name, u.avatar_file_id, u.timezone, u.locale, u.updated_at,
		       ue.email, ue.is_verified as email_verified,
		       r.name as role
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	var userID_ string
	var firstName, lastName, email, role string
	var avatarURL, timezone, locale sql.NullString
	var emailVerified bool
	var updatedAt time.Time

	err := s.db.QueryRow(query, userID).Scan(
		&userID_, &firstName, &lastName, &avatarURL, &timezone, &locale, &updatedAt,
		&email, &emailVerified, &role,
	)
	if err != nil {
		return nil, err
	}

	userInfo := map[string]interface{}{
		"user_id":        userID_,
		"name":           firstName + " " + lastName,
		"given_name":     firstName,
		"family_name":    lastName,
		"email":          email,
		"email_verified": emailVerified,
		"role":           role,
		"updated_at":     updatedAt.Unix(),
	}

	if avatarURL.Valid {
		userInfo["picture"] = avatarURL.String
	}
	if timezone.Valid {
		userInfo["zoneinfo"] = timezone.String
	}
	if locale.Valid {
		userInfo["locale"] = locale.String
	}

	return userInfo, nil
}

func (s *OAuth2Service) storeAccessTokenMetadata(jti, clientID, userID string, scopes []string, expiry time.Duration) error {
	query := `
		INSERT INTO oauth_access_tokens (id, jti, client_id, user_id, scopes, issued_at, expires_at, use_count)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, 0)
	`

	now := time.Now().UTC()
	expiresAt := now.Add(expiry)

	_, err := s.db.Exec(query, uuid.New().String(), jti, clientID, userID, pq.Array(scopes), now, expiresAt)
	return err
}

func (s *OAuth2Service) generateRefreshToken(userID, clientID string, scopes []string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	familyID := uuid.New().String()
	tokenHash := s.hashToken(tokenID)

	query := `
		INSERT INTO oauth_refresh_tokens (id, token_hash, client_id, user_id, scopes, token_family_id, issued_at, expires_at)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8)
	`

	now := time.Now().UTC()
	expiresAt := now.Add(expiry)

	_, err := s.db.Exec(query, uuid.New().String(), tokenHash, clientID, userID, pq.Array(scopes), familyID, now, expiresAt)
	if err != nil {
		return "", "", err
	}

	// Generate JWT
	refreshToken, _, err := s.jwtUtil.GenerateRefreshToken(userID, clientID, expiry)
	if err != nil {
		return "", "", err
	}

	return refreshToken, tokenID, nil
}

func (s *OAuth2Service) rotateRefreshToken(oldToken, userID, clientID string, scopes []string, expiry time.Duration) (string, string, error) {
	// TODO: Implement proper refresh token rotation with family tracking
	return s.generateRefreshToken(userID, clientID, scopes, expiry)
}

func (s *OAuth2Service) validateRefreshToken(token string) (map[string]string, error) {
	claims, err := s.jwtUtil.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	tokenHash := s.hashToken(claims.TokenID)

	// Check if token exists and is valid
	var userID, clientID string
	var scopes []string
	var revokedAt sql.NullTime

	query := `
		SELECT user_id, client_id, scopes, revoked_at
		FROM oauth_refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()
	`

	err = s.db.QueryRow(query, tokenHash).Scan(&userID, &clientID, pq.Array(&scopes), &revokedAt)
	if err != nil {
		return nil, err
	}

	if revokedAt.Valid {
		return nil, fmt.Errorf("token revoked")
	}

	return map[string]string{
		"user_id":   userID,
		"client_id": clientID,
		"scopes":    strings.Join(scopes, " "),
	}, nil
}

func (s *OAuth2Service) validateClientCredentials(clientID, clientSecret string) (*models.OAuthClient, error) {
	client, err := s.GetClientByClientID(clientID)
	if err != nil {
		return nil, err
	}

	if client.ClientType == "public" {
		return client, nil
	}

	// Verify secret
	if !client.ClientSecretHash.Valid || !utils.VerifyPassword(clientSecret, client.ClientSecretHash.String) {
		return nil, fmt.Errorf("invalid credentials")
	}

	return client, nil
}

func (s *OAuth2Service) validateClientScopes(client *models.OAuthClient, requestedScopes []string) bool {
	// TODO: Implement scope validation against client allowed scopes
	return true
}

func (s *OAuth2Service) isTokenRevoked(jti string) (bool, error) {
	var revokedAt sql.NullTime
	query := `SELECT revoked_at FROM oauth_access_tokens WHERE jti = $1`
	err := s.db.QueryRow(query, jti).Scan(&revokedAt)
	if err != nil {
		return false, err
	}
	return revokedAt.Valid, nil
}

func (s *OAuth2Service) revokeAccessToken(jti string) error {
	query := `UPDATE oauth_access_tokens SET revoked_at = $1 WHERE jti = $2`
	_, err := s.db.Exec(query, time.Now().UTC(), jti)
	return err
}

func (s *OAuth2Service) revokeRefreshToken(token string) error {
	claims, err := s.jwtUtil.ValidateToken(token)
	if err != nil {
		return nil
	}

	tokenHash := s.hashToken(claims.TokenID)
	query := `UPDATE oauth_refresh_tokens SET revoked_at = $1 WHERE token_hash = $2`
	_, err = s.db.Exec(query, time.Now().UTC(), tokenHash)
	return err
}

func (s *OAuth2Service) revokeRefreshTokenFamily(familyID string) error {
	query := `UPDATE oauth_refresh_tokens SET revoked_at = $1 WHERE token_family_id = $2`
	_, err := s.db.Exec(query, time.Now().UTC(), familyID)
	return err
}

func (s *OAuth2Service) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
