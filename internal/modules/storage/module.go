package storage

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// StorageModule handles file storage
type StorageModule struct {
	service        *StorageService
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewStorageModule creates a new storage module
func NewStorageModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *StorageModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewStorageService(db, cfg)

	return &StorageModule{
		service:        service,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers storage routes
func (m *StorageModule) RegisterRoutes(router *gin.RouterGroup) {
	storage := router.Group("/storage")
	{
		// Upload route - requires authentication
		storage.POST("/upload", m.authMiddleware.RequireAuth(), m.uploadFile)

		// Files routes - public access with optional auth for private files
		files := storage.Group("/files")
		{
			// List files - public endpoint, shows public files + user's private files if authenticated
			files.GET("", m.authMiddleware.OptionalAuth(), m.listFiles)

			// Get file metadata - public for public files, requires auth for private files
			files.GET("/:id", m.authMiddleware.OptionalAuth(), m.getFile)

			// Download file - public for public files, requires auth for private files
			files.GET("/:id/download", m.authMiddleware.OptionalAuth(), m.downloadFile)

			// Update file - requires authentication
			files.PUT("/:id", m.authMiddleware.RequireAuth(), m.updateFile)

			// Delete file - requires authentication
			files.DELETE("/:id", m.authMiddleware.RequireAuth(), m.deleteFile)
		}
	}
}
