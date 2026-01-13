package models

import (
	"time"
)

// InviteStatus represents the status of an organization invitation
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusRejected InviteStatus = "rejected"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// OrganizationInvite represents an invitation to join an organization
type OrganizationInvite struct {
	ID                string       `json:"id" db:"id"`
	OrganizationID    string       `json:"organization_id" db:"organization_id"`
	InvitedByUserID   string       `json:"invited_by_user_id" db:"invited_by_user_id"`
	Email             string       `json:"email" db:"email"`
	Role              string       `json:"role" db:"role"`
	InviteToken       string       `json:"-" db:"invite_token"` // Never expose in JSON
	Status            InviteStatus `json:"status" db:"status"`
	ExpiresAt         time.Time    `json:"expires_at" db:"expires_at"`
	AcceptedAt        *time.Time   `json:"accepted_at,omitempty" db:"accepted_at"`
	RejectedAt        *time.Time   `json:"rejected_at,omitempty" db:"rejected_at"`
	RevokedAt         *time.Time   `json:"revoked_at,omitempty" db:"revoked_at"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at" db:"updated_at"`
}

// IsExpired checks if the invitation has expired
func (i *OrganizationInvite) IsExpired() bool {
	return time.Now().UTC().After(i.ExpiresAt)
}

// CanBeAccepted checks if the invitation can be accepted
func (i *OrganizationInvite) CanBeAccepted() bool {
	return i.Status == InviteStatusPending && !i.IsExpired()
}

// CanBeRejected checks if the invitation can be rejected
func (i *OrganizationInvite) CanBeRejected() bool {
	return i.Status == InviteStatusPending && !i.IsExpired()
}

// CanBeRevoked checks if the invitation can be revoked
func (i *OrganizationInvite) CanBeRevoked() bool {
	return i.Status == InviteStatusPending
}
