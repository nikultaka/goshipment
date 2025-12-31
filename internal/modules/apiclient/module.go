package apiclient

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// APIClientModule handles API client management
type APIClientModule struct {
	db          *clients.Database
	redis       *clients.RedisClient
	config      *config.Config
	service     *APIClientService
	redisHelper *redishelper.RedisHelper
	jwtUtil     *utils.JWTUtil
}

// NewAPIClientModule creates a new API client module
func NewAPIClientModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *APIClientModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	service := NewAPIClientService(db, redisHelper)

	return &APIClientModule{
		db:          db,
		redis:       redis,
		config:      cfg,
		service:     service,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
	}
}

// RegisterRoutes registers API client routes
func (m *APIClientModule) RegisterRoutes(router *gin.RouterGroup) {
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)

	clients := router.Group("/clients")
	clients.Use(authMiddleware.RequireAuth(), middleware.RequireAdmin())
	{
		clients.POST("", m.createClient)
		clients.GET("", m.listClients)
		clients.GET("/:id", m.getClient)
		clients.PUT("/:id", m.updateClient)
		clients.DELETE("/:id", m.deleteClient)
		clients.POST("/:id/regenerate-secret", m.regenerateSecret)
		clients.PUT("/:id/status", m.updateStatus)
	}
}
