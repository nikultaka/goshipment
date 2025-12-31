package models

import (
	"database/sql"
	"time"
)

// Review represents a user review
type Review struct {
	ID          string         `json:"id" db:"id"`
	UserID      string         `json:"user_id" db:"user_id"`
	ResourceType string        `json:"resource_type" db:"resource_type"` // product, service, etc.
	ResourceID  string         `json:"resource_id" db:"resource_id"`
	Rating      int            `json:"rating" db:"rating"` // 1-5
	Title       sql.NullString `json:"title,omitempty" db:"title"`
	Content     string         `json:"content" db:"content"`
	Status      string         `json:"status" db:"status"` // pending, approved, rejected
	ModeratedBy sql.NullString `json:"moderated_by,omitempty" db:"moderated_by"`
	ModeratedAt sql.NullTime   `json:"moderated_at,omitempty" db:"moderated_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt   sql.NullTime   `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsApproved returns true if the review is approved
func (r *Review) IsApproved() bool {
	return r.Status == "approved"
}

// IsPending returns true if the review is pending moderation
func (r *Review) IsPending() bool {
	return r.Status == "pending"
}
