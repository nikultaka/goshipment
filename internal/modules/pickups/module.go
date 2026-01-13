package pickups

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// PickupModule represents the pickup module
type PickupModule struct {
	handlers       *PickupHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewPickupModule creates a new pickup module
func NewPickupModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *PickupModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewPickupService(db)
	handlers := NewPickupHandlers(service)

	return &PickupModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the pickup routes
func (m *PickupModule) RegisterRoutes(router *gin.RouterGroup) {
	pickups := router.Group("/pickups")
	pickups.Use(m.authMiddleware.RequireAuth())
	{
		// POST /pickups - Schedule a pickup (requires pickups.create scope)
		// Scope: pickups.create
		pickups.POST("", m.handlers.SchedulePickup)

		// POST /pickups/:id/cancel - Cancel a pickup (requires pickups.cancel scope)
		// Scope: pickups.cancel
		// Manager/Admin can cancel any pickup, users can cancel their own
		pickups.POST("/:id/cancel", m.handlers.CancelPickup)

		// GET /pickups - List pickups for organization
		// Scope: pickups.create (read allowed for creators)
		// Users see their own shipments' pickups, managers/admins see all
		pickups.GET("", m.handlers.ListPickups)
	}
}
