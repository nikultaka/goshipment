package core

import (
	"gogin/internal/clients"
	"gogin/internal/config"

	"github.com/gin-gonic/gin"
)

// CoreModule handles core functionality
type CoreModule struct {
	db     *clients.Database
	redis  *clients.RedisClient
	nats   *clients.NATSClient
	config *config.Config
}

// NewCoreModule creates a new core module
func NewCoreModule(db *clients.Database, redis *clients.RedisClient, nats *clients.NATSClient, cfg *config.Config) *CoreModule {
	return &CoreModule{
		db:     db,
		redis:  redis,
		nats:   nats,
		config: cfg,
	}
}

// RegisterRoutes registers core routes
func (m *CoreModule) RegisterRoutes(router *gin.RouterGroup) {
	// Health check endpoints
	router.GET("/health", m.healthCheck)
	router.GET("/status", m.status)
}
