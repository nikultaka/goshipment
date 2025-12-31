package models

import (
	"database/sql"
	"time"
)

// SupportTicket represents a user support ticket
type SupportTicket struct {
	ID          string         `json:"id" db:"id"`
	ExternalID  string         `json:"external_id" db:"external_id"`
	UserID      string         `json:"user_id" db:"user_id"`
	Subject     string         `json:"subject" db:"subject"`
	Description string         `json:"description" db:"description"`
	Status      string         `json:"status" db:"status"` // open, in_progress, resolved, closed
	Priority    string         `json:"priority" db:"priority"` // low, medium, high, urgent
	Category    sql.NullString `json:"category,omitempty" db:"category"`
	AssignedTo  sql.NullString `json:"assigned_to,omitempty" db:"assigned_to"`
	ResolvedAt  sql.NullTime   `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt    sql.NullTime   `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// SupportTicketReply represents a reply to a support ticket
type SupportTicketReply struct {
	ID        string       `json:"id" db:"id"`
	TicketID  string       `json:"ticket_id" db:"ticket_id"`
	UserID    string       `json:"user_id" db:"user_id"`
	IsStaff   bool         `json:"is_staff" db:"is_staff"`
	Content   string       `json:"content" db:"content"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsOpen returns true if the ticket is open
func (t *SupportTicket) IsOpen() bool {
	return t.Status == "open" || t.Status == "in_progress"
}

// IsResolved returns true if the ticket is resolved
func (t *SupportTicket) IsResolved() bool {
	return t.Status == "resolved" || t.Status == "closed"
}

// SupportTicketAttachment represents a file attached to a support ticket
type SupportTicketAttachment struct {
	ID                 string         `json:"id" db:"id"`
	TicketID           string         `json:"ticket_id" db:"ticket_id"`
	MessageID          sql.NullString `json:"message_id,omitempty" db:"message_id"`
	FileID             string         `json:"file_id" db:"file_id"`
	AttachmentType     string         `json:"attachment_type" db:"attachment_type"` // document, screenshot, log_file
	Description        sql.NullString `json:"description,omitempty" db:"description"`
	VisibleToCustomer  bool           `json:"visible_to_customer" db:"visible_to_customer"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UploadedBy         sql.NullString `json:"uploaded_by,omitempty" db:"uploaded_by"`
}
