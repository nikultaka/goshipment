package storage

import (
	"time"
)

// UploadRequest represents file metadata for upload
type UploadRequest struct {
	Visibility string `form:"visibility" binding:"required,oneof=public private"`
	Metadata   string `form:"metadata"` // Optional JSON metadata
}

// FileResponse represents a file response
type FileResponse struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id,omitempty"`
	FileName     string            `json:"file_name"`
	OriginalName string            `json:"original_name"`
	MimeType     string            `json:"mime_type"`
	Size         int64             `json:"size"`
	StorageType  string            `json:"storage_type"`
	Visibility   string            `json:"visibility"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	DownloadURL  string            `json:"download_url"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// UpdateFileRequest represents a file update request
type UpdateFileRequest struct {
	Visibility string `json:"visibility" binding:"omitempty,oneof=public private"`
	Metadata   string `json:"metadata"` // Optional JSON metadata
}

// FilesListResponse represents a paginated list of files
type FilesListResponse struct {
	Files      []*FileResponse `json:"files"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// FileUploadResponse represents the response after successful upload
type FileUploadResponse struct {
	File *FileResponse `json:"file"`
}
