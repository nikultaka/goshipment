package social_auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
	GetProviderName() string
}

// ============================================================================
// GOOGLE PROVIDER
// ============================================================================

// GoogleProvider implements OAuth for Google
type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetProviderName returns the provider name
func (g *GoogleProvider) GetProviderName() string {
	return "google"
}

// GetAuthURL generates the Google OAuth authorization URL
func (g *GoogleProvider) GetAuthURL(state string) string {
	baseURL := "https://accounts.google.com/o/oauth2/v2/auth"
	params := url.Values{}
	params.Set("client_id", g.clientID)
	params.Set("redirect_uri", g.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "offline") // Request refresh token
	params.Set("prompt", "consent")      // Force consent to always get refresh token

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeCode exchanges authorization code for access token
func (g *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{}
	data.Set("client_id", g.clientID)
	data.Set("client_secret", g.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", g.redirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokens OAuthTokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokens, nil
}

// GetUserInfo fetches user information from Google
func (g *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info (status %d): %s", resp.StatusCode, string(body))
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &OAuthUserInfo{
		ProviderUserID: googleUser.ID,
		Email:          googleUser.Email,
		EmailVerified:  googleUser.VerifiedEmail,
		FirstName:      googleUser.GivenName,
		LastName:       googleUser.FamilyName,
		FullName:       googleUser.Name,
		AvatarURL:      googleUser.Picture,
		Locale:         googleUser.Locale,
	}, nil
}

// ============================================================================
// FACEBOOK PROVIDER
// ============================================================================

// FacebookProvider implements OAuth for Facebook
type FacebookProvider struct {
	appID       string
	appSecret   string
	redirectURL string
}

// NewFacebookProvider creates a new Facebook OAuth provider
func NewFacebookProvider(appID, appSecret, redirectURL string) *FacebookProvider {
	return &FacebookProvider{
		appID:       appID,
		appSecret:   appSecret,
		redirectURL: redirectURL,
	}
}

// GetProviderName returns the provider name
func (f *FacebookProvider) GetProviderName() string {
	return "facebook"
}

// GetAuthURL generates the Facebook OAuth authorization URL
func (f *FacebookProvider) GetAuthURL(state string) string {
	baseURL := "https://www.facebook.com/v18.0/dialog/oauth"
	params := url.Values{}
	params.Set("client_id", f.appID)
	params.Set("redirect_uri", f.redirectURL)
	params.Set("state", state)
	params.Set("scope", "email,public_profile")
	params.Set("response_type", "code")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeCode exchanges authorization code for access token
func (f *FacebookProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	tokenURL := "https://graph.facebook.com/v18.0/oauth/access_token"

	params := url.Values{}
	params.Set("client_id", f.appID)
	params.Set("client_secret", f.appSecret)
	params.Set("redirect_uri", f.redirectURL)
	params.Set("code", code)

	fullURL := fmt.Sprintf("%s?%s", tokenURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokens OAuthTokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokens, nil
}

// GetUserInfo fetches user information from Facebook
func (f *FacebookProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := "https://graph.facebook.com/me?fields=id,email,name,first_name,last_name,picture.type(large)"

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info (status %d): %s", resp.StatusCode, string(body))
	}

	var fbUser struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Picture   struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fbUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &OAuthUserInfo{
		ProviderUserID: fbUser.ID,
		Email:          fbUser.Email,
		EmailVerified:  true, // Facebook emails are considered verified
		FirstName:      fbUser.FirstName,
		LastName:       fbUser.LastName,
		FullName:       fbUser.Name,
		AvatarURL:      fbUser.Picture.Data.URL,
		Locale:         "",
	}, nil
}
