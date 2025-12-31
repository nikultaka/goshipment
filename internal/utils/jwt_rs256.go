package utils

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTUtilRS256 provides JWT operations with RS256 signing
type JWTUtilRS256 struct {
	keyPair *RSAKeyPair
	issuer  string
}

// NewJWTUtilRS256 creates a new JWT utility with RS256 signing
func NewJWTUtilRS256(keyPair *RSAKeyPair, issuer string) *JWTUtilRS256 {
	return &JWTUtilRS256{
		keyPair: keyPair,
		issuer:  issuer,
	}
}

// GenerateAccessToken generates a new access token signed with RS256
func (j *JWTUtilRS256) GenerateAccessToken(userID, clientID, role string, scopes []string, expiry time.Duration) (string, string, error) {
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
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyPair.KeyID

	tokenString, err := token.SignedString(j.keyPair.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateRefreshToken generates a new refresh token signed with RS256
func (j *JWTUtilRS256) GenerateRefreshToken(userID, clientID string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		UserID:   userID,
		ClientID: clientID,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyPair.KeyID

	tokenString, err := token.SignedString(j.keyPair.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateClientToken generates a token for client credentials flow (no user)
func (j *JWTUtilRS256) GenerateClientToken(clientID string, scopes []string, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		ClientID: clientID,
		Scopes:   scopes,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   clientID,
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyPair.KeyID

	tokenString, err := token.SignedString(j.keyPair.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign client token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateOIDCAccessToken generates an OIDC-compliant access token with RS256
func (j *JWTUtilRS256) GenerateOIDCAccessToken(userID, clientID, role string, scopes []string, userInfo map[string]interface{}, expiry time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := OIDCClaims{
		UserID:   userID,
		ClientID: clientID,
		Role:     role,
		Scopes:   scopes,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	// Add OIDC profile claims if present
	populateOIDCClaims(&claims, userInfo)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyPair.KeyID

	tokenString, err := token.SignedString(j.keyPair.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateIDToken generates an OIDC ID Token with RS256
func (j *JWTUtilRS256) GenerateIDToken(userID, clientID, nonce string, userInfo map[string]interface{}, expiry time.Duration) (string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	// Extended OIDC claims with nonce
	claims := struct {
		OIDCClaims
		Nonce string `json:"nonce,omitempty"`
	}{
		OIDCClaims: OIDCClaims{
			UserID:   userID,
			ClientID: clientID,
			TokenID:  tokenID,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    j.issuer,
				Subject:   userID,
				Audience:  jwt.ClaimStrings{clientID},
				ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				ID:        tokenID,
			},
		},
		Nonce: nonce,
	}

	// Add user info
	populateOIDCClaims(&claims.OIDCClaims, userInfo)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyPair.KeyID

	tokenString, err := token.SignedString(j.keyPair.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign ID token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTUtilRS256) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Verify key ID matches
		if kid, ok := token.Header["kid"].(string); ok {
			if kid != j.keyPair.KeyID {
				return nil, fmt.Errorf("key ID mismatch: expected %s, got %s", j.keyPair.KeyID, kid)
			}
		}
		return j.keyPair.PublicKey, nil
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

// ValidateOIDCToken validates and parses an OIDC token
func (j *JWTUtilRS256) ValidateOIDCToken(tokenString string) (*OIDCClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &OIDCClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if kid, ok := token.Header["kid"].(string); ok {
			if kid != j.keyPair.KeyID {
				return nil, fmt.Errorf("key ID mismatch")
			}
		}
		return j.keyPair.PublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*OIDCClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ParseTokenWithoutValidation parses a token without validating
func (j *JWTUtilRS256) ParseTokenWithoutValidation(tokenString string) (*JWTClaims, error) {
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
func (j *JWTUtilRS256) GetTokenID(tokenString string) (string, error) {
	claims, err := j.ParseTokenWithoutValidation(tokenString)
	if err != nil {
		return "", err
	}
	return claims.TokenID, nil
}

// GetKeyPair returns the RSA key pair used by this utility
func (j *JWTUtilRS256) GetKeyPair() *RSAKeyPair {
	return j.keyPair
}

// GetPublicKey returns the public key
func (j *JWTUtilRS256) GetPublicKey() *rsa.PublicKey {
	return j.keyPair.PublicKey
}

// GetKeyID returns the key ID
func (j *JWTUtilRS256) GetKeyID() string {
	return j.keyPair.KeyID
}

// Helper function to populate OIDC claims from userInfo map
func populateOIDCClaims(claims *OIDCClaims, userInfo map[string]interface{}) {
	if name, ok := userInfo["name"].(string); ok {
		claims.Name = name
	}
	if givenName, ok := userInfo["given_name"].(string); ok {
		claims.GivenName = givenName
	}
	if familyName, ok := userInfo["family_name"].(string); ok {
		claims.FamilyName = familyName
	}
	if middleName, ok := userInfo["middle_name"].(string); ok {
		claims.MiddleName = middleName
	}
	if nickname, ok := userInfo["nickname"].(string); ok {
		claims.Nickname = nickname
	}
	if preferredUsername, ok := userInfo["preferred_username"].(string); ok {
		claims.PreferredUsername = preferredUsername
	}
	if email, ok := userInfo["email"].(string); ok {
		claims.Email = email
	}
	if emailVerified, ok := userInfo["email_verified"].(bool); ok {
		claims.EmailVerified = emailVerified
	}
	if picture, ok := userInfo["picture"].(string); ok {
		claims.Picture = picture
	}
	if website, ok := userInfo["website"].(string); ok {
		claims.Website = website
	}
	if gender, ok := userInfo["gender"].(string); ok {
		claims.Gender = gender
	}
	if birthdate, ok := userInfo["birthdate"].(string); ok {
		claims.Birthdate = birthdate
	}
	if locale, ok := userInfo["locale"].(string); ok {
		claims.Locale = locale
	}
	if zoneinfo, ok := userInfo["zoneinfo"].(string); ok {
		claims.Zoneinfo = zoneinfo
	}
	if phoneNumber, ok := userInfo["phone_number"].(string); ok {
		claims.PhoneNumber = phoneNumber
	}
	if phoneVerified, ok := userInfo["phone_number_verified"].(bool); ok {
		claims.PhoneVerified = phoneVerified
	}
	if updatedAt, ok := userInfo["updated_at"].(int64); ok {
		claims.UpdatedAt = updatedAt
	}
}
