package commissions

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
	// Number prefixes
	PayoutRequestNumberPrefix = "PRQ-"
	PayoutNumberPrefix        = "PAY-"
)

// CommissionService handles commission business logic
type CommissionService struct {
	db *clients.Database
}

// NewCommissionService creates a new commission service
func NewCommissionService(db *clients.Database) *CommissionService {
	return &CommissionService{db: db}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *CommissionService) GetOrganizationIDFromUser(userID string) (string, error) {
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
func (s *CommissionService) GetUserRole(userID string) (string, error) {
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

// generatePayoutRequestNumber generates a unique payout request number
func (s *CommissionService) generatePayoutRequestNumber() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return PayoutRequestNumberPrefix + hex.EncodeToString(bytes), nil
}

// generatePayoutNumber generates a unique payout number
func (s *CommissionService) generatePayoutNumber() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return PayoutNumberPrefix + hex.EncodeToString(bytes), nil
}

// ListCommissions retrieves commission transactions
func (s *CommissionService) ListCommissions(req ListCommissionsRequest, organizationID, userID string, isAdmin bool) (*ListCommissionsResponse, error) {
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
		SELECT id, commission_transaction_number, organization_id, beneficiary_user_id,
			shipment_id, commission_amount, currency, status,
			earned_at, eligible_at, paid_at, payout_id, created_at, updated_at
		FROM commission_transactions
		WHERE organization_id = $1
	`
	args := []interface{}{organizationID}
	argIndex := 2

	// Non-admins can only see their own commissions
	if !isAdmin {
		query += fmt.Sprintf(" AND beneficiary_user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.ShipmentID != "" {
		query += fmt.Sprintf(" AND shipment_id = $%d", argIndex)
		args = append(args, req.ShipmentID)
		argIndex++
	}

	if req.FromDate != "" {
		query += fmt.Sprintf(" AND earned_at >= $%d", argIndex)
		args = append(args, req.FromDate)
		argIndex++
	}

	if req.ToDate != "" {
		query += fmt.Sprintf(" AND earned_at <= $%d", argIndex)
		args = append(args, req.ToDate+" 23:59:59")
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var totalCount int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count commissions: %w", err)
	}

	// Calculate totals
	var totalEarnings, eligibleBalance, paidBalance float64
	totalsQuery := `
		SELECT
			COALESCE(SUM(commission_amount), 0) as total,
			COALESCE(SUM(CASE WHEN status = 'eligible' THEN commission_amount ELSE 0 END), 0) as eligible,
			COALESCE(SUM(CASE WHEN status = 'paid' THEN commission_amount ELSE 0 END), 0) as paid
		FROM (` + query + `) AS totals_query
	`
	err = s.db.DB.QueryRow(totalsQuery, args...).Scan(&totalEarnings, &eligibleBalance, &paidBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate totals: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY earned_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commissions: %w", err)
	}
	defer rows.Close()

	commissions := []*CommissionTransactionResponse{}
	for rows.Next() {
		var tx models.CommissionTransaction
		err := rows.Scan(
			&tx.ID, &tx.CommissionTransactionNumber, &tx.OrganizationID, &tx.BeneficiaryUserID,
			&tx.ShipmentID, &tx.CommissionAmount, &tx.Currency, &tx.Status,
			&tx.EarnedAt, &tx.EligibleAt, &tx.PaidAt, &tx.PayoutID, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan commission: %w", err)
		}
		commissions = append(commissions, ToCommissionTransactionResponse(&tx))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commissions: %w", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	return &ListCommissionsResponse{
		Commissions:     commissions,
		TotalCount:      totalCount,
		Page:            page,
		PageSize:        pageSize,
		TotalPages:      totalPages,
		TotalEarnings:   totalEarnings,
		EligibleBalance: eligibleBalance,
		PaidBalance:     paidBalance,
	}, nil
}

// ListPayouts retrieves payout records
func (s *CommissionService) ListPayouts(req ListPayoutsRequest, organizationID, userID string, isAdmin bool) (*ListPayoutsResponse, error) {
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
		SELECT id, payout_number, organization_id, beneficiary_user_id,
			amount, currency, method, status, user_visible_note,
			requested_at, processed_at, paid_at, created_at, updated_at
		FROM commission_payouts
		WHERE organization_id = $1
	`
	args := []interface{}{organizationID}
	argIndex := 2

	// Non-admins can only see their own payouts
	if !isAdmin {
		query += fmt.Sprintf(" AND beneficiary_user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.FromDate != "" {
		query += fmt.Sprintf(" AND requested_at >= $%d", argIndex)
		args = append(args, req.FromDate)
		argIndex++
	}

	if req.ToDate != "" {
		query += fmt.Sprintf(" AND requested_at <= $%d", argIndex)
		args = append(args, req.ToDate+" 23:59:59")
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var totalCount int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count payouts: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY requested_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payouts: %w", err)
	}
	defer rows.Close()

	payouts := []*PayoutResponse{}
	for rows.Next() {
		var payout models.CommissionPayout
		err := rows.Scan(
			&payout.ID, &payout.PayoutNumber, &payout.OrganizationID, &payout.BeneficiaryUserID,
			&payout.Amount, &payout.Currency, &payout.Method, &payout.Status, &payout.UserVisibleNote,
			&payout.RequestedAt, &payout.ProcessedAt, &payout.PaidAt, &payout.CreatedAt, &payout.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout: %w", err)
		}

		// Get transaction count for this payout
		var txCount int
		s.db.DB.QueryRow(`SELECT COUNT(*) FROM commission_payout_items WHERE payout_id = $1`, payout.ID).Scan(&txCount)

		resp := ToPayoutResponse(&payout)
		resp.TransactionCount = txCount
		payouts = append(payouts, resp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payouts: %w", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	return &ListPayoutsResponse{
		Payouts:    payouts,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// RequestPayout creates a payout request for a sales rep
func (s *CommissionService) RequestPayout(req RequestPayoutRequest, organizationID, userID string) (*RequestPayoutResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify all commission transactions exist, are eligible, and belong to this user
	var totalAmount float64
	var currency string
	for i, txID := range req.CommissionTransactionIDs {
		var commTx models.CommissionTransaction
		err = tx.QueryRow(`
			SELECT id, beneficiary_user_id, organization_id, commission_amount, currency, status
			FROM commission_transactions
			WHERE id = $1
		`, txID).Scan(
			&commTx.ID, &commTx.BeneficiaryUserID, &commTx.OrganizationID,
			&commTx.CommissionAmount, &commTx.Currency, &commTx.Status,
		)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("commission transaction %s not found", txID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch commission transaction: %w", err)
		}

		// Verify ownership and organization
		if commTx.BeneficiaryUserID != userID {
			return nil, fmt.Errorf("commission transaction %s does not belong to you", txID)
		}
		if commTx.OrganizationID != organizationID {
			return nil, fmt.Errorf("commission transaction %s does not belong to your organization", txID)
		}

		// Must be eligible
		if commTx.Status != models.CommissionTxStatusEligible {
			return nil, fmt.Errorf("commission transaction %s is not eligible (status: %s)", txID, commTx.Status)
		}

		// Accumulate total
		totalAmount += commTx.CommissionAmount
		if i == 0 {
			currency = commTx.Currency
		} else if commTx.Currency != currency {
			return nil, fmt.Errorf("all commission transactions must have the same currency")
		}
	}

	if totalAmount <= 0 {
		return nil, fmt.Errorf("no eligible commissions to pay out")
	}

	// Generate payout request number
	payoutRequestNumber, err := s.generatePayoutRequestNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payout request number: %w", err)
	}

	now := time.Now()

	// Create payout request
	var payoutRequestID string
	err = tx.QueryRow(`
		INSERT INTO commission_payout_requests (
			payout_request_number, organization_id, requested_by_user_id, beneficiary_user_id,
			amount_requested, currency, method, status, request_note,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, 'pending', $8,
			$9, $10
		)
		RETURNING id
	`, payoutRequestNumber, organizationID, userID, userID,
		totalAmount, currency, req.PayoutMethod, req.Note,
		now, now).Scan(&payoutRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to create payout request: %w", err)
	}

	// Log audit entry
	eventID := fmt.Sprintf("evt_%s_%d", payoutRequestID[:8], now.UnixNano())
	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			event_id, actor_type, user_id, action, resource_type, resource_id,
			business_purpose, status, occurred_at
		) VALUES (
			$1, 'user', $2, 'commission.payout_requested', 'commission_payout_request', $3,
			'commission_management', 'success', $4
		)
	`, eventID, userID, payoutRequestID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to log audit entry: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &RequestPayoutResponse{
		PayoutRequestID:     payoutRequestID,
		PayoutRequestNumber: payoutRequestNumber,
		AmountRequested:     totalAmount,
		Currency:            currency,
		Status:              string(models.PayoutRequestStatusPending),
		TransactionCount:    len(req.CommissionTransactionIDs),
		Message:             "Payout request submitted successfully and is pending admin review",
		CreatedAt:           now,
	}, nil
}

// ProcessPayout approves or rejects a payout request (admin only)
func (s *CommissionService) ProcessPayout(req ProcessPayoutRequest, organizationID, userID string) (*ProcessPayoutResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Fetch payout request
	var payoutReq models.CommissionPayoutRequest
	err = tx.QueryRow(`
		SELECT id, payout_request_number, organization_id, beneficiary_user_id,
			amount_requested, currency, method, status
		FROM commission_payout_requests
		WHERE id = $1 AND organization_id = $2
	`, req.PayoutRequestID, organizationID).Scan(
		&payoutReq.ID, &payoutReq.PayoutRequestNumber, &payoutReq.OrganizationID,
		&payoutReq.BeneficiaryUserID, &payoutReq.AmountRequested, &payoutReq.Currency,
		&payoutReq.Method, &payoutReq.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payout request not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payout request: %w", err)
	}

	// Must be pending
	if payoutReq.Status != models.PayoutRequestStatusPending {
		return nil, fmt.Errorf("payout request is not pending (status: %s)", payoutReq.Status)
	}

	now := time.Now()

	if req.Action == "reject" {
		// Reject the payout request
		_, err = tx.Exec(`
			UPDATE commission_payout_requests
			SET status = 'rejected', reviewed_by_user_id = $1, reviewed_at = $2, review_note = $3, updated_at = $4
			WHERE id = $5
		`, userID, now, req.ReviewNote, now, req.PayoutRequestID)
		if err != nil {
			return nil, fmt.Errorf("failed to reject payout request: %w", err)
		}

		// Log audit entry
		rejectEventID := fmt.Sprintf("evt_%s_%d", req.PayoutRequestID[:8], now.UnixNano())
		_, err = tx.Exec(`
			INSERT INTO audit_logs (
				event_id, actor_type, user_id, action, resource_type, resource_id,
				business_purpose, status, occurred_at
			) VALUES (
				$1, 'user', $2, 'commission.payout_rejected', 'commission_payout_request', $3,
				'commission_management', 'success', $4
			)
		`, rejectEventID, userID, req.PayoutRequestID, now)

		if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return &ProcessPayoutResponse{
			Status:      string(models.PayoutRequestStatusRejected),
			Message:     "Payout request rejected",
			ProcessedAt: now,
		}, nil
	}

	// Approve and create payout
	// Generate payout number
	payoutNumber, err := s.generatePayoutNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payout number: %w", err)
	}

	// Create payout
	var payoutID string
	err = tx.QueryRow(`
		INSERT INTO commission_payouts (
			payout_number, organization_id, beneficiary_user_id, payout_request_id,
			method, amount, currency, status, payout_note, manual_reference,
			created_by_user_id, requested_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, 'paid', 'Approved by admin', $8,
			$9, $10, $11, $12
		)
		RETURNING id
	`, payoutNumber, organizationID, payoutReq.BeneficiaryUserID, req.PayoutRequestID,
		payoutReq.Method, payoutReq.AmountRequested, payoutReq.Currency, req.ManualReference,
		userID, now, now, now).Scan(&payoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to create payout: %w", err)
	}

	// Update payout request status
	_, err = tx.Exec(`
		UPDATE commission_payout_requests
		SET status = 'approved', reviewed_by_user_id = $1, reviewed_at = $2, review_note = $3, updated_at = $4
		WHERE id = $5
	`, userID, now, req.ReviewNote, now, req.PayoutRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to update payout request: %w", err)
	}

	// Mark commission transactions as paid
	// This is a simplified version - in production you'd get the specific transactions from the request
	_, err = tx.Exec(`
		UPDATE commission_transactions
		SET status = 'paid', payout_id = $1, paid_at = $2, updated_at = $3
		WHERE beneficiary_user_id = $4 AND status = 'eligible' AND organization_id = $5
	`, payoutID, now, now, payoutReq.BeneficiaryUserID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to update commission transactions: %w", err)
	}

	// Log audit entry
	processEventID := fmt.Sprintf("evt_%s_%d", payoutID[:8], now.UnixNano())
	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			event_id, actor_type, user_id, action, resource_type, resource_id,
			business_purpose, status, occurred_at
		) VALUES (
			$1, 'user', $2, 'commission.payout_processed', 'commission_payout', $3,
			'commission_management', 'success', $4
		)
	`, processEventID, userID, payoutID, now)

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &ProcessPayoutResponse{
		PayoutID:     payoutID,
		PayoutNumber: payoutNumber,
		Status:       string(models.PayoutStatusPaid),
		Amount:       payoutReq.AmountRequested,
		Message:      "Payout processed successfully",
		ProcessedAt:  now,
	}, nil
}

// CreateCommissionOnShipmentComplete creates a commission transaction when a shipment is delivered
// This should be called when shipment status changes to "delivered"
func (s *CommissionService) CreateCommissionOnShipmentComplete(shipmentID, organizationID string, revenueAmount, costAmount float64, currency string) error {
	// Find if the shipment creator was referred by someone
	var referredUserID string
	var referrerUserID sql.NullString
	var commissionPlanID sql.NullString
	var commissionNumber, commissionBase, commissionValueType string
	var commissionValue float64

	// Get the user who created the shipment and check if they were referred
	err := s.db.DB.QueryRow(`
		SELECT s.created_by_user_id, u.referred_by_user_id
		FROM shipments s
		JOIN users u ON u.id = s.created_by_user_id
		WHERE s.id = $1 AND s.organization_id = $2
	`, shipmentID, organizationID).Scan(&referredUserID, &referrerUserID)

	if err != nil {
		return fmt.Errorf("failed to get shipment creator: %w", err)
	}

	// If user wasn't referred, no commission to create
	if !referrerUserID.Valid || referrerUserID.String == "" {
		return nil
	}

	// Get the referral redemption to get the commission plan snapshot
	err = s.db.DB.QueryRow(`
		SELECT commission_plan_id, commission_number, commission_base, commission_value_type, commission_value
		FROM referral_redemptions
		WHERE referred_user_id = $1 AND referrer_user_id = $2 AND status = 'active'
		LIMIT 1
	`, referredUserID, referrerUserID.String).Scan(&commissionPlanID, &commissionNumber, &commissionBase, &commissionValueType, &commissionValue)

	if err == sql.ErrNoRows {
		// No active referral relationship, no commission
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get referral redemption: %w", err)
	}

	// Check if commission already exists for this shipment+beneficiary+plan combination
	var existingCount int
	err = s.db.DB.QueryRow(`
		SELECT COUNT(*) FROM commission_transactions
		WHERE shipment_id = $1 AND beneficiary_user_id = $2 AND commission_number = $3
	`, shipmentID, referrerUserID.String, commissionNumber).Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed to check existing commission: %w", err)
	}
	if existingCount > 0 {
		// Commission already exists
		return nil
	}

	// Calculate commission amount based on commission_base
	profitAmount := revenueAmount - costAmount
	var commissionAmount float64

	switch commissionBase {
	case "profit":
		if commissionValueType == "percent" {
			commissionAmount = profitAmount * (commissionValue / 100.0)
		} else {
			commissionAmount = commissionValue
		}
	case "revenue":
		if commissionValueType == "percent" {
			commissionAmount = revenueAmount * (commissionValue / 100.0)
		} else {
			commissionAmount = commissionValue
		}
	case "per_shipment":
		commissionAmount = commissionValue // Fixed amount per shipment
	default:
		commissionAmount = profitAmount * (commissionValue / 100.0) // Default to profit-based percent
	}

	// Ensure commission is non-negative
	if commissionAmount < 0 {
		commissionAmount = 0
	}

	// Generate commission transaction number
	txNumber, err := s.generateCommissionTransactionNumber()
	if err != nil {
		return fmt.Errorf("failed to generate commission transaction number: %w", err)
	}

	now := time.Now()
	commissionID := s.generateUUID()

	// Create the commission transaction (status: eligible since shipment is completed)
	_, err = s.db.DB.Exec(`
		INSERT INTO commission_transactions (
			id, commission_transaction_number, organization_id, beneficiary_user_id,
			source_user_id, shipment_id, commission_plan_id, commission_number,
			commission_base, commission_value_type, commission_value,
			currency, revenue_amount, cost_amount, profit_amount, commission_amount,
			status, earned_at, eligible_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			'eligible', $17, $18, $19, $20
		)
	`, commissionID, txNumber, organizationID, referrerUserID.String,
		referredUserID, shipmentID, commissionPlanID, commissionNumber,
		commissionBase, commissionValueType, commissionValue,
		currency, revenueAmount, costAmount, profitAmount, commissionAmount,
		now, now, now, now)

	if err != nil {
		return fmt.Errorf("failed to create commission transaction: %w", err)
	}

	return nil
}

// generateCommissionTransactionNumber generates a unique commission transaction number
func (s *CommissionService) generateCommissionTransactionNumber() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "CTX-" + hex.EncodeToString(bytes), nil
}

// generateUUID generates a new UUID string
func (s *CommissionService) generateUUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
