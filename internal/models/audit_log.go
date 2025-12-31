package models

import (
	"database/sql"
	"time"
)

// AuditLog represents an immutable audit log entry
type AuditLog struct {
	ID          string         `json:"id" db:"id"`
	UserID      sql.NullString `json:"user_id,omitempty" db:"user_id"`
	ClientID    sql.NullString `json:"client_id,omitempty" db:"client_id"`
	Action      string         `json:"action" db:"action"` // login, logout, create, update, delete, etc.
	Resource    string         `json:"resource" db:"resource"` // users, oauth_clients, etc.
	ResourceID  sql.NullString `json:"resource_id,omitempty" db:"resource_id"`
	IPAddress   string         `json:"ip_address" db:"ip_address"`
	UserAgent   sql.NullString `json:"user_agent,omitempty" db:"user_agent"`
	Metadata    sql.NullString `json:"metadata,omitempty" db:"metadata"` // JSON
	Status      string         `json:"status" db:"status"` // success, failure
	ErrorMsg    sql.NullString `json:"error_msg,omitempty" db:"error_msg"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}
