package middleware

import (
	"strings"

	"gogin/internal/modules/redishelper"
	"gogin/internal/response"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT tokens
type AuthMiddleware struct {
	jwtUtil     *utils.JWTUtil
	redisHelper *redishelper.RedisHelper
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtUtil *utils.JWTUtil, redisHelper *redishelper.RedisHelper) *AuthMiddleware {
	return &AuthMiddleware{
		jwtUtil:     jwtUtil,
		redisHelper: redisHelper,
	}
}

// RequireAuth validates JWT token and sets user context
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := am.jwtUtil.ValidateToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Check if token is revoked
		revoked, err := am.redisHelper.IsTokenRevoked(claims.TokenID)
		if err == nil && revoked {
			response.Unauthorized(c, "Token has been revoked")
			c.Abort()
			return
		}

		// Set user context
		if claims.UserID != "" {
			c.Set("user_id", claims.UserID)
		}
		c.Set("client_id", claims.ClientID)
		if claims.Role != "" {
			c.Set("role", claims.Role)
		}
		c.Set("scopes", claims.Scopes)
		c.Set("token_id", claims.TokenID)

		c.Next()
	}
}

// OptionalAuth validates JWT if present, but doesn't require it
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := am.jwtUtil.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		// Check if token is revoked
		revoked, err := am.redisHelper.IsTokenRevoked(claims.TokenID)
		if err == nil && revoked {
			c.Next()
			return
		}

		// Set user context
		if claims.UserID != "" {
			c.Set("user_id", claims.UserID)
		}
		c.Set("client_id", claims.ClientID)
		if claims.Role != "" {
			c.Set("role", claims.Role)
		}
		c.Set("scopes", claims.Scopes)
		c.Set("token_id", claims.TokenID)

		c.Next()
	}
}

// RequireRole checks if the user has the required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "Access denied: role information missing")
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		allowed := false

		for _, role := range roles {
			if roleStr == role {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Forbidden(c, "Access denied: insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireScope checks if the token has the required scope
func RequireScope(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopesInterface, exists := c.Get("scopes")
		if !exists {
			response.Forbidden(c, "Access denied: scope information missing")
			c.Abort()
			return
		}

		scopes, ok := scopesInterface.([]string)
		if !ok {
			response.Forbidden(c, "Access denied: invalid scope information")
			c.Abort()
			return
		}

		// Check if token has any of the required scopes
		hasScope := false
		for _, required := range requiredScopes {
			for _, scope := range scopes {
				if scope == required || scope == "*" {
					hasScope = true
					break
				}
			}
			if hasScope {
				break
			}
		}

		if !hasScope {
			response.Forbidden(c, "Access denied: required scope not present")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin is a convenience middleware for admin-only routes
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin", "superadmin")
}
