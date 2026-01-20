package referrals

import (
	"database/sql"
	"fmt"

	"gogin/internal/clients"
	"gogin/internal/models"
)

// ReferralService handles referral business logic
type ReferralService struct {
	db *clients.Database
}

// NewReferralService creates a new referral service
func NewReferralService(db *clients.Database) *ReferralService {
	return &ReferralService{db: db}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *ReferralService) GetOrganizationIDFromUser(userID string) (string, error) {
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

// GetUserRole retrieves the role of a user from their organization membership
func (s *ReferralService) GetUserRole(userID string) (string, error) {
	query := `
		SELECT role
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`

	var role string
	err := s.db.DB.QueryRow(query, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user not found in any organization")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// ListReferrals retrieves referral redemptions
func (s *ReferralService) ListReferrals(req ListReferralsRequest, userID string, isAdmin bool) (*ListReferralsResponse, error) {
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
		SELECT rr.id, rr.reference_number, rr.organization_id, rr.referrer_user_id,
			rr.referred_user_id, rr.status, rr.redeemed_at,
			rr.commission_plan_id, rr.commission_number, rr.commission_base,
			rr.commission_value_type, rr.commission_value, rr.snapshot_taken_at,
			rr.metadata, rr.created_at,
			ue.email as referred_user_email
		FROM referral_redemptions rr
		LEFT JOIN users u ON u.id = rr.referred_user_id
		LEFT JOIN user_emails ue ON ue.id = u.primary_email_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Non-admins (Sales Reps) can only see their own referrals
	if !isAdmin {
		query += fmt.Sprintf(" AND rr.referrer_user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND rr.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.FromDate != "" {
		query += fmt.Sprintf(" AND rr.redeemed_at >= $%d", argIndex)
		args = append(args, req.FromDate)
		argIndex++
	}

	if req.ToDate != "" {
		query += fmt.Sprintf(" AND rr.redeemed_at <= $%d", argIndex)
		args = append(args, req.ToDate+" 23:59:59")
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var totalCount int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count referrals: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY rr.redeemed_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query referrals: %w", err)
	}
	defer rows.Close()

	var referrals []*models.ReferralRedemption
	var emails []string

	for rows.Next() {
		var ref models.ReferralRedemption
		var referredEmail sql.NullString

		err := rows.Scan(
			&ref.ID,
			&ref.ReferenceNumber,
			&ref.OrganizationID,
			&ref.ReferrerUserID,
			&ref.ReferredUserID,
			&ref.Status,
			&ref.RedeemedAt,
			&ref.CommissionPlanID,
			&ref.CommissionNumber,
			&ref.CommissionBase,
			&ref.CommissionValueType,
			&ref.CommissionValue,
			&ref.SnapshotTakenAt,
			&ref.Metadata,
			&ref.CreatedAt,
			&referredEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan referral: %w", err)
		}

		referrals = append(referrals, &ref)
		if referredEmail.Valid {
			emails = append(emails, referredEmail.String)
		} else {
			emails = append(emails, "")
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating referrals: %w", err)
	}

	// Get status counts
	statusCounts, err := s.getStatusCounts(userID, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	// Build response
	totalPages := (totalCount + pageSize - 1) / pageSize
	response := &ListReferralsResponse{
		Referrals:    make([]*ReferralRedemptionResponse, 0, len(referrals)),
		TotalCount:   totalCount,
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		ActiveCount:  statusCounts["active"],
		RevokedCount: statusCounts["revoked"],
		ExpiredCount: statusCounts["expired"],
	}

	for i, ref := range referrals {
		response.Referrals = append(response.Referrals, ToReferralRedemptionResponse(ref, emails[i]))
	}

	return response, nil
}

// getStatusCounts retrieves count of referrals by status
func (s *ReferralService) getStatusCounts(userID string, isAdmin bool) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM referral_redemptions
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if !isAdmin {
		query += fmt.Sprintf(" AND referrer_user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	query += " GROUP BY status"

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query status counts: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{
		"active":  0,
		"revoked": 0,
		"expired": 0,
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		counts[status] = count
	}

	return counts, nil
}

// GetReferralByID retrieves a single referral by ID
func (s *ReferralService) GetReferralByID(referralID string, userID string, isAdmin bool) (*ReferralDetailResponse, error) {
	query := `
		SELECT rr.id, rr.reference_number, rr.organization_id, rr.referrer_user_id,
			rr.referred_user_id, rr.status, rr.redeemed_at,
			rr.commission_plan_id, rr.commission_number, rr.commission_base,
			rr.commission_value_type, rr.commission_value, rr.snapshot_taken_at,
			rr.metadata, rr.created_at,
			ue1.email as referrer_email,
			ue2.email as referred_email
		FROM referral_redemptions rr
		LEFT JOIN users u1 ON u1.id = rr.referrer_user_id
		LEFT JOIN user_emails ue1 ON ue1.id = u1.primary_email_id
		LEFT JOIN users u2 ON u2.id = rr.referred_user_id
		LEFT JOIN user_emails ue2 ON ue2.id = u2.primary_email_id
		WHERE rr.id = $1
	`
	args := []interface{}{referralID}
	argIndex := 2

	// Non-admins can only see their own referrals
	if !isAdmin {
		query += fmt.Sprintf(" AND rr.referrer_user_id = $%d", argIndex)
		args = append(args, userID)
	}

	var ref models.ReferralRedemption
	var referrerEmail, referredEmail sql.NullString

	err := s.db.DB.QueryRow(query, args...).Scan(
		&ref.ID,
		&ref.ReferenceNumber,
		&ref.OrganizationID,
		&ref.ReferrerUserID,
		&ref.ReferredUserID,
		&ref.Status,
		&ref.RedeemedAt,
		&ref.CommissionPlanID,
		&ref.CommissionNumber,
		&ref.CommissionBase,
		&ref.CommissionValueType,
		&ref.CommissionValue,
		&ref.SnapshotTakenAt,
		&ref.Metadata,
		&ref.CreatedAt,
		&referrerEmail,
		&referredEmail,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("referral not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query referral: %w", err)
	}

	referrerEmailStr := ""
	if referrerEmail.Valid {
		referrerEmailStr = referrerEmail.String
	}

	referredEmailStr := ""
	if referredEmail.Valid {
		referredEmailStr = referredEmail.String
	}

	return ToReferralDetailResponse(&ref, referrerEmailStr, referredEmailStr), nil
}
