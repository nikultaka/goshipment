package models

import (
	"database/sql"
	"time"
)

// TicketCategory represents the category of a support ticket
type TicketCategory string

const (
	TicketCategoryShipment  TicketCategory = "shipment"
	TicketCategoryWallet    TicketCategory = "wallet"
	TicketCategoryPickup    TicketCategory = "pickup"
	TicketCategoryInvoice   TicketCategory = "invoice"
	TicketCategoryBilling   TicketCategory = "billing"
	TicketCategoryAccount   TicketCategory = "account"
	TicketCategoryTechnical TicketCategory = "technical"
	TicketCategoryOther     TicketCategory = "other"
)

// TicketPriority represents the priority level of a support ticket
type TicketPriority string

const (
	TicketPriorityCritical TicketPriority = "critical"
	TicketPriorityHigh     TicketPriority = "high"
	TicketPriorityNormal   TicketPriority = "normal"
	TicketPriorityLow      TicketPriority = "low"
)

// TicketStatus represents the status of a support ticket
type TicketStatus string

const (
	TicketStatusOpen     TicketStatus = "open"
	TicketStatusResolved TicketStatus = "resolved"
	TicketStatusClosed   TicketStatus = "closed"
)

// TicketResolutionReason represents why a ticket was resolved
type TicketResolutionReason string

const (
	ResolutionIssueResolved     TicketResolutionReason = "issue_resolved"
	ResolutionNoLongerNeeded    TicketResolutionReason = "no_longer_needed"
	ResolutionCreatedByMistake  TicketResolutionReason = "created_by_mistake"
	ResolutionWrongRequestType  TicketResolutionReason = "wrong_request_type"
	ResolutionWillContactLater  TicketResolutionReason = "will_contact_later"
	ResolutionOther             TicketResolutionReason = "other"
)

// TicketMessageSender represents who sent a message
type TicketMessageSender string

const (
	MessageSenderUser   TicketMessageSender = "user"
	MessageSenderStaff  TicketMessageSender = "staff"
	MessageSenderSystem TicketMessageSender = "system"
)

// SupportTicket represents a support ticket in the database
type SupportTicket struct {
	ID             string         `json:"id" db:"id"`
	ExternalID     string         `json:"external_id" db:"external_id"`
	TicketNumber   string         `json:"ticket_number" db:"ticket_number"`
	OrganizationID sql.NullString `json:"organization_id,omitempty" db:"organization_id"`

	// User fields (support both old and new schema)
	UserID           string         `json:"user_id" db:"user_id"`
	CreatedByUserID  string         `json:"created_by_user_id" db:"created_by_user_id"`
	AssignedTo       sql.NullString `json:"assigned_to,omitempty" db:"assigned_to"`
	AssignedToUserID sql.NullString `json:"assigned_to_user_id,omitempty" db:"assigned_to_user_id"`

	// Classification
	Category sql.NullString `json:"category" db:"category"`
	Priority string         `json:"priority" db:"priority"`
	Status   string         `json:"status" db:"status"`

	// Content
	Subject     string         `json:"subject" db:"subject"`
	Description string         `json:"description" db:"description"`
	AdminNotes  sql.NullString `json:"admin_notes,omitempty" db:"admin_notes"`

	// Resolution
	ResolutionReason sql.NullString `json:"resolution_reason,omitempty" db:"resolution_reason"`
	ResolvedAt       sql.NullTime   `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt         sql.NullTime   `json:"closed_at,omitempty" db:"closed_at"`

	// Activity
	LastMessageAt time.Time `json:"last_message_at" db:"last_message_at"`

	// Audit
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// SupportTicketMessage represents a message in a support ticket thread
type SupportTicketMessage struct {
	ID           string              `json:"id" db:"id"`
	TicketID     string              `json:"ticket_id" db:"ticket_id"`
	SenderType   TicketMessageSender `json:"sender_type" db:"sender_type"`
	SenderUserID sql.NullString      `json:"sender_user_id,omitempty" db:"sender_user_id"`
	Message      string              `json:"message" db:"message"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
}

// SupportTicketReply represents a reply to a support ticket (alias for compatibility)
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

// SupportTicketAttachment represents a file attached to a support ticket
type SupportTicketAttachment struct {
	ID                string         `json:"id" db:"id"`
	TicketID          string         `json:"ticket_id" db:"ticket_id"`
	MessageID         sql.NullString `json:"message_id,omitempty" db:"message_id"`
	FileID            string         `json:"file_id" db:"file_id"`
	AttachmentType    string         `json:"attachment_type" db:"attachment_type"`
	Description       sql.NullString `json:"description,omitempty" db:"description"`
	VisibleToCustomer bool           `json:"visible_to_customer" db:"visible_to_customer"`
	UploadedBy        sql.NullString `json:"uploaded_by,omitempty" db:"uploaded_by"`
	UploadedByUserID  string         `json:"uploaded_by_user_id" db:"uploaded_by_user_id"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
}

// Helper methods

// IsOpen returns true if the ticket is open
func (t *SupportTicket) IsOpen() bool {
	return t.Status == string(TicketStatusOpen)
}

// IsResolved returns true if the ticket is resolved
func (t *SupportTicket) IsResolved() bool {
	return t.Status == string(TicketStatusResolved)
}

// IsClosed returns true if the ticket is closed
func (t *SupportTicket) IsClosed() bool {
	return t.Status == string(TicketStatusClosed)
}

// CanUserReply returns true if a user can reply to this ticket
func (t *SupportTicket) CanUserReply() bool {
	return t.Status != string(TicketStatusClosed)
}
