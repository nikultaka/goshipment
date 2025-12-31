package tickets

import "time"

// CreateTicketRequest represents the request body for creating a ticket
type CreateTicketRequest struct {
	Subject       string `json:"subject" binding:"required,min=5,max=255"`
	Description   string `json:"description" binding:"required,min=10"`
	Priority      string `json:"priority" binding:"required,oneof=low medium high urgent"`
	Category      string `json:"category"`
	CustomerEmail string `json:"customer_email"` // Optional, will use user's email if not provided
}

// UpdateTicketRequest represents the request body for updating a ticket
type UpdateTicketRequest struct {
	Subject     string `json:"subject" binding:"omitempty,min=5,max=255"`
	Description string `json:"description" binding:"omitempty,min=10"`
	Priority    string `json:"priority" binding:"omitempty,oneof=low medium high urgent"`
	Category    string `json:"category"`
}

// UpdateTicketStatusRequest represents the request body for updating ticket status
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open in_progress resolved closed"`
}

// AssignTicketRequest represents the request body for assigning a ticket
type AssignTicketRequest struct {
	AssignedTo string `json:"assigned_to" binding:"required,uuid"`
}

// CreateReplyRequest represents the request body for creating a reply
type CreateReplyRequest struct {
	Content string `json:"content" binding:"required,min=1"`
}

// TicketResponse represents a sanitized ticket response
type TicketResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Category    *string    `json:"category,omitempty"`
	AssignedTo  *string    `json:"assigned_to,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ReplyCount  int        `json:"reply_count,omitempty"`
}

// ReplyResponse represents a sanitized reply response
type ReplyResponse struct {
	ID        string     `json:"id"`
	TicketID  string     `json:"ticket_id"`
	UserID    string     `json:"user_id"`
	IsStaff   bool       `json:"is_staff"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// TicketDetailResponse represents a ticket with all its replies
type TicketDetailResponse struct {
	Ticket  *TicketResponse  `json:"ticket"`
	Replies []*ReplyResponse `json:"replies"`
}

// TicketsListResponse represents a paginated list of tickets
type TicketsListResponse struct {
	Tickets    []*TicketResponse `json:"tickets"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// AttachmentResponse represents a ticket attachment response
type AttachmentResponse struct {
	ID                string    `json:"id"`
	TicketID          string    `json:"ticket_id"`
	FileID            string    `json:"file_id"`
	FileName          string    `json:"file_name"`
	OriginalName      string    `json:"original_name"`
	MimeType          string    `json:"mime_type"`
	Size              int64     `json:"size"`
	AttachmentType    string    `json:"attachment_type"`
	Description       *string   `json:"description,omitempty"`
	VisibleToCustomer bool      `json:"visible_to_customer"`
	DownloadURL       string    `json:"download_url"`
	CreatedAt         time.Time `json:"created_at"`
	UploadedBy        *string   `json:"uploaded_by,omitempty"`
}

// AttachmentsListResponse represents a list of ticket attachments
type AttachmentsListResponse struct {
	Attachments []*AttachmentResponse `json:"attachments"`
	Total       int                   `json:"total"`
}
