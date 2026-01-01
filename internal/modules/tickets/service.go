package tickets

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"time"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/models"
	"gogin/internal/modules/redishelper"
	"gogin/internal/modules/storage"

	"github.com/google/uuid"
)

type TicketsService struct {
	db             *clients.Database
	redisHelper    *redishelper.RedisHelper
	config         *config.Config
	storageService *storage.StorageService
}

func NewTicketsService(db *clients.Database, redisHelper *redishelper.RedisHelper, cfg *config.Config, storageService *storage.StorageService) *TicketsService {
	return &TicketsService{
		db:             db,
		redisHelper:    redisHelper,
		config:         cfg,
		storageService: storageService,
	}
}

// toTicketResponse converts a models.SupportTicket to TicketResponse
func (s *TicketsService) toTicketResponse(ticket *models.SupportTicket) *TicketResponse {
	response := &TicketResponse{
		ID:          ticket.ID,
		UserID:      ticket.UserID,
		Subject:     ticket.Subject,
		Description: ticket.Description,
		Status:      ticket.Status,
		Priority:    ticket.Priority,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
	}

	if ticket.Category.Valid {
		category := ticket.Category.String
		response.Category = &category
	}

	if ticket.AssignedTo.Valid {
		assignedTo := ticket.AssignedTo.String
		response.AssignedTo = &assignedTo
	}

	if ticket.ResolvedAt.Valid {
		response.ResolvedAt = &ticket.ResolvedAt.Time
	}

	if ticket.ClosedAt.Valid {
		response.ClosedAt = &ticket.ClosedAt.Time
	}

	return response
}

// toReplyResponse converts a models.SupportTicketReply to ReplyResponse
func (s *TicketsService) toReplyResponse(reply *models.SupportTicketReply) *ReplyResponse {
	response := &ReplyResponse{
		ID:        reply.ID,
		TicketID:  reply.TicketID,
		UserID:    reply.UserID,
		IsStaff:   reply.IsStaff,
		Content:   reply.Content,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
	}

	if reply.DeletedAt.Valid {
		response.DeletedAt = &reply.DeletedAt.Time
	}

	return response
}

