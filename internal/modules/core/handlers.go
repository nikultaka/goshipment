package core

import (
	"net/http"
	"time"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// healthCheck performs a simple health check
// @Summary Health check
// @Description Check if the API is running
// @Tags Core
// @Produce json
// @Success 200 {object} response.Response{data=object{status=string,time=string}}
// @Router /health [get]
func (m *CoreModule) healthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "OK", gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// status returns detailed system status
// @Summary System status
// @Description Get detailed system status including database, Redis, and NATS health
// @Tags Core
// @Produce json
// @Success 200 {object} response.Response{data=object{status=string,timestamp=string,services=object,app=object}}
// @Failure 503 {object} response.Response{data=object{status=string,timestamp=string,services=object,app=object}}
// @Router /status [get]
func (m *CoreModule) status(c *gin.Context) {
	// Check database health
	dbHealthy := true
	if err := m.db.HealthCheck(); err != nil {
		dbHealthy = false
	}

	// Check Redis health
	redisHealthy := true
	if err := m.redis.HealthCheck(); err != nil {
		redisHealthy = false
	}

	// Check NATS health
	natsHealthy := true
	if err := m.nats.HealthCheck(); err != nil {
		natsHealthy = false
	}

	// Get database stats
	dbStats := m.db.Stats()

	// Overall status
	overallStatus := "healthy"
	if !dbHealthy || !redisHealthy || !natsHealthy {
		overallStatus = "degraded"
	}

	statusCode := http.StatusOK
	if overallStatus == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	response.Success(c, statusCode, "System status", gin.H{
		"status": overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services": gin.H{
			"database": gin.H{
				"healthy": dbHealthy,
				"stats": gin.H{
					"open_connections": dbStats.OpenConnections,
					"in_use":          dbStats.InUse,
					"idle":            dbStats.Idle,
				},
			},
			"redis": gin.H{
				"healthy": redisHealthy,
			},
			"nats": gin.H{
				"healthy": natsHealthy,
			},
		},
		"app": gin.H{
			"name":    m.config.App.Name,
			"version": m.config.App.Version,
			"env":     m.config.App.Env,
		},
	})
}
