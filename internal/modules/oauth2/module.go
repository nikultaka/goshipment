package oauth2

import (
	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/middleware"
	"gogin/internal/modules/redishelper"
	"gogin/internal/utils"

	"github.com/gin-gonic/gin"
)

// OAuth2Module handles OAuth 2.0 authorization server
type OAuth2Module struct {
	db           *clients.Database
	redis        *clients.RedisClient
	config       *config.Config
	service      *OAuth2Service
	redisHelper  *redishelper.RedisHelper
	jwtUtil      *utils.JWTUtil
	jwtUtilRS256 *utils.JWTUtilRS256 // RS256 JWT utility (optional)
}

// NewOAuth2Module creates a new OAuth2 module
func NewOAuth2Module(db *clients.Database, redis *clients.RedisClient, cfg *config.Config) *OAuth2Module {
	redisHelper := redishelper.NewRedisHelper(redis)
	jwtUtil := utils.NewJWTUtil(cfg.OAuth.JWTSecret, cfg.OAuth.JWTIssuer)

	// Try to load RS256 keys if configured
	var jwtUtilRS256 *utils.JWTUtilRS256
	if cfg.OAuth.JWTAlgorithm == "RS256" && cfg.OAuth.JWTPrivateKeyPath != "" {
		keyPair, err := utils.LoadRSAKeyPairFromFiles(
			cfg.OAuth.JWTPrivateKeyPath,
			cfg.OAuth.JWTPublicKeyPath,
			cfg.OAuth.JWTKeyID,
		)
		if err == nil {
			jwtUtilRS256 = utils.NewJWTUtilRS256(keyPair, cfg.OAuth.JWTIssuer)
		}
		// If error loading keys, log but continue with HMAC
	}

	service := NewOAuth2Service(db, redisHelper, jwtUtil, cfg)

	return &OAuth2Module{
		db:           db,
		redis:        redis,
		config:       cfg,
		service:      service,
		redisHelper:  redisHelper,
		jwtUtil:      jwtUtil,
		jwtUtilRS256: jwtUtilRS256,
	}
}

// RegisterRoutes registers OAuth2 routes
func (m *OAuth2Module) RegisterRoutes(router *gin.RouterGroup) {
	oauth := router.Group("/oauth")
	authMiddleware := middleware.NewAuthMiddleware(m.jwtUtil, m.redisHelper)
	{
		// Authorization endpoints
		oauth.GET("/authorize", authMiddleware.RequireAuth(), m.authorize)     // GET for browser redirects
		oauth.POST("/authorize", authMiddleware.RequireAuth(), m.authorize)    // POST for API calls
		oauth.POST("/token", m.token)                                          // Public endpoint

		// Token management
		oauth.POST("/revoke", authMiddleware.RequireAuth(), m.revoke)
		oauth.POST("/introspect", authMiddleware.RequireAuth(), m.introspect)

		// OIDC endpoints
		oauth.GET("/jwks.json", m.getJWKS)                                     // Public JWKS endpoint
		oauth.GET("/userinfo", authMiddleware.RequireAuth(), m.getUserInfo)   // UserInfo endpoint
	}

	// OIDC Discovery endpoint (at well-known path)
	router.GET("/.well-known/openid-configuration", m.getOIDCDiscovery)
}
