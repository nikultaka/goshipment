package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"gogin/internal/clients"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditLogger middleware logs API requests to audit_logs table
type AuditLogger struct {
	db *clients.Database
}

// NewAuditLogger creates a new audit logger middleware
func NewAuditLogger(db *clients.Database) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log returns middleware that logs requests to audit log
func (a *AuditLogger) Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health/status endpoints
		if c.Request.URL.Path == "/api/v1/health" || c.Request.URL.Path == "/api/v1/status" {
			c.Next()
			return
		}

		// Record start time
		startTime := time.Now()

		// Get request body
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			requestBody = string(bodyBytes)
		}

		// Process request
		c.Next()

		// Get user info from context
		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			userID = uid.(string)
		}

		clientID := ""
		if cid, exists := c.Get("api_client_id"); exists {
			clientID = cid.(string)
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"query":          c.Request.URL.RawQuery,
			"ip":             c.ClientIP(),
			"user_agent":     c.Request.UserAgent(),
			"status_code":    c.Writer.Status(),
			"duration_ms":    time.Since(startTime).Milliseconds(),
			"request_id":     c.GetString("request_id"),
		}

		metadataJSON, _ := json.Marshal(metadata)

		// Insert audit log asynchronously
		go a.insertAuditLog(
			userID,
			clientID,
			c.Request.Method+" "+c.Request.URL.Path,
			requestBody,
			string(metadataJSON),
			c.ClientIP(),
		)
	}
}

func (a *AuditLogger) insertAuditLog(userID, clientID, action, requestData, metadata, ipAddress string) {
	// Generate event_id
	eventID := "evt_" + uuid.New().String()[:8]

	// Determine actor type
	actorType := "system"
	if userID != "" {
		actorType = "user"
	} else if clientID != "" {
		actorType = "api_client"
	}

	// Parse action to extract HTTP method and path
	// e.g., "GET /api/v1/notifications" -> action: "list_notifications", resource_type: "notifications"
	resourceType := "api"

	// Simple status determination
	status := "success"

	// Validate if clientID is a valid UUID, otherwise set to NULL
	var apiClientID *string
	if clientID != "" {
		if _, err := uuid.Parse(clientID); err == nil {
			// It's a valid UUID
			apiClientID = &clientID
		}
		// If not a valid UUID (like "web"), leave as NULL and use actor_identifier instead
	}

	query := `
		INSERT INTO audit_logs (
			id, event_id, actor_type, user_id, api_client_id,
			actor_identifier, actor_ip, action, resource_type,
			business_purpose, status, occurred_at
		)
		VALUES (
			$1, $2, $3, NULLIF($4, '')::uuid, $5,
			$6, $7::inet, $8, $9,
			$10, $11, NOW()
		)
	`

	_, err := a.db.Exec(query,
		uuid.New(),           // id
		eventID,              // event_id
		actorType,            // actor_type
		userID,               // user_id
		apiClientID,          // api_client_id (NULL if not a valid UUID)
		clientID,             // actor_identifier (stores "web" or UUID)
		ipAddress,            // actor_ip
		action,               // action
		resourceType,         // resource_type
		"api_access",         // business_purpose
		status,               // status
	)

	if err != nil {
		// Log error but don't fail the request
		println("Failed to insert audit log:", err.Error())
	}
}
