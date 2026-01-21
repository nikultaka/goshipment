package shipment_estimates

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// EstimatesModule handles shipment estimate management
type EstimatesModule struct {
	handlers       *EstimateHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewEstimatesModule creates a new shipment estimates module
func NewEstimatesModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *EstimatesModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewEstimateService(db)
	handlers := NewEstimateHandlers(service)

	return &EstimatesModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers shipment estimate routes
func (m *EstimatesModule) RegisterRoutes(router *gin.RouterGroup) {
	// Shipment estimates routes
	// Note: POST is public (guest-friendly), GET requires optional auth
	estimates := router.Group("/shipment-estimates")
	{
		// POST /shipment-estimates - Create estimate (public, auth optional)
		// Uses OptionalAuth so guest users can create quotes
		estimates.POST("", m.authMiddleware.OptionalAuth(), m.handlers.CreateEstimate)

		// GET /shipment-estimates/:id - Get estimate (public, auth optional)
		// Guest estimates accessible by anyone, org estimates require auth + membership
		estimates.GET("/:id", m.authMiddleware.OptionalAuth(), m.handlers.GetEstimate)

		// POST /shipment-estimates/:id/rates - Request rates for estimate
		estimates.POST("/:id/rates", m.authMiddleware.OptionalAuth(), m.handlers.RequestRates)

		// PATCH /shipment-estimates/:id/select-rate - Select a carrier rate
		estimates.PATCH("/:id/select-rate", m.authMiddleware.OptionalAuth(), m.handlers.SelectRate)
	}
}
