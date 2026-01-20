package shipments

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ShipmentsModule handles shipment management
type ShipmentsModule struct {
	handlers       *ShipmentHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewShipmentsModule creates a new shipments module
func NewShipmentsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *ShipmentsModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewSimpleShipmentService(db)
	handlers := NewShipmentHandlers(service)

	return &ShipmentsModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers shipment routes
func (m *ShipmentsModule) RegisterRoutes(router *gin.RouterGroup) {
	// Shipments routes (all require authentication)
	// Organization membership is verified in handlers
	shipments := router.Group("/shipments")
	shipments.Use(m.authMiddleware.RequireAuth())
	{
		// POST /shipments - Create shipment from estimate
		shipments.POST("", m.handlers.CreateShipment)

		// GET /shipments - List shipments
		shipments.GET("", m.handlers.ListShipments)

		// GET /shipments/:id - Get shipment by ID
		shipments.GET("/:id", m.handlers.GetShipment)

		// POST /shipments/:id/cancel - Cancel shipment
		shipments.POST("/:id/cancel", m.handlers.CancelShipment)

		// POST /shipments/:id/void - Void shipment
		shipments.POST("/:id/void", m.handlers.VoidShipment)

		// POST /shipments/:id/archive - Archive/unarchive shipment
		shipments.POST("/:id/archive", m.handlers.ArchiveShipment)

		// PATCH /shipments/:id/status - Update shipment status
		shipments.PATCH("/:id/status", m.handlers.UpdateStatus)
	}
}
