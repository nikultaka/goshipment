package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Response represents the standard API response structure
type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    interface{}    `json:"data,omitempty"`
	Meta    Meta           `json:"meta"`
	Errors  []ResponseError `json:"errors,omitempty"`
}

// Meta contains metadata about the response
type Meta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
	Version   string `json:"version"`
	Actor     Actor  `json:"actor"`
}

// Actor contains information about who made the request
type Actor struct {
	UserID   string `json:"user_id,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	Role     string `json:"role,omitempty"`
}

// ResponseError represents a single error in the response
type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Success sends a successful response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	resp := Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    buildMeta(c),
	}
	c.JSON(statusCode, resp)
}

// Fail sends a failed response with errors
func Fail(c *gin.Context, statusCode int, message string, errors []ResponseError) {
	resp := Response{
		Success: false,
		Message: message,
		Meta:    buildMeta(c),
		Errors:  errors,
	}
	c.JSON(statusCode, resp)
}

// Error sends a single error response
func Error(c *gin.Context, statusCode int, message string, errorCode string) {
	Fail(c, statusCode, message, []ResponseError{{Code: errorCode, Message: message}})
}

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, errors []ResponseError) {
	Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
}

// Unauthorized sends an unauthorized response
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, "UNAUTHORIZED")
}

// Forbidden sends a forbidden response
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, "FORBIDDEN")
}

// NotFound sends a not found response
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message, "NOT_FOUND")
}

// InternalError sends an internal server error response
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message, "INTERNAL_ERROR")
}

// BadRequest sends a bad request response
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message, "BAD_REQUEST")
}

// TooManyRequests sends a rate limit exceeded response
func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, message, "RATE_LIMIT_EXCEEDED")
}

// buildMeta creates the meta information for the response
func buildMeta(c *gin.Context) Meta {
	requestID, _ := c.Get("request_id")
	if requestID == nil {
		requestID = uuid.New().String()
	}

	version, _ := c.Get("version")
	if version == nil {
		version = "v1"
	}

	actor := Actor{}
	if userID, exists := c.Get("user_id"); exists {
		actor.UserID = userID.(string)
	}
	if clientID, exists := c.Get("client_id"); exists {
		actor.ClientID = clientID.(string)
	}
	if role, exists := c.Get("role"); exists {
		actor.Role = role.(string)
	}

	return Meta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID.(string),
		Version:   version.(string),
		Actor:     actor,
	}
}

// NewError creates a new ResponseError instance
func NewError(code, message, field string) ResponseError {
	return ResponseError{
		Code:    code,
		Message: message,
		Field:   field,
	}
}
