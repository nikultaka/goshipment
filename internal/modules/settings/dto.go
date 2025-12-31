package settings

import "time"

// CreateSettingRequest represents the request body for creating a setting
type CreateSettingRequest struct {
	Category    string `json:"category"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=string number boolean json"`
	IsEncrypted bool   `json:"is_encrypted"`
	Description string `json:"description"`
}

// UpdateSettingRequest represents the request body for updating a setting
type UpdateSettingRequest struct {
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=string number boolean json"`
	IsEncrypted bool   `json:"is_encrypted"`
	Description string `json:"description"`
}

// SettingResponse represents a sanitized setting response
type SettingResponse struct {
	ID          string    `json:"id"`
	UserID      *string   `json:"user_id,omitempty"`
	Category    string    `json:"category,omitempty"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Type        string    `json:"type"`
	IsEncrypted bool      `json:"is_encrypted"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SettingsListResponse represents a list of settings with pagination
type SettingsListResponse struct {
	Settings   []*SettingResponse `json:"settings"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}
