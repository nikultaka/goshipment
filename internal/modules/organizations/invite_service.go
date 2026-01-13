package organizations

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// OrganizationInviteService handles organization invitation business logic
type OrganizationInviteService struct {
	db *clients.Database
}

// NewOrganizationInviteService creates a new organization invite service
func NewOrganizationInviteService(db *clients.Database) *OrganizationInviteService {
	return &OrganizationInviteService{db: db}
}

// CreateInvite creates a new organization invitation
func (s *OrganizationInviteService) CreateInvite(
	req CreateInviteRequest,
	organizationID string,
	invitedByUserID string,
) (*models.OrganizationInvite, error) {
	// 1. Check if user is already a member
	var existingMember string
	err := s.db.QueryRow(`
		SELECT om.id FROM organization_members om
		JOIN user_emails ue ON om.user_id = ue.user_id
		WHERE om.organization_id = $1 AND ue.email = $2 AND om.is_active = true
	`, organizationID, req.Email).Scan(&existingMember)

	if err == nil {
		return nil, errors.New("user is already a member of this organization")
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing membership: %w", err)
	}

	// 2. Check for pending invitations
	var pendingInvite string
	err = s.db.QueryRow(`
		SELECT id FROM organization_invites
		WHERE organization_id = $1 AND email = $2 AND status = 'pending' AND expires_at > NOW()
	`, organizationID, req.Email).Scan(&pendingInvite)

	if err == nil {
		return nil, errors.New("a pending invitation already exists for this email")
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check pending invitations: %w", err)
	}

	// 3. Generate secure invite token
	inviteToken, err := generateSecureToken(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}

	// 4. Create invitation record
	inviteID := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 days expiry

	invite := &models.OrganizationInvite{
		ID:              inviteID,
		OrganizationID:  organizationID,
		InvitedByUserID: invitedByUserID,
		Email:           req.Email,
		Role:            req.Role,
		InviteToken:     inviteToken,
		Status:          models.InviteStatusPending,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = s.db.Exec(`
		INSERT INTO organization_invites (
			id, organization_id, invited_by_user_id, email, role,
			invite_token, status, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, invite.ID, invite.OrganizationID, invite.InvitedByUserID, invite.Email, invite.Role,
		invite.InviteToken, invite.Status, invite.ExpiresAt, invite.CreatedAt, invite.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// 5. TODO: Send email notification (implement later)
	// This would call your email service to send the invitation email
	// with the invite link containing the token

	return invite, nil
}

// AcceptInvite accepts an organization invitation
func (s *OrganizationInviteService) AcceptInvite(token string) (*models.OrganizationInvite, error) {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get invitation by token
	var invite models.OrganizationInvite
	err = tx.QueryRow(`
		SELECT id, organization_id, invited_by_user_id, email, role,
		       invite_token, status, expires_at, created_at, updated_at
		FROM organization_invites
		WHERE invite_token = $1
	`, token).Scan(
		&invite.ID, &invite.OrganizationID, &invite.InvitedByUserID, &invite.Email, &invite.Role,
		&invite.InviteToken, &invite.Status, &invite.ExpiresAt, &invite.CreatedAt, &invite.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired invitation token")
		}
		return nil, fmt.Errorf("failed to fetch invitation: %w", err)
	}

	// 2. Validate invitation can be accepted
	if !invite.CanBeAccepted() {
		if invite.IsExpired() {
			return nil, errors.New("invitation has expired")
		}
		return nil, fmt.Errorf("invitation cannot be accepted (current status: %s)", invite.Status)
	}

	// 3. Find or verify user by email
	var userID string
	err = tx.QueryRow(`
		SELECT user_id FROM user_emails
		WHERE email = $1 AND is_primary = true
		LIMIT 1
	`, invite.Email).Scan(&userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no user account found with this email. Please register first")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// 4. Check if already a member
	var existingMember string
	err = tx.QueryRow(`
		SELECT id FROM organization_members
		WHERE organization_id = $1 AND user_id = $2 AND is_active = true
	`, invite.OrganizationID, userID).Scan(&existingMember)

	if err == nil {
		return nil, errors.New("you are already a member of this organization")
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	// 5. Create organization member
	memberID := uuid.New().String()
	now := time.Now().UTC()

	_, err = tx.Exec(`
		INSERT INTO organization_members (
			id, organization_id, user_id, role, is_active, joined_at
		) VALUES ($1, $2, $3, $4, true, $5)
	`, memberID, invite.OrganizationID, userID, invite.Role, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create organization member: %w", err)
	}

	// 6. Update invitation status
	_, err = tx.Exec(`
		UPDATE organization_invites
		SET status = 'accepted', accepted_at = $1, updated_at = $2
		WHERE id = $3
	`, now, now, invite.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to update invitation status: %w", err)
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	invite.Status = models.InviteStatusAccepted
	invite.AcceptedAt = &now

	return &invite, nil
}

// RejectInvite rejects an organization invitation
func (s *OrganizationInviteService) RejectInvite(token string) (*models.OrganizationInvite, error) {
	// 1. Get invitation by token
	var invite models.OrganizationInvite
	err := s.db.QueryRow(`
		SELECT id, organization_id, invited_by_user_id, email, role,
		       invite_token, status, expires_at, created_at, updated_at
		FROM organization_invites
		WHERE invite_token = $1
	`, token).Scan(
		&invite.ID, &invite.OrganizationID, &invite.InvitedByUserID, &invite.Email, &invite.Role,
		&invite.InviteToken, &invite.Status, &invite.ExpiresAt, &invite.CreatedAt, &invite.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired invitation token")
		}
		return nil, fmt.Errorf("failed to fetch invitation: %w", err)
	}

	// 2. Validate invitation can be rejected
	if !invite.CanBeRejected() {
		if invite.IsExpired() {
			return nil, errors.New("invitation has expired")
		}
		return nil, fmt.Errorf("invitation cannot be rejected (current status: %s)", invite.Status)
	}

	// 3. Update invitation status
	now := time.Now().UTC()
	_, err = s.db.Exec(`
		UPDATE organization_invites
		SET status = 'rejected', rejected_at = $1, updated_at = $2
		WHERE id = $3
	`, now, now, invite.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to update invitation status: %w", err)
	}

	invite.Status = models.InviteStatusRejected
	invite.RejectedAt = &now

	return &invite, nil
}

// ListInvites lists invitations for an organization
func (s *OrganizationInviteService) ListInvites(
	organizationID string,
	query ListInvitesQuery,
) ([]InviteResponse, int, error) {
	// Set defaults
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Build WHERE clause
	whereClause := "oi.organization_id = $1"
	args := []interface{}{organizationID}
	argIndex := 2

	if query.Status != "" {
		whereClause += fmt.Sprintf(" AND oi.status = $%d", argIndex)
		args = append(args, query.Status)
		argIndex++
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organization_invites oi WHERE %s", whereClause)
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get invitations with inviter details
	listQuery := fmt.Sprintf(`
		SELECT
			oi.id, oi.organization_id, oi.email, oi.role, oi.status,
			oi.expires_at, oi.accepted_at, oi.rejected_at, oi.revoked_at, oi.created_at,
			u.id as inviter_id, u.first_name, u.last_name,
			ue.email as inviter_email
		FROM organization_invites oi
		JOIN users u ON oi.invited_by_user_id = u.id
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE %s
		ORDER BY oi.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.Query(listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch invitations: %w", err)
	}
	defer rows.Close()

	var invites []InviteResponse
	for rows.Next() {
		var invite InviteResponse
		var inviter UserInfo
		var inviterEmail sql.NullString

		err := rows.Scan(
			&invite.ID, &invite.OrganizationID, &invite.Email, &invite.Role, &invite.Status,
			&invite.ExpiresAt, &invite.AcceptedAt, &invite.RejectedAt, &invite.RevokedAt, &invite.CreatedAt,
			&inviter.ID, &inviter.FirstName, &inviter.LastName, &inviterEmail,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan invitation: %w", err)
		}

		if inviterEmail.Valid {
			inviter.Email = inviterEmail.String
		}
		invite.InvitedBy = inviter
		invite.IsExpired = time.Now().UTC().After(invite.ExpiresAt)

		invites = append(invites, invite)
	}

	return invites, total, nil
}

// RevokeInvite revokes a pending invitation
func (s *OrganizationInviteService) RevokeInvite(inviteID, organizationID string) error {
	now := time.Now().UTC()

	result, err := s.db.Exec(`
		UPDATE organization_invites
		SET status = 'revoked', revoked_at = $1, updated_at = $2
		WHERE id = $3 AND organization_id = $4 AND status = 'pending'
	`, now, now, inviteID, organizationID)

	if err != nil {
		return fmt.Errorf("failed to revoke invitation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invitation not found or cannot be revoked")
	}

	return nil
}

// GetOrganizationInfo retrieves basic organization information
func (s *OrganizationInviteService) GetOrganizationInfo(organizationID string) (*OrgInfo, error) {
	var org OrgInfo
	err := s.db.QueryRow(`
		SELECT id, external_id, name, plan
		FROM organizations
		WHERE id = $1
	`, organizationID).Scan(&org.ID, &org.ExternalID, &org.Name, &org.Plan)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}

	return &org, nil
}

// =====================================================
// HELPER FUNCTIONS
// =====================================================

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length/2) // hex encoding doubles the length
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
