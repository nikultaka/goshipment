package users

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/modules/storage"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// UsersModule handles user management
type UsersModule struct {
	service        *UserService
	sessionService *SessionService
	twofaService   *TwoFAService
	storageService *storage.StorageService
	redisHelper    *redishelper.RedisHelper
	authMiddleware *middleware.AuthMiddleware
	config         *config.Config
}

// NewUsersModule creates a new users module
func NewUsersModule(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *UsersModule {
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)
	redisHelper := redishelper.NewRedisHelper(redis)
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil, redisHelper)

	service := NewUserService(db, jwtUtil, redisHelper, cfg)
	sessionService := NewSessionService(db)
	twofaService := NewTwoFAService(db.DB, service)
	storageService := storage.NewStorageService(db, cfg)

	return &UsersModule{
		service:        service,
		sessionService: sessionService,
		twofaService:   twofaService,
		storageService: storageService,
		redisHelper:    redisHelper,
		authMiddleware: authMiddleware,
		config:         cfg,
	}
}

// RegisterRoutes registers user routes
func (m *UsersModule) RegisterRoutes(router *gin.RouterGroup) {
	// Authentication routes (public and protected)
	auth := router.Group("/auth")
	{
		// Phone OTP authentication
		auth.POST("/phone/send-otp", m.sendPhoneOTP)
		auth.POST("/phone/verify-otp", m.verifyPhoneOTP)

		// Token refresh (public)
		auth.POST("/refresh", m.refreshToken)

		// 2FA login verification (public)
		auth.POST("/2fa/login/verify", m.verify2FALogin)

		// 2FA management (protected)
		twofa := auth.Group("/2fa")
		twofa.Use(m.authMiddleware.RequireAuth())
		{
			twofa.GET("/status", m.get2FAStatus)
			twofa.POST("/setup", m.setup2FA)
			twofa.POST("/verify", m.verify2FASetup)
			twofa.POST("/disable", m.disable2FA)

			// Backup code management
			twofa.GET("/backup-codes", m.getBackupCodes)
			twofa.POST("/backup-codes/regenerate", m.regenerateBackupCodes)
		}
	}

	// Email verification (public route - accessible from email link)
	router.GET("/verify-email", m.verifyEmailWithToken)

	// Password reset (public routes - accessible without authentication)
	router.POST("/forgot-password", m.forgotPassword)
	router.POST("/reset-password", m.resetPassword)

	users := router.Group("/users")
	{
		// Public routes
		users.POST("/register", m.register)
		users.POST("/login", m.login)

		// Protected routes
		auth := users.Group("")
		auth.Use(m.authMiddleware.RequireAuth())
		{
			auth.GET("/me", m.getProfile)
			auth.PUT("/me", m.updateProfile)
			auth.PUT("/me/password", m.changePassword)
			auth.POST("/me/avatar", m.uploadAvatar)
			auth.DELETE("/me/avatar", m.deleteAvatar)

			// Multi-email management
			auth.GET("/me/emails", m.listUserEmails)
			auth.POST("/me/emails", m.addUserEmail)
			auth.PUT("/me/emails/:email_id/primary", m.setPrimaryEmail)
			auth.DELETE("/me/emails/:email_id", m.deleteUserEmail)
			auth.POST("/me/emails/:email_id/verify", m.resendEmailVerification)

			// Multi-phone management
			auth.GET("/me/phones", m.listUserPhones)
			auth.POST("/me/phones", m.addUserPhone)
			auth.PUT("/me/phones/:phone_id/primary", m.setPrimaryPhone)
			auth.DELETE("/me/phones/:phone_id", m.deleteUserPhone)
			auth.POST("/me/phones/:phone_id/verify", m.resendPhoneVerification)

			auth.POST("/logout", m.logout)
			auth.DELETE("/me", m.deleteAccount)

			// Session management (multi-device support)
			auth.GET("/me/sessions", m.getSessions)
			auth.DELETE("/me/sessions/:session_id", m.terminateSession)
			auth.POST("/me/sessions/logout-all", m.logoutAllDevices)
			auth.POST("/me/sessions/logout-others", m.logoutOtherDevices)
			auth.PATCH("/me/sessions/:session_id/trust", m.trustSession)
		}

		// Admin routes
		admin := users.Group("")
		admin.Use(m.authMiddleware.RequireAuth())
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("", m.adminCreateUser)
			admin.GET("", m.listUsers)
			admin.GET("/:id", m.getUserByID)
			admin.PUT("/:id", m.updateUser)
			admin.DELETE("/:id", m.adminDeleteUser)
			admin.PUT("/:id/status", m.updateUserStatus)

			// Admin session management
			admin.GET("/:id/sessions", m.adminGetUserSessions)
			admin.DELETE("/:id/sessions/:session_id", m.adminTerminateUserSession)
			admin.DELETE("/:id/sessions", m.adminTerminateAllUserSessions)

			// Admin user settings
			admin.GET("/:id/settings", m.adminGetUserSettings)
			admin.PUT("/:id/settings", m.adminUpdateUserSettings)
		}
	}
}
