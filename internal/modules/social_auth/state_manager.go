package social_auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"gogin/internal/modules/redishelper"
)

// StateManager handles OAuth state parameter generation and validation
type StateManager struct {
	redisHelper *redishelper.RedisHelper
}

// NewStateManager creates a new state manager
func NewStateManager(redisHelper *redishelper.RedisHelper) *StateManager {
	return &StateManager{
		redisHelper: redisHelper,
	}
}

// GenerateState generates a cryptographically random state parameter
// and stores it in Redis with a 10-minute expiry for CSRF protection
func (sm *StateManager) GenerateState() (string, error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}

	// Encode to base64 URL-safe string
	state := base64.URLEncoding.EncodeToString(b)

	// Store in Redis with 10-minute expiry
	key := fmt.Sprintf("oauth_state:%s", state)
	data := map[string]interface{}{
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := sm.redisHelper.CacheSet(key, data, 10*time.Minute); err != nil {
		return "", fmt.Errorf("failed to store state in Redis: %w", err)
	}

	return state, nil
}

// ValidateState validates an OAuth state parameter
// Returns true if valid, false otherwise
// The state is deleted after validation (one-time use)
func (sm *StateManager) ValidateState(state string) bool {
	if state == "" {
		return false
	}

	key := fmt.Sprintf("oauth_state:%s", state)

	// Try to get the state from cache
	var data map[string]interface{}
	err := sm.redisHelper.CacheGet(key, &data)
	if err != nil {
		// State doesn't exist or has expired
		return false
	}

	// Delete the state (one-time use for CSRF protection)
	sm.redisHelper.CacheDelete(key)

	return true
}
