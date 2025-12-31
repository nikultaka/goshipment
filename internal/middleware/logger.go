package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger middleware logs HTTP requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID
		requestID, _ := c.Get("request_id")

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Get method
		method := c.Request.Method

		// Get error if any
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Build query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Log format: [TIME] STATUS | LATENCY | IP | METHOD | PATH | REQUEST_ID | ERROR
		logMessage := fmt.Sprintf("[%s] %d | %13v | %15s | %-7s | %s | %s",
			time.Now().Format("2006-01-02 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			requestID,
		)

		if errorMessage != "" {
			logMessage += " | " + errorMessage
		}

		// Color-code by status
		if statusCode >= 500 {
			fmt.Printf("\033[31m%s\033[0m\n", logMessage) // Red for 5xx
		} else if statusCode >= 400 {
			fmt.Printf("\033[33m%s\033[0m\n", logMessage) // Yellow for 4xx
		} else if statusCode >= 300 {
			fmt.Printf("\033[36m%s\033[0m\n", logMessage) // Cyan for 3xx
		} else {
			fmt.Printf("\033[32m%s\033[0m\n", logMessage) // Green for 2xx
		}
	}
}
