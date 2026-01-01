package redishelper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gogin/internal/clients"
)

// RedisHelper provides utility functions for Redis operations
type RedisHelper struct {
	redis *clients.RedisClient
}

// NewRedisHelper creates a new Redis helper
func NewRedisHelper(redis *clients.RedisClient) *RedisHelper {
	return &RedisHelper{redis: redis}
}

// Session Management

// SaveSession stores a user session
func (r *RedisHelper) SaveSession(userID string, sessionID string, data map[string]interface{}, expiry time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("session:%s", sessionID)

	// Add user_id to session data
	data["user_id"] = userID
	data["created_at"] = time.Now().UTC().Unix()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := r.redis.Set(ctx, key, string(jsonData), expiry); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Add to user's session list
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	if err := r.redis.SAdd(ctx, userSessionsKey, sessionID); err != nil {
		return fmt.Errorf("failed to add session to user list: %w", err)
	}
	r.redis.Expire(ctx, userSessionsKey, expiry)

	return nil
}

// GetSession retrieves a user session
func (r *RedisHelper) GetSession(sessionID string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("session:%s", sessionID)

	jsonData, err := r.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return data, nil
}

// DeleteSession removes a user session
func (r *RedisHelper) DeleteSession(sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get session to find user_id
	session, err := r.GetSession(sessionID)
	if err == nil && session["user_id"] != nil {
		userID := session["user_id"].(string)
		userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
		r.redis.SRem(ctx, userSessionsKey, sessionID)
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return r.redis.Del(ctx, key)
}

// DeleteAllUserSessions removes all sessions for a user
func (r *RedisHelper) DeleteAllUserSessions(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)

	// Get all session IDs
	sessionIDs, err := r.redis.GetClient().SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Delete each session
	for _, sessionID := range sessionIDs {
		key := fmt.Sprintf("session:%s", sessionID)
		r.redis.Del(ctx, key)
	}

	// Delete the user sessions set
	return r.redis.Del(ctx, userSessionsKey)
}

// JWT Revocation

// RevokeToken adds a JWT token to the revocation list
func (r *RedisHelper) RevokeToken(tokenID string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("revoked_token:%s", tokenID)
	ttl := time.Until(expiresAt)

	if ttl <= 0 {
		return nil // Token already expired
	}

	return r.redis.Set(ctx, key, "revoked", ttl)
}

// IsTokenRevoked checks if a JWT token is revoked
func (r *RedisHelper) IsTokenRevoked(tokenID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("revoked_token:%s", tokenID)
	return r.redis.Exists(ctx, key)
}

// RevokeAllUserTokens revokes all tokens for a user
func (r *RedisHelper) RevokeAllUserTokens(userID string, tokenIDs []string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}

	for _, tokenID := range tokenIDs {
		key := fmt.Sprintf("revoked_token:%s", tokenID)
		if err := r.redis.Set(ctx, key, "revoked", ttl); err != nil {
			return fmt.Errorf("failed to revoke token %s: %w", tokenID, err)
		}
	}

	return nil
}

// SetLogoutAllTimestamp sets a timestamp when user logged out from all devices
func (r *RedisHelper) SetLogoutAllTimestamp(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("logout_all:%s", userID)
	timestamp := time.Now().Unix()
	// Set with 24 hour expiry (should match or exceed access token expiry)
	return r.redis.Set(ctx, key, timestamp, 24*time.Hour)
}

// GetLogoutAllTimestamp gets the timestamp when user logged out from all devices
func (r *RedisHelper) GetLogoutAllTimestamp(userID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("logout_all:%s", userID)
	result, err := r.redis.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var timestamp int64
	fmt.Sscanf(result, "%d", &timestamp)
	return timestamp, nil
}

// Cache Operations

// CacheSet stores data in cache with expiration
func (r *RedisHelper) CacheSet(key string, data interface{}, expiry time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	cacheKey := fmt.Sprintf("cache:%s", key)
	return r.redis.Set(ctx, cacheKey, string(jsonData), expiry)
}

// CacheGet retrieves data from cache
func (r *RedisHelper) CacheGet(key string, dest interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("cache:%s", key)

	jsonData, err := r.redis.Get(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("cache miss: %w", err)
	}

	if err := json.Unmarshal([]byte(jsonData), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return nil
}

// CacheDelete removes data from cache
func (r *RedisHelper) CacheDelete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("cache:%s", key)
	return r.redis.Del(ctx, cacheKey)
}

// CacheExists checks if a cache key exists
func (r *RedisHelper) CacheExists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("cache:%s", key)
	return r.redis.Exists(ctx, cacheKey)
}

// CacheInvalidatePattern removes all cache entries matching a pattern
func (r *RedisHelper) CacheInvalidatePattern(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cachePattern := fmt.Sprintf("cache:%s", pattern)

	// Scan for matching keys
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = r.redis.GetClient().Scan(ctx, cursor, cachePattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// Delete all matching keys
	if len(keys) > 0 {
		return r.redis.Del(ctx, keys...)
	}

	return nil
}

// Rate Limiting Helpers

// IncrementCounter increments a counter with expiration
func (r *RedisHelper) IncrementCounter(key string, expiry time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.redis.Incr(ctx, key)
	if err != nil {
		return 0, err
	}

	if count == 1 {
		r.redis.Expire(ctx, key, expiry)
	}

	return count, nil
}

// GetCounter retrieves a counter value
func (r *RedisHelper) GetCounter(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	val, err := r.redis.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var count int64
	fmt.Sscanf(val, "%d", &count)
	return count, nil
}

// Lock Operations (distributed locking)

// AcquireLock acquires a distributed lock
func (r *RedisHelper) AcquireLock(key string, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lockKey := fmt.Sprintf("lock:%s", key)
	result, err := r.redis.GetClient().SetNX(ctx, lockKey, "locked", ttl).Result()
	return result, err
}

// ReleaseLock releases a distributed lock
func (r *RedisHelper) ReleaseLock(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lockKey := fmt.Sprintf("lock:%s", key)
	return r.redis.Del(ctx, lockKey)
}
