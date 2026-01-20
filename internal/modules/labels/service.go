package labels

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"gogin/internal/clients"
)

const (
	// TransactionNumberPrefix is the prefix for transaction numbers
	TransactionNumberPrefix = "TRX-"
)

// LabelService handles label purchase business logic
type LabelService struct {
	db *clients.Database
}

// NewLabelService creates a new label service
func NewLabelService(db *clients.Database) *LabelService {
	return &LabelService{
		db: db,
	}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *LabelService) GetOrganizationIDFromUser(userID string) (string, error) {
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

// shipmentInfo holds shipment details for label purchase
type shipmentInfo struct {
	ID                 string
	ShipmentNumber     string
	OrganizationID     string
	ShipmentStatus     string
	PaymentStatus      string
	BaseAmount         float64
	Currency           string
	CreatedByUserID    string
	CarrierID          string
	ServicelevelCode   string
	ServicelevelName   string
}

// packageInfo holds package details for label purchase
type packageInfo struct {
	ID            string
	PackageNumber string
	PackageIndex  int
	PackageName   sql.NullString
	PackageStatus string
	Length        float64
	Width         float64
	Height        float64
	DimensionUnit string
	Weight        float64
	WeightUnit    string
}

// walletHoldInfo holds wallet hold details
type walletHoldInfo struct {
	ID          string
	HoldNumber  string
	WalletID    string
	Amount      float64
	Currency    string
	Status      string
}

// walletInfo holds wallet details
type walletInfo struct {
	ID            string
	WalletNumber  string
	Balance       float64
	HeldBalance   float64
	Status        string
	Version       int64
}

// carrierLabelResponse represents mock carrier API response
type carrierLabelResponse struct {
	ParcelID           string
	LabelTransactionID string
	LabelURL           string
	LabelFileType      string
	TrackingNumber     string
	TrackingURL        string
}

// PurchaseLabels purchases shipping labels for all packages in a shipment
func (s *LabelService) PurchaseLabels(
	req PurchaseLabelsRequest,
	organizationID string,
	userID string,
) (*PurchaseLabelsResponse, error) {

	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Lock shipment for update and verify
	shipment, err := s.getShipmentForUpdate(tx, req.ShipmentID, organizationID)
	if err != nil {
		return nil, err
	}

	// 2. Verify shipment is in pre-label state
	if err := s.verifyShipmentState(shipment); err != nil {
		return nil, err
	}

	// 3. Get all packages for this shipment
	packages, err := s.getPackagesForUpdate(tx, req.ShipmentID)
	if err != nil {
		return nil, err
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages found for shipment")
	}

	// 4. Verify all packages are in pending state
	for _, pkg := range packages {
		if pkg.PackageStatus != "pending" {
			return nil, fmt.Errorf("package %s is not in pending state (current: %s)", pkg.PackageNumber, pkg.PackageStatus)
		}
	}

	// 5. Handle payment (wallet or carrier billing)
	var walletTransactionID *string
	var transactionNumber *string
	useCarrierBilling := req.UseCarrierBilling != nil && *req.UseCarrierBilling

	if !useCarrierBilling {
		// Wallet payment - verify hold and capture it
		hold, err := s.getWalletHoldForUpdate(tx, req.ShipmentID)
		if err != nil {
			return nil, fmt.Errorf("wallet hold not found: %w", err)
		}

		if hold.Status != "pending" {
			return nil, fmt.Errorf("wallet hold is not in pending state (current: %s)", hold.Status)
		}

		// Get wallet with optimistic lock
		wallet, err := s.getWalletForUpdate(tx, hold.WalletID)
		if err != nil {
			return nil, fmt.Errorf("failed to get wallet: %w", err)
		}

		if wallet.Status != "active" {
			return nil, fmt.Errorf("wallet is not active (current: %s)", wallet.Status)
		}

		// Capture hold
		txnID, txnNum, err := s.captureWalletHold(tx, hold, wallet, shipment, userID, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to capture wallet hold: %w", err)
		}
		walletTransactionID = &txnID
		transactionNumber = &txnNum
	}

	// 6. Purchase labels for each package (call carrier API)
	packageLabels := make([]PackageLabelInfo, 0, len(packages))
	now := time.Now().UTC()

	for _, pkg := range packages {
		// Call carrier API to purchase label (MOCK for now)
		labelResp, err := s.purchaseLabelFromCarrier(shipment, pkg)
		if err != nil {
			return nil, fmt.Errorf("failed to purchase label for package %s: %w", pkg.PackageNumber, err)
		}

		// Update package with label info
		if err := s.updatePackageLabel(tx, pkg.ID, labelResp, now); err != nil {
			return nil, fmt.Errorf("failed to update package label: %w", err)
		}

		// Create package-level event
		if err := s.createPackageLabelEvent(tx, shipment, pkg.ID, userID); err != nil {
			return nil, fmt.Errorf("failed to create package event: %w", err)
		}

		// Build response
		var packageName *string
		if pkg.PackageName.Valid {
			packageName = &pkg.PackageName.String
		}
		labelPurchasedAt := now

		packageLabels = append(packageLabels, PackageLabelInfo{
			PackageID:                 pkg.ID,
			PackageNumber:             pkg.PackageNumber,
			PackageIndex:              pkg.PackageIndex,
			PackageName:               packageName,
			LabelURL:                  &labelResp.LabelURL,
			LabelFileType:             &labelResp.LabelFileType,
			TrackingNumber:            &labelResp.TrackingNumber,
			TrackingURL:               &labelResp.TrackingURL,
			CarrierParcelID:           &labelResp.ParcelID,
			CarrierLabelTransactionID: &labelResp.LabelTransactionID,
			PackageStatus:             "label_purchased",
			LabelPurchasedAt:          &labelPurchasedAt,
		})
	}

	// 7. Update shipment status
	newPaymentStatus := "paid"
	if useCarrierBilling {
		newPaymentStatus = "carrier_billed"
	}

	if err := s.updateShipmentStatus(tx, req.ShipmentID, walletTransactionID, newPaymentStatus); err != nil {
		return nil, fmt.Errorf("failed to update shipment status: %w", err)
	}

	// 8. Create shipment-level event
	if err := s.createShipmentLabelEvent(tx, shipment, userID); err != nil {
		return nil, fmt.Errorf("failed to create shipment event: %w", err)
	}

	// 9. Update invoice
	if err := s.updateInvoiceStatus(tx, req.ShipmentID, newPaymentStatus, now); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build response
	resp := &PurchaseLabelsResponse{
		ShipmentID:     shipment.ID,
		ShipmentNumber: shipment.ShipmentNumber,
		ShipmentStatus: "label_purchased",
		PaymentStatus:  newPaymentStatus,
		TotalAmount:    shipment.BaseAmount,
		Currency:       shipment.Currency,
		Packages:       packageLabels,
		PurchasedAt:    now,
		PurchasedBy:    userID,
	}

	if walletTransactionID != nil {
		resp.WalletTransactionID = walletTransactionID
		resp.TransactionNumber = transactionNumber
	}

	return resp, nil
}

// GetLabelInfo retrieves label and tracking info for a package
func (s *LabelService) GetLabelInfo(packageID string, organizationID string) (*GetLabelResponse, error) {
	query := `
		SELECT
			sp.id, sp.package_number, sp.package_index, sp.package_name,
			sp.shipment_id, s.shipment_number, s.shipment_status, s.organization_id,
			sp.label_url, sp.label_file_type, sp.tracking_number, sp.tracking_url,
			sp.carrier_parcel_id, sp.carrier_label_transaction_id,
			sp.length, sp.width, sp.height, sp.dimension_unit,
			sp.weight, sp.weight_unit, sp.declared_value, sp.declared_currency,
			sp.package_status, sp.label_purchased_at, sp.failure_reason
		FROM shipment_packages sp
		JOIN shipments s ON sp.shipment_id = s.id
		WHERE sp.id = $1 AND s.organization_id = $2
	`

	var resp GetLabelResponse
	var labelURL, labelFileType, trackingNumber, trackingURL sql.NullString
	var carrierParcelID, carrierLabelTransactionID sql.NullString
	var packageName sql.NullString
	var declaredValue sql.NullFloat64
	var declaredCurrency sql.NullString
	var labelPurchasedAt sql.NullTime
	var failureReason sql.NullString
	var shipmentID, shipmentNumber, shipmentStatus, orgID string

	err := s.db.DB.QueryRow(query, packageID, organizationID).Scan(
		&resp.PackageID, &resp.PackageNumber, &resp.PackageIndex, &packageName,
		&shipmentID, &shipmentNumber, &shipmentStatus, &orgID,
		&labelURL, &labelFileType, &trackingNumber, &trackingURL,
		&carrierParcelID, &carrierLabelTransactionID,
		&resp.Length, &resp.Width, &resp.Height, &resp.DimensionUnit,
		&resp.Weight, &resp.WeightUnit, &declaredValue, &declaredCurrency,
		&resp.PackageStatus, &labelPurchasedAt, &failureReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("package not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query package: %w", err)
	}

	// Set shipment context
	resp.ShipmentID = shipmentID
	resp.ShipmentNumber = shipmentNumber
	resp.ShipmentStatus = shipmentStatus

	// Set optional fields
	if packageName.Valid {
		resp.PackageName = &packageName.String
	}
	if labelURL.Valid {
		resp.LabelURL = &labelURL.String
	}
	if labelFileType.Valid {
		resp.LabelFileType = &labelFileType.String
	}
	if trackingNumber.Valid {
		resp.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		resp.TrackingURL = &trackingURL.String
	}
	if carrierParcelID.Valid {
		resp.CarrierParcelID = &carrierParcelID.String
	}
	if carrierLabelTransactionID.Valid {
		resp.CarrierLabelTransactionID = &carrierLabelTransactionID.String
	}
	if declaredValue.Valid {
		resp.DeclaredValue = &declaredValue.Float64
	}
	if declaredCurrency.Valid {
		resp.DeclaredCurrency = &declaredCurrency.String
	}
	if labelPurchasedAt.Valid {
		resp.LabelPurchasedAt = &labelPurchasedAt.Time
	}
	if failureReason.Valid {
		resp.FailureReason = &failureReason.String
	}

	return &resp, nil
}

// =====================================================
// PRIVATE HELPER METHODS
// =====================================================

func (s *LabelService) getShipmentForUpdate(tx *sql.Tx, shipmentID string, organizationID string) (*shipmentInfo, error) {
	query := `
		SELECT id, shipment_number, organization_id, shipment_status, payment_status,
		       base_amount, currency, created_by_user_id, carrier_id, servicelevel_code, servicelevel_name
		FROM shipments
		WHERE id = $1 AND organization_id = $2
		FOR UPDATE
	`

	var shipment shipmentInfo
	err := tx.QueryRow(query, shipmentID, organizationID).Scan(
		&shipment.ID, &shipment.ShipmentNumber, &shipment.OrganizationID,
		&shipment.ShipmentStatus, &shipment.PaymentStatus,
		&shipment.BaseAmount, &shipment.Currency,
		&shipment.CreatedByUserID, &shipment.CarrierID,
		&shipment.ServicelevelCode, &shipment.ServicelevelName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("shipment not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query shipment: %w", err)
	}

	return &shipment, nil
}

func (s *LabelService) verifyShipmentState(shipment *shipmentInfo) error {
	// Verify shipment is in pre-label state
	validStates := map[string]bool{
		"draft":            true,
		"awaiting_payment": true,
		"payment_failed":   true,
	}

	if !validStates[shipment.ShipmentStatus] {
		return fmt.Errorf("shipment is not in pre-label state (current: %s)", shipment.ShipmentStatus)
	}

	return nil
}

func (s *LabelService) getPackagesForUpdate(tx *sql.Tx, shipmentID string) ([]packageInfo, error) {
	query := `
		SELECT id, package_number, package_index, package_name, package_status,
		       length, width, height, dimension_unit, weight, weight_unit
		FROM shipment_packages
		WHERE shipment_id = $1
		ORDER BY package_index ASC
		FOR UPDATE
	`

	rows, err := tx.Query(query, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query packages: %w", err)
	}
	defer rows.Close()

	packages := make([]packageInfo, 0)
	for rows.Next() {
		var pkg packageInfo
		if err := rows.Scan(
			&pkg.ID, &pkg.PackageNumber, &pkg.PackageIndex, &pkg.PackageName,
			&pkg.PackageStatus, &pkg.Length, &pkg.Width, &pkg.Height,
			&pkg.DimensionUnit, &pkg.Weight, &pkg.WeightUnit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (s *LabelService) getWalletHoldForUpdate(tx *sql.Tx, shipmentID string) (*walletHoldInfo, error) {
	query := `
		SELECT id, hold_number, wallet_id, amount, currency, status
		FROM wallet_holds
		WHERE reference_type = 'shipment' AND reference_id = $1
		FOR UPDATE
	`

	var hold walletHoldInfo
	err := tx.QueryRow(query, shipmentID).Scan(
		&hold.ID, &hold.HoldNumber, &hold.WalletID,
		&hold.Amount, &hold.Currency, &hold.Status,
	)

	if err != nil {
		return nil, err
	}

	return &hold, nil
}

func (s *LabelService) getWalletForUpdate(tx *sql.Tx, walletID string) (*walletInfo, error) {
	query := `
		SELECT id, wallet_number, balance, held_balance, status, version
		FROM wallets
		WHERE id = $1
		FOR UPDATE
	`

	var wallet walletInfo
	err := tx.QueryRow(query, walletID).Scan(
		&wallet.ID, &wallet.WalletNumber, &wallet.Balance,
		&wallet.HeldBalance, &wallet.Status, &wallet.Version,
	)

	if err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (s *LabelService) captureWalletHold(
	tx *sql.Tx,
	hold *walletHoldInfo,
	wallet *walletInfo,
	shipment *shipmentInfo,
	userID string,
	idempotencyKey string,
) (string, string, error) {

	// Generate transaction number
	transactionNumber, err := s.generateTransactionNumber()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate transaction number: %w", err)
	}

	// Create wallet transaction
	txnQuery := `
		INSERT INTO wallet_transactions (
			transaction_number, wallet_id, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			performed_by_user_id, is_system_generated, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore - hold.Amount
	description := fmt.Sprintf("Label purchase for shipment %s", shipment.ShipmentNumber)

	var transactionID string
	err = tx.QueryRow(
		txnQuery,
		transactionNumber, wallet.ID, "debit", "shipment_payment",
		hold.Amount, hold.Currency, balanceBefore, balanceAfter,
		description, "shipment", shipment.ID,
		userID, true, idempotencyKey,
	).Scan(&transactionID)

	if err != nil {
		return "", "", fmt.Errorf("failed to create wallet transaction: %w", err)
	}

	// Update wallet balances
	walletQuery := `
		UPDATE wallets
		SET balance = balance - $1,
		    held_balance = held_balance - $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE id = $3 AND version = $4
	`

	result, err := tx.Exec(walletQuery, hold.Amount, hold.Amount, wallet.ID, wallet.Version)
	if err != nil {
		return "", "", fmt.Errorf("failed to update wallet: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "", "", fmt.Errorf("wallet version mismatch (concurrent update detected)")
	}

	// Update hold status
	holdQuery := `
		UPDATE wallet_holds
		SET status = 'captured',
		    captured_at = NOW(),
		    capture_transaction_id = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	if _, err := tx.Exec(holdQuery, transactionID, hold.ID); err != nil {
		return "", "", fmt.Errorf("failed to update hold: %w", err)
	}

	return transactionID, transactionNumber, nil
}

func (s *LabelService) purchaseLabelFromCarrier(shipment *shipmentInfo, pkg packageInfo) (*carrierLabelResponse, error) {
	// TODO: Replace with actual carrier API call (Shippo)
	// For now, return mock data

	// Generate mock IDs
	parcelID := fmt.Sprintf("parcel_%s", generateRandomID(8))
	labelTxnID := fmt.Sprintf("label_txn_%s", generateRandomID(8))
	trackingNumber := fmt.Sprintf("1Z999AA%d", time.Now().Unix()%100000000)

	return &carrierLabelResponse{
		ParcelID:           parcelID,
		LabelTransactionID: labelTxnID,
		LabelURL:           fmt.Sprintf("https://mock-cdn.example.com/labels/%s.pdf", labelTxnID),
		LabelFileType:      "pdf",
		TrackingNumber:     trackingNumber,
		TrackingURL:        fmt.Sprintf("https://track.example.com/track?num=%s", trackingNumber),
	}, nil
}

func (s *LabelService) updatePackageLabel(tx *sql.Tx, packageID string, label *carrierLabelResponse, now time.Time) error {
	query := `
		UPDATE shipment_packages
		SET carrier_parcel_id = $1,
		    carrier_label_transaction_id = $2,
		    label_url = $3,
		    label_file_type = $4,
		    tracking_number = $5,
		    tracking_url = $6,
		    package_status = 'label_purchased',
		    label_purchased_at = $7
		WHERE id = $8
	`

	_, err := tx.Exec(
		query,
		label.ParcelID, label.LabelTransactionID,
		label.LabelURL, label.LabelFileType,
		label.TrackingNumber, label.TrackingURL,
		now, packageID,
	)

	return err
}

func (s *LabelService) updateShipmentStatus(tx *sql.Tx, shipmentID string, walletTxnID *string, paymentStatus string) error {
	query := `
		UPDATE shipments
		SET shipment_status = 'label_purchased',
		    wallet_transaction_id = $1,
		    payment_status = $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := tx.Exec(query, walletTxnID, paymentStatus, shipmentID)
	return err
}

func (s *LabelService) createPackageLabelEvent(tx *sql.Tx, shipment *shipmentInfo, packageID string, userID string) error {
	query := `
		INSERT INTO shipment_events (
			shipment_id, organization_id, package_id,
			entity_type, event_key, source, severity,
			title, message, provider, occurred_at, actor_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), $11)
	`

	_, err := tx.Exec(
		query,
		shipment.ID, shipment.OrganizationID, packageID,
		"package", "package.label_purchased", "system", "info",
		"Label Purchased", "Shipping label purchased successfully",
		"shippo", userID,
	)

	return err
}

func (s *LabelService) createShipmentLabelEvent(tx *sql.Tx, shipment *shipmentInfo, userID string) error {
	query := `
		INSERT INTO shipment_events (
			shipment_id, organization_id,
			entity_type, event_key, source, severity,
			title, message, occurred_at, actor_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)
	`

	_, err := tx.Exec(
		query,
		shipment.ID, shipment.OrganizationID,
		"shipment", "shipment.label_purchased", "system", "info",
		"Labels Purchased", "All shipping labels purchased successfully",
		userID,
	)

	return err
}

func (s *LabelService) updateInvoiceStatus(tx *sql.Tx, shipmentID string, paymentStatus string, paidAt time.Time) error {
	// Determine payment method based on payment status
	paymentMethod := "wallet"
	if paymentStatus == "carrier_billed" {
		paymentMethod = "carrier_account"
	}

	query := `
		UPDATE invoices
		SET payment_status = $1,
		    payment_method = $2,
		    amount_paid = total_amount,
		    balance_due = 0,
		    refundable_amount = total_amount,
		    paid_at = $3,
		    updated_at = NOW()
		WHERE shipment_id = $4
	`

	_, err := tx.Exec(query, paymentStatus, paymentMethod, paidAt, shipmentID)
	return err
}

func (s *LabelService) generateTransactionNumber() (string, error) {
	return generateNumber(TransactionNumberPrefix, 8)
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
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}
