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
	TicketNumber   string         `json:"ticket_number" db:"ticket_number"`
	OrganizationID sql.NullString `json:"organization_id,omitempty" db:"organization_id"`
	CreatedByUserID string        `json:"created_by_user_id" db:"created_by_user_id"`
	AssignedToUserID sql.NullString `json:"assigned_to_user_id,omitempty" db:"assigned_to_user_id"`

	// Classification
	Category TicketCategory `json:"category" db:"category"`
	Priority TicketPriority `json:"priority" db:"priority"`
	Status   TicketStatus   `json:"status" db:"status"`

	// Content
	Subject    string         `json:"subject" db:"subject"`
	AdminNotes sql.NullString `json:"admin_notes,omitempty" db:"admin_notes"`

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

// SupportTicketAttachment represents a file attached to a support ticket
type SupportTicketAttachment struct {
	ID               string         `json:"id" db:"id"`
	TicketID         string         `json:"ticket_id" db:"ticket_id"`
	MessageID        sql.NullString `json:"message_id,omitempty" db:"message_id"`
	FileID           string         `json:"file_id" db:"file_id"`
	UploadedByUserID string         `json:"uploaded_by_user_id" db:"uploaded_by_user_id"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// Helper methods

// IsOpen returns true if the ticket is open
func (t *SupportTicket) IsOpen() bool {
	return t.Status == TicketStatusOpen
}

// IsResolved returns true if the ticket is resolved
func (t *SupportTicket) IsResolved() bool {
	return t.Status == TicketStatusResolved
}

// IsClosed returns true if the ticket is closed
func (t *SupportTicket) IsClosed() bool {
	return t.Status == TicketStatusClosed
}

// CanUserReply returns true if a user can reply to this ticket
func (t *SupportTicket) CanUserReply() bool {
	return t.Status != TicketStatusClosed
}
