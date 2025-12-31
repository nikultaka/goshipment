package models

import (
	"database/sql"
	"time"
)

// Setting represents a system or user setting
type Setting struct {
	ID          string         `json:"id" db:"id"`
	UserID      sql.NullString `json:"user_id,omitempty" db:"user_id"` // NULL for system settings
	Category    sql.NullString `json:"category,omitempty" db:"category"` // Category for system settings
	Key         string         `json:"key" db:"key"`
	Value       string         `json:"value" db:"value"` // JSON value
	Type        string         `json:"type" db:"type"` // string, number, boolean, json
	IsEncrypted bool           `json:"is_encrypted" db:"is_encrypted"`
	Description sql.NullString `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// IsSystemSetting returns true if this is a system-wide setting
func (s *Setting) IsSystemSetting() bool {
	return !s.UserID.Valid
}

// IsUserSetting returns true if this is a user-specific setting
func (s *Setting) IsUserSetting() bool {
	return s.UserID.Valid
}
