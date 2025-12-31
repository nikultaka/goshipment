package audit

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuditModule handles audit log management
type AuditModule struct {
	db          *clients.Database
	redis       *clients.RedisClient
	config      *config.Config
	service     *AuditService
	redisHelper *redishelper.RedisHelper
	jwtUtil     *utils.JWTUtil
}

// NewAuditModule creates a new audit module
func NewAuditModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *AuditModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	service := NewAuditService(db)

	return &AuditModule{
		db:          db,
		redis:       redis,
		config:      cfg,
		service:     service,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
	}
}

// RegisterRoutes registers audit routes
func (m *AuditModule) RegisterRoutes(router *gin.RouterGroup) {
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)

	// Admin routes (require authentication and admin role)
	admin := router.Group("/admin/audit-logs")
	admin.Use(authMiddleware.RequireAuth(), middleware.RequireAdmin())
	{
		admin.GET("", m.getAuditLogs)
		admin.GET("/export", m.exportAuditLogs)
	}
}
