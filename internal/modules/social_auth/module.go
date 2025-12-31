package social_auth

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// NewSocialAuthModule creates a new social auth module instance
func NewSocialAuthModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *SocialAuthModule {
	// Initialize JWT utility
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)

	// Initialize Redis helper
	redisHelper := redishelper.NewRedisHelper(redis)

	// Initialize social auth service
	service := NewSocialAuthService(db, jwtUtil, redisHelper, cfg)

	return &SocialAuthModule{
		service: service,
	}
}

// RegisterRoutes registers all social auth routes
func (m *SocialAuthModule) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		// Google OAuth routes
		auth.GET("/google/login", m.googleLogin)
		auth.GET("/google/callback", m.googleCallback)

		// Facebook OAuth routes
		auth.GET("/facebook/login", m.facebookLogin)
		auth.GET("/facebook/callback", m.facebookCallback)
	}
}
