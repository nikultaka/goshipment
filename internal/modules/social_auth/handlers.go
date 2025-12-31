package social_auth

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// SocialAuthModule contains the handlers for social authentication
type SocialAuthModule struct {
	service *SocialAuthService
}

// googleLogin generates Google OAuth authorization URL
// @Summary Google OAuth login
// @Description Generate Google OAuth authorization URL. Copy the URL, open it in your browser, authenticate with Google, and you'll be redirected back with a code and state parameter.
// @Tags Social Auth
// @Produce json
// @Success 200 {object} response.Response{data=OAuthURLResponse} "Authorization URL generated successfully"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /auth/google/login [get]
func (m *SocialAuthModule) googleLogin(c *gin.Context) {
	authURL, err := m.service.GetAuthorizationURL(m.service.googleProvider)
	if err != nil {
		response.InternalError(c, "Failed to generate Google authorization URL")
		return
	}

	response.Success(c, http.StatusOK, "Google OAuth URL generated successfully", OAuthURLResponse{
		AuthorizationURL: authURL,
		Provider:         "google",
		Message:          "Copy the URL and open it in your browser to authenticate with Google",
	})
}

// googleCallback handles Google OAuth callback
// @Summary Google OAuth callback
// @Description Handle Google OAuth callback. After authenticating with Google, you'll be redirected here with a code and state parameter. This endpoint exchanges the code for tokens and returns JWT tokens for your application.
// @Tags Social Auth
// @Produce json
// @Param code query string true "Authorization code from Google (obtained after user authentication)"
// @Param state query string true "State parameter for CSRF protection (generated during login)"
// @Success 200 {object} response.Response{data=OAuthLoginResponse} "Login successful - JWT tokens and user info returned"
// @Failure 400 {object} response.Response "Bad request - missing code or state parameter"
// @Failure 401 {object} response.Response "Unauthorized - invalid state or authentication failed"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /auth/google/callback [get]
func (m *SocialAuthModule) googleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "Missing code or state parameter")
		return
	}

	loginResp, err := m.service.HandleCallback(c.Request.Context(), m.service.googleProvider, code, state)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Google login successful", loginResp)
}

// facebookLogin generates Facebook OAuth authorization URL
// @Summary Facebook OAuth login
// @Description Generate Facebook OAuth authorization URL. Copy the URL, open it in your browser, authenticate with Facebook, and you'll be redirected back with a code and state parameter.
// @Tags Social Auth
// @Produce json
// @Success 200 {object} response.Response{data=OAuthURLResponse} "Authorization URL generated successfully"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /auth/facebook/login [get]
func (m *SocialAuthModule) facebookLogin(c *gin.Context) {
	authURL, err := m.service.GetAuthorizationURL(m.service.facebookProvider)
	if err != nil {
		response.InternalError(c, "Failed to generate Facebook authorization URL")
		return
	}

	response.Success(c, http.StatusOK, "Facebook OAuth URL generated successfully", OAuthURLResponse{
		AuthorizationURL: authURL,
		Provider:         "facebook",
		Message:          "Copy the URL and open it in your browser to authenticate with Facebook",
	})
}

// facebookCallback handles Facebook OAuth callback
// @Summary Facebook OAuth callback
// @Description Handle Facebook OAuth callback. After authenticating with Facebook, you'll be redirected here with a code and state parameter. This endpoint exchanges the code for tokens and returns JWT tokens for your application.
// @Tags Social Auth
// @Produce json
// @Param code query string true "Authorization code from Facebook (obtained after user authentication)"
// @Param state query string true "State parameter for CSRF protection (generated during login)"
// @Success 200 {object} response.Response{data=OAuthLoginResponse} "Login successful - JWT tokens and user info returned"
// @Failure 400 {object} response.Response "Bad request - missing code or state parameter"
// @Failure 401 {object} response.Response "Unauthorized - invalid state or authentication failed"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /auth/facebook/callback [get]
func (m *SocialAuthModule) facebookCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.BadRequest(c, "Missing code or state parameter")
		return
	}

	loginResp, err := m.service.HandleCallback(c.Request.Context(), m.service.facebookProvider, code, state)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Facebook login successful", loginResp)
}
