package referrals

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ReferralModule represents the referral module
type ReferralModule struct {
	handlers       *ReferralHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewReferralModule creates a new referral module
func NewReferralModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *ReferralModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewReferralService(db)
	handlers := NewReferralHandlers(service)

	return &ReferralModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the referral routes
func (m *ReferralModule) RegisterRoutes(router *gin.RouterGroup) {
	referrals := router.Group("/referrals")
	referrals.Use(m.authMiddleware.RequireAuth())
	{
		// GET /referrals - List referral redemptions
		// Scope: commissions.read (referrals drive commissions)
		// Sales Reps see only their own referrals (as referrer)
		// Admins/Managers see all organization referrals
		// Read-only - referrals are immutable financial records
		referrals.GET("", m.handlers.ListReferrals)

		// GET /referrals/:id - Get referral redemption by ID
		// Scope: commissions.read
		// Must be owned by requesting Sales Rep OR accessed by Admin/Manager
		// Read-only - no modification allowed (immutable ledger)
		referrals.GET("/:id", m.handlers.GetReferralByID)

		// NO POST/PATCH/DELETE endpoints
		// Referrals are created automatically during user registration
		// They are immutable for audit/financial integrity
		// Only status can be changed (revoke/expire) via admin tools (not exposed here)
	}
}
