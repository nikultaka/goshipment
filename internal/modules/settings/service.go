package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/models"
	"gogin/internal/modules/redishelper"
)

type SettingsService struct {
	db          *clients.Database
	redisHelper *redishelper.RedisHelper
	config      *config.Config
}

func NewSettingsService(db *clients.Database, redisHelper *redishelper.RedisHelper, cfg *config.Config) *SettingsService {
	return &SettingsService{
		db:          db,
		redisHelper: redisHelper,
		config:      cfg,
	}
}

// validateKey checks if the setting key is valid
func (s *SettingsService) validateKey(key string) error {
	// Key should contain only alphanumeric characters, underscores, and dots
	validKey := regexp.MustCompile(`^[a-zA-Z0-9_.]+$`)
	if !validKey.MatchString(key) {
		return fmt.Errorf("invalid key format: only alphanumeric characters, underscores, and dots are allowed")
	}
	if len(key) > 255 {
		return fmt.Errorf("key too long: maximum 255 characters")
	}
	return nil
}

// validateValue checks if the value matches the declared type
func (s *SettingsService) validateValue(value, valueType string) error {
	switch valueType {
	case "string":
		return nil
	case "number":
		var n float64
		if err := json.Unmarshal([]byte(value), &n); err != nil {
			return fmt.Errorf("value is not a valid number")
		}
	case "boolean":
		var b bool
		if err := json.Unmarshal([]byte(value), &b); err != nil {
			return fmt.Errorf("value is not a valid boolean")
		}
	case "json":
		var j interface{}
		if err := json.Unmarshal([]byte(value), &j); err != nil {
			return fmt.Errorf("value is not valid JSON")
		}
	default:
		return fmt.Errorf("invalid type: must be one of string, number, boolean, json")
	}
	return nil
}

