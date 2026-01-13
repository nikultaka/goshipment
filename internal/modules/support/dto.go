package support

import (
	"time"

	"gogin/internal/models"
)

// CreateSupportTicketRequest represents the request to create a support ticket
type CreateSupportTicketRequest struct {
	Subject             string `json:"subject" binding:"required,min=5,max=255"`
	Message             string `json:"message" binding:"required,min=10"`
	Category            string `json:"category" binding:"required,oneof=shipment wallet pickup invoice billing account technical other"`
	Priority            string `json:"priority" binding:"omitempty,oneof=critical high normal low"`
	RelatedShipmentID   string `json:"related_shipment_id,omitempty" binding:"omitempty,uuid"`
	RelatedInvoiceID    string `json:"related_invoice_id,omitempty" binding:"omitempty,uuid"`
	RelatedPickupID     string `json:"related_pickup_id,omitempty" binding:"omitempty,uuid"`
}

// ReplySupportTicketRequest represents a reply to a support ticket
type ReplySupportTicketRequest struct {
	Message string `json:"message" binding:"required,min=1"`
}

// ManageSupportTicketRequest represents admin/manager actions on a ticket
type ManageSupportTicketRequest struct {
	Action         string `json:"action" binding:"required,oneof=assign escalate close reopen"`
	AssignedUserID string `json:"assigned_user_id,omitempty" binding:"omitempty,uuid"`
	AdminNotes     string `json:"admin_notes,omitempty"`
	ResolutionReason string `json:"resolution_reason,omitempty" binding:"omitempty,oneof=issue_resolved no_longer_needed created_by_mistake wrong_request_type will_contact_later other"`
}

// ListSupportTicketsRequest represents query parameters for listing tickets
type ListSupportTicketsRequest struct {
	Status     string `form:"status" binding:"omitempty,oneof=open resolved closed"`
	Category   string `form:"category" binding:"omitempty,oneof=shipment wallet pickup invoice billing account technical other"`
	Priority   string `form:"priority" binding:"omitempty,oneof=critical high normal low"`
	AssignedTo string `form:"assigned_to" binding:"omitempty,uuid"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// SupportTicketResponse represents a support ticket in API responses
type SupportTicketResponse struct {
	ID               string    `json:"id"`
	TicketNumber     string    `json:"ticket_number"`
	OrganizationID   string    `json:"organization_id,omitempty"`
	CreatedByUserID  string    `json:"created_by_user_id"`
	AssignedToUserID string    `json:"assigned_to_user_id,omitempty"`
	Category         string    `json:"category"`
	Priority         string    `json:"priority"`
	Status           string    `json:"status"`
	Subject          string    `json:"subject"`
	AdminNotes       string    `json:"admin_notes,omitempty"` // Only visible to admins
	ResolutionReason string    `json:"resolution_reason,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
	LastMessageAt    time.Time `json:"last_message_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	MessageCount     int       `json:"message_count,omitempty"`
}

// SupportTicketMessageResponse represents a message in a support ticket
type SupportTicketMessageResponse struct {
	ID           string    `json:"id"`
	TicketID     string    `json:"ticket_id"`
	SenderType   string    `json:"sender_type"` // user | staff | system
	SenderUserID string    `json:"sender_user_id,omitempty"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

// SupportTicketDetailResponse represents a ticket with all its messages
type SupportTicketDetailResponse struct {
	Ticket   *SupportTicketResponse          `json:"ticket"`
	Messages []*SupportTicketMessageResponse `json:"messages"`
}

// ListSupportTicketsResponse represents a paginated list of support tickets
type ListSupportTicketsResponse struct {
	Tickets    []*SupportTicketResponse `json:"tickets"`
	TotalCount int                      `json:"total_count"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// ManageSupportTicketResponse represents the response after managing a ticket
type ManageSupportTicketResponse struct {
	ID           string    `json:"id"`
	TicketNumber string    `json:"ticket_number"`
	Status       string    `json:"status"`
	Action       string    `json:"action"`
	Message      string    `json:"message"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToSupportTicketResponse converts a SupportTicket model to response DTO
func ToSupportTicketResponse(ticket *models.SupportTicket, isAdmin bool) *SupportTicketResponse {
	resp := &SupportTicketResponse{
		ID:              ticket.ID,
		TicketNumber:    ticket.TicketNumber,
		CreatedByUserID: ticket.CreatedByUserID,
		Category:        string(ticket.Category),
		Priority:        string(ticket.Priority),
		Status:          string(ticket.Status),
		Subject:         ticket.Subject,
		LastMessageAt:   ticket.LastMessageAt,
		CreatedAt:       ticket.CreatedAt,
		UpdatedAt:       ticket.UpdatedAt,
	}

	if ticket.OrganizationID.Valid {
		resp.OrganizationID = ticket.OrganizationID.String
	}

	if ticket.AssignedToUserID.Valid {
		resp.AssignedToUserID = ticket.AssignedToUserID.String
	}

	// Admin notes only visible to admins
	if isAdmin && ticket.AdminNotes.Valid {
		resp.AdminNotes = ticket.AdminNotes.String
	}

	if ticket.ResolutionReason.Valid {
		resp.ResolutionReason = ticket.ResolutionReason.String
	}

	if ticket.ResolvedAt.Valid {
		resp.ResolvedAt = &ticket.ResolvedAt.Time
	}

	if ticket.ClosedAt.Valid {
		resp.ClosedAt = &ticket.ClosedAt.Time
	}

	return resp
}

// ToSupportTicketMessageResponse converts a SupportTicketMessage model to response DTO
func ToSupportTicketMessageResponse(message *models.SupportTicketMessage) *SupportTicketMessageResponse {
	resp := &SupportTicketMessageResponse{
		ID:         message.ID,
		TicketID:   message.TicketID,
		SenderType: string(message.SenderType),
		Message:    message.Message,
		CreatedAt:  message.CreatedAt,
	}

	if message.SenderUserID.Valid {
		resp.SenderUserID = message.SenderUserID.String
	}

	return resp
}
