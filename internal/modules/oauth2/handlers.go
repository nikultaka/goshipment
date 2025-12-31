package oauth2

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// authorize handles authorization requests
// @Summary OAuth2 Authorization
// @Description Request authorization code with PKCE support
// @Tags OAuth2.1
// @Accept json
// @Produce json
// @Param request body AuthorizeRequest true "Authorization request"
// @Success 200 {object} response.Response{data=AuthorizeResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Security BearerAuth
// @Router /oauth/authorize [post]
func (m *OAuth2Module) authorize(c *gin.Context) {
	var req AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User authentication required")
		return
	}

	// Create authorization code
	authCode, err := m.service.CreateAuthorizationCode(userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Authorization code generated", &AuthorizeResponse{
		Code:  authCode.Code,
		State: req.State,
	})
}

// token handles token requests
// @Summary OAuth2 Token
// @Description Exchange authorization code, refresh token, or client credentials for access token
// @Tags OAuth2.1
// @Accept json
// @Produce json
// @Param request body TokenRequest true "Token request"
// @Success 200 {object} response.Response{data=TokenResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /oauth/token [post]
func (m *OAuth2Module) token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	var tokenResp *TokenResponse
	var err error

	switch req.GrantType {
	case "authorization_code":
		tokenResp, err = m.service.ExchangeAuthorizationCode(&req)
	case "client_credentials":
		tokenResp, err = m.service.ClientCredentials(&req)
	case "refresh_token":
		tokenResp, err = m.service.RefreshAccessToken(&req)
	default:
		response.BadRequest(c, "Unsupported grant type")
		return
	}

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Token generated successfully", tokenResp)
}

// revoke handles token revocation
// @Summary Revoke Token
// @Description Revoke an access or refresh token
// @Tags OAuth2.1
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RevokeRequest true "Revoke request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /oauth/revoke [post]
func (m *OAuth2Module) revoke(c *gin.Context) {
	var req RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.RevokeToken(req.Token, req.TokenTypeHint)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Token revoked successfully", nil)
}

// introspect handles token introspection
// @Summary Introspect Token
// @Description Get information about a token
// @Tags OAuth2.1
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IntrospectRequest true "Introspect request"
// @Success 200 {object} response.Response{data=IntrospectResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /oauth/introspect [post]
func (m *OAuth2Module) introspect(c *gin.Context) {
	var req IntrospectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	result, err := m.service.IntrospectToken(req.Token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Token introspected successfully", result)
}
