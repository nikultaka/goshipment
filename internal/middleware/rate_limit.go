package middleware

import (
	"context"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting using Redis
type RateLimiter struct {
	redis       *clients.RedisClient
	maxRequests int
	window      time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis *clients.RedisClient, maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:       redis,
		maxRequests: maxRequests,
		window:      window,
	}
}

// Limit returns a middleware that limits requests per IP
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP or user ID if authenticated)
		identifier := rl.getIdentifier(c)

		// Check rate limit
		allowed, err := rl.checkLimit(identifier)
		if err != nil {
			// Log error but allow request to proceed
			fmt.Printf("[RATE LIMIT ERROR] %v\n", err)
			c.Next()
			return
		}

		if !allowed {
			response.TooManyRequests(c, "Rate limit exceeded. Please try again later.")
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkLimit checks if the request is within rate limit
func (rl *RateLimiter) checkLimit(identifier string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("rate_limit:%s", identifier)

	// Increment counter
	count, err := rl.redis.Incr(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	// Set expiration on first request
	if count == 1 {
		if err := rl.redis.Expire(ctx, key, rl.window); err != nil {
			return false, fmt.Errorf("failed to set rate limit expiration: %w", err)
		}
	}

	// Check if limit exceeded
	return count <= int64(rl.maxRequests), nil
}

// getIdentifier returns a unique identifier for the client
func (rl *RateLimiter) getIdentifier(c *gin.Context) string {
	// Prefer user ID if authenticated
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// RateLimitByKey limits requests by a custom key
func RateLimitByKey(redis *clients.RedisClient, key string, maxRequests int, window time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	// Increment counter
	count, err := redis.Incr(ctx, rateLimitKey)
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	// Set expiration on first request
	if count == 1 {
		if err := redis.Expire(ctx, rateLimitKey, window); err != nil {
			return false, fmt.Errorf("failed to set rate limit expiration: %w", err)
		}
	}

	// Check if limit exceeded
	return count <= int64(maxRequests), nil
}
