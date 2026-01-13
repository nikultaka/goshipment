package commissions

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// CommissionModule represents the commission module
type CommissionModule struct {
	handlers       *CommissionHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewCommissionModule creates a new commission module
func NewCommissionModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *CommissionModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewCommissionService(db)
	handlers := NewCommissionHandlers(service)

	return &CommissionModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the commission routes
func (m *CommissionModule) RegisterRoutes(router *gin.RouterGroup) {
	commissions := router.Group("/commissions")
	commissions.Use(m.authMiddleware.RequireAuth())
	{
		// GET /commissions - List commission transactions
		// Scope: commissions.read
		// Sales Reps see only their own commissions
		// Admins/Managers see all organization commissions
		commissions.GET("", m.handlers.ListCommissions)

		// GET /commissions/payouts - List payout records
		// Scope: commissions.read
		// Sales Reps see only their own payouts
		// Admins/Managers see all organization payouts
		commissions.GET("/payouts", m.handlers.ListPayouts)

		// POST /commissions/payout - Request payout (Sales Rep)
		// Scope: commissions.read (allows sales reps to request)
		// Creates a payout request that requires admin approval
		commissions.POST("/payout", m.handlers.RequestPayout)

		// POST /commissions/payout/process - Process payout (Admin/Manager Only)
		// Scope: commissions.manage (admin only)
		// Approve or reject payout requests
		commissions.POST("/payout/process", m.handlers.ProcessPayout)
	}
}
