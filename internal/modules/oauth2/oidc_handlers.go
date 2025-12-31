package oauth2

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gogin/internal/response"
)

// OIDCDiscoveryResponse represents the OpenID Connect discovery document
// swagger:model OIDCDiscoveryResponse
type OIDCDiscoveryResponse struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	UserinfoEndpoint                           string   `json:"userinfo_endpoint,omitempty"`
	JwksURI                                    string   `json:"jwks_uri"`
	RegistrationEndpoint                       string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	RevocationEndpoint                         string   `json:"revocation_endpoint,omitempty"`
	RevocationEndpointAuthMethodsSupported     []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint,omitempty"`
	IntrospectionEndpointAuthMethodsSupported  []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	SubjectTypesSupported                      []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                            []string `json:"claims_supported,omitempty"`
	RequestParameterSupported                  bool     `json:"request_parameter_supported,omitempty"`
	RequestURIParameterSupported               bool     `json:"request_uri_parameter_supported,omitempty"`
	RequireRequestURIRegistration              bool     `json:"require_request_uri_registration,omitempty"`
}

// getOIDCDiscovery godoc
// @Summary      OIDC Discovery Document
// @Description  Returns the OpenID Connect discovery document (RFC 8414)
// @Tags         OAuth2.1 / OIDC
// @Produce      json
// @Success      200  {object}  OIDCDiscoveryResponse
// @Router       /.well-known/openid-configuration [get]
func (m *OAuth2Module) getOIDCDiscovery(c *gin.Context) {
	// Construct base URL from request
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	// If JWT_ISSUER is set in config, use it as the issuer
	issuer := m.config.OAuth.JWTIssuer
	if issuer == "" || issuer == "goapi" {
		// Default to baseURL if not configured
		issuer = baseURL
	}

	discovery := OIDCDiscoveryResponse{
		Issuer:                baseURL,
		AuthorizationEndpoint: fmt.Sprintf("%s/api/v1/oauth/authorize", baseURL),
		TokenEndpoint:         fmt.Sprintf("%s/api/v1/oauth/token", baseURL),
		JwksURI:               fmt.Sprintf("%s/api/v1/oauth/jwks.json", baseURL),
		RevocationEndpoint:    fmt.Sprintf("%s/api/v1/oauth/revoke", baseURL),
		IntrospectionEndpoint: fmt.Sprintf("%s/api/v1/oauth/introspect", baseURL),

		ScopesSupported: []string{
			"openid",
			"profile",
			"email",
			"phone",
			"address",
			"offline_access",
		},
		ResponseTypesSupported: []string{
			"code",
		},
		ResponseModesSupported: []string{
			"query",
			"fragment",
		},
		GrantTypesSupported: []string{
			"authorization_code",
			"refresh_token",
			"client_credentials",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
			"none", // For public clients
		},
		RevocationEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
		},
		IntrospectionEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
		},
		CodeChallengeMethodsSupported: []string{
			"S256",
			"plain",
		},
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"RS256",
		},
		ClaimsSupported: []string{
			"sub",
			"iss",
			"aud",
			"exp",
			"iat",
			"nbf",
			"jti",
			"name",
			"given_name",
			"family_name",
			"middle_name",
			"nickname",
			"preferred_username",
			"profile",
			"picture",
			"website",
			"email",
			"email_verified",
			"gender",
			"birthdate",
			"zoneinfo",
			"locale",
			"phone_number",
			"phone_number_verified",
			"updated_at",
		},
		RequestParameterSupported:     false,
		RequestURIParameterSupported:  false,
		RequireRequestURIRegistration: false,
	}

	c.JSON(http.StatusOK, discovery)
}

// getJWKS godoc
// @Summary      JSON Web Key Set
// @Description  Returns the public keys used for JWT signature verification (RFC 7517)
// @Tags         OAuth2.1 / OIDC
// @Produce      json
// @Success      200  {object}  object
// @Router       /oauth/jwks.json [get]
func (m *OAuth2Module) getJWKS(c *gin.Context) {
	// Check if RS256 is being used
	if m.jwtUtilRS256 == nil {
		// Fallback: return empty JWKS if using HMAC (HMAC keys should never be exposed)
		c.JSON(http.StatusOK, gin.H{
			"keys": []interface{}{},
		})
		return
	}

	jwks := m.jwtUtilRS256.GetKeyPair().ToJWK()

	c.JSON(http.StatusOK, gin.H{
		"keys": []interface{}{jwks},
	})
}

// getUserInfo godoc
// @Summary      OIDC UserInfo Endpoint
// @Description  Returns claims about the authenticated user (RFC 7662)
// @Tags         OAuth2.1 / OIDC
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  object
// @Failure      401  {object}  response.Response
// @Router       /oauth/userinfo [get]
func (m *OAuth2Module) getUserInfo(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}

	// Query user information from database directly
	var user struct {
		ID            string  `db:"id"`
		FirstName     *string `db:"first_name"`
		LastName      *string `db:"last_name"`
		DisplayName   *string `db:"display_name"`
		Email         string  `db:"email"`
		EmailVerified bool    `db:"oidc_email_verified"`
		Phone         *string `db:"phone"`
		PhoneVerified bool    `db:"oidc_phone_verified"`
		AvatarURL     *string `db:"avatar_file_id"`
		Locale        *string `db:"locale"`
		Timezone      *string `db:"timezone"`
		UpdatedAt     int64   `db:"updated_at"`
	}

	// Get primary email
	query := `
		SELECT
			u.id,
			u.first_name,
			u.last_name,
			u.display_name,
			COALESCE(ue.email, '') as email,
			u.oidc_email_verified,
			up.phone_e164 as phone,
			u.oidc_phone_verified,
			u.avatar_file_id,
			u.locale,
			u.timezone,
			EXTRACT(EPOCH FROM u.updated_at)::bigint as updated_at
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		LEFT JOIN user_phones up ON u.primary_phone_id = up.id
		WHERE u.id = $1 AND u.is_active = true
	`

	err := m.db.DB.QueryRow(query, userID).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.DisplayName,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.AvatarURL,
		&user.Locale,
		&user.Timezone,
		&user.UpdatedAt,
	)

	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	// Build response with OIDC standard claims
	userInfo := gin.H{
		"sub":            user.ID,
		"email_verified": user.EmailVerified,
	}

	// Add optional claims only if they exist
	if user.FirstName != nil && user.LastName != nil {
		userInfo["name"] = *user.FirstName + " " + *user.LastName
	}
	if user.FirstName != nil {
		userInfo["given_name"] = *user.FirstName
	}
	if user.LastName != nil {
		userInfo["family_name"] = *user.LastName
	}
	if user.DisplayName != nil {
		userInfo["preferred_username"] = *user.DisplayName
	}
	if user.Email != "" {
		userInfo["email"] = user.Email
	}
	if user.Phone != nil {
		userInfo["phone_number"] = *user.Phone
		userInfo["phone_number_verified"] = user.PhoneVerified
	}
	if user.AvatarURL != nil {
		userInfo["picture"] = *user.AvatarURL
	}
	if user.Locale != nil {
		userInfo["locale"] = *user.Locale
	}
	if user.Timezone != nil {
		userInfo["zoneinfo"] = *user.Timezone
	}
	if user.UpdatedAt > 0 {
		userInfo["updated_at"] = user.UpdatedAt
	}

	c.JSON(http.StatusOK, userInfo)
}
