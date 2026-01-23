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

		// Check if token was issued before logout-all
		if claims.UserID != "" {
			logoutAllTimestamp, err := am.redisHelper.GetLogoutAllTimestamp(claims.UserID)
			if err == nil && logoutAllTimestamp > 0 && claims.IssuedAt != nil {
				if claims.IssuedAt.Unix() < logoutAllTimestamp {
					response.Unauthorized(c, "Token has been invalidated")
					c.Abort()
					return
				}
			}
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

// isAdmin checks if the given role is an admin role
func isAdmin(role string) bool {
	return role == "admin" || role == "superadmin"
}

// RequireRole checks if the user has the required role
// Admin users (admin, superadmin) can access all APIs
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "Access denied: role information missing")
			c.Abort()
			return
		}

		roleStr := userRole.(string)

		// Admin users can access all APIs
		if isAdmin(roleStr) {
			c.Next()
			return
		}

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
// Admin users (admin, superadmin) can access all APIs regardless of scope
func RequireScope(requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Admin users can access all APIs regardless of scope
		if userRole, exists := c.Get("role"); exists {
			if roleStr, ok := userRole.(string); ok && isAdmin(roleStr) {
				c.Next()
				return
			}
		}

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

// GetOrganizationContext gets the organization context for the current user
// For admin users, it returns the organization_id from query param if provided, or empty string (all orgs)
// For non-admin users, it returns empty string and false, indicating the handler should do normal org lookup
// Returns: (organizationID, isAdmin)
func GetOrganizationContext(c *gin.Context) (string, bool) {
	if userRole, exists := c.Get("role"); exists {
		if roleStr, ok := userRole.(string); ok && isAdmin(roleStr) {
			// Admin user - check for organization_id query param
			orgID := c.Query("organization_id")
			return orgID, true
		}
	}
	return "", false
}

// IsAdminUser checks if the current user is an admin (admin or superadmin)
func IsAdminUser(c *gin.Context) bool {
	if userRole, exists := c.Get("role"); exists {
		if roleStr, ok := userRole.(string); ok {
			return isAdmin(roleStr)
		}
	}
	return false
}
