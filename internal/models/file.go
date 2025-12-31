package models

import (
	"database/sql"
	"time"
)

// File represents an uploaded file
type File struct {
	ID          string         `json:"id" db:"id"`
	UserID      sql.NullString `json:"user_id,omitempty" db:"user_id"`
	FileName    string         `json:"file_name" db:"file_name"`
	OriginalName string        `json:"original_name" db:"original_name"`
	MimeType    string         `json:"mime_type" db:"mime_type"`
	Size        int64          `json:"size" db:"size"` // bytes
	Path        string         `json:"path" db:"path"`
	StorageType string         `json:"storage_type" db:"storage_type"` // local, s3
	Visibility  string         `json:"visibility" db:"visibility"` // public, private
	Metadata    sql.NullString `json:"metadata,omitempty" db:"metadata"` // JSON
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt   sql.NullTime   `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsPublic returns true if the file is publicly accessible
func (f *File) IsPublic() bool {
	return f.Visibility == "public"
}
