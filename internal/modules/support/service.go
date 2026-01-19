package support

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
)

const (
	// TicketNumberPrefix is the prefix for ticket numbers
	TicketNumberPrefix = "TCK-"
)

// SupportService handles support ticket business logic
type SupportService struct {
	db *clients.Database
}

// NewSupportService creates a new support service
func NewSupportService(db *clients.Database) *SupportService {
	return &SupportService{db: db}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *SupportService) GetOrganizationIDFromUser(userID string) (string, error) {
	query := `
		SELECT organization_id
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`

	var organizationID string
	err := s.db.DB.QueryRow(query, userID).Scan(&organizationID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user is not a member of any organization")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get organization: %w", err)
	}

	return organizationID, nil
}

// GetUserRole retrieves the role of a user
func (s *SupportService) GetUserRole(userID string) (string, error) {
	query := `
		SELECT role
		FROM users
		WHERE id = $1
	`

	var role string
	err := s.db.DB.QueryRow(query, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// generateTicketNumber generates a unique ticket number
func (s *SupportService) generateTicketNumber() (string, error) {
	// Generate 8 random characters
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return TicketNumberPrefix + hex.EncodeToString(bytes), nil
}

// CreateSupportTicket creates a new support ticket
func (s *SupportService) CreateSupportTicket(req CreateSupportTicketRequest, organizationID, userID string) (*SupportTicketDetailResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate ticket number
	ticketNumber, err := s.generateTicketNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ticket number: %w", err)
	}

	// Set default priority if not provided
	priority := req.Priority
	if priority == "" {
		priority = string(models.TicketPriorityNormal)
	}

	now := time.Now()

	// Create ticket
	var ticketID string
	err = tx.QueryRow(`
		INSERT INTO support_tickets (
			ticket_number, organization_id, created_by_user_id,
			category, priority, status, subject, last_message_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, 'open', $6, $7,
			$8, $9
		)
		RETURNING id
	`, ticketNumber, organizationID, userID,
		req.Category, priority, req.Subject, now,
		now, now).Scan(&ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// Create initial message
	var messageID string
	err = tx.QueryRow(`
		INSERT INTO support_ticket_messages (
			ticket_id, sender_type, sender_user_id, message, created_at
		) VALUES (
			$1, 'user', $2, $3, $4
		)
		RETURNING id
	`, ticketID, userID, req.Message, now).Scan(&messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial message: %w", err)
	}

	// Log audit entry
	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id, status, created_at
		) VALUES (
			$1, 'support.ticket_created', 'support_ticket', $2, 'success', $3
		)
	`, userID, ticketID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to log audit entry: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return the created ticket with message
	return s.GetSupportTicketDetail(ticketID, organizationID, userID, false)
}

// ReplySupportTicket adds a reply to an existing support ticket
func (s *SupportService) ReplySupportTicket(ticketID, organizationID, userID string, req ReplySupportTicketRequest, isStaff bool) (*SupportTicketMessageResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify ticket exists and belongs to organization
	var ticket models.SupportTicket
	err = tx.QueryRow(`
		SELECT id, organization_id, status, created_by_user_id
		FROM support_tickets
		WHERE id = $1 AND organization_id = $2
	`, ticketID, organizationID).Scan(
		&ticket.ID, &ticket.OrganizationID,
		&ticket.Status, &ticket.CreatedByUserID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket: %w", err)
	}

	// Check if ticket is closed
	if ticket.Status == string(models.TicketStatusClosed) {
		return nil, fmt.Errorf("cannot reply to a closed ticket")
	}

	// Determine sender type
	senderType := models.MessageSenderUser
	if isStaff {
		senderType = models.MessageSenderStaff
	}

	now := time.Now()

	// Create message
	var messageID string
	err = tx.QueryRow(`
		INSERT INTO support_ticket_messages (
			ticket_id, sender_type, sender_user_id, message, created_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING id
	`, ticketID, senderType, userID, req.Message, now).Scan(&messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Update ticket's last_message_at
	_, err = tx.Exec(`
		UPDATE support_tickets
		SET last_message_at = $1, updated_at = $2
		WHERE id = $3
	`, now, now, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &SupportTicketMessageResponse{
		ID:           messageID,
		TicketID:     ticketID,
		SenderType:   string(senderType),
		SenderUserID: userID,
		Message:      req.Message,
		CreatedAt:    now,
	}, nil
}

// ManageSupportTicket performs admin/manager actions on a ticket
func (s *SupportService) ManageSupportTicket(ticketID, organizationID, userID string, req ManageSupportTicketRequest) (*ManageSupportTicketResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify ticket exists and belongs to organization
	var ticket models.SupportTicket
	err = tx.QueryRow(`
		SELECT id, ticket_number, organization_id, status
		FROM support_tickets
		WHERE id = $1 AND organization_id = $2
	`, ticketID, organizationID).Scan(
		&ticket.ID, &ticket.TicketNumber,
		&ticket.OrganizationID, &ticket.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket: %w", err)
	}

	now := time.Now()
	var statusMessage string

	switch req.Action {
	case "assign":
		if req.AssignedUserID == "" {
			return nil, fmt.Errorf("assigned_user_id is required for assign action")
		}
		_, err = tx.Exec(`
			UPDATE support_tickets
			SET assigned_to_user_id = $1, updated_at = $2
			WHERE id = $3
		`, req.AssignedUserID, now, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to assign ticket: %w", err)
		}
		statusMessage = "Ticket assigned successfully"

		// Create system message
		_, err = tx.Exec(`
			INSERT INTO support_ticket_messages (
				ticket_id, sender_type, message, created_at
			) VALUES (
				$1, 'system', 'Ticket assigned to user', $2
			)
		`, ticketID, now)

	case "escalate":
		_, err = tx.Exec(`
			UPDATE support_tickets
			SET priority = 'critical', updated_at = $1
			WHERE id = $2
		`, now, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to escalate ticket: %w", err)
		}
		statusMessage = "Ticket escalated successfully"

		// Create system message
		_, err = tx.Exec(`
			INSERT INTO support_ticket_messages (
				ticket_id, sender_type, message, created_at
			) VALUES (
				$1, 'system', 'Ticket escalated to critical priority', $2
			)
		`, ticketID, now)

	case "close":
		resolutionReason := req.ResolutionReason
		if resolutionReason == "" {
			resolutionReason = "other"
		}
		_, err = tx.Exec(`
			UPDATE support_tickets
			SET status = 'closed', resolution_reason = $1, closed_at = $2, updated_at = $3
			WHERE id = $4
		`, resolutionReason, now, now, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to close ticket: %w", err)
		}
		statusMessage = "Ticket closed successfully"

		// Create system message
		_, err = tx.Exec(`
			INSERT INTO support_ticket_messages (
				ticket_id, sender_type, message, created_at
			) VALUES (
				$1, 'system', 'Ticket closed: ' || $2, $3
			)
		`, ticketID, resolutionReason, now)

	case "reopen":
		if ticket.Status != string(models.TicketStatusClosed) {
			return nil, fmt.Errorf("ticket is not closed")
		}
		_, err = tx.Exec(`
			UPDATE support_tickets
			SET status = 'open', resolution_reason = NULL, closed_at = NULL, updated_at = $1
			WHERE id = $2
		`, now, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to reopen ticket: %w", err)
		}
		statusMessage = "Ticket reopened successfully"

		// Create system message
		_, err = tx.Exec(`
			INSERT INTO support_ticket_messages (
				ticket_id, sender_type, message, created_at
			) VALUES (
				$1, 'system', 'Ticket reopened', $2
			)
		`, ticketID, now)

	default:
		return nil, fmt.Errorf("invalid action: %s", req.Action)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create system message: %w", err)
	}

	// Update admin notes if provided
	if req.AdminNotes != "" {
		_, err = tx.Exec(`
			UPDATE support_tickets
			SET admin_notes = $1
			WHERE id = $2
		`, req.AdminNotes, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to update admin notes: %w", err)
		}
	}

	// Log audit entry
	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id, status, created_at
		) VALUES (
			$1, 'support.ticket_' || $2, 'support_ticket', $3, 'success', $4
		)
	`, userID, req.Action, ticketID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to log audit entry: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get updated status
	var newStatus string
	s.db.DB.QueryRow(`SELECT status FROM support_tickets WHERE id = $1`, ticketID).Scan(&newStatus)

	return &ManageSupportTicketResponse{
		ID:           ticket.ID,
		TicketNumber: ticket.TicketNumber,
		Status:       newStatus,
		Action:       req.Action,
		Message:      statusMessage,
		UpdatedAt:    now,
	}, nil
}

// GetSupportTicketDetail retrieves a ticket with all its messages
func (s *SupportService) GetSupportTicketDetail(ticketID, organizationID, userID string, isAdmin bool) (*SupportTicketDetailResponse, error) {
	// Fetch ticket
	var ticket models.SupportTicket
	err := s.db.DB.QueryRow(`
		SELECT id, ticket_number, organization_id, created_by_user_id, assigned_to_user_id,
			category, priority, status, subject, admin_notes, resolution_reason,
			resolved_at, closed_at, last_message_at, created_at, updated_at
		FROM support_tickets
		WHERE id = $1 AND organization_id = $2
	`, ticketID, organizationID).Scan(
		&ticket.ID, &ticket.TicketNumber, &ticket.OrganizationID,
		&ticket.CreatedByUserID, &ticket.AssignedToUserID,
		&ticket.Category, &ticket.Priority, &ticket.Status, &ticket.Subject,
		&ticket.AdminNotes, &ticket.ResolutionReason,
		&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.LastMessageAt,
		&ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket: %w", err)
	}

	// Check authorization: user can only view their own tickets unless admin
	if !isAdmin && ticket.CreatedByUserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Fetch messages
	rows, err := s.db.DB.Query(`
		SELECT id, ticket_id, sender_type, sender_user_id, message, created_at
		FROM support_ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	messages := []*SupportTicketMessageResponse{}
	for rows.Next() {
		var msg models.SupportTicketMessage
		err := rows.Scan(
			&msg.ID, &msg.TicketID, &msg.SenderType,
			&msg.SenderUserID, &msg.Message, &msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, ToSupportTicketMessageResponse(&msg))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return &SupportTicketDetailResponse{
		Ticket:   ToSupportTicketResponse(&ticket, isAdmin),
		Messages: messages,
	}, nil
}

// ListSupportTickets retrieves a list of support tickets
func (s *SupportService) ListSupportTickets(req ListSupportTicketsRequest, organizationID, userID string, isAdmin bool) (*ListSupportTicketsResponse, error) {
	// Default pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// Build query
	query := `
		SELECT id, ticket_number, organization_id, created_by_user_id, assigned_to_user_id,
			category, priority, status, subject, admin_notes, resolution_reason,
			resolved_at, closed_at, last_message_at, created_at, updated_at
		FROM support_tickets
		WHERE organization_id = $1
	`
	args := []interface{}{organizationID}
	argIndex := 2

	// Non-admins can only see their own tickets
	if !isAdmin {
		query += fmt.Sprintf(" AND created_by_user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, req.Category)
		argIndex++
	}

	if req.Priority != "" {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, req.Priority)
		argIndex++
	}

	if req.AssignedTo != "" {
		query += fmt.Sprintf(" AND assigned_to_user_id = $%d", argIndex)
		args = append(args, req.AssignedTo)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var totalCount int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY last_message_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tickets: %w", err)
	}
	defer rows.Close()

	tickets := []*SupportTicketResponse{}
	for rows.Next() {
		var ticket models.SupportTicket
		err := rows.Scan(
			&ticket.ID, &ticket.TicketNumber, &ticket.OrganizationID,
			&ticket.CreatedByUserID, &ticket.AssignedToUserID,
			&ticket.Category, &ticket.Priority, &ticket.Status, &ticket.Subject,
			&ticket.AdminNotes, &ticket.ResolutionReason,
			&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.LastMessageAt,
			&ticket.CreatedAt, &ticket.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		// Get message count for each ticket
		var messageCount int
		s.db.DB.QueryRow(`SELECT COUNT(*) FROM support_ticket_messages WHERE ticket_id = $1`, ticket.ID).Scan(&messageCount)

		resp := ToSupportTicketResponse(&ticket, isAdmin)
		resp.MessageCount = messageCount
		tickets = append(tickets, resp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickets: %w", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	return &ListSupportTicketsResponse{
		Tickets:    tickets,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
