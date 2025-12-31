package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// StorageService handles file storage business logic
type StorageService struct {
	db     *clients.Database
	config *config.Config
}

// NewStorageService creates a new storage service
func NewStorageService(db *clients.Database, cfg *config.Config) *StorageService {
	return &StorageService{
		db:     db,
		config: cfg,
	}
}

// UploadFile handles file upload
func (s *StorageService) UploadFile(file *multipart.FileHeader, req *UploadRequest, userID string) (*models.File, error) {
	// Validate file size
	if file.Size > s.config.Storage.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.config.Storage.MaxFileSize)
	}

	// Generate unique filename
	fileID := uuid.New().String()
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s%s", fileID, ext)

	// Determine storage path
	var filePath string
	var storageType string

	if s.config.Storage.Type == "s3" {
		// TODO: Implement S3 upload
		return nil, fmt.Errorf("S3 storage not yet implemented")
	} else {
		// Local storage
		storageType = "local"
		filePath = filepath.Join(s.config.Storage.BasePath, fileName)

		// Ensure storage directory exists
		if err := os.MkdirAll(s.config.Storage.BasePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}

		// Save file to disk
		if err := s.saveFile(file, filePath); err != nil {
			return nil, fmt.Errorf("failed to save file: %w", err)
		}
	}

	// Generate external_id for the file
	externalID := fmt.Sprintf("FILE-%s", uuid.New().String()[:8])

	// Create file record
	fileModel := &models.File{
		ID:           fileID,
		UserID:       sql.NullString{String: userID, Valid: userID != ""},
		FileName:     fileName,
		OriginalName: file.Filename,
		MimeType:     file.Header.Get("Content-Type"),
		Size:         file.Size,
		Path:         filePath,
		StorageType:  storageType,
		Visibility:   req.Visibility,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Store metadata if provided
	if req.Metadata != "" {
		fileModel.Metadata = sql.NullString{String: req.Metadata, Valid: true}
	}

	// Insert into database using correct column names
	query := `
		INSERT INTO files (
			id, external_id, original_name, file_size, mime_type,
			storage_path, storage_provider, uploaded_by, owned_by,
			visibility, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9, true, $10, $11)
	`

	_, err := s.db.DB.Exec(query,
		fileModel.ID,
		externalID,
		fileModel.OriginalName,
		fileModel.Size,
		fileModel.MimeType,
		fileModel.Path,
		fileModel.StorageType,
		userID, // Both uploaded_by and owned_by
		fileModel.Visibility,
		fileModel.CreatedAt,
		fileModel.UpdatedAt,
	)

	if err != nil {
		// Clean up file if database insert fails
		if storageType == "local" {
			os.Remove(filePath)
		}
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return fileModel, nil
}

// saveFile saves uploaded file to disk
func (s *StorageService) saveFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// GetFile retrieves a file by ID
func (s *StorageService) GetFile(fileID string, userID string) (*models.File, error) {
	query := `
		SELECT id, uploaded_by as user_id, original_name, original_name as file_name,
		       mime_type, file_size as size, storage_path as path,
		       storage_provider as storage_type, visibility,
		       created_at, updated_at, deleted_at
		FROM files
		WHERE id = $1 AND deleted_at IS NULL
	`

	var file models.File
	var metadata sql.NullString
	err := s.db.DB.QueryRow(query, fileID).Scan(
		&file.ID,
		&file.UserID,
		&file.OriginalName,
		&file.FileName,
		&file.MimeType,
		&file.Size,
		&file.Path,
		&file.StorageType,
		&file.Visibility,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	file.Metadata = metadata

	// Check permissions for private files
	if file.Visibility == "private" {
		// If file has a user, only that user can access it
		if file.UserID.Valid && file.UserID.String != userID {
			return nil, fmt.Errorf("access denied")
		}
	}

	return &file, nil
}

// ListFiles retrieves files with pagination
func (s *StorageService) ListFiles(userID string, visibility string, page, limit int) ([]*models.File, int, error) {
	offset := (page - 1) * limit

	// Build query based on filters
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argCount := 1

	// Handle visibility and access control
	if visibility == "public" {
		// Show only public files
		conditions = append(conditions, "visibility = 'public'")
	} else if visibility == "private" {
		// Show only private files (user must be authenticated)
		if userID != "" {
			conditions = append(conditions, fmt.Sprintf("visibility = 'private' AND uploaded_by = $%d", argCount))
			args = append(args, userID)
			argCount++
		} else {
			// No unauthenticated user can see private files, return empty
			conditions = append(conditions, "1 = 0")
		}
	} else {
		// No visibility filter: show public files + authenticated user's private files
		if userID != "" {
			conditions = append(conditions, fmt.Sprintf("(visibility = 'public' OR uploaded_by = $%d)", argCount))
			args = append(args, userID)
			argCount++
		} else {
			// If no user, only show public files
			conditions = append(conditions, "visibility = 'public'")
		}
	}

	whereClause := strings.Join(conditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM files WHERE %s", whereClause)
	var total int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	// Get files
	query := fmt.Sprintf(`
		SELECT id, uploaded_by as user_id, original_name as file_name, original_name, mime_type, file_size as size, storage_path as path, storage_provider as storage_type, visibility, '{}' as metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	files := []*models.File{}
	for rows.Next() {
		var file models.File
		err := rows.Scan(
			&file.ID,
			&file.UserID,
			&file.FileName,
			&file.OriginalName,
			&file.MimeType,
			&file.Size,
			&file.Path,
			&file.StorageType,
			&file.Visibility,
			&file.Metadata,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, &file)
	}

	return files, total, nil
}

// DeleteFile soft deletes a file
func (s *StorageService) DeleteFile(fileID string, userID string) error {
	// First check if file exists and user has permission
	file, err := s.GetFile(fileID, userID)
	if err != nil {
		return err
	}

	// Check ownership for deletion
	if file.UserID.Valid && file.UserID.String != userID {
		return fmt.Errorf("access denied")
	}

	// Soft delete
	query := `
		UPDATE files
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3
	`

	_, err = s.db.DB.Exec(query, time.Now().UTC(), time.Now().UTC(), fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Optionally: Delete physical file after soft delete
	// For now, we'll keep the file on disk for recovery purposes

	return nil
}

// UpdateFile updates file metadata and visibility
func (s *StorageService) UpdateFile(fileID string, req *UpdateFileRequest, userID string) (*models.File, error) {
	// First check if file exists and user has permission
	file, err := s.GetFile(fileID, userID)
	if err != nil {
		return nil, err
	}

	// Check ownership for update
	if file.UserID.Valid && file.UserID.String != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Build update query dynamically based on provided fields
	updates := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argCount := 2

	if req.Visibility != "" {
		updates = append(updates, fmt.Sprintf("visibility = $%d", argCount))
		args = append(args, req.Visibility)
		argCount++
		file.Visibility = req.Visibility
	}

	// Note: metadata field doesn't exist in current schema, skipping
	// if req.Metadata != "" {
	// 	updates = append(updates, fmt.Sprintf("metadata = $%d", argCount))
	// 	args = append(args, req.Metadata)
	// 	argCount++
	// 	file.Metadata = sql.NullString{String: req.Metadata, Valid: true}
	// }

	args = append(args, fileID)

	query := fmt.Sprintf(`
		UPDATE files
		SET %s
		WHERE id = $%d
	`, strings.Join(updates, ", "), argCount)

	_, err = s.db.DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update file: %w", err)
	}

	file.UpdatedAt = time.Now().UTC()

	return file, nil
}

// ToFileResponse converts a File model to FileResponse DTO
func (s *StorageService) ToFileResponse(file *models.File, baseURL string) *FileResponse {
	response := &FileResponse{
		ID:           file.ID,
		FileName:     file.FileName,
		OriginalName: file.OriginalName,
		MimeType:     file.MimeType,
		Size:         file.Size,
		StorageType:  file.StorageType,
		Visibility:   file.Visibility,
		DownloadURL:  fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, file.ID),
		CreatedAt:    file.CreatedAt,
		UpdatedAt:    file.UpdatedAt,
	}

	if file.UserID.Valid {
		response.UserID = file.UserID.String
	}

	// Parse metadata if exists
	if file.Metadata.Valid && file.Metadata.String != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(file.Metadata.String), &metadata); err == nil {
			response.Metadata = metadata
		}
	}

	return response
}
