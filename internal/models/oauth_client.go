package models

import (
	"database/sql"
	"time"
)

// OAuthScope represents an OAuth scope
type OAuthScope struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"` // e.g. shipments.create
	Description sql.NullString `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// OAuthClient represents an OAuth 2.1 client application (updated schema)
type OAuthClient struct {
	ID                    string         `json:"id" db:"id"`
	ClientID              string         `json:"client_id" db:"client_id"`
	ClientSecretHash      sql.NullString `json:"-" db:"client_secret_hash"` // NULL for public clients
	Name                  string         `json:"name" db:"name"`
	Description           sql.NullString `json:"description,omitempty" db:"description"`
	RedirectURIs          []string       `json:"redirect_uris" db:"redirect_uris"` // TEXT[] - must be HTTPS
	Scopes                []string       `json:"scopes" db:"scopes"` // TEXT[] - allowed scopes
	ClientType            string         `json:"client_type" db:"client_type"` // confidential, public
	AllowedGrantTypes     []string       `json:"allowed_grant_types" db:"allowed_grant_types"` // TEXT[]
	RequirePKCE           bool           `json:"require_pkce" db:"require_pkce"`
	PKCEChallengeMethod   string         `json:"pkce_challenge_method" db:"pkce_challenge_method"` // S256
	AccessTokenLifetime   int            `json:"access_token_lifetime" db:"access_token_lifetime"` // seconds
	RefreshTokenLifetime  int            `json:"refresh_token_lifetime" db:"refresh_token_lifetime"` // seconds
	IsActive              bool           `json:"is_active" db:"is_active"`
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at" db:"updated_at"`
}

// OAuthClientScope represents the many-to-many relationship between clients and scopes
type OAuthClientScope struct {
	ID       string `json:"id" db:"id"`
	ClientID string `json:"client_id" db:"client_id"`
	ScopeID  string `json:"scope_id" db:"scope_id"`
}

// OAuthAuthorizationCode represents an OAuth 2.1 authorization code with PKCE
type OAuthAuthorizationCode struct {
	ID                  string       `json:"id" db:"id"`
	Code                string       `json:"code" db:"code"`
	ClientID            string       `json:"client_id" db:"client_id"` // FK to oauth_clients
	UserID              string       `json:"user_id" db:"user_id"` // FK to users
	RedirectURI         string       `json:"redirect_uri" db:"redirect_uri"`
	Scopes              []string     `json:"scopes" db:"scopes"` // TEXT[]
	State               sql.NullString `json:"state,omitempty" db:"state"`
	CodeChallenge       string       `json:"-" db:"code_challenge"` // Required PKCE
	CodeChallengeMethod string       `json:"code_challenge_method" db:"code_challenge_method"` // S256
	ExpiresAt           time.Time    `json:"expires_at" db:"expires_at"`
	UsedAt              sql.NullTime `json:"used_at,omitempty" db:"used_at"`
	CreatedAt           time.Time    `json:"created_at" db:"created_at"`
}

// OAuthAccessToken represents OAuth access token metadata (JWT metadata)
type OAuthAccessToken struct {
	ID                    string         `json:"id" db:"id"`
	JTI                   string         `json:"jti" db:"jti"` // JWT ID
	ClientID              string         `json:"client_id" db:"client_id"`
	UserID                sql.NullString `json:"user_id,omitempty" db:"user_id"` // null for client_credentials
	Scopes                []string       `json:"scopes" db:"scopes"` // TEXT[]
	IssuedAt              time.Time      `json:"issued_at" db:"issued_at"`
	ExpiresAt             time.Time      `json:"expires_at" db:"expires_at"`
	RevokedAt             sql.NullTime   `json:"revoked_at,omitempty" db:"revoked_at"`
	LastUsedAt            sql.NullTime   `json:"last_used_at,omitempty" db:"last_used_at"`
	UseCount              int            `json:"use_count" db:"use_count"`
	IssuingIPAddress      sql.NullString `json:"issuing_ip_address,omitempty" db:"issuing_ip_address"`
	IssuingUserAgentHash  sql.NullString `json:"issuing_user_agent_hash,omitempty" db:"issuing_user_agent_hash"`
}

// OAuthRefreshToken represents OAuth refresh token with rotation support
type OAuthRefreshToken struct {
	ID            string         `json:"id" db:"id"`
	TokenHash     string         `json:"-" db:"token_hash"` // SHA-256 hashed
	ClientID      string         `json:"client_id" db:"client_id"`
	UserID        sql.NullString `json:"user_id,omitempty" db:"user_id"`
	Scopes        []string       `json:"scopes" db:"scopes"` // TEXT[]
	TokenFamilyID string         `json:"token_family_id" db:"token_family_id"` // Links all tokens in chain
	ParentTokenID sql.NullString `json:"parent_token_id,omitempty" db:"parent_token_id"` // Previous RT
	ReplacedBy    sql.NullString `json:"replaced_by,omitempty" db:"replaced_by"` // Next RT in chain
	IssuedAt      time.Time      `json:"issued_at" db:"issued_at"`
	ExpiresAt     time.Time      `json:"expires_at" db:"expires_at"`
	RevokedAt     sql.NullTime   `json:"revoked_at,omitempty" db:"revoked_at"`
}

// JWTSigningKey represents a JWT signing key (for key rotation)
type JWTSigningKey struct {
	ID            string         `json:"id" db:"id"`
	KID           string         `json:"kid" db:"kid"` // Key ID
	Algorithm     string         `json:"algorithm" db:"algorithm"` // RS256, ES256
	KeyType       string         `json:"key_type" db:"key_type"` // RSA, EC
	RSAPublicKey  sql.NullString `json:"rsa_public_key,omitempty" db:"rsa_public_key"`
	RSAPrivateKey sql.NullString `json:"-" db:"rsa_private_key"` // Encrypted PEM
	ECPublicKey   sql.NullString `json:"ec_public_key,omitempty" db:"ec_public_key"`
	ECPrivateKey  sql.NullString `json:"-" db:"ec_private_key"` // Encrypted PEM
	IsActive      bool           `json:"is_active" db:"is_active"`
	IsPrimary     bool           `json:"is_primary" db:"is_primary"` // Which key signs new tokens
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

// IsExpired returns true if the authorization code is expired
func (c *OAuthAuthorizationCode) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsValid returns true if the authorization code is valid
func (c *OAuthAuthorizationCode) IsValid() bool {
	return !c.UsedAt.Valid && !c.IsExpired()
}

// IsRevoked returns true if the access token is revoked
func (t *OAuthAccessToken) IsRevoked() bool {
	return t.RevokedAt.Valid
}

// IsExpired returns true if the access token is expired
func (t *OAuthAccessToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked returns true if the refresh token is revoked
func (t *OAuthRefreshToken) IsRevoked() bool {
	return t.RevokedAt.Valid
}

// IsExpired returns true if the refresh token is expired
func (t *OAuthRefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
