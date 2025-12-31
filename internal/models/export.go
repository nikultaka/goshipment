package models

import (
	"database/sql"
	"time"
)

// Export represents a user export job
type Export struct {
	ID             string         `json:"id" db:"id"`
	UserID         string         `json:"user_id" db:"user_id"`
	ExportType     string         `json:"export_type" db:"export_type"`       // wallet_transactions, shipments, invoices, etc.
	Format         string         `json:"format" db:"format"`                 // csv, json, xlsx
	DeliveryMethod string         `json:"delivery_method" db:"delivery_method"` // download, email
	EmailSentAt    sql.NullTime   `json:"email_sent_at,omitempty" db:"email_sent_at"`
	Status         string         `json:"status" db:"status"`                 // queued, processing, ready, failed, expired
	ErrorMessage   sql.NullString `json:"error_message,omitempty" db:"error_message"`
	FileID         sql.NullString `json:"file_id,omitempty" db:"file_id"`
	FileSize       sql.NullInt64  `json:"file_size,omitempty" db:"file_size"`
	RowCount       sql.NullInt32  `json:"row_count,omitempty" db:"row_count"`
	ExpiresAt      sql.NullTime   `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	CompletedAt    sql.NullTime   `json:"completed_at,omitempty" db:"completed_at"`
}

// IsReady returns true if the export is ready for download
func (e *Export) IsReady() bool {
	return e.Status == "ready"
}

// IsFailed returns true if the export has failed
func (e *Export) IsFailed() bool {
	return e.Status == "failed"
}

// IsProcessing returns true if the export is currently being processed
func (e *Export) IsProcessing() bool {
	return e.Status == "queued" || e.Status == "processing"
}
