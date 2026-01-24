package discounts

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// DiscountModule represents the discount module
type DiscountModule struct {
	handlers       *DiscountHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewDiscountModule creates a new discount module
func NewDiscountModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *DiscountModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewDiscountService(db)
	handlers := NewDiscountHandlers(service)

	return &DiscountModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the discount routes
func (m *DiscountModule) RegisterRoutes(router *gin.RouterGroup) {
	discounts := router.Group("/discounts")
	discounts.Use(m.authMiddleware.RequireAuth())
	{
		// GET /discounts - List all discounts
		discounts.GET("", m.handlers.ListDiscounts)

		// POST /discounts - Create a new discount
		discounts.POST("",
			middleware.RequireRole("manager", "admin"),
			m.handlers.CreateDiscount)

		// POST /discounts/validate - Validate a discount code
		discounts.POST("/validate", m.handlers.ValidateDiscount)

		// POST /discounts/apply - Apply a discount code
		discounts.POST("/apply", m.handlers.ApplyDiscount)

		// GET /discounts/:id - Get discount by ID
		discounts.GET("/:id", m.handlers.GetDiscount)

		// PUT /discounts/:id - Update a discount
		discounts.PUT("/:id",
			middleware.RequireRole("manager", "admin"),
			m.handlers.UpdateDiscount)

		// DELETE /discounts/:id - Delete a discount
		discounts.DELETE("/:id",
			middleware.RequireRole("manager", "admin"),
			m.handlers.DeleteDiscount)
	}
}
