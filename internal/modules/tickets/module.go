package tickets

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/modules/storage"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

type TicketsModule struct {
	service        *TicketsService
	authMiddleware *middleware.AuthMiddleware
}

// NewTicketsModule creates a new instance of the tickets module
func NewTicketsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *TicketsModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	storageService := storage.NewStorageService(db, cfg)
	service := NewTicketsService(db, redisHelper, cfg, storageService)

	return &TicketsModule{
		service:        service,
		authMiddleware: middleware.NewAuthMiddleware(jwtUtil, redisHelper),
	}
}

// RegisterRoutes registers all ticket-related routes
func (m *TicketsModule) RegisterRoutes(router *gin.RouterGroup) {
	tickets := router.Group("/tickets")
	tickets.Use(m.authMiddleware.RequireAuth())

	// User routes (authenticated users)
	{
		tickets.POST("", m.createTicket)              // Create ticket
		tickets.GET("/my", m.listMyTickets)           // List my tickets
		tickets.GET("/:id", m.getTicket)              // Get ticket details
		tickets.PUT("/:id", m.updateTicket)           // Update ticket
		tickets.DELETE("/:id", m.deleteTicket)        // Delete ticket
		tickets.POST("/:id/replies", m.createReply)   // Add reply

		// Attachment routes
		tickets.POST("/:id/attachments", m.uploadAttachment)                    // Upload attachment
		tickets.GET("/:id/attachments", m.listAttachments)                      // List attachments
		tickets.DELETE("/:id/attachments/:attachment_id", m.deleteAttachment)   // Delete attachment
	}

	// Admin routes
	admin := tickets.Group("")
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("", m.listAllTickets)                // List all tickets
		admin.PUT("/:id/status", m.updateTicketStatus) // Update status
		admin.PUT("/:id/assign", m.assignTicket)       // Assign ticket
	}
}
