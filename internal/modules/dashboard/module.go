package dashboard

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// DashboardModule handles dashboard analytics and metrics
type DashboardModule struct {
	db          *clients.Database
	redis       *clients.RedisClient
	config      *config.Config
	service     *DashboardService
	handlers    *DashboardHandlers
	redisHelper *redishelper.RedisHelper
	jwtUtil     *utils.JWTUtil
}

// NewDashboardModule creates a new dashboard module
func NewDashboardModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *DashboardModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	service := NewDashboardService(db)
	handlers := NewDashboardHandlers(service)

	return &DashboardModule{
		db:          db,
		redis:       redis,
		config:      cfg,
		service:     service,
		handlers:    handlers,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
	}
}

// RegisterRoutes registers dashboard routes
func (m *DashboardModule) RegisterRoutes(router *gin.RouterGroup) {
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)

	// Admin dashboard routes (require authentication and admin role)
	admin := router.Group("/admin/dashboard")
	admin.Use(authMiddleware.RequireAuth(), middleware.RequireAdmin())
	{
		// GET /admin/dashboard/financial-metrics - Get financial metrics
		admin.GET("/financial-metrics", m.handlers.GetFinancialMetrics)

		// GET /admin/dashboard/revenue-chart - Get revenue chart data
		admin.GET("/revenue-chart", m.handlers.GetRevenueChart)

		// GET /admin/dashboard/top-carriers - Get top carriers by revenue
		admin.GET("/top-carriers", m.handlers.GetTopCarriers)

		// GET /admin/dashboard/shipment-stats - Get shipment statistics
		admin.GET("/shipment-stats", m.handlers.GetShipmentStats)

		// GET /admin/dashboard/user-overview - Get user overview metrics
		admin.GET("/user-overview", m.handlers.GetUserOverview)

		// GET /admin/dashboard/top-spending-businesses - Get top spending businesses (last 7 days)
		admin.GET("/top-spending-businesses", m.handlers.GetTopSpendingBusinesses)

		// GET /admin/dashboard/pickups-exceptions - Get pickups and exceptions list
		admin.GET("/pickups-exceptions", m.handlers.GetPickupsAndExceptions)
	}
}
