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
	"gogin/internal/modules/addresses"
	"gogin/internal/modules/apiclient"
	"gogin/internal/modules/audit"
	"gogin/internal/modules/commissions"
	"gogin/internal/modules/dashboard"
	"gogin/internal/modules/core"
	"gogin/internal/modules/invoices"
	"gogin/internal/modules/labels"
	"gogin/internal/modules/notifications"
	"gogin/internal/modules/oauth2"
	"gogin/internal/modules/organizations"
	"gogin/internal/modules/pickups"
	"gogin/internal/modules/carriers"
	"gogin/internal/modules/geo"
	"gogin/internal/modules/referrals"
	"gogin/internal/modules/reviews"
	"gogin/internal/modules/settings"
	"gogin/internal/modules/shipment_estimates"
	"gogin/internal/modules/shipments"
	"gogin/internal/modules/social_auth"
	"gogin/internal/modules/storage"
	"gogin/internal/modules/support"
	// "gogin/internal/modules/tickets" // OLD - replaced by support module
	"gogin/internal/modules/users"
	"gogin/internal/modules/wallet"
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

// @host      localhost:8081
// @BasePath  /api/v1
// @schemes   http https

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

// @tag.name Admin - Dashboard
// @tag.description Admin dashboard financial metrics and analytics (requires admin role)

// @tag.name Admin - Notification Templates
// @tag.description Admin notification template management (requires admin role)

// @tag.name Shipment Estimates
// @tag.description Shipment rate quotes and cost estimation (guest-friendly, no auth required)

// @tag.name Labels
// @tag.description Purchase and manage shipping labels (MONETIZATION POINT - requires authentication)

// @tag.name Wallet
// @tag.description Organization wallet management - balance, top-ups, refunds, and transactions (PAYMENT ENGINE)

// @tag.name Invoices
// @tag.description Legal and financial records - immutable invoices, credit notes, and PDF downloads (ACCOUNTING & COMPLIANCE)

// @tag.name Pickups
// @tag.description Schedule and manage carrier pickups for shipments (LOGISTICS & FULFILLMENT)

// @tag.name Support
// @tag.description Customer support ticket system - create tickets, reply to conversations, manage resolutions (CUSTOMER SERVICE)

// @tag.name Commissions
// @tag.description Commission earnings tracking for Sales Representatives - view commissions, request payouts, process payments (FINANCIAL RECORDS)

// @tag.name Referrals
// @tag.description Referral redemption records - immutable ledger tracking referred users and commission eligibility (READ-ONLY FINANCIAL RECORDS)

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

	// Tickets module (OLD - replaced by Support module)
	// ticketsModule := tickets.NewTicketsModule(db, redis, cfg)
	// ticketsModule.RegisterRoutes(v1)
	// log.Println("✓ Tickets module registered")

	// Storage module
	storageModule := storage.NewStorageModule(db, redis, cfg)
	storageModule.RegisterRoutes(v1)
	log.Println("✓ Storage module registered")

	// Audit module (admin only)
	auditModule := audit.NewAuditModule(db, redis, cfg)
	auditModule.RegisterRoutes(v1)
	log.Println("✓ Audit module registered")

	// Dashboard module (admin only - financial metrics and analytics)
	dashboardModule := dashboard.NewDashboardModule(db, redis, cfg)
	dashboardModule.RegisterRoutes(v1)
	log.Println("✓ Dashboard module registered")

	// Organizations module
	organizationsModule := organizations.NewOrganizationsModule(db, redis, cfg)
	organizationsModule.RegisterRoutes(v1)
	log.Println("✓ Organizations module registered")

	// Shipment Estimates module (guest-friendly, no auth required)
	estimatesModule := shipment_estimates.NewEstimatesModule(db, redis, cfg)
	estimatesModule.RegisterRoutes(v1)
	log.Println("✓ Shipment Estimates module registered")

	// Shipments module (create and manage shipments from estimates)
	shipmentsModule := shipments.NewShipmentsModule(db, redis, cfg)
	shipmentsModule.RegisterRoutes(v1)
	log.Println("✓ Shipments module registered")

	// Address Book module (manage sender and recipient addresses)
	addressModule := addresses.NewAddressModule(db, redis, cfg)
	addressModule.RegisterRoutes(v1)
	log.Println("✓ Address Book module registered")

	// Labels module (shipping label purchase and tracking)
	labelsModule := labels.NewLabelsModule(db, redis, cfg)
	labelsModule.RegisterRoutes(v1)
	log.Println("✓ Labels module registered")

	// Wallet module (payment engine - balance, topups, refunds)
	walletModule := wallet.NewWalletModule(db, redis, cfg)
	walletModule.RegisterRoutes(v1)
	log.Println("✓ Wallet module registered")

	// Invoices module (legal & financial records - immutable invoices, credit notes, PDF)
	invoicesModule := invoices.NewInvoicesModule(db, redis, cfg)
	invoicesModule.RegisterRoutes(v1)
	log.Println("✓ Invoices module registered")

	// Pickups module (carrier pickup scheduling and management)
	pickupsModule := pickups.NewPickupModule(db, redis, cfg)
	pickupsModule.RegisterRoutes(v1)
	log.Println("✓ Pickups module registered")

	// Support module (customer support tickets and conversations)
	supportModule := support.NewSupportModule(db, redis, cfg)
	supportModule.RegisterRoutes(v1)
	log.Println("✓ Support module registered")

	// Commissions module (sales rep commission tracking and payouts)
	commissionsModule := commissions.NewCommissionModule(db, redis, cfg)
	commissionsModule.RegisterRoutes(v1)
	log.Println("✓ Commissions module registered")

	// Referrals module (referral redemptions - immutable ledger, read-only)
	referralsModule := referrals.NewReferralModule(db, redis, cfg)
	referralsModule.RegisterRoutes(v1)
	log.Println("✓ Referrals module registered")

	// Geo module (countries, regions - reference data)
	geoModule := geo.NewGeoModule(db, redis, cfg)
	geoModule.RegisterRoutes(v1)
	log.Println("✓ Geo module registered")

	// Carriers module (carrier info and availability)
	carriersModule := carriers.NewCarrierModule(db, redis, cfg)
	carriersModule.RegisterRoutes(v1)
	log.Println("✓ Carriers module registered")

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
