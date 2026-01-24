package wallet

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gogin/internal/clients"
)

const (
	// Number prefixes
	WalletNumberPrefix      = "WAL-"
	TransactionNumberPrefix = "TRX-"
	TopUpNumberPrefix       = "TOP-"

	// Retry configuration for optimistic locking
	MaxRetries    = 3
	RetryDelayMS  = 100
)

// WalletService handles wallet business logic
type WalletService struct {
	db *clients.Database
}

// NewWalletService creates a new wallet service
func NewWalletService(db *clients.Database) *WalletService {
	return &WalletService{
		db: db,
	}
}

// walletInfo holds wallet details
type walletInfo struct {
	ID             string
	WalletNumber   string
	OrganizationID string
	Balance        float64
	HeldBalance    float64
	Status         string
	Version        int64
	MinTopUpAmount float64
	MaxTopUpAmount sql.NullFloat64
	MaxBalance     sql.NullFloat64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// stripePaymentResponse represents mock Stripe response
type stripePaymentResponse struct {
	PaymentIntentID string
	ChargeID        string
	PaymentMethodID string
	ReceiptURL      string
	Status          string
	Amount          float64
	Currency        string
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *WalletService) GetOrganizationIDFromUser(userID string) (string, error) {
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

// GetWallet retrieves wallet information for an organization
func (s *WalletService) GetWallet(organizationID string) (*WalletResponse, error) {
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Get carrier refund in progress amount
	carrierRefundInProgress, err := s.getCarrierRefundInProgress(wallet.ID)
	if err != nil {
		// Log error but don't fail - just set to 0
		carrierRefundInProgress = 0
	}

	// Get total in-transit (pending wallet holds)
	var totalInTransit float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_holds
		WHERE wallet_id = $1 AND status IN ('pending', 'active')
	`, wallet.ID).Scan(&totalInTransit)
	if err != nil && err != sql.ErrNoRows {
		totalInTransit = 0
	}

	// Get this month's deposits and outflow
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var totalDeposits float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_transactions
		WHERE wallet_id = $1 AND direction = 'credit' AND created_at >= $2
	`, wallet.ID, startOfMonth).Scan(&totalDeposits)
	if err != nil && err != sql.ErrNoRows {
		totalDeposits = 0
	}

	var totalOutflow float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_transactions
		WHERE wallet_id = $1 AND direction = 'debit' AND created_at >= $2
	`, wallet.ID, startOfMonth).Scan(&totalOutflow)
	if err != nil && err != sql.ErrNoRows {
		totalOutflow = 0
	}

	netMovement := totalDeposits - totalOutflow

	// Build response
	resp := &WalletResponse{
		ID:                      wallet.ID,
		WalletNumber:            wallet.WalletNumber,
		OrganizationID:          wallet.OrganizationID,

		// New balance fields
		AvailableBalance:        wallet.Balance,
		TotalInTransit:          totalInTransit,
		Reserved:                wallet.HeldBalance,
		TotalBalance:            wallet.Balance + wallet.HeldBalance + totalInTransit,

		// Financial summary
		TotalDeposits:           totalDeposits,
		TotalOutflow:            totalOutflow,
		NetMovement:             netMovement,

		// Legacy fields
		Balance:                 wallet.Balance,
		HeldBalance:             wallet.HeldBalance,
		CarrierRefundInProgress: carrierRefundInProgress,

		Status:                  wallet.Status,
		Currency:                "CAD",
		MinTopUpAmount:          wallet.MinTopUpAmount,
		CreatedAt:               wallet.CreatedAt,
		UpdatedAt:               wallet.UpdatedAt,
	}

	if wallet.MaxTopUpAmount.Valid {
		resp.MaxTopUpAmount = &wallet.MaxTopUpAmount.Float64
	}
	if wallet.MaxBalance.Valid {
		resp.MaxBalance = &wallet.MaxBalance.Float64
	}

	return resp, nil
}

// GetWalletWithTransactions retrieves wallet with recent transactions
func (s *WalletService) GetWalletWithTransactions(organizationID string, page, pageSize int) (*WalletResponse, error) {
	wallet, err := s.GetWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Get recent transactions
	transactions, err := s.getTransactions(wallet.ID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	wallet.RecentTransactions = transactions

	return wallet, nil
}

// ListTransactions retrieves paginated transactions with filters
func (s *WalletService) ListTransactions(organizationID string, req ListTransactionsRequest) (*TransactionsListResponse, error) {
	// Get wallet
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build count query
	countQuery := `SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1`
	countArgs := []interface{}{wallet.ID}
	argIndex := 2

	// Build select query
	selectQuery := `
		SELECT
			id, transaction_number, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			performed_by_user_id, is_system_generated, created_at
		FROM wallet_transactions
		WHERE wallet_id = $1
	`
	selectArgs := []interface{}{wallet.ID}

	// Add filters
	if req.TransactionID != nil && *req.TransactionID != "" {
		countQuery += fmt.Sprintf(" AND id = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND id = $%d", argIndex)
		countArgs = append(countArgs, *req.TransactionID)
		selectArgs = append(selectArgs, *req.TransactionID)
		argIndex++
	}

	if req.TransactionNumber != nil && *req.TransactionNumber != "" {
		countQuery += fmt.Sprintf(" AND transaction_number ILIKE $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND transaction_number ILIKE $%d", argIndex)
		searchTerm := "%" + *req.TransactionNumber + "%"
		countArgs = append(countArgs, searchTerm)
		selectArgs = append(selectArgs, searchTerm)
		argIndex++
	}

	if req.Direction != nil && *req.Direction != "" {
		countQuery += fmt.Sprintf(" AND direction = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND direction = $%d", argIndex)
		countArgs = append(countArgs, *req.Direction)
		selectArgs = append(selectArgs, *req.Direction)
		argIndex++
	}

	if req.TxType != nil && *req.TxType != "" {
		countQuery += fmt.Sprintf(" AND tx_type = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND tx_type = $%d", argIndex)
		countArgs = append(countArgs, *req.TxType)
		selectArgs = append(selectArgs, *req.TxType)
		argIndex++
	}

	if req.MinAmount != nil {
		countQuery += fmt.Sprintf(" AND amount >= $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND amount >= $%d", argIndex)
		countArgs = append(countArgs, *req.MinAmount)
		selectArgs = append(selectArgs, *req.MinAmount)
		argIndex++
	}

	if req.MaxAmount != nil {
		countQuery += fmt.Sprintf(" AND amount <= $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND amount <= $%d", argIndex)
		countArgs = append(countArgs, *req.MaxAmount)
		selectArgs = append(selectArgs, *req.MaxAmount)
		argIndex++
	}

	if req.FromDate != nil && *req.FromDate != "" {
		countQuery += fmt.Sprintf(" AND created_at >= $%d::date", argIndex)
		selectQuery += fmt.Sprintf(" AND created_at >= $%d::date", argIndex)
		countArgs = append(countArgs, *req.FromDate)
		selectArgs = append(selectArgs, *req.FromDate)
		argIndex++
	}

	if req.ToDate != nil && *req.ToDate != "" {
		countQuery += fmt.Sprintf(" AND created_at < ($%d::date + INTERVAL '1 day')", argIndex)
		selectQuery += fmt.Sprintf(" AND created_at < ($%d::date + INTERVAL '1 day')", argIndex)
		countArgs = append(countArgs, *req.ToDate)
		selectArgs = append(selectArgs, *req.ToDate)
		argIndex++
	}

	// Get total count
	var total int
	err = s.db.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Add sorting
	sortBy := "created_at"
	if req.SortBy != nil && *req.SortBy != "" {
		sortBy = *req.SortBy
	}
	sortOrder := "DESC"
	if req.SortOrder != nil && *req.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	selectQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	selectArgs = append(selectArgs, limit, offset)

	// Execute query
	rows, err := s.db.DB.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	// Parse results
	transactions := make([]TransactionResponse, 0)
	for rows.Next() {
		var t TransactionResponse
		var description, refType, refID, performedBy sql.NullString

		if err := rows.Scan(
			&t.ID, &t.TransactionNumber, &t.Direction, &t.TxType,
			&t.Amount, &t.Currency, &t.BalanceBefore, &t.BalanceAfter,
			&description, &refType, &refID,
			&performedBy, &t.IsSystemGenerated, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if description.Valid {
			t.Description = &description.String
		}
		if refType.Valid {
			t.ReferenceType = &refType.String
		}
		if refID.Valid {
			t.ReferenceID = &refID.String
		}
		if performedBy.Valid {
			t.PerformedByUserID = &performedBy.String
		}

		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	return &TransactionsListResponse{
		Transactions: transactions,
		Pagination: PaginationMeta{
			Page:       page,
			PerPage:    limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// TopUp adds funds to wallet via Stripe
func (s *WalletService) TopUp(req TopUpRequest, organizationID string, userID string) (*TopUpResponse, error) {
	// Auto-generate idempotency key if not provided
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = fmt.Sprintf("topup_%s_%s", organizationID, generateRandomID(16))
	}

	// Get wallet
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Verify wallet is active
	if wallet.Status != "active" {
		return nil, fmt.Errorf("wallet is %s and cannot accept top-ups", wallet.Status)
	}

	// Verify amount constraints
	if req.Amount < wallet.MinTopUpAmount {
		return nil, fmt.Errorf("amount must be at least %.2f %s", wallet.MinTopUpAmount, "CAD")
	}
	if wallet.MaxTopUpAmount.Valid && req.Amount > wallet.MaxTopUpAmount.Float64 {
		return nil, fmt.Errorf("amount cannot exceed %.2f %s", wallet.MaxTopUpAmount.Float64, "CAD")
	}

	// Create Stripe Payment Intent (MOCK for now)
	stripeResp, err := s.createStripePayment(req.Amount, req.PaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("stripe payment failed: %w", err)
	}

	if stripeResp.Status != "succeeded" {
		return nil, fmt.Errorf("stripe payment failed with status: %s", stripeResp.Status)
	}

	// Process top-up with retries (optimistic locking)
	var topUpResp *TopUpResponse
	for attempt := 0; attempt < MaxRetries; attempt++ {
		topUpResp, err = s.processTopUp(wallet, stripeResp, req.IdempotencyKey, userID)
		if err == nil {
			break
		}
		if !isVersionMismatchError(err) {
			return nil, err
		}
		// Version mismatch - retry
		time.Sleep(time.Duration(RetryDelayMS*(attempt+1)) * time.Millisecond)
		// Reload wallet
		wallet, err = s.getWallet(organizationID)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", MaxRetries, err)
	}

	return topUpResp, nil
}

// Refund processes a shipment refund to wallet
func (s *WalletService) Refund(req RefundRequest, organizationID string, userID string) (*RefundResponse, error) {
	// Auto-generate idempotency key if not provided
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = fmt.Sprintf("refund_%s_%s", req.ShipmentID, generateRandomID(16))
	}

	// Get wallet
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Verify shipment exists and belongs to organization
	shipment, err := s.getShipment(req.ShipmentID, organizationID)
	if err != nil {
		return nil, err
	}

	// Verify shipment is refundable
	if shipment.PaymentStatus == "refunded" {
		return nil, fmt.Errorf("shipment has already been refunded")
	}

	// Process refund with retries
	var refundResp *RefundResponse
	for attempt := 0; attempt < MaxRetries; attempt++ {
		refundResp, err = s.processRefund(wallet, shipment, req, userID)
		if err == nil {
			break
		}
		if !isVersionMismatchError(err) {
			return nil, err
		}
		// Version mismatch - retry
		time.Sleep(time.Duration(RetryDelayMS*(attempt+1)) * time.Millisecond)
		// Reload wallet
		wallet, err = s.getWallet(organizationID)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", MaxRetries, err)
	}

	return refundResp, nil
}

// LockWallet locks a wallet (admin only)
func (s *WalletService) LockWallet(organizationID string, userID string) (*WalletStatusResponse, error) {
	return s.updateWalletStatus(organizationID, "locked", userID)
}

// UnlockWallet unlocks a wallet (admin only)
func (s *WalletService) UnlockWallet(organizationID string, userID string) (*WalletStatusResponse, error) {
	return s.updateWalletStatus(organizationID, "active", userID)
}

// =====================================================
// PRIVATE HELPER METHODS
// =====================================================

func (s *WalletService) getWallet(organizationID string) (*walletInfo, error) {
	query := `
		SELECT
			id, wallet_number, organization_id,
			balance, held_balance, status, version,
			min_topup_amount, max_topup_amount, max_balance,
			created_at, updated_at
		FROM wallets
		WHERE organization_id = $1
	`

	var wallet walletInfo
	err := s.db.DB.QueryRow(query, organizationID).Scan(
		&wallet.ID, &wallet.WalletNumber, &wallet.OrganizationID,
		&wallet.Balance, &wallet.HeldBalance, &wallet.Status, &wallet.Version,
		&wallet.MinTopUpAmount, &wallet.MaxTopUpAmount, &wallet.MaxBalance,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet: %w", err)
	}

	return &wallet, nil
}

// getCarrierRefundInProgress calculates total amount of carrier refunds in progress
func (s *WalletService) getCarrierRefundInProgress(walletID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(requested_amount), 0)
		FROM carrier_refunds
		WHERE wallet_id = $1
		  AND status IN ('requested', 'processing')
	`

	var amount float64
	err := s.db.DB.QueryRow(query, walletID).Scan(&amount)
	if err != nil {
		return 0, fmt.Errorf("failed to get carrier refund in progress: %w", err)
	}

	return amount, nil
}

func (s *WalletService) getTransactions(walletID string, page, pageSize int) ([]TransactionResponse, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT
			id, transaction_number, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			performed_by_user_id, is_system_generated, created_at
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.DB.Query(query, walletID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]TransactionResponse, 0)
	for rows.Next() {
		var t TransactionResponse
		var description, refType, refID, performedBy sql.NullString

		if err := rows.Scan(
			&t.ID, &t.TransactionNumber, &t.Direction, &t.TxType,
			&t.Amount, &t.Currency, &t.BalanceBefore, &t.BalanceAfter,
			&description, &refType, &refID,
			&performedBy, &t.IsSystemGenerated, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if description.Valid {
			t.Description = &description.String
		}
		if refType.Valid {
			t.ReferenceType = &refType.String
		}
		if refID.Valid {
			t.ReferenceID = &refID.String
		}
		if performedBy.Valid {
			t.PerformedByUserID = &performedBy.String
		}

		transactions = append(transactions, t)
	}

	return transactions, nil
}

func (s *WalletService) processTopUp(
	wallet *walletInfo,
	stripeResp *stripePaymentResponse,
	idempotencyKey string,
	userID string,
) (*TopUpResponse, error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate numbers
	topUpNumber, err := generateNumber(TopUpNumberPrefix, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate topup number: %w", err)
	}

	transactionNumber, err := generateNumber(TransactionNumberPrefix, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction number: %w", err)
	}

	// Create wallet_topup record
	topUpQuery := `
		INSERT INTO wallet_topups (
			topup_number, wallet_id, organization_id,
			amount, currency, status,
			stripe_payment_intent_id, stripe_charge_id,
			stripe_payment_method_id, stripe_receipt_url,
			refundable_amount, created_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var topUpID string
	err = tx.QueryRow(
		topUpQuery,
		topUpNumber, wallet.ID, wallet.OrganizationID,
		stripeResp.Amount, stripeResp.Currency, stripeResp.Status,
		stripeResp.PaymentIntentID, stripeResp.ChargeID,
		stripeResp.PaymentMethodID, stripeResp.ReceiptURL,
		stripeResp.Amount, userID,
	).Scan(&topUpID)

	if err != nil {
		return nil, fmt.Errorf("failed to create topup record: %w", err)
	}

	// Create wallet_transaction (immutable ledger)
	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore + stripeResp.Amount

	txnQuery := `
		INSERT INTO wallet_transactions (
			transaction_number, wallet_id, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			stripe_payment_intent_id, stripe_charge_id,
			performed_by_user_id, is_system_generated, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at
	`

	description := fmt.Sprintf("Wallet top-up via Stripe - %s", stripeResp.Currency)
	var transactionID string
	var createdAt time.Time

	err = tx.QueryRow(
		txnQuery,
		transactionNumber, wallet.ID, "credit", "topup",
		stripeResp.Amount, stripeResp.Currency, balanceBefore, balanceAfter,
		description, "topup", topUpID,
		stripeResp.PaymentIntentID, stripeResp.ChargeID,
		userID, false, idempotencyKey,
	).Scan(&transactionID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update wallet_topup with transaction link
	updateTopUpQuery := `
		UPDATE wallet_topups
		SET wallet_transaction_id = $1
		WHERE id = $2
	`
	if _, err := tx.Exec(updateTopUpQuery, transactionID, topUpID); err != nil {
		return nil, fmt.Errorf("failed to update topup: %w", err)
	}

	// Update wallet balance with optimistic locking
	updateWalletQuery := `
		UPDATE wallets
		SET balance = balance + $1,
		    version = version + 1,
		    updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := tx.Exec(updateWalletQuery, stripeResp.Amount, wallet.ID, wallet.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("wallet version mismatch (concurrent update detected)")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build response
	resp := &TopUpResponse{
		WalletID:          wallet.ID,
		TransactionID:     transactionID,
		TransactionNumber: transactionNumber,
		TopUpNumber:       topUpNumber,
		Amount:            stripeResp.Amount,
		Currency:          stripeResp.Currency,
		NewBalance:        balanceAfter,
		Status:            stripeResp.Status,
		CreatedAt:         createdAt,
	}

	if stripeResp.PaymentIntentID != "" {
		resp.StripePaymentIntentID = &stripeResp.PaymentIntentID
	}
	if stripeResp.ReceiptURL != "" {
		resp.StripeReceiptURL = &stripeResp.ReceiptURL
	}

	return resp, nil
}

type shipmentInfo struct {
	ID            string
	ShipmentNumber string
	PaymentStatus string
	BaseAmount    float64
	Currency      string
}

func (s *WalletService) getShipment(shipmentID string, organizationID string) (*shipmentInfo, error) {
	query := `
		SELECT id, shipment_number, payment_status, base_amount, currency
		FROM shipments
		WHERE id = $1 AND organization_id = $2
	`

	var shipment shipmentInfo
	err := s.db.DB.QueryRow(query, shipmentID, organizationID).Scan(
		&shipment.ID, &shipment.ShipmentNumber, &shipment.PaymentStatus,
		&shipment.BaseAmount, &shipment.Currency,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("shipment not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query shipment: %w", err)
	}

	return &shipment, nil
}

func (s *WalletService) processRefund(
	wallet *walletInfo,
	shipment *shipmentInfo,
	req RefundRequest,
	userID string,
) (*RefundResponse, error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate transaction number
	transactionNumber, err := generateNumber(TransactionNumberPrefix, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction number: %w", err)
	}

	// Create wallet_transaction (credit)
	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore + req.Amount

	txnQuery := `
		INSERT INTO wallet_transactions (
			transaction_number, wallet_id, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			performed_by_user_id, is_system_generated, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	description := fmt.Sprintf("Refund for shipment %s - %s", shipment.ShipmentNumber, req.Reason)
	var transactionID string
	var createdAt time.Time

	err = tx.QueryRow(
		txnQuery,
		transactionNumber, wallet.ID, "credit", "shipment_refund",
		req.Amount, shipment.Currency, balanceBefore, balanceAfter,
		description, "shipment", shipment.ID,
		userID, true, req.IdempotencyKey,
	).Scan(&transactionID, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update wallet balance with optimistic locking
	updateWalletQuery := `
		UPDATE wallets
		SET balance = balance + $1,
		    version = version + 1,
		    updated_at = NOW()
		WHERE id = $2 AND version = $3
	`

	result, err := tx.Exec(updateWalletQuery, req.Amount, wallet.ID, wallet.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("wallet version mismatch (concurrent update detected)")
	}

	// Update shipment payment_status
	updateShipmentQuery := `
		UPDATE shipments
		SET payment_status = 'refunded',
		    updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.Exec(updateShipmentQuery, shipment.ID); err != nil {
		return nil, fmt.Errorf("failed to update shipment: %w", err)
	}

	// Update invoice (set to 'unpaid' since payment was refunded)
	// Valid enum values: unpaid, partial, paid, overpaid
	updateInvoiceQuery := `
		UPDATE invoices
		SET payment_status = 'unpaid',
		    updated_at = NOW()
		WHERE shipment_id = $1
	`
	if _, err := tx.Exec(updateInvoiceQuery, shipment.ID); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build response
	resp := &RefundResponse{
		WalletID:          wallet.ID,
		TransactionID:     transactionID,
		TransactionNumber: transactionNumber,
		ShipmentID:        shipment.ID,
		Amount:            req.Amount,
		Currency:          shipment.Currency,
		NewBalance:        balanceAfter,
		Reason:            req.Reason,
		CreatedAt:         createdAt,
	}

	return resp, nil
}

func (s *WalletService) updateWalletStatus(organizationID string, status string, userID string) (*WalletStatusResponse, error) {
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE wallets
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`

	var updatedAt time.Time
	err = s.db.DB.QueryRow(query, status, wallet.ID).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet status: %w", err)
	}

	return &WalletStatusResponse{
		WalletID:     wallet.ID,
		WalletNumber: wallet.WalletNumber,
		Status:       status,
		UpdatedAt:    updatedAt,
		UpdatedBy:    userID,
	}, nil
}

func (s *WalletService) createStripePayment(amount float64, paymentMethodID string) (*stripePaymentResponse, error) {
	// TODO: Replace with actual Stripe integration
	// Mock implementation for now

	// Simulate Stripe API call
	paymentIntentID := fmt.Sprintf("pi_%s", generateRandomID(24))
	chargeID := fmt.Sprintf("ch_%s", generateRandomID(24))
	receiptURL := fmt.Sprintf("https://pay.stripe.com/receipts/%s", generateRandomID(32))

	return &stripePaymentResponse{
		PaymentIntentID: paymentIntentID,
		ChargeID:        chargeID,
		PaymentMethodID: paymentMethodID,
		ReceiptURL:      receiptURL,
		Status:          "succeeded",
		Amount:          amount,
		Currency:        "CAD",
	}, nil
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func generateNumber(prefix string, length int) (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed ambiguous chars
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return prefix + string(b), nil
}

func generateRandomID(length int) string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func isVersionMismatchError(err error) bool {
	return err != nil && (err.Error() == "wallet version mismatch (concurrent update detected)")
}

// ExportTransactions retrieves all transactions for export with optional filters
func (s *WalletService) ExportTransactions(organizationID string, req *ExportTransactionsRequest) (*ExportTransactionsResponse, error) {
	// Get wallet
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Build query with filters
	query := `
		SELECT
			transaction_number, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id, created_at
		FROM wallet_transactions
		WHERE wallet_id = $1
	`
	args := []interface{}{wallet.ID}
	argIndex := 2

	// Add date filters
	if req.FromDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, req.FromDate+" 00:00:00")
		argIndex++
	}
	if req.ToDate != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, req.ToDate+" 23:59:59")
		argIndex++
	}

	// Add direction filter
	if req.Direction != "" {
		query += fmt.Sprintf(" AND direction = $%d", argIndex)
		args = append(args, req.Direction)
		argIndex++
	}

	// Add tx_type filter
	if req.TxType != "" {
		query += fmt.Sprintf(" AND tx_type = $%d", argIndex)
		args = append(args, req.TxType)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]ExportTransactionRow, 0)
	for rows.Next() {
		var t ExportTransactionRow
		var description, refType, refID sql.NullString
		var createdAt time.Time

		if err := rows.Scan(
			&t.TransactionNumber, &t.Direction, &t.Type,
			&t.Amount, &t.Currency, &t.BalanceBefore, &t.BalanceAfter,
			&description, &refType, &refID, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		t.Date = createdAt.Format("2006-01-02")
		t.Time = createdAt.Format("15:04:05")

		if description.Valid {
			t.Description = description.String
		}
		if refType.Valid {
			t.ReferenceType = refType.String
		}
		if refID.Valid {
			t.ReferenceID = refID.String
		}

		transactions = append(transactions, t)
	}

	resp := &ExportTransactionsResponse{
		WalletNumber:   wallet.WalletNumber,
		OrganizationID: wallet.OrganizationID,
		ExportedAt:     time.Now().Format(time.RFC3339),
		FromDate:       req.FromDate,
		ToDate:         req.ToDate,
		TotalRecords:   len(transactions),
		Transactions:   transactions,
	}

	return resp, nil
}

// GenerateCSV generates CSV content from export data
func (s *WalletService) GenerateCSV(data *ExportTransactionsResponse) string {
	var sb strings.Builder

	// CSV Header
	sb.WriteString("Transaction Number,Date,Time,Direction,Type,Amount,Currency,Balance Before,Balance After,Description,Reference Type,Reference ID\n")

	// CSV Rows
	for _, t := range data.Transactions {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,%s,%.2f,%.2f,\"%s\",%s,%s\n",
			t.TransactionNumber,
			t.Date,
			t.Time,
			t.Direction,
			t.Type,
			t.Amount,
			t.Currency,
			t.BalanceBefore,
			t.BalanceAfter,
			escapeCSV(t.Description),
			t.ReferenceType,
			t.ReferenceID,
		))
	}

	return sb.String()
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(s string) string {
	// Replace double quotes with two double quotes
	return strings.ReplaceAll(s, "\"", "\"\"")
}

// ListCarrierRefunds retrieves paginated carrier refunds with filters
func (s *WalletService) ListCarrierRefunds(organizationID string, req ListCarrierRefundsRequest) (*CarrierRefundsListResponse, error) {
	// Get wallet
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build count query
	countQuery := `SELECT COUNT(*) FROM carrier_refunds WHERE organization_id = $1`
	countArgs := []interface{}{organizationID}
	argIndex := 2

	// Build select query
	selectQuery := `
		SELECT
			cr.id, cr.refund_number, cr.organization_id, cr.wallet_id,
			cr.shipment_id, cr.invoice_id, cr.carrier_id,
			cr.carrier_tracking_number, cr.carrier_refund_reference,
			cr.requested_amount, cr.approved_amount, cr.currency, cr.status,
			cr.refund_reason, cr.reason_notes,
			cr.requested_at, cr.carrier_acknowledged_at, cr.completed_at,
			cr.expected_completion_date, cr.wallet_transaction_id,
			cr.requested_by_user_id, cr.processed_by_user_id,
			cr.carrier_response_code, cr.carrier_response_message,
			cr.notes, cr.created_at, cr.updated_at,
			s.shipment_number, c.name as carrier_name
		FROM carrier_refunds cr
		LEFT JOIN shipments s ON cr.shipment_id = s.id
		LEFT JOIN carriers c ON cr.carrier_id = c.id
		WHERE cr.organization_id = $1
	`
	selectArgs := []interface{}{organizationID}

	// Add filters
	if req.RefundNumber != nil && *req.RefundNumber != "" {
		countQuery += fmt.Sprintf(" AND refund_number ILIKE $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.refund_number ILIKE $%d", argIndex)
		searchTerm := "%" + *req.RefundNumber + "%"
		countArgs = append(countArgs, searchTerm)
		selectArgs = append(selectArgs, searchTerm)
		argIndex++
	}

	if req.ShipmentID != nil && *req.ShipmentID != "" {
		countQuery += fmt.Sprintf(" AND shipment_id = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.shipment_id = $%d", argIndex)
		countArgs = append(countArgs, *req.ShipmentID)
		selectArgs = append(selectArgs, *req.ShipmentID)
		argIndex++
	}

	if req.CarrierID != nil && *req.CarrierID != "" {
		countQuery += fmt.Sprintf(" AND carrier_id = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.carrier_id = $%d", argIndex)
		countArgs = append(countArgs, *req.CarrierID)
		selectArgs = append(selectArgs, *req.CarrierID)
		argIndex++
	}

	if req.Status != nil && *req.Status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.status = $%d", argIndex)
		countArgs = append(countArgs, *req.Status)
		selectArgs = append(selectArgs, *req.Status)
		argIndex++
	}

	if req.RefundReason != nil && *req.RefundReason != "" {
		countQuery += fmt.Sprintf(" AND refund_reason = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.refund_reason = $%d", argIndex)
		countArgs = append(countArgs, *req.RefundReason)
		selectArgs = append(selectArgs, *req.RefundReason)
		argIndex++
	}

	if req.FromDate != nil && *req.FromDate != "" {
		countQuery += fmt.Sprintf(" AND requested_at >= $%d::date", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.requested_at >= $%d::date", argIndex)
		countArgs = append(countArgs, *req.FromDate)
		selectArgs = append(selectArgs, *req.FromDate)
		argIndex++
	}

	if req.ToDate != nil && *req.ToDate != "" {
		countQuery += fmt.Sprintf(" AND requested_at < ($%d::date + INTERVAL '1 day')", argIndex)
		selectQuery += fmt.Sprintf(" AND cr.requested_at < ($%d::date + INTERVAL '1 day')", argIndex)
		countArgs = append(countArgs, *req.ToDate)
		selectArgs = append(selectArgs, *req.ToDate)
		argIndex++
	}

	// Get total count
	var total int
	err = s.db.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count carrier refunds: %w", err)
	}

	// Add sorting
	sortBy := "cr.requested_at"
	if req.SortBy != nil && *req.SortBy != "" {
		switch *req.SortBy {
		case "requested_at":
			sortBy = "cr.requested_at"
		case "requested_amount":
			sortBy = "cr.requested_amount"
		case "status":
			sortBy = "cr.status"
		}
	}
	sortOrder := "DESC"
	if req.SortOrder != nil && *req.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	selectQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	selectArgs = append(selectArgs, limit, offset)

	// Execute query
	rows, err := s.db.DB.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query carrier refunds: %w", err)
	}
	defer rows.Close()

	// Parse results
	refunds := make([]CarrierRefundResponse, 0)
	for rows.Next() {
		var r CarrierRefundResponse
		var invoiceID, carrierTrackingNumber, carrierRefundReference sql.NullString
		var approvedAmount sql.NullFloat64
		var reasonNotes, walletTransactionID, requestedByUserID, processedByUserID sql.NullString
		var carrierResponseCode, carrierResponseMessage, notes sql.NullString
		var carrierAcknowledgedAt, completedAt, expectedCompletionDate sql.NullTime
		var shipmentNumber, carrierName sql.NullString
		var walletID string

		if err := rows.Scan(
			&r.ID, &r.RefundNumber, &r.OrganizationID, &walletID,
			&r.ShipmentID, &invoiceID, &r.CarrierID,
			&carrierTrackingNumber, &carrierRefundReference,
			&r.RequestedAmount, &approvedAmount, &r.Currency, &r.Status,
			&r.RefundReason, &reasonNotes,
			&r.RequestedAt, &carrierAcknowledgedAt, &completedAt,
			&expectedCompletionDate, &walletTransactionID,
			&requestedByUserID, &processedByUserID,
			&carrierResponseCode, &carrierResponseMessage,
			&notes, &r.CreatedAt, &r.UpdatedAt,
			&shipmentNumber, &carrierName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan carrier refund: %w", err)
		}

		// Handle nullable fields
		if carrierTrackingNumber.Valid {
			r.CarrierTrackingNumber = &carrierTrackingNumber.String
		}
		if carrierRefundReference.Valid {
			r.CarrierRefundReference = &carrierRefundReference.String
		}
		if approvedAmount.Valid {
			r.ApprovedAmount = &approvedAmount.Float64
		}
		if reasonNotes.Valid {
			r.ReasonNotes = &reasonNotes.String
		}
		if carrierAcknowledgedAt.Valid {
			r.CarrierAcknowledgedAt = &carrierAcknowledgedAt.Time
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		if expectedCompletionDate.Valid {
			dateStr := expectedCompletionDate.Time.Format("2006-01-02")
			r.ExpectedCompletionDate = &dateStr
		}
		if walletTransactionID.Valid {
			r.WalletTransactionID = &walletTransactionID.String
		}
		if requestedByUserID.Valid {
			r.RequestedByUserID = &requestedByUserID.String
		}
		if carrierResponseCode.Valid {
			r.CarrierResponseCode = &carrierResponseCode.String
		}
		if carrierResponseMessage.Valid {
			r.CarrierResponseMessage = &carrierResponseMessage.String
		}
		if notes.Valid {
			r.Notes = &notes.String
		}
		if shipmentNumber.Valid {
			r.ShipmentNumber = &shipmentNumber.String
		}
		if carrierName.Valid {
			r.CarrierName = &carrierName.String
		}

		refunds = append(refunds, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating carrier refunds: %w", err)
	}

	// Get summary statistics
	summary, err := s.getCarrierRefundSummary(wallet.ID)
	if err != nil {
		// Don't fail, just use empty summary
		summary = &CarrierRefundSummary{Currency: "CAD"}
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	return &CarrierRefundsListResponse{
		Refunds: refunds,
		Summary: *summary,
		Pagination: PaginationMeta{
			Page:       page,
			PerPage:    limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// getCarrierRefundSummary retrieves summary statistics for carrier refunds
func (s *WalletService) getCarrierRefundSummary(walletID string) (*CarrierRefundSummary, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'requested') as total_requested,
			COUNT(*) FILTER (WHERE status = 'processing') as total_processing,
			COUNT(*) FILTER (WHERE status = 'completed') as total_completed,
			COUNT(*) FILTER (WHERE status = 'rejected') as total_rejected,
			COUNT(*) FILTER (WHERE status = 'failed') as total_failed,
			COUNT(*) FILTER (WHERE status = 'cancelled') as total_cancelled,
			COALESCE(SUM(requested_amount) FILTER (WHERE status IN ('requested', 'processing')), 0) as amount_pending,
			COALESCE(SUM(COALESCE(approved_amount, requested_amount)) FILTER (WHERE status = 'completed'), 0) as amount_completed
		FROM carrier_refunds
		WHERE wallet_id = $1
	`

	var summary CarrierRefundSummary
	err := s.db.DB.QueryRow(query, walletID).Scan(
		&summary.TotalRequested,
		&summary.TotalProcessing,
		&summary.TotalCompleted,
		&summary.TotalRejected,
		&summary.TotalFailed,
		&summary.TotalCancelled,
		&summary.AmountPending,
		&summary.AmountCompleted,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get carrier refund summary: %w", err)
	}

	summary.Currency = "CAD"
	return &summary, nil
}

// GetWalletOverview retrieves wallet overview with financial summary
func (s *WalletService) GetWalletOverview(organizationID string, period string) (*WalletOverviewResponse, error) {
	// Get wallet info
	wallet, err := s.getWallet(organizationID)
	if err != nil {
		return nil, err
	}

	// Determine date range based on period
	now := time.Now()
	var startDate, endDate time.Time
	var periodLabel string

	switch period {
	case "this_week":
		// Start of this week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday is 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		endDate = now
		periodLabel = "This Week"
	case "this_year":
		// Start of this year
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = now
		periodLabel = "This Year"
	default:
		// Default to this_month
		period = "this_month"
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = now
		periodLabel = "This Month"
	}

	// Get total in-transit (pending wallet holds)
	var totalInTransit float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_holds
		WHERE wallet_id = $1 AND status IN ('pending', 'active')
	`, wallet.ID).Scan(&totalInTransit)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get in-transit amount: %w", err)
	}

	// Get total deposits (credits) in period
	var totalDeposits float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_transactions
		WHERE wallet_id = $1
		  AND direction = 'credit'
		  AND created_at >= $2
		  AND created_at <= $3
	`, wallet.ID, startDate, endDate).Scan(&totalDeposits)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get total deposits: %w", err)
	}

	// Get total outflow (debits) in period
	var totalOutflow float64
	err = s.db.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM wallet_transactions
		WHERE wallet_id = $1
		  AND direction = 'debit'
		  AND created_at >= $2
		  AND created_at <= $3
	`, wallet.ID, startDate, endDate).Scan(&totalOutflow)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get total outflow: %w", err)
	}

	// Calculate net movement
	netMovement := totalDeposits - totalOutflow

	resp := &WalletOverviewResponse{
		AvailableBalance: wallet.Balance,
		TotalInTransit:   totalInTransit,
		Reserved:         wallet.HeldBalance,
		TotalDeposits:    totalDeposits,
		TotalOutflow:     totalOutflow,
		NetMovement:      netMovement,
		Period: WalletOverviewPeriod{
			Type:      period,
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
			Label:     periodLabel,
		},
		Currency:    "CAD",
		GeneratedAt: now,
	}

	return resp, nil
}
