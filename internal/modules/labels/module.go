package labels

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// LabelsModule represents the labels module
type LabelsModule struct {
	handlers       *LabelHandlers
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewLabelsModule creates a new labels module
func NewLabelsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *LabelsModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewLabelService(db)
	handlers := NewLabelHandlers(service)

	return &LabelsModule{
		handlers:       handlers,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers the label routes
func (m *LabelsModule) RegisterRoutes(router *gin.RouterGroup) {
	labels := router.Group("/labels")
	labels.Use(m.authMiddleware.RequireAuth())
	{
		// Purchase labels for a shipment (POST only - irreversible operation)
		// Requires: labels.purchase scope (enforced by RBAC at handler level if needed)
		labels.POST("/purchase", m.handlers.PurchaseLabels)

		// Get label and tracking info for a package
		// Requires: labels.read scope (enforced by RBAC at handler level if needed)
		labels.GET("/:package_id", m.handlers.GetLabel)
	}
}
