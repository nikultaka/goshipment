package invoices

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// InvoicesModule represents the invoices module
type InvoicesModule struct {
	handlers       *InvoiceHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewInvoicesModule creates a new invoices module
func NewInvoicesModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *InvoicesModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewInvoiceService(db)
	handlers := NewInvoiceHandlers(service)

	return &InvoicesModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the invoices routes
func (m *InvoicesModule) RegisterRoutes(router *gin.RouterGroup) {
	invoices := router.Group("/invoices")
	invoices.Use(m.authMiddleware.RequireAuth())
	{
		// GET /invoices - List invoices (all authenticated users)
		// Users see only their own shipments' invoices
		// Managers and admins see all organization invoices
		// Scope: invoices.read (user), invoices.read_all (manager/admin)
		invoices.GET("", m.handlers.ListInvoices)

		// GET /invoices/{id} - Get invoice details (all authenticated users)
		// Scope: invoices.read
		invoices.GET("/:id", m.handlers.GetInvoice)

		// GET /invoices/{id}/download - Download invoice PDF (all authenticated users)
		// Scope: invoices.download
		invoices.GET("/:id/download", m.handlers.DownloadInvoice)

		// POST /invoices/{id}/refund - Issue refund via credit note (Manager/Admin only)
		// Scope: invoices.refund
		// Creates credit note and refunds to wallet
		invoices.POST("/:id/refund",
			middleware.RequireRole("manager", "admin"),
			m.handlers.RefundInvoice)

		// POST /invoices/{id}/void - Void invoice (Admin only)
		// Scope: invoices.void
		// Marks invoice as void (irreversible)
		invoices.POST("/:id/void",
			middleware.RequireRole("admin"),
			m.handlers.VoidInvoice)
	}
}
