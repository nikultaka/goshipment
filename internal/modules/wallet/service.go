package wallet

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
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

	// Build response
	resp := &WalletResponse{
		ID:             wallet.ID,
		WalletNumber:   wallet.WalletNumber,
		OrganizationID: wallet.OrganizationID,
		Balance:        wallet.Balance,
		HeldBalance:    wallet.HeldBalance,
		TotalBalance:   wallet.Balance + wallet.HeldBalance,
		Status:         wallet.Status,
		MinTopUpAmount: wallet.MinTopUpAmount,
		CreatedAt:      wallet.CreatedAt,
		UpdatedAt:      wallet.UpdatedAt,
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

// TopUp adds funds to wallet via Stripe
func (s *WalletService) TopUp(req TopUpRequest, organizationID string, userID string) (*TopUpResponse, error) {
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
