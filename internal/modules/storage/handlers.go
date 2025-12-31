package storage

import (
	"fmt"
	"net/http"
	"strconv"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// uploadFile handles file upload
// @Summary Upload a file
// @Description Upload a file to storage (public or private)
// @Tags Storage
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File to upload"
// @Param visibility formData string true "File visibility (public or private)"
// @Param metadata formData string false "Optional JSON metadata"
// @Success 201 {object} response.Response{data=FileUploadResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 413 {object} response.Response
// @Router /storage/upload [post]
func (m *StorageModule) uploadFile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID := ""
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(string)
	}

	// Parse multipart form
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "No file provided")
		return
	}

	// Upload file
	uploadedFile, err := m.service.UploadFile(file, &req, userID)
	if err != nil {
		if err.Error() == fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", m.config.Storage.MaxFileSize) {
			response.Error(c, http.StatusRequestEntityTooLarge, "File too large", err.Error())
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	// Get base URL for download links
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://" + c.Request.Host
	}

	fileResp := m.service.ToFileResponse(uploadedFile, baseURL)

	response.Success(c, http.StatusCreated, "File uploaded successfully", FileUploadResponse{
		File: fileResp,
	})
}

// listFiles retrieves files with pagination
// @Summary List files
// @Description Get a paginated list of files (public files + user's private files if authenticated)
// @Tags Storage
// @Produce json
// @Param visibility query string false "Filter by visibility (public or private)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} response.Response{data=FilesListResponse}
// @Failure 400 {object} response.Response
// @Router /storage/files [get]
func (m *StorageModule) listFiles(c *gin.Context) {
	// Get user ID from context (optional)
	userID := ""
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(string)
	}

	// Get query parameters
	visibility := c.Query("visibility")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// List files
	files, total, err := m.service.ListFiles(userID, visibility, page, limit)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get base URL for download links
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://" + c.Request.Host
	}

	// Convert to response DTOs
	fileResponses := make([]*FileResponse, len(files))
	for i, file := range files {
		fileResponses[i] = m.service.ToFileResponse(file, baseURL)
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "Files retrieved successfully", FilesListResponse{
		Files:      fileResponses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// getFile retrieves file metadata by ID
// @Summary Get file metadata
// @Description Get metadata for a specific file by ID
// @Tags Storage
// @Produce json
// @Param id path string true "File ID"
// @Success 200 {object} response.Response{data=object{file=FileResponse}}
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /storage/files/{id} [get]
func (m *StorageModule) getFile(c *gin.Context) {
	fileID := c.Param("id")

	// Get user ID from context (optional)
	userID := ""
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(string)
	}

	file, err := m.service.GetFile(fileID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(c, "Access denied")
			return
		}
		response.NotFound(c, "File not found")
		return
	}

	// Get base URL for download links
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://" + c.Request.Host
	}

	fileResp := m.service.ToFileResponse(file, baseURL)

	response.Success(c, http.StatusOK, "File retrieved successfully", gin.H{
		"file": fileResp,
	})
}

// downloadFile handles file download
// @Summary Download a file
// @Description Download a file by ID
// @Tags Storage
// @Produce application/octet-stream
// @Param id path string true "File ID"
// @Success 200 {file} binary "File content"
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /storage/files/{id}/download [get]
func (m *StorageModule) downloadFile(c *gin.Context) {
	fileID := c.Param("id")

	// Get user ID from context (optional)
	userID := ""
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(string)
	}

	file, err := m.service.GetFile(fileID, userID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(c, "Access denied")
			return
		}
		response.NotFound(c, "File not found")
		return
	}

	// Set headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.OriginalName))
	c.Header("Content-Type", file.MimeType)

	// Serve the file
	c.File(file.Path)
}

// updateFile updates file metadata
// @Summary Update file metadata
// @Description Update file visibility and metadata
// @Tags Storage
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "File ID"
// @Param request body UpdateFileRequest true "File update details"
// @Success 200 {object} response.Response{data=object{file=FileResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /storage/files/{id} [put]
func (m *StorageModule) updateFile(c *gin.Context) {
	fileID := c.Param("id")

	// Get user ID from context (required for update)
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}

	var req UpdateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	file, err := m.service.UpdateFile(fileID, &req, userID.(string))
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(c, "Access denied")
			return
		}
		if err.Error() == "file not found" {
			response.NotFound(c, "File not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	// Get base URL for download links
	baseURL := fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host)
	if baseURL == "://" {
		baseURL = "http://" + c.Request.Host
	}

	fileResp := m.service.ToFileResponse(file, baseURL)

	response.Success(c, http.StatusOK, "File updated successfully", gin.H{
		"file": fileResp,
	})
}

// deleteFile soft deletes a file
// @Summary Delete a file
// @Description Soft delete a file by ID
// @Tags Storage
// @Produce json
// @Security BearerAuth
// @Param id path string true "File ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /storage/files/{id} [delete]
func (m *StorageModule) deleteFile(c *gin.Context) {
	fileID := c.Param("id")

	// Get user ID from context (required for delete)
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Authentication required")
		return
	}

	err := m.service.DeleteFile(fileID, userID.(string))
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(c, "Access denied")
			return
		}
		if err.Error() == "file not found" {
			response.NotFound(c, "File not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "File deleted successfully", nil)
}
