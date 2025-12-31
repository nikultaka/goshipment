package settings

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

type SettingsModule struct {
	service        *SettingsService
	authMiddleware *middleware.AuthMiddleware
}

// NewSettingsModule creates a new instance of the settings module
func NewSettingsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *SettingsModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	service := NewSettingsService(db, redisHelper, cfg)

	return &SettingsModule{
		service:        service,
		authMiddleware: middleware.NewAuthMiddleware(jwtUtil, redisHelper),
	}
}

// RegisterRoutes registers all settings-related routes
func (m *SettingsModule) RegisterRoutes(router *gin.RouterGroup) {
	settings := router.Group("/settings")

	// System settings routes (admin only)
	system := settings.Group("/system")
	system.Use(m.authMiddleware.RequireAuth(), middleware.RequireAdmin())
	{
		system.POST("", m.createSystemSetting)
		system.GET("", m.listSystemSettings)
		system.GET("/:key", m.getSystemSetting)
		system.PUT("/:key", m.updateSystemSetting)
		system.DELETE("/:key", m.deleteSystemSetting)
	}

	// User settings routes (authenticated users)
	user := settings.Group("/user")
	user.Use(m.authMiddleware.RequireAuth())
	{
		user.GET("", m.listUserSettings)
		user.GET("/:key", m.getUserSetting)
		user.PUT("/:key", m.createOrUpdateUserSetting)
		user.DELETE("/:key", m.deleteUserSetting)
	}
}
