package addresses

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// AddressesModule handles address book management
type AddressesModule struct {
	service        *AddressService
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewAddressesModule creates a new addresses module
func NewAddressesModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *AddressesModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewAddressService(db)

	return &AddressesModule{
		service:        service,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers address book routes
func (m *AddressesModule) RegisterRoutes(router *gin.RouterGroup) {
	addresses := router.Group("/addresses")
	addresses.Use(m.authMiddleware.RequireAuth())
	{
		// GET /addresses - List all addresses (requires addresses.read scope)
		addresses.GET("", middleware.RequireScope("addresses.read", "*"), m.listAddresses)

		// GET /addresses/:id - Get single address (requires addresses.read scope)
		addresses.GET("/:id", middleware.RequireScope("addresses.read", "*"), m.getAddress)

		// POST /addresses - Create address (requires addresses.write scope)
		addresses.POST("", middleware.RequireScope("addresses.write", "*"), m.createAddress)

		// PATCH /addresses/:id - Update address (requires addresses.write scope)
		addresses.PATCH("/:id", middleware.RequireScope("addresses.write", "*"), m.updateAddress)
	}
}
