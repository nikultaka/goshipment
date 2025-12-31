package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID   string   `json:"user_id,omitempty"`
	ClientID string   `json:"client_id"`
	Role     string   `json:"role,omitempty"`
	Scopes   []string `json:"scopes"`
	TokenID  string   `json:"jti"`
	jwt.RegisteredClaims
}

// JWTUtil provides JWT operations
type JWTUtil struct {
	secret string
	issuer string
}

// NewJWTUtil creates a new JWT utility
func NewJWTUtil(secret, issuer string) *JWTUtil {
	return &JWTUtil{
		secret: secret,
		issuer: issuer,
	}
}

// GenerateAccessToken generates a new access token
func (j *JWTUtil) GenerateAccessToken(userID, clientID, role string, scopes []string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		UserID:   userID,
		ClientID: clientID,
		Role:     role,
		Scopes:   scopes,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateRefreshToken generates a new refresh token
func (j *JWTUtil) GenerateRefreshToken(userID, clientID string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		UserID:   userID,
		ClientID: clientID,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateClientToken generates a token for client credentials flow (no user)
func (j *JWTUtil) GenerateClientToken(clientID string, scopes []string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		ClientID: clientID,
		Scopes:   scopes,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   clientID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign client token: %w", err)
	}

	return tokenString, tokenID, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTUtil) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Additional validation
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("token has expired")
		}

		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ParseTokenWithoutValidation parses a token without validating (useful for getting claims from expired tokens)
func (j *JWTUtil) ParseTokenWithoutValidation(tokenString string) (*JWTClaims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GetTokenID extracts the token ID from a token without full validation
func (j *JWTUtil) GetTokenID(tokenString string) (string, error) {
	claims, err := j.ParseTokenWithoutValidation(tokenString)
	if err != nil {
		return "", err
	}
	return claims.TokenID, nil
}

// HasScope checks if the token has a specific scope
func (c *JWTClaims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the token has any of the specified scopes
func (c *JWTClaims) HasAnyScope(scopes []string) bool {
	for _, requiredScope := range scopes {
		if c.HasScope(requiredScope) {
			return true
		}
	}
	return false
}

// HasAllScopes checks if the token has all of the specified scopes
func (c *JWTClaims) HasAllScopes(scopes []string) bool {
	for _, requiredScope := range scopes {
		if !c.HasScope(requiredScope) {
			return false
		}
	}
	return true
}
