package addresses

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AddressModule handles address book management
type AddressModule struct {
	service        *AddressService
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewAddressModule creates a new address module
func NewAddressModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *AddressModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewAddressService(db)

	return &AddressModule{
		service:        service,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers address book routes
func (m *AddressModule) RegisterRoutes(router *gin.RouterGroup) {
	// Address book routes (all require authentication)
	addresses := router.Group("/addresses")
	addresses.Use(m.authMiddleware.RequireAuth())
	{
		// GET /addresses - List all addresses
		addresses.GET("", m.listAddresses)

		// POST /addresses - Create a new address
		addresses.POST("", m.createAddress)

		// GET /addresses/default/sender - Get default sender address
		addresses.GET("/default/sender", m.getDefaultSender)

		// GET /addresses/default/recipient - Get default recipient address
		addresses.GET("/default/recipient", m.getDefaultRecipient)

		// GET /addresses/:id - Get a specific address
		addresses.GET("/:id", m.getAddress)

		// PATCH /addresses/:id - Update an address
		addresses.PATCH("/:id", m.updateAddress)

		// DELETE /addresses/:id - Delete an address
		addresses.DELETE("/:id", m.deleteAddress)

		// POST /addresses/:id/set-default - Set as default sender or recipient
		addresses.POST("/:id/set-default", m.setDefaultAddress)

		// POST /addresses/:id/clear-default - Clear default status
		addresses.POST("/:id/clear-default", m.clearDefaultAddress)
	}
}
