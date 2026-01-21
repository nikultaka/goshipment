package carriers

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// CarrierModule handles carrier management
type CarrierModule struct {
	service        *CarrierService
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewCarrierModule creates a new carrier module
func NewCarrierModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *CarrierModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewCarrierService(db)

	return &CarrierModule{
		service:        service,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers carrier routes
func (m *CarrierModule) RegisterRoutes(router *gin.RouterGroup) {
	carriers := router.Group("/carriers")
	carriers.Use(m.authMiddleware.RequireAuth())
	{
		// GET /carriers - List all carriers
		carriers.GET("", middleware.RequireScope("carriers.read", "shipments.read", "read", "*"), m.listCarriers)

		// GET /carriers/availability/:code - Get carrier availability
		carriers.GET("/availability/:code", middleware.RequireScope("carriers.read", "shipments.read", "read", "*"), m.getCarrierAvailability)

		// GET /carriers/:id - Get carrier by ID
		carriers.GET("/:id", middleware.RequireScope("carriers.read", "shipments.read", "read", "*"), m.getCarrier)

		// GET /carriers/:id/services - Get carrier service levels
		carriers.GET("/:id/services", middleware.RequireScope("carriers.read", "shipments.read", "read", "*"), m.getCarrierServiceLevels)
	}
}