// encrypt encrypts a string value using AES
func (s *SettingsService) encrypt(plaintext string) (string, error) {
	// Use JWT secret as encryption key (should be 32 bytes for AES-256)
	key := []byte(s.config.OAuth.JWTSecret)
	if len(key) < 32 {
		// Pad the key if it's too short
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		// Truncate if too long
		key = key[:32]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts an encrypted string value
func (s *SettingsService) decrypt(ciphertext string) (string, error) {
	// Use JWT secret as encryption key
	key := []byte(s.config.OAuth.JWTSecret)
	if len(key) < 32 {
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		key = key[:32]
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// getCacheKey returns the Redis cache key for a setting
func (s *SettingsService) getCacheKey(userID *string, key string) string {
	if userID == nil {
		return fmt.Sprintf("setting:system:%s", key)
	}
	return fmt.Sprintf("setting:user:%s:%s", *userID, key)
}

// toResponse converts a models.Setting to SettingResponse
func (s *SettingsService) toResponse(setting *models.Setting) *SettingResponse {
	response := &SettingResponse{
		ID:          setting.ID,
		Key:         setting.Key,
		Value:       setting.Value,
		Type:        setting.Type,
		IsEncrypted: setting.IsEncrypted,
		IsSystem:    setting.IsSystemSetting(),
		CreatedAt:   setting.CreatedAt,
		UpdatedAt:   setting.UpdatedAt,
	}

	if setting.UserID.Valid {
		userID := setting.UserID.String
		response.UserID = &userID
	}

	if setting.Category.Valid {
		response.Category = setting.Category.String
	}

	if setting.Description.Valid {
		response.Description = setting.Description.String
	}

	return response
}

// CreateSystemSetting creates a new system-wide setting
func (s *SettingsService) CreateSystemSetting(req *CreateSettingRequest) (*SettingResponse, error) {
	// Validate key
	if err := s.validateKey(req.Key); err != nil {
		return nil, err
	}

	// Validate value type
	if err := s.validateValue(req.Value, req.Type); err != nil {
		return nil, err
	}

	// Encrypt if needed
	value := req.Value
	if req.IsEncrypted {
		encrypted, err := s.encrypt(req.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}
		value = encrypted
	}

	// Insert into database
	query := `
		INSERT INTO system_settings (category, key, value, value_type, is_encrypted, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, category, key, value, value_type, is_encrypted, description, created_at, updated_at
	`

	now := time.Now().UTC()
	var setting models.Setting

	err := s.db.QueryRow(
		query,
		sql.NullString{String: req.Category, Valid: req.Category != ""},
		req.Key,
		value,
		req.Type,
		req.IsEncrypted,
		sql.NullString{String: req.Description, Valid: req.Description != ""},
		now,
		now,
	).Scan(
		&setting.ID,
		&setting.Category,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.IsEncrypted,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create system setting: %w", err)
	}

	// Decrypt for response if needed
	if setting.IsEncrypted {
		decrypted, err := s.decrypt(setting.Value)
		if err == nil {
			setting.Value = decrypted
		}
	}

	// Cache the setting
	cacheKey := s.getCacheKey(nil, setting.Key)
	s.redisHelper.CacheSet(cacheKey, &setting, 24*time.Hour)

	return s.toResponse(&setting), nil
}

// GetSystemSetting retrieves a system setting by key
func (s *SettingsService) GetSystemSetting(key string) (*SettingResponse, error) {
	// Try cache first
	cacheKey := s.getCacheKey(nil, key)
	var cached models.Setting
	if s.redisHelper.CacheGet(cacheKey, &cached) == nil {
		// Decrypt if needed
		if cached.IsEncrypted {
			decrypted, err := s.decrypt(cached.Value)
			if err == nil {
				cached.Value = decrypted
			}
		}
		return s.toResponse(&cached), nil
	}

	// Query database
	query := `
		SELECT id, category, key, value, value_type, is_encrypted, description, created_at, updated_at
		FROM system_settings
		WHERE key = $1
	`

	var setting models.Setting
	err := s.db.QueryRow(query, key).Scan(
		&setting.ID,
		&setting.Category,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.IsEncrypted,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("system setting not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get system setting: %w", err)
	}

	// Decrypt if needed
	if setting.IsEncrypted {
		decrypted, err := s.decrypt(setting.Value)
		if err == nil {
			setting.Value = decrypted
		}
	}

	// Cache the setting
	s.redisHelper.CacheSet(cacheKey, &setting, 24*time.Hour)

	return s.toResponse(&setting), nil
}

// ListSystemSettings retrieves all system settings with pagination
func (s *SettingsService) ListSystemSettings(page, limit int) (*SettingsListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM system_settings`
	if err := s.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count system settings: %w", err)
	}

	// Query settings
	query := `
		SELECT id, category, key, value, value_type, is_encrypted, description, created_at, updated_at
		FROM system_settings
		ORDER BY key ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list system settings: %w", err)
	}
	defer rows.Close()

	var settings []*SettingResponse
	for rows.Next() {
		var setting models.Setting
		if err := rows.Scan(
			&setting.ID,
			&setting.Category,
			&setting.Key,
			&setting.Value,
			&setting.Type,
			&setting.IsEncrypted,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan system setting: %w", err)
		}

		// Decrypt if needed
		if setting.IsEncrypted {
			decrypted, err := s.decrypt(setting.Value)
			if err == nil {
				setting.Value = decrypted
			}
		}

		settings = append(settings, s.toResponse(&setting))
	}

	if settings == nil {
		settings = []*SettingResponse{}
	}

	totalPages := (total + limit - 1) / limit

	return &SettingsListResponse{
		Settings:   settings,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// UpdateSystemSetting updates a system setting by key
func (s *SettingsService) UpdateSystemSetting(key string, req *UpdateSettingRequest) (*SettingResponse, error) {
	// Validate value type
	if err := s.validateValue(req.Value, req.Type); err != nil {
		return nil, err
	}

	// Encrypt if needed
	value := req.Value
	if req.IsEncrypted {
		encrypted, err := s.encrypt(req.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}
		value = encrypted
	}

	// Update in database
	query := `
		UPDATE settings
		SET value = $1, type = $2, is_encrypted = $3, description = $4, updated_at = $5
		WHERE user_id IS NULL AND key = $6
		RETURNING id, user_id, key, value, type, is_encrypted, description, created_at, updated_at
	`

	var setting models.Setting
	err := s.db.QueryRow(
		query,
		value,
		req.Type,
		req.IsEncrypted,
		sql.NullString{String: req.Description, Valid: req.Description != ""},
		time.Now().UTC(),
		key,
	).Scan(
		&setting.ID,
		&setting.UserID,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.IsEncrypted,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("system setting not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update system setting: %w", err)
	}

	// Decrypt for response if needed
	if setting.IsEncrypted {
		decrypted, err := s.decrypt(setting.Value)
		if err == nil {
			setting.Value = decrypted
		}
	}

	// Invalidate cache
	cacheKey := s.getCacheKey(nil, key)
	s.redisHelper.CacheDelete(cacheKey)

	return s.toResponse(&setting), nil
}

// DeleteSystemSetting deletes a system setting by key
func (s *SettingsService) DeleteSystemSetting(key string) error {
	query := `DELETE FROM system_settings WHERE key = $1`

	result, err := s.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete system setting: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("system setting not found")
	}

	// Invalidate cache
	cacheKey := s.getCacheKey(nil, key)
	s.redisHelper.CacheDelete(cacheKey)

	return nil
}

// GetUserSetting retrieves a user setting by key
func (s *SettingsService) GetUserSetting(userID, key string) (*SettingResponse, error) {
	// Try cache first
	cacheKey := s.getCacheKey(&userID, key)
	var cached models.Setting
	if s.redisHelper.CacheGet(cacheKey, &cached) == nil {
		// Decrypt if needed
		if cached.IsEncrypted {
			decrypted, err := s.decrypt(cached.Value)
			if err == nil {
				cached.Value = decrypted
			}
		}
		return s.toResponse(&cached), nil
	}

	// Query database
	query := `
		SELECT id, user_id, key, value, value_type, description, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1 AND key = $2
	`

	var setting models.Setting
	err := s.db.QueryRow(query, userID, key).Scan(
		&setting.ID,
		&setting.UserID,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.IsEncrypted,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user setting not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user setting: %w", err)
	}

	// Decrypt if needed
	if setting.IsEncrypted {
		decrypted, err := s.decrypt(setting.Value)
		if err == nil {
			setting.Value = decrypted
		}
	}

	// Cache the setting
	s.redisHelper.CacheSet(cacheKey, &setting, 24*time.Hour)

	return s.toResponse(&setting), nil
}

// ListUserSettings retrieves all user settings
func (s *SettingsService) ListUserSettings(userID string, page, limit int) (*SettingsListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM user_settings WHERE user_id = $1`
	if err := s.db.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count user settings: %w", err)
	}

	// Query settings
	query := `
		SELECT id, user_id, key, value, value_type, description, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
		ORDER BY key ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user settings: %w", err)
	}
	defer rows.Close()

	var settings []*SettingResponse
	for rows.Next() {
		var setting models.Setting
		if err := rows.Scan(
			&setting.ID,
			&setting.UserID,
			&setting.Key,
			&setting.Value,
			&setting.Type,
			&setting.IsEncrypted,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user setting: %w", err)
		}

		// Decrypt if needed
		if setting.IsEncrypted {
			decrypted, err := s.decrypt(setting.Value)
			if err == nil {
				setting.Value = decrypted
			}
		}

		settings = append(settings, s.toResponse(&setting))
	}

	if settings == nil {
		settings = []*SettingResponse{}
	}

	totalPages := (total + limit - 1) / limit

	return &SettingsListResponse{
		Settings:   settings,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// CreateOrUpdateUserSetting creates or updates a user setting
func (s *SettingsService) CreateOrUpdateUserSetting(userID, key string, req *UpdateSettingRequest) (*SettingResponse, error) {
	// Validate key
	if err := s.validateKey(key); err != nil {
		return nil, err
	}

	// Validate value type
	if err := s.validateValue(req.Value, req.Type); err != nil {
		return nil, err
	}

	// Encrypt if needed
	value := req.Value
	if req.IsEncrypted {
		encrypted, err := s.encrypt(req.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}
		value = encrypted
	}

	// Upsert in database
	query := `
		INSERT INTO settings (user_id, key, value, type, is_encrypted, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, key)
		DO UPDATE SET value = EXCLUDED.value, type = EXCLUDED.type, is_encrypted = EXCLUDED.is_encrypted,
		              description = EXCLUDED.description, updated_at = EXCLUDED.updated_at
		RETURNING id, user_id, key, value, type, is_encrypted, description, created_at, updated_at
	`

	now := time.Now().UTC()
	var setting models.Setting

	err := s.db.QueryRow(
		query,
		userID,
		key,
		value,
		req.Type,
		req.IsEncrypted,
		sql.NullString{String: req.Description, Valid: req.Description != ""},
		now,
		now,
	).Scan(
		&setting.ID,
		&setting.UserID,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.IsEncrypted,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create or update user setting: %w", err)
	}

	// Decrypt for response if needed
	if setting.IsEncrypted {
		decrypted, err := s.decrypt(setting.Value)
		if err == nil {
			setting.Value = decrypted
		}
	}

	// Invalidate cache
	cacheKey := s.getCacheKey(&userID, key)
	s.redisHelper.CacheDelete(cacheKey)

	return s.toResponse(&setting), nil
}

// DeleteUserSetting deletes a user setting by key
func (s *SettingsService) DeleteUserSetting(userID, key string) error {
	query := `DELETE FROM user_settings WHERE user_id = $1 AND key = $2`

	result, err := s.db.Exec(query, userID, key)
	if err != nil {
		return fmt.Errorf("failed to delete user setting: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user setting not found")
	}

	// Invalidate cache
	cacheKey := s.getCacheKey(&userID, key)
	s.redisHelper.CacheDelete(cacheKey)

	return nil
}
