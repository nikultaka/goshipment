package organizations

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// OrganizationsModule handles organization management
type OrganizationsModule struct {
	handlers       *OrganizationHandlers
	inviteHandlers *OrganizationInviteHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewOrganizationsModule creates a new organizations module
func NewOrganizationsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *OrganizationsModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewOrganizationService(db)
	handlers := NewOrganizationHandlers(service)

	inviteService := NewOrganizationInviteService(db)
	inviteHandlers := NewOrganizationInviteHandlers(inviteService)

	return &OrganizationsModule{
		handlers:       handlers,
		inviteHandlers: inviteHandlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers organization routes
func (m *OrganizationsModule) RegisterRoutes(router *gin.RouterGroup) {
	// Organization routes (authenticated only)
	orgRoutes := router.Group("/organization")
	orgRoutes.Use(m.authMiddleware.RequireAuth())
	{
		// Get current user's organization
		orgRoutes.GET("/current", m.handlers.GetMyOrganization)

		// Organization members (manager+ required)
		orgRoutes.GET("/members", m.handlers.GetOrganizationMembers)

		// Organization settings (admin+ required)
		orgRoutes.GET("/settings", m.handlers.GetOrganizationSettings)
		orgRoutes.PATCH("/settings", m.handlers.UpdateOrganizationSettings)

		// Organization preferences (all authenticated users can read, manager+ can write)
		orgRoutes.GET("/preferences", m.handlers.GetOrganizationPreferences)
		orgRoutes.PATCH("/preferences", m.handlers.UpdateOrganizationPreferences)

		// Organization invitations (admin/owner only for create, list, revoke)
		orgRoutes.POST("/invites", m.inviteHandlers.CreateInvite)
		orgRoutes.GET("/invites", m.inviteHandlers.ListInvites)
		orgRoutes.DELETE("/invites/:id", m.inviteHandlers.RevokeInvite)
	}

	// Public invitation routes (no auth required - token-based)
	inviteRoutes := router.Group("/organization/invites")
	{
		// Accept/reject invitations (public with token)
		inviteRoutes.POST("/accept", m.inviteHandlers.AcceptInvite)
		inviteRoutes.POST("/reject", m.inviteHandlers.RejectInvite)
	}
}
