package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"net/http" // for /health endpoint - testing only

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/apiclient"
	"gogin/internal/modules/audit"
	"gogin/internal/modules/core"
	"gogin/internal/modules/notifications"
	"gogin/internal/modules/oauth2"
	"gogin/internal/modules/reviews"
	"gogin/internal/modules/settings"
	"gogin/internal/modules/social_auth"
	"gogin/internal/modules/storage"
	"gogin/internal/modules/tickets"
	"gogin/internal/modules/users"
	"gogin/internal/response"
	"gogin/internal/workers"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "gogin/docs" // Import generated docs
)

// @title           Gogin API
// @version         1.0
// @description     A comprehensive REST API built with Go and Gin framework
// @description     This API provides user authentication, profile management, and admin operations.

// @contact.name   API Support
// @contact.email  support@gogin.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      
// @BasePath  /api/v1
// @schemes   https http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Core
// @tag.description System health and status endpoints

// @tag.name Users
// @tag.description User authentication and profile management

// @tag.name Admin
// @tag.description Administrative user management endpoints (requires admin role)

// @tag.name OAuth2.1
// @tag.description OAuth 2.1 authorization server endpoints

// @tag.name API Clients
// @tag.description OAuth client management (admin only)

// @tag.name Notifications
// @tag.description User notifications and messaging

// @tag.name Reviews
// @tag.description Review and rating system

// @tag.name Settings
// @tag.description System and user settings management

// @tag.name Tickets
// @tag.description Support ticket management

// @tag.name Storage
// @tag.description File storage and management (upload/download public and private files)

// @tag.name Social Auth
// @tag.description Google and Facebook OAuth2 login integration

// @tag.name Admin - Audit Logs
// @tag.description Admin audit log query and export (requires admin role)

// @tag.name Admin - Notification Templates
// @tag.description Admin notification template management (requires admin role)

func main() {
	// testing add by Zubair
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	/*log.Println("[API] listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}*/
	// end of testing
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := clients.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database connected")

	// Initialize Redis
	redis, err := clients.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Println("✓ Redis connected")

	// Initialize NATS
	nats, err := clients.NewNATSClient(cfg.NATS)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()
	log.Println("✓ NATS connected")

	// Start background workers
	workerManager := workers.NewWorkerManager(db, nats, cfg)
	if err := workerManager.Start(); err != nil {
		log.Printf("Warning: Failed to start workers: %v", err)
	}
	defer workerManager.Stop()

	// Create Gin router
	router := gin.New()

	// Apply global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.CORS(cfg.App.AllowOrigins))

	// Add audit logging middleware
	auditLogger := middleware.NewAuditLogger(db)
	router.Use(auditLogger.Log())

	// Set version in context
	router.Use(func(c *gin.Context) {
		c.Set("version", cfg.App.Version)
		c.Next()
	})

	// Trust proxies
	if len(cfg.App.TrustedProxies) > 0 {
		router.SetTrustedProxies(cfg.App.TrustedProxies)
	}

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		response.Success(c, 200, "Go API System is running", gin.H{
			"name":    cfg.App.Name,
			"version": cfg.App.Version,
			"env":     cfg.App.Env,
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 group
	v1 := router.Group("/api/v1")

	// Core routes (health, status)
	coreModule := core.NewCoreModule(db, redis, nats, cfg)
	coreModule.RegisterRoutes(v1)

	// Users module (authentication)
	usersModule := users.NewUsersModule(db, redis, cfg)
	usersModule.RegisterRoutes(v1)
	log.Println("✓ Users module registered")

	// Social Auth module (Google, Facebook OAuth)
	socialAuthModule := social_auth.NewSocialAuthModule(db, redis, cfg)
	socialAuthModule.RegisterRoutes(v1)
	log.Println("✓ Social Auth module registered")

	// OAuth2 authorization server
	oauth2Module := oauth2.NewOAuth2Module(db, redis, cfg)
	oauth2Module.RegisterRoutes(v1)
	log.Println("✓ OAuth2 module registered")

	// API Client management (admin only)
	apiClientModule := apiclient.NewAPIClientModule(db, redis, cfg)
	apiClientModule.RegisterRoutes(v1)
	log.Println("✓ API Client module registered")

	// Notifications module
	notificationsModule := notifications.NewNotificationsModule(db, redis, nats, cfg)
	notificationsModule.RegisterRoutes(v1)
	log.Println("✓ Notifications module registered")

	// Reviews module
	reviewsModule := reviews.NewReviewsModule(db, redis, cfg)
	reviewsModule.RegisterRoutes(v1)
	log.Println("✓ Reviews module registered")

	// Settings module
	settingsModule := settings.NewSettingsModule(db, redis, cfg)
	settingsModule.RegisterRoutes(v1)
	log.Println("✓ Settings module registered")

	// Tickets module
	ticketsModule := tickets.NewTicketsModule(db, redis, cfg)
	ticketsModule.RegisterRoutes(v1)
	log.Println("✓ Tickets module registered")

	// Storage module
	storageModule := storage.NewStorageModule(db, redis, cfg)
	storageModule.RegisterRoutes(v1)
	log.Println("✓ Storage module registered")

	// Audit module (admin only)
	auditModule := audit.NewAuditModule(db, redis, cfg)
	auditModule.RegisterRoutes(v1)
	log.Println("✓ Audit module registered")

	// Apply rate limiting after authentication routes
	rateLimiter := middleware.NewRateLimiter(redis, cfg.App.RateLimitRPS, 60)
	v1.Use(rateLimiter.Limit())

	// Handle 404
	router.NoRoute(middleware.NotFoundHandler())

	// Handle 405
	router.NoMethod(middleware.MethodNotAllowedHandler())

	// Start server
	serverAddr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("🚀 Server starting on %s", serverAddr)
	log.Printf("   Environment: %s", cfg.App.Env)
	log.Printf("   Version: %s", cfg.App.Version)

	// Graceful shutdown
	go func() {
		if err := router.Run(serverAddr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	log.Println("Server stopped")
}
