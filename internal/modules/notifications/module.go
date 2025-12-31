package notifications

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/modules/sendgrid"
	"gogin/internal/modules/twilio"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// NotificationsModule handles notifications
type NotificationsModule struct {
	db           *clients.Database
	redis        *clients.RedisClient
	nats         *clients.NATSClient
	config       *config.Config
	service      *NotificationsService
	sendgrid     *sendgrid.SendGridClient
	twilio       *twilio.TwilioClient
	redisHelper  *redishelper.RedisHelper
	jwtUtil      *utils.JWTUtil
}

// NewNotificationsModule creates a new notifications module
func NewNotificationsModule(db *clients.Database, redis *clients.RedisClient, nats *clients.NATSClient, cfg *config.Config) *NotificationsModule {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	sendgridClient := sendgrid.NewSendGridClient(cfg.SMTP)
	twilioClient := twilio.NewTwilioClient(cfg.Twilio)
	service := NewNotificationsService(db, nats, sendgridClient, twilioClient)

	return &NotificationsModule{
		db:          db,
		redis:       redis,
		nats:        nats,
		config:      cfg,
		service:     service,
		sendgrid:    sendgridClient,
		twilio:      twilioClient,
		redisHelper: redisHelper,
		jwtUtil:     jwtUtil,
	}
}

// RegisterRoutes registers notification routes
func (m *NotificationsModule) RegisterRoutes(router *gin.RouterGroup) {
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)

	notifications := router.Group("/notifications")
	notifications.Use(authMiddleware.RequireAuth())
	{
		notifications.POST("", m.createNotification)
		notifications.GET("", m.listNotifications)
		notifications.GET("/:id", m.getNotification)
		notifications.PUT("/:id/read", m.markAsRead)
		notifications.PUT("/mark-all-read", m.markAllAsRead)
		notifications.DELETE("/:id", m.deleteNotification)
		notifications.POST("/test-email", m.testEmail)
		notifications.POST("/test-sms", m.testSMS)
	}

	// Admin routes for notification templates
	adminTemplates := router.Group("/admin/notification-templates")
	adminTemplates.Use(authMiddleware.RequireAuth(), middleware.RequireAdmin())
	{
		adminTemplates.GET("", m.listNotificationTemplates)
		adminTemplates.POST("", m.createNotificationTemplate)
		adminTemplates.PUT("/:id", m.updateNotificationTemplate)
	}
}
