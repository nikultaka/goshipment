package reviews

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// ReviewsModule handles reviews
type ReviewsModule struct {
	db          *clients.Database
	redis       *clients.RedisClient
	config      *config.Config
	service     *ReviewsService
	redisHelper *redishelper.RedisHelper
	jwtUtil     *utils.JWTUtil
}

// NewReviewsModule creates a new reviews module
func NewReviewsModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *ReviewsModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	service := NewReviewsService(db)

	return &ReviewsModule{
		db:          db,
		redis:       redis,
		config:      cfg,
		service:     service,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
	}
}

// RegisterRoutes registers review routes
func (m *ReviewsModule) RegisterRoutes(router *gin.RouterGroup) {
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)

	reviews := router.Group("/reviews")
	{
		reviews.GET("", m.listReviews) // Public
		reviews.GET("/:id", m.getReview) // Public
	}

	reviewsAuth := router.Group("/reviews")
	reviewsAuth.Use(authMiddleware.RequireAuth())
	{
		reviewsAuth.POST("", m.createReview)
		reviewsAuth.PUT("/:id", m.updateReview)
		reviewsAuth.DELETE("/:id", m.deleteReview)
	}
}