// CreateTicket creates a new support ticket
func (s *TicketsService) CreateTicket(userID string, req *CreateTicketRequest) (*TicketResponse, error) {
	// Generate external_id and ticket_number
	externalID := fmt.Sprintf("TKT-%s", time.Now().Format("20060102-150405"))
	ticketNumber := fmt.Sprintf("TICKET-%d", time.Now().Unix())

	// Get customer email - TODO: fetch from users table if needed
	customerEmail := req.CustomerEmail
	if customerEmail == "" {
		customerEmail = "unknown@example.com" // Fallback, should be fetched from user
	}

	query := `
		INSERT INTO support_tickets (
			external_id, ticket_number, user_id, customer_email,
			subject, description, priority, category, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, external_id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
	`

	now := time.Now().UTC()
	var ticket models.SupportTicket

	category := sql.NullString{String: req.Category, Valid: req.Category != ""}

	err := s.db.QueryRow(
		query,
		externalID,
		ticketNumber,
		userID,
		customerEmail,
		req.Subject,
		req.Description,
		req.Priority,
		category,
		"open",
		now,
		now,
	).Scan(
		&ticket.ID,
		&ticket.ExternalID,
		&ticket.UserID,
		&ticket.Subject,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Category,
		&ticket.AssignedTo,
		&ticket.ResolvedAt,
		&ticket.ClosedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// Invalidate user tickets cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user_tickets:%s", userID))

	return s.toTicketResponse(&ticket), nil
}

// GetTicketByID retrieves a ticket by ID
func (s *TicketsService) GetTicketByID(ticketID string) (*TicketResponse, error) {
	query := `
		SELECT id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
		FROM support_tickets
		WHERE id = $1
	`

	var ticket models.SupportTicket
	err := s.db.QueryRow(query, ticketID).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.Subject,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Category,
		&ticket.AssignedTo,
		&ticket.ResolvedAt,
		&ticket.ClosedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return s.toTicketResponse(&ticket), nil
}

// GetTicketWithReplies retrieves a ticket with all its replies
func (s *TicketsService) GetTicketWithReplies(ticketID string) (*TicketDetailResponse, error) {
	// Get ticket
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Get replies
	query := `
		SELECT
			id,
			ticket_id,
			COALESCE(sender_user_id::text, '') as user_id,
			CASE WHEN sender_type IN ('agent', 'system') THEN true ELSE false END as is_staff,
			message_text as content,
			created_at,
			created_at as updated_at,
			NULL::timestamp as deleted_at
		FROM support_ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}
	defer rows.Close()

	var replies []*ReplyResponse
	for rows.Next() {
		var reply models.SupportTicketReply
		if err := rows.Scan(
			&reply.ID,
			&reply.TicketID,
			&reply.UserID,
			&reply.IsStaff,
			&reply.Content,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reply: %w", err)
		}
		replies = append(replies, s.toReplyResponse(&reply))
	}

	if replies == nil {
		replies = []*ReplyResponse{}
	}

	return &TicketDetailResponse{
		Ticket:  ticket,
		Replies: replies,
	}, nil
}

// ListUserTickets lists all tickets for a specific user
func (s *TicketsService) ListUserTickets(userID string, status string, page, limit int) (*TicketsListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build query
	countQuery := `SELECT COUNT(*) FROM support_tickets WHERE user_id = $1`
	query := `
		SELECT id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
		FROM support_tickets
		WHERE user_id = $1
	`

	args := []interface{}{userID}

	if status != "" {
		countQuery += ` AND status = $2`
		query += ` AND status = $2`
		args = append(args, status)
	}

	// Count total
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Query tickets
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*TicketResponse
	for rows.Next() {
		var ticket models.SupportTicket
		if err := rows.Scan(
			&ticket.ID,
			&ticket.UserID,
			&ticket.Subject,
			&ticket.Description,
			&ticket.Status,
			&ticket.Priority,
			&ticket.Category,
			&ticket.AssignedTo,
			&ticket.ResolvedAt,
			&ticket.ClosedAt,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, s.toTicketResponse(&ticket))
	}

	if tickets == nil {
		tickets = []*TicketResponse{}
	}

	totalPages := (total + limit - 1) / limit

	return &TicketsListResponse{
		Tickets:    tickets,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// ListAllTickets lists all tickets (admin only)
func (s *TicketsService) ListAllTickets(status, priority string, page, limit int) (*TicketsListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build query
	countQuery := `SELECT COUNT(*) FROM support_tickets WHERE 1=1`
	query := `
		SELECT id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
		FROM support_tickets
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		countQuery += fmt.Sprintf(` AND status = $%d`, argCount)
		query += fmt.Sprintf(` AND status = $%d`, argCount)
		args = append(args, status)
	}

	if priority != "" {
		argCount++
		countQuery += fmt.Sprintf(` AND priority = $%d`, argCount)
		query += fmt.Sprintf(` AND priority = $%d`, argCount)
		args = append(args, priority)
	}

	// Count total
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Query tickets
	argCount++
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*TicketResponse
	for rows.Next() {
		var ticket models.SupportTicket
		if err := rows.Scan(
			&ticket.ID,
			&ticket.UserID,
			&ticket.Subject,
			&ticket.Description,
			&ticket.Status,
			&ticket.Priority,
			&ticket.Category,
			&ticket.AssignedTo,
			&ticket.ResolvedAt,
			&ticket.ClosedAt,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, s.toTicketResponse(&ticket))
	}

	if tickets == nil {
		tickets = []*TicketResponse{}
	}

	totalPages := (total + limit - 1) / limit

	return &TicketsListResponse{
		Tickets:    tickets,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// UpdateTicket updates a ticket
func (s *TicketsService) UpdateTicket(ticketID, userID string, req *UpdateTicketRequest) (*TicketResponse, error) {
	// Build dynamic update query
	query := `UPDATE support_tickets SET updated_at = $1`
	args := []interface{}{time.Now().UTC()}
	argCount := 1

	if req.Subject != "" {
		argCount++
		query += fmt.Sprintf(`, subject = $%d`, argCount)
		args = append(args, req.Subject)
	}

	if req.Description != "" {
		argCount++
		query += fmt.Sprintf(`, description = $%d`, argCount)
		args = append(args, req.Description)
	}

	if req.Priority != "" {
		argCount++
		query += fmt.Sprintf(`, priority = $%d`, argCount)
		args = append(args, req.Priority)
	}

	if req.Category != "" {
		argCount++
		query += fmt.Sprintf(`, category = $%d`, argCount)
		args = append(args, req.Category)
	}

	argCount++
	query += fmt.Sprintf(` WHERE id = $%d AND user_id = $%d`, argCount, argCount+1)
	query += ` RETURNING id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at`
	args = append(args, ticketID, userID)

	var ticket models.SupportTicket
	err := s.db.QueryRow(query, args...).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.Subject,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Category,
		&ticket.AssignedTo,
		&ticket.ResolvedAt,
		&ticket.ClosedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Invalidate cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user_tickets:%s", userID))

	return s.toTicketResponse(&ticket), nil
}

// UpdateTicketStatus updates the status of a ticket (admin only)
func (s *TicketsService) UpdateTicketStatus(ticketID string, req *UpdateTicketStatusRequest) (*TicketResponse, error) {
	now := time.Now().UTC()
	var resolvedAt, closedAt sql.NullTime

	// Set timestamps based on status
	if req.Status == "resolved" {
		resolvedAt = sql.NullTime{Time: now, Valid: true}
	} else if req.Status == "closed" {
		closedAt = sql.NullTime{Time: now, Valid: true}
	}

	query := `
		UPDATE support_tickets
		SET status = $1, resolved_at = $2, closed_at = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
	`

	var ticket models.SupportTicket
	err := s.db.QueryRow(query, req.Status, resolvedAt, closedAt, now, ticketID).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.Subject,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Category,
		&ticket.AssignedTo,
		&ticket.ResolvedAt,
		&ticket.ClosedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	// Invalidate cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user_tickets:%s", ticket.UserID))

	return s.toTicketResponse(&ticket), nil
}

// AssignTicket assigns a ticket to an admin (admin only)
func (s *TicketsService) AssignTicket(ticketID string, req *AssignTicketRequest) (*TicketResponse, error) {
	query := `
		UPDATE support_tickets
		SET assigned_to = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
	`

	now := time.Now().UTC()
	var ticket models.SupportTicket

	err := s.db.QueryRow(query, req.AssignedTo, now, ticketID).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.Subject,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Category,
		&ticket.AssignedTo,
		&ticket.ResolvedAt,
		&ticket.ClosedAt,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to assign ticket: %w", err)
	}

	return s.toTicketResponse(&ticket), nil
}

// CreateReply creates a reply to a ticket
func (s *TicketsService) CreateReply(ticketID, userID string, isStaff bool, req *CreateReplyRequest) (*ReplyResponse, error) {
	// Generate external ID for the message
	externalID := fmt.Sprintf("msg_%s", uuid.New().String()[:8])

	// Determine sender type
	senderType := "customer"
	if isStaff {
		senderType = "agent"
	}

	query := `
		INSERT INTO support_ticket_messages (
			external_id, ticket_id, message_text, sender_type, sender_user_id,
			message_type, is_internal, delivery_method
		)
		VALUES ($1, $2, $3, $4, $5, 'reply', false, 'web')
		RETURNING id, external_id, ticket_id, message_text, sender_type, sender_user_id,
		          message_type, created_at
	`

	var reply struct {
		ID           string
		ExternalID   string
		TicketID     string
		MessageText  string
		SenderType   string
		SenderUserID sql.NullString
		MessageType  string
		CreatedAt    time.Time
	}

	err := s.db.QueryRow(query, externalID, ticketID, req.Content, senderType, userID).Scan(
		&reply.ID,
		&reply.ExternalID,
		&reply.TicketID,
		&reply.MessageText,
		&reply.SenderType,
		&reply.SenderUserID,
		&reply.MessageType,
		&reply.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	return &ReplyResponse{
		ID:        reply.ID,
		TicketID:  reply.TicketID,
		UserID:    userID,
		IsStaff:   isStaff,
		Content:   reply.MessageText,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.CreatedAt,
	}, nil
}

// DeleteTicket deletes a ticket (user can only delete their own open tickets)
func (s *TicketsService) DeleteTicket(ticketID, userID string) error {
	query := `DELETE FROM support_tickets WHERE id = $1 AND user_id = $2 AND status = 'open'`

	result, err := s.db.Exec(query, ticketID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("ticket not found or cannot be deleted")
	}

	// Invalidate cache
	s.redisHelper.CacheDelete(fmt.Sprintf("user_tickets:%s", userID))

	return nil
}

// UploadAttachment uploads a file and creates an attachment for a ticket
func (s *TicketsService) UploadAttachment(ticketID, userID string, file *multipart.FileHeader, attachmentType, description string) (*AttachmentResponse, error) {
	// Verify ticket exists and user has access
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Check if user owns the ticket (or is admin)
	if ticket.UserID != userID {
		return nil, fmt.Errorf("access denied: you can only upload attachments to your own tickets")
	}

	// Upload file using storage service
	uploadReq := &storage.UploadRequest{
		Visibility: "private", // Ticket attachments are private
		Metadata:   fmt.Sprintf(`{"ticket_id":"%s","attachment_type":"%s"}`, ticketID, attachmentType),
	}

	uploadedFile, err := s.storageService.UploadFile(file, uploadReq, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create attachment record
	attachmentID := uuid.New().String()
	now := time.Now().UTC()

	query := `
		INSERT INTO support_ticket_attachments (
			id, ticket_id, file_id, attachment_type, description,
			visible_to_customer, created_at, uploaded_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, ticket_id, file_id, attachment_type, description, visible_to_customer, created_at, uploaded_by
	`

	var attachment models.SupportTicketAttachment
	err = s.db.QueryRow(
		query,
		attachmentID,
		ticketID,
		uploadedFile.ID,
		attachmentType,
		sql.NullString{String: description, Valid: description != ""},
		true, // visible to customer by default
		now,
		sql.NullString{String: userID, Valid: true},
	).Scan(
		&attachment.ID,
		&attachment.TicketID,
		&attachment.FileID,
		&attachment.AttachmentType,
		&attachment.Description,
		&attachment.VisibleToCustomer,
		&attachment.CreatedAt,
		&attachment.UploadedBy,
	)

	if err != nil {
		// Clean up uploaded file if attachment record creation fails
		s.storageService.DeleteFile(uploadedFile.ID, userID)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	// Build response
	baseURL := s.config.OAuth.JWTIssuer
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	response := &AttachmentResponse{
		ID:                attachment.ID,
		TicketID:          attachment.TicketID,
		FileID:            attachment.FileID,
		FileName:          uploadedFile.FileName,
		OriginalName:      uploadedFile.OriginalName,
		MimeType:          uploadedFile.MimeType,
		Size:              uploadedFile.Size,
		AttachmentType:    attachment.AttachmentType,
		VisibleToCustomer: attachment.VisibleToCustomer,
		DownloadURL:       fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, uploadedFile.ID),
		CreatedAt:         attachment.CreatedAt,
	}

	if attachment.Description.Valid {
		desc := attachment.Description.String
		response.Description = &desc
	}

	if attachment.UploadedBy.Valid {
		uploadedBy := attachment.UploadedBy.String
		response.UploadedBy = &uploadedBy
	}

	return response, nil
}

// ListAttachments lists all attachments for a ticket
func (s *TicketsService) ListAttachments(ticketID, userID string) (*AttachmentsListResponse, error) {
	// Verify ticket exists and user has access
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Check if user owns the ticket (or is admin)
	if ticket.UserID != userID {
		return nil, fmt.Errorf("access denied: you can only view attachments for your own tickets")
	}

	query := `
		SELECT
			a.id, a.ticket_id, a.file_id, a.attachment_type, a.description,
			a.visible_to_customer, a.created_at, a.uploaded_by,
			f.original_name, f.original_name as file_name, f.mime_type, f.file_size
		FROM support_ticket_attachments a
		JOIN files f ON a.file_id = f.id
		WHERE a.ticket_id = $1
		ORDER BY a.created_at DESC
	`

	rows, err := s.db.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer rows.Close()

	baseURL := s.config.OAuth.JWTIssuer
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var attachments []*AttachmentResponse
	for rows.Next() {
		var attachment models.SupportTicketAttachment
		var fileName, originalName, mimeType string
		var fileSize int64

		if err := rows.Scan(
			&attachment.ID,
			&attachment.TicketID,
			&attachment.FileID,
			&attachment.AttachmentType,
			&attachment.Description,
			&attachment.VisibleToCustomer,
			&attachment.CreatedAt,
			&attachment.UploadedBy,
			&originalName,
			&fileName,
			&mimeType,
			&fileSize,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}

		response := &AttachmentResponse{
			ID:                attachment.ID,
			TicketID:          attachment.TicketID,
			FileID:            attachment.FileID,
			FileName:          fileName,
			OriginalName:      originalName,
			MimeType:          mimeType,
			Size:              fileSize,
			AttachmentType:    attachment.AttachmentType,
			VisibleToCustomer: attachment.VisibleToCustomer,
			DownloadURL:       fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, attachment.FileID),
			CreatedAt:         attachment.CreatedAt,
		}

		if attachment.Description.Valid {
			desc := attachment.Description.String
			response.Description = &desc
		}

		if attachment.UploadedBy.Valid {
			uploadedBy := attachment.UploadedBy.String
			response.UploadedBy = &uploadedBy
		}

		attachments = append(attachments, response)
	}

	if attachments == nil {
		attachments = []*AttachmentResponse{}
	}

	return &AttachmentsListResponse{
		Attachments: attachments,
		Total:       len(attachments),
	}, nil
}

// DeleteAttachment deletes an attachment
func (s *TicketsService) DeleteAttachment(ticketID, attachmentID, userID string) error {
	// Verify ticket exists and user has access
	ticket, err := s.GetTicketByID(ticketID)
	if err != nil {
		return err
	}

	// Check if user owns the ticket
	if ticket.UserID != userID {
		return fmt.Errorf("access denied: you can only delete attachments from your own tickets")
	}

	// Get attachment to verify it exists and belongs to the ticket
	var fileID string
	query := `SELECT file_id FROM support_ticket_attachments WHERE id = $1 AND ticket_id = $2`
	err = s.db.QueryRow(query, attachmentID, ticketID).Scan(&fileID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("attachment not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get attachment: %w", err)
	}

	// Delete attachment record
	deleteQuery := `DELETE FROM support_ticket_attachments WHERE id = $1 AND ticket_id = $2`
	result, err := s.db.Exec(deleteQuery, attachmentID, ticketID)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("attachment not found")
	}

	// Delete the file
	if err := s.storageService.DeleteFile(fileID, userID); err != nil {
		// Log error but don't fail - attachment record is already deleted
		fmt.Printf("Warning: failed to delete file %s: %v\n", fileID, err)
	}

	return nil
}

// ========== Admin Service Methods ==========

// AdminCreateTicket creates a ticket on behalf of a user (admin only)
func (s *TicketsService) AdminCreateTicket(adminID string, req *AdminCreateTicketRequest) (*TicketResponse, error) {
	ticketID := uuid.New().String()
	externalID := fmt.Sprintf("TKT-%s", time.Now().Format("20060102-150405"))
	ticketNumber := fmt.Sprintf("TICKET-%d", time.Now().Unix())

	// Get user's email
	var customerEmail string
	emailQuery := `
		SELECT ue.email
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE u.id = $1
	`
	err := s.db.QueryRow(emailQuery, req.UserID).Scan(&customerEmail)
	if err != nil || customerEmail == "" {
		customerEmail = "noreply@support.com" // Fallback email
	}

	query := `
		INSERT INTO support_tickets (id, external_id, ticket_number, user_id, customer_email, subject, description, status, priority, category, assigned_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, external_id, user_id, subject, description, status, priority, category, assigned_to, created_at, updated_at
	`

	var ticket models.SupportTicket
	err = s.db.QueryRow(query, ticketID, externalID, ticketNumber, req.UserID, customerEmail, req.Subject, req.Description, "open", req.Priority,
		sql.NullString{String: req.Category, Valid: req.Category != ""}, req.AssignedTo).
		Scan(&ticket.ID, &ticket.ExternalID, &ticket.UserID, &ticket.Subject, &ticket.Description, &ticket.Status, &ticket.Priority,
			&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	return s.toTicketResponse(&ticket), nil
}

// AdminCreateReply creates a reply to a ticket (admin only)
func (s *TicketsService) AdminCreateReply(ticketID, adminID, content string) (*ReplyResponse, error) {
	// Verify ticket exists
	_, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Generate external ID for the message
	externalID := fmt.Sprintf("msg_%s", uuid.New().String()[:8])

	query := `
		INSERT INTO support_ticket_messages (
			external_id, ticket_id, message_text, sender_type, sender_user_id,
			message_type, is_internal, delivery_method
		)
		VALUES ($1, $2, $3, 'agent', $4, 'reply', false, 'web')
		RETURNING id, external_id, ticket_id, message_text, sender_type, sender_user_id,
		          message_type, created_at
	`

	var reply struct {
		ID           string
		ExternalID   string
		TicketID     string
		MessageText  string
		SenderType   string
		SenderUserID sql.NullString
		MessageType  string
		CreatedAt    time.Time
	}

	err = s.db.QueryRow(query, externalID, ticketID, content, adminID).
		Scan(&reply.ID, &reply.ExternalID, &reply.TicketID, &reply.MessageText,
			&reply.SenderType, &reply.SenderUserID, &reply.MessageType, &reply.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	// Update ticket's updated_at timestamp
	updateQuery := `UPDATE support_tickets SET updated_at = NOW() WHERE id = $1`
	_, _ = s.db.Exec(updateQuery, ticketID)

	return &ReplyResponse{
		ID:        reply.ID,
		TicketID:  reply.TicketID,
		UserID:    adminID,
		IsStaff:   true,
		Content:   reply.MessageText,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.CreatedAt,
	}, nil
}

// AdminUploadAttachment uploads an attachment to a ticket (admin only)
func (s *TicketsService) AdminUploadAttachment(ticketID, adminID string, file *multipart.FileHeader, attachmentType, description string) (*AttachmentResponse, error) {
	// Verify ticket exists
	_, err := s.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Upload file using storage service
	uploadReq := &storage.UploadRequest{
		Visibility: "private",
		Metadata:   "",
	}
	uploadedFile, err := s.storageService.UploadFile(file, uploadReq, adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create attachment record
	attachmentID := uuid.New().String()
	query := `
		INSERT INTO support_ticket_attachments (id, ticket_id, file_id, attachment_type, description, visible_to_customer, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, ticket_id, file_id, attachment_type, description, visible_to_customer, uploaded_by, created_at
	`

	var attachment models.SupportTicketAttachment
	var desc sql.NullString
	if description != "" {
		desc = sql.NullString{String: description, Valid: true}
	}

	err = s.db.QueryRow(query, attachmentID, ticketID, uploadedFile.ID, attachmentType, desc, true, adminID).
		Scan(&attachment.ID, &attachment.TicketID, &attachment.FileID, &attachment.AttachmentType, &attachment.Description,
			&attachment.VisibleToCustomer, &attachment.UploadedBy, &attachment.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	return &AttachmentResponse{
		ID:                attachment.ID,
		TicketID:          attachment.TicketID,
		FileID:            attachment.FileID,
		FileName:          uploadedFile.FileName,
		OriginalName:      uploadedFile.OriginalName,
		MimeType:          uploadedFile.MimeType,
		Size:              uploadedFile.Size,
		AttachmentType:    attachment.AttachmentType,
		Description:       &description,
		VisibleToCustomer: attachment.VisibleToCustomer,
		DownloadURL:       fmt.Sprintf("/api/v1/storage/files/%s/download", uploadedFile.ID),
		CreatedAt:         attachment.CreatedAt,
	}, nil
}

// GetTicketsByUser gets all tickets for a specific user (admin only)
func (s *TicketsService) GetTicketsByUser(userID string, page, limit int, status, priority string) ([]*TicketResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build query with filters
	baseQuery := `FROM support_tickets WHERE user_id = $1`
	args := []interface{}{userID}
	argCount := 1

	if status != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if priority != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, priority)
	}

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) ` + baseQuery
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Get tickets
	query := `
		SELECT id, user_id, subject, description, status, priority, category, assigned_to, resolved_at, closed_at, created_at, updated_at
		` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)

	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*TicketResponse
	for rows.Next() {
		var ticket models.SupportTicket
		err := rows.Scan(&ticket.ID, &ticket.UserID, &ticket.Subject, &ticket.Description, &ticket.Status,
			&ticket.Priority, &ticket.Category, &ticket.AssignedTo, &ticket.ResolvedAt, &ticket.ClosedAt,
			&ticket.CreatedAt, &ticket.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, s.toTicketResponse(&ticket))
	}

	if tickets == nil {
		tickets = []*TicketResponse{}
	}

	return tickets, total, nil
}
