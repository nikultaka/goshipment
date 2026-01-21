package geo

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// GeoModule handles geography reference data (countries, regions)
type GeoModule struct {
	service        *GeoService
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewGeoModule creates a new geo module
func NewGeoModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *GeoModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewGeoService(db)

	return &GeoModule{
		service:        service,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers geography routes
func (m *GeoModule) RegisterRoutes(router *gin.RouterGroup) {
	geo := router.Group("/geo")
	geo.Use(m.authMiddleware.RequireAuth())
	{
		// GET /geo/countries - List all countries
		geo.GET("/countries", middleware.RequireScope("geo.read", "shipments.read", "read", "*"), m.listCountries)

		// GET /geo/countries/:code/regions - List regions for a country
		geo.GET("/countries/:code/regions", middleware.RequireScope("geo.read", "shipments.read", "read", "*"), m.listRegions)
	}
}
