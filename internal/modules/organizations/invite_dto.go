package organizations

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// CreateInviteRequest represents the request to invite a user to organization
type CreateInviteRequest struct {
	Email string `json:"email" binding:"required,email" example:"newuser@example.com"`
	Role  string `json:"role" binding:"required,oneof=owner administrator manager operator viewer" example:"operator"`
}

// AcceptInviteRequest represents the request to accept an invitation
type AcceptInviteRequest struct {
	InviteToken string `json:"invite_token" binding:"required,len=64" example:"a1b2c3d4e5f6...64chars"`
}

// RejectInviteRequest represents the request to reject an invitation
type RejectInviteRequest struct {
	InviteToken string `json:"invite_token" binding:"required,len=64" example:"a1b2c3d4e5f6...64chars"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// InviteResponse represents an organization invitation in API responses
type InviteResponse struct {
	ID              string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID  string     `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Email           string     `json:"email" example:"newuser@example.com"`
	Role            string     `json:"role" example:"operator"`
	Status          string     `json:"status" example:"pending"`
	InvitedBy       UserInfo   `json:"invited_by"`
	ExpiresAt       time.Time  `json:"expires_at" example:"2026-01-20T10:00:00Z"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty" example:"2026-01-14T10:30:00Z"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at" example:"2026-01-13T10:00:00Z"`
	IsExpired       bool       `json:"is_expired" example:"false"`
}

// UserInfo represents basic user information in invite responses
type UserInfo struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
	Email     string `json:"email" example:"john.doe@example.com"`
}

// CreateInviteResponse represents the response after creating an invitation
type CreateInviteResponse struct {
	Invite     InviteResponse `json:"invite"`
	InviteLink string         `json:"invite_link" example:"https://app.example.com/invites/accept?token=a1b2c3d4..."`
	Message    string         `json:"message" example:"Invitation sent successfully to newuser@example.com"`
}

// AcceptInviteResponse represents the response after accepting an invitation
type AcceptInviteResponse struct {
	Message        string         `json:"message" example:"Invitation accepted successfully. You are now a member of the organization."`
	Organization   OrgInfo        `json:"organization"`
	YourRole       string         `json:"your_role" example:"operator"`
	AccessGranted  bool           `json:"access_granted" example:"true"`
}

// RejectInviteResponse represents the response after rejecting an invitation
type RejectInviteResponse struct {
	Message        string  `json:"message" example:"Invitation rejected successfully."`
	OrganizationName string `json:"organization_name" example:"Acme Corp"`
}

// OrgInfo represents basic organization information
type OrgInfo struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ExternalID  string `json:"external_id" example:"org_ABC12345"`
	Name        string `json:"name" example:"Acme Corp"`
	Plan        string `json:"plan" example:"tier_1"`
}

// ListInvitesQuery represents query parameters for listing invitations
type ListInvitesQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=pending accepted rejected expired revoked" example:"pending"`
	Page   int    `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
}

// ListInvitesResponse represents paginated list of invitations
type ListInvitesResponse struct {
	Invites    []InviteResponse `json:"invites"`
	Pagination PaginationInfo   `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int `json:"page" example:"1"`
	Limit      int `json:"limit" example:"20"`
	Total      int `json:"total" example:"45"`
	TotalPages int `json:"total_pages" example:"3"`
}

// RoleInfo represents an organization role
type RoleInfo struct {
	Name        string `json:"name" example:"operator"`
	DisplayName string `json:"display_name" example:"Operator"`
	Description string `json:"description" example:"Can create and manage shipments"`
	Level       int    `json:"level" example:"30"`
}
