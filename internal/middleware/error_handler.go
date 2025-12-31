package middleware

import (
	"fmt"
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// ErrorHandler middleware handles panics and errors globally
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				requestID, _ := c.Get("request_id")
				fmt.Printf("[PANIC] Request ID: %v | Error: %v\n", requestID, err)

				// Return internal server error
				response.InternalError(c, "An unexpected error occurred")

				// Abort the request
				c.Abort()
			}
		}()

		// Process request
		c.Next()

		// Check for errors after request processing
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Log error
			requestID, _ := c.Get("request_id")
			fmt.Printf("[ERROR] Request ID: %v | Error: %v\n", requestID, err.Error())

			// If no response has been written yet, send error response
			if !c.Writer.Written() {
				response.InternalError(c, "An error occurred while processing your request")
			}
		}
	}
}

// Recovery middleware recovers from panics and returns 500
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for broken connection
				if isBrokenPipe(err) {
					c.Abort()
					return
				}

				requestID, _ := c.Get("request_id")
				fmt.Printf("[PANIC RECOVERY] Request ID: %v | Error: %v\n", requestID, err)

				response.InternalError(c, "Internal server error")
				c.Abort()
			}
		}()

		c.Next()
	}
}

// isBrokenPipe checks if the error is a broken pipe error
func isBrokenPipe(err interface{}) bool {
	if err == nil {
		return false
	}

	errStr := fmt.Sprintf("%v", err)
	return contains(errStr, "broken pipe") || contains(errStr, "connection reset by peer")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.NotFound(c, "Route not found")
	}
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
	}
}
