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

		// GET /commissions/summary - Get commission summary
		// Scope: commissions.read
		// Returns this month's pending/earned and all-time paid/earned
		commissions.GET("/summary", m.handlers.GetCommissionSummary)

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

		// =====================================================
		// COMMISSION PLANS (Admin/Manager Only)
		// =====================================================

		// GET /commissions/plans - List commission plans
		// Scope: commissions.manage (admin only)
		commissions.GET("/plans", m.handlers.ListPlans)

		// POST /commissions/plans - Create a commission plan
		// Scope: commissions.manage (admin only)
		commissions.POST("/plans", m.handlers.CreatePlan)

		// GET /commissions/plans/:id - Get a commission plan
		// Scope: commissions.manage (admin only)
		commissions.GET("/plans/:id", m.handlers.GetPlan)

		// PATCH /commissions/plans/:id - Update a commission plan
		// Scope: commissions.manage (admin only)
		commissions.PATCH("/plans/:id", m.handlers.UpdatePlan)

		// DELETE /commissions/plans/:id - Delete a commission plan
		// Scope: commissions.manage (admin only)
		commissions.DELETE("/plans/:id", m.handlers.DeletePlan)

		// =====================================================
		// MANUAL COMMISSION TRANSACTIONS (Admin/Manager Only)
		// =====================================================

		// POST /commissions/transactions - Create a commission transaction manually
		// Scope: commissions.manage (admin only)
		// Allows admins to manually create commission transactions
		commissions.POST("/transactions", m.handlers.CreateCommissionTransaction)
	}
}
