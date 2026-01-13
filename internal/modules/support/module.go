package support

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// SupportModule represents the support ticket module
type SupportModule struct {
	handlers       *SupportHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewSupportModule creates a new support module
func NewSupportModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *SupportModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewSupportService(db)
	handlers := NewSupportHandlers(service)

	return &SupportModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the support ticket routes
func (m *SupportModule) RegisterRoutes(router *gin.RouterGroup) {
	support := router.Group("/support")
	support.Use(m.authMiddleware.RequireAuth())
	{
		// POST /support - Create a support ticket (requires support.create scope)
		// Scope: support.create
		// All authenticated users can create tickets
		support.POST("", m.handlers.CreateSupportTicket)

		// GET /support - List support tickets
		// Scope: support.read (users see own tickets)
		// Scope: support.read_all (managers/admins see all org tickets - implicit by role)
		support.GET("", m.handlers.ListSupportTickets)

		// GET /support/:id - Get ticket details
		// Scope: support.read
		// Users can view own tickets, managers/admins can view all org tickets
		support.GET("/:id", m.handlers.GetSupportTicket)

		// POST /support/:id/reply - Reply to a ticket
		// Scope: support.reply
		// Users can reply to own tickets, managers/admins can reply to any org ticket
		support.POST("/:id/reply", m.handlers.ReplySupportTicket)

		// POST /support/:id/manage - Manage ticket (Admin/Manager only)
		// Scope: support.manage
		// Actions: assign, escalate, close, reopen
		// Only managers and admins can use this endpoint
		support.POST("/:id/manage", m.handlers.ManageSupportTicket)
	}
}
