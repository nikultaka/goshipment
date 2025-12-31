package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// OIDCClaims represents OIDC-compliant JWT claims (extends JWTClaims)
type OIDCClaims struct {
	// Standard OIDC claims
	Name              string `json:"name,omitempty"`              // Full name
	GivenName         string `json:"given_name,omitempty"`        // First name
	FamilyName        string `json:"family_name,omitempty"`       // Last name
	MiddleName        string `json:"middle_name,omitempty"`       // Middle name
	Nickname          string `json:"nickname,omitempty"`          // Casual name
	PreferredUsername string `json:"preferred_username,omitempty"` // Username
	Profile           string `json:"profile,omitempty"`           // Profile page URL
	Picture           string `json:"picture,omitempty"`           // Avatar URL
	Website           string `json:"website,omitempty"`           // Web page
	Email             string `json:"email,omitempty"`             // Email address
	EmailVerified     bool   `json:"email_verified,omitempty"`    // Email verified
	Gender            string `json:"gender,omitempty"`            // Gender
	Birthdate         string `json:"birthdate,omitempty"`         // Birthday YYYY-MM-DD
	Zoneinfo          string `json:"zoneinfo,omitempty"`          // Timezone
	Locale            string `json:"locale,omitempty"`            // Locale
	PhoneNumber       string `json:"phone_number,omitempty"`      // Phone number
	PhoneVerified     bool   `json:"phone_number_verified,omitempty"` // Phone verified
	UpdatedAt         int64  `json:"updated_at,omitempty"`        // Unix timestamp

	// Custom claims
	UserID   string   `json:"user_id,omitempty"`
	ClientID string   `json:"client_id"`
	Role     string   `json:"role,omitempty"`
	Scopes   []string `json:"scopes"`
	TokenID  string   `json:"jti"`

	jwt.RegisteredClaims
}

// GenerateOIDCAccessToken generates an OIDC-compliant access token
func (j *JWTUtil) GenerateOIDCAccessToken(userID, clientID, role string, scopes []string, userInfo map[string]interface{}, expiry time.Duration) (string, string, error) {
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
	if name, ok := userInfo["name"].(string); ok {
		claims.Name = name
	}
	if givenName, ok := userInfo["given_name"].(string); ok {
		claims.GivenName = givenName
	}
	if familyName, ok := userInfo["family_name"].(string); ok {
		claims.FamilyName = familyName
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
	if preferredUsername, ok := userInfo["preferred_username"].(string); ok {
		claims.PreferredUsername = preferredUsername
	}
	if locale, ok := userInfo["locale"].(string); ok {
		claims.Locale = locale
	}
	if zoneinfo, ok := userInfo["zoneinfo"].(string); ok {
		claims.Zoneinfo = zoneinfo
	}
	if updatedAt, ok := userInfo["updated_at"].(int64); ok {
		claims.UpdatedAt = updatedAt
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, tokenID, nil
}

// GenerateIDToken generates an OIDC ID Token
func (j *JWTUtil) GenerateIDToken(userID, clientID string, nonce string, userInfo map[string]interface{}, expiry time.Duration) (string, error) {
	tokenID := uuid.New().String()
	now := time.Now()

	claims := OIDCClaims{
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

	// Add user info
	if name, ok := userInfo["name"].(string); ok {
		claims.Name = name
	}
	if givenName, ok := userInfo["given_name"].(string); ok {
		claims.GivenName = givenName
	}
	if familyName, ok := userInfo["family_name"].(string); ok {
		claims.FamilyName = familyName
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
	if preferredUsername, ok := userInfo["preferred_username"].(string); ok {
		claims.PreferredUsername = preferredUsername
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign ID token: %w", err)
	}

	return tokenString, nil
}

// ValidateOIDCToken validates and parses an OIDC token
func (j *JWTUtil) ValidateOIDCToken(tokenString string) (*OIDCClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &OIDCClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*OIDCClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
