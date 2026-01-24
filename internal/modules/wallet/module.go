package wallet

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// WalletModule represents the wallet module
type WalletModule struct {
	handlers       *WalletHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewWalletModule creates a new wallet module
func NewWalletModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *WalletModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewWalletService(db)
	handlers := NewWalletHandlers(service)

	return &WalletModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the wallet routes
func (m *WalletModule) RegisterRoutes(router *gin.RouterGroup) {
	wallet := router.Group("/wallet")
	wallet.Use(m.authMiddleware.RequireAuth())
	{
		// GET /wallet - Get wallet status (all authenticated users)
		// Scope: wallet.read
		wallet.GET("", m.handlers.GetWallet)

		// GET /wallet/overview - Get wallet overview with financial summary
		// Scope: wallet.read
		wallet.GET("/overview", m.handlers.GetWalletOverview)

		// GET /wallet/transactions - List transactions with filters
		// Scope: wallet.read
		wallet.GET("/transactions", m.handlers.ListTransactions)

		// GET /wallet/transactions/export - Export transactions to CSV or JSON
		// Scope: wallet.read
		wallet.GET("/transactions/export", m.handlers.ExportTransactions)

		// GET /wallet/refunds - List carrier refund requests
		// Scope: wallet.read
		wallet.GET("/refunds", m.handlers.ListCarrierRefunds)

		// POST /wallet/topup - Add funds via Stripe (all authenticated users)
		// Scope: wallet.topup
		wallet.POST("/topup", m.handlers.TopUp)

		// POST /wallet/refund - Refund to wallet (Manager/Admin only)
		// Scope: wallet.refund
		// Role check should be enforced in handler or via middleware
		wallet.POST("/refund",
			middleware.RequireRole("manager", "admin"),
			m.handlers.Refund)

		// POST /wallet/lock - Lock wallet (Admin only)
		// Scope: wallet.manage
		wallet.POST("/lock",
			middleware.RequireRole("admin"),
			m.handlers.LockWallet)

		// POST /wallet/unlock - Unlock wallet (Admin only)
		// Scope: wallet.manage
		wallet.POST("/unlock",
			middleware.RequireRole("admin"),
			m.handlers.UnlockWallet)
	}
}
