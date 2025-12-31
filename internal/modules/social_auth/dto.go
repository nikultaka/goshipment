package social_auth

import "time"

// OAuthURLResponse is the response containing the OAuth authorization URL
type OAuthURLResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	Provider         string `json:"provider"`
	Message          string `json:"message,omitempty"`
}

// OAuthLoginResponse is the response after successful OAuth login
type OAuthLoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// UserResponse contains user information returned after OAuth login
type UserResponse struct {
	ID            string    `json:"id"`
	ExternalID    string    `json:"external_id,omitempty"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name,omitempty"`
	LastName      string    `json:"last_name,omitempty"`
	Avatar        string    `json:"avatar,omitempty"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	Provider      string    `json:"provider,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// OAuthTokens represents tokens received from OAuth provider
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"` // For OIDC providers like Google
}

// OAuthUserInfo represents user information from OAuth provider
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	FirstName      string
	LastName       string
	FullName       string
	AvatarURL      string
	Locale         string
}
