package shipments

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// SimpleShipmentService handles simplified shipment operations
type SimpleShipmentService struct {
	db *clients.Database
}

// NewSimpleShipmentService creates a new simple shipment service
func NewSimpleShipmentService(db *clients.Database) *SimpleShipmentService {
	return &SimpleShipmentService{db: db}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *SimpleShipmentService) GetOrganizationIDFromUser(userID string) (string, error) {
	query := `
		SELECT organization_id
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`

	var organizationID string
	err := s.db.QueryRow(query, userID).Scan(&organizationID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user is not a member of any organization")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get organization: %w", err)
	}

	return organizationID, nil
}

// CreateShipment creates a simplified shipment from an estimate
func (s *SimpleShipmentService) CreateShipment(
	req CreateShipmentRequest,
	organizationID string,
	userID string,
) (*models.Shipment, error) {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify estimate exists and get basic info including carrier rates
	var estimate struct {
		ID               string
		FromCountry      string
		FromRegionCode   sql.NullString
		FromCity         string
		FromPostalCode   string
		ToCountry        string
		ToRegionCode     sql.NullString
		ToCity           string
		ToPostalCode     string
		ExpiresAt        time.Time
		CarrierRatesJSON []byte
	}

	err = tx.QueryRow(`
		SELECT id, from_country, from_region_code, from_city, from_postal_code,
		       to_country, to_region_code, to_city, to_postal_code, expires_at,
		       COALESCE(carrier_rates_json, '[]'::jsonb)
		FROM shipment_estimates
		WHERE id = $1 AND (organization_id = $2 OR organization_id IS NULL) AND is_deleted = FALSE
	`, req.EstimateID, organizationID).Scan(
		&estimate.ID, &estimate.FromCountry, &estimate.FromRegionCode, &estimate.FromCity, &estimate.FromPostalCode,
		&estimate.ToCountry, &estimate.ToRegionCode, &estimate.ToCity, &estimate.ToPostalCode,
		&estimate.ExpiresAt, &estimate.CarrierRatesJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("estimate not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch estimate: %w", err)
	}

	// 2. Check if estimate is expired
	if time.Now().UTC().After(estimate.ExpiresAt) {
		return nil, errors.New("estimate has expired")
	}

	// 3. Get rate based on selected carrier and service code
	// First try to find rate in stored carrier_rates_json from estimate
	var carrierRates []models.CarrierRate
	if err := json.Unmarshal(estimate.CarrierRatesJSON, &carrierRates); err != nil {
		return nil, fmt.Errorf("failed to parse carrier rates: %w", err)
	}

	var selectedRate *models.CarrierRate
	for i, rate := range carrierRates {
		if rate.CarrierID == req.SelectedCarrierID && rate.ServiceCode == req.SelectedServiceCode {
			selectedRate = &carrierRates[i]
			break
		}
	}

	// Fallback to mock rates if not found in stored rates (for backwards compatibility)
	if selectedRate == nil {
		selectedRate = getMockRate(req.SelectedCarrierID, req.SelectedServiceCode)
	}

	if selectedRate == nil {
		return nil, errors.New("selected rate not found - invalid carrier or service code")
	}

	// 4. Look up real carrier UUID from carrier code
	var realCarrierID string
	err = tx.QueryRow(`
		SELECT id FROM carriers WHERE code = $1 AND is_active = true
	`, selectedRate.CarrierCode).Scan(&realCarrierID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("carrier %s not found", selectedRate.CarrierCode)
		}
		return nil, fmt.Errorf("failed to look up carrier: %w", err)
	}

	// 5. Create shipment record
	shipmentID := uuid.New().String()
	shipmentNumber := fmt.Sprintf("SIF-%s", generateRandomCode(8))
	now := time.Now().UTC()

	shipment := &models.Shipment{
		ID:                 shipmentID,
		ShipmentNumber:     shipmentNumber,
		OrganizationID:     organizationID,
		CreatedByUserID:    userID,
		ShipmentEstimateID: req.EstimateID,
		CarrierID:          realCarrierID,
		ServicelevelCode:   sql.NullString{String: req.SelectedServiceCode, Valid: true},
		ServicelevelName:   sql.NullString{String: selectedRate.ServiceName, Valid: true},
		FromCountry:        estimate.FromCountry,
		FromRegionCode:     estimate.FromRegionCode,
		FromCity:           estimate.FromCity,
		FromPostalCode:     estimate.FromPostalCode,
		FromAddressLine1:   "",
		ToCountry:          estimate.ToCountry,
		ToRegionCode:       estimate.ToRegionCode,
		ToCity:             estimate.ToCity,
		ToPostalCode:       estimate.ToPostalCode,
		ToAddressLine1:     "",
		TotalPackages:      1,
		TotalWeight:        0,
		WeightUnit:         "kg",
		BaseAmount:         selectedRate.BaseRate,
		TaxesAmount:        sql.NullFloat64{Float64: selectedRate.Taxes, Valid: selectedRate.Taxes > 0},
		Currency:           "CAD",
		ShipmentStatus:     models.ShipmentStatusDraft,
		PaymentStatus:      models.PaymentStatusPending,
		IsArchived:         false,
		IsDeleted:          false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Insert shipment
	_, err = tx.Exec(`
		INSERT INTO shipments (
			id, shipment_number, organization_id, created_by_user_id, shipment_estimate_id,
			carrier_id, servicelevel_code, servicelevel_name,
			from_country, from_region_code, from_city, from_postal_code, from_address_line1,
			to_country, to_region_code, to_city, to_postal_code, to_address_line1,
			total_packages, total_weight, weight_unit,
			base_amount, taxes_amount, currency,
			shipment_status, payment_status,
			is_archived, is_deleted,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21,
			$22, $23, $24,
			$25, $26,
			$27, $28,
			$29, $30
		)
	`,
		shipment.ID, shipment.ShipmentNumber, shipment.OrganizationID, shipment.CreatedByUserID, shipment.ShipmentEstimateID,
		shipment.CarrierID, shipment.ServicelevelCode, shipment.ServicelevelName,
		shipment.FromCountry, shipment.FromRegionCode, shipment.FromCity, shipment.FromPostalCode, shipment.FromAddressLine1,
		shipment.ToCountry, shipment.ToRegionCode, shipment.ToCity, shipment.ToPostalCode, shipment.ToAddressLine1,
		shipment.TotalPackages, shipment.TotalWeight, shipment.WeightUnit,
		shipment.BaseAmount, shipment.TaxesAmount, shipment.Currency,
		shipment.ShipmentStatus, shipment.PaymentStatus,
		shipment.IsArchived, shipment.IsDeleted,
		shipment.CreatedAt, shipment.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// 6. Copy packages from estimate to shipment
	// Extract alphanumeric part from shipment number (remove "SIF-" prefix)
	shipmentCode := shipmentNumber[4:] // Remove "SIF-" to get just "8D9SYOR9"

	_, err = tx.Exec(`
		INSERT INTO shipment_packages (
			id, shipment_id, package_number, package_index,
			package_name, length, width, height, dimension_unit,
			weight, weight_unit, declared_value, declared_currency,
			package_status, created_at
		)
		SELECT
			uuid_generate_v4(), $1::uuid, ('PKG-' || $2::text || '-' || LPAD(package_index::text, 2, '0')), package_index,
			package_name, length, width, height, dimension_unit,
			weight, weight_unit, declared_value, 'CAD',
			'pending', NOW()
		FROM shipment_packages
		WHERE shipment_estimate_id = $3::uuid
	`, shipmentID, shipmentCode, req.EstimateID)
	if err != nil {
		return nil, fmt.Errorf("failed to copy packages: %w", err)
	}

	// 7. Create wallet hold for payment
	// Get organization's wallet
	var walletID string
	err = tx.QueryRow(`
		SELECT id FROM wallets WHERE organization_id = $1 AND status = 'active'
	`, organizationID).Scan(&walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create wallet hold with shipment as reference
	holdNumber := fmt.Sprintf("HLD-%s", generateRandomCode(8))
	holdAmount := selectedRate.TotalAmount
	expiresAt := now.Add(30 * time.Minute) // Hold expires in 30 minutes

	_, err = tx.Exec(`
		INSERT INTO wallet_holds (
			id, hold_number, wallet_id, amount, currency, status,
			reference_type, reference_id, description, expires_at,
			created_by_user_id, created_at, updated_at
		) VALUES (
			uuid_generate_v4(), $1, $2, $3, $4, 'pending',
			'shipment', $5, $6, $7,
			$8, NOW(), NOW()
		)
	`, holdNumber, walletID, holdAmount, "CAD",
		shipmentID, fmt.Sprintf("Hold for shipment %s", shipmentNumber), expiresAt,
		userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet hold: %w", err)
	}

	// Update wallet held_balance
	_, err = tx.Exec(`
		UPDATE wallets
		SET held_balance = held_balance + $1, updated_at = NOW()
		WHERE id = $2
	`, holdAmount, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet held balance: %w", err)
	}

	// 8. Create invoice for the shipment
	invoiceID := uuid.New().String()
	invoiceNumber := fmt.Sprintf("INV-%s", generateRandomCode(8))

	// Get organization billing info for invoice
	var orgName string
	err = tx.QueryRow(`SELECT name FROM organizations WHERE id = $1`, organizationID).Scan(&orgName)
	if err != nil {
		orgName = "" // Non-critical, continue without it
	}

	// Calculate invoice amounts
	subtotalAmount := selectedRate.BaseRate + selectedRate.FuelSurcharge + selectedRate.OtherFees
	taxesAmount := selectedRate.Taxes
	totalAmount := selectedRate.TotalAmount

	// Build shipment summaries for invoice snapshot
	fromSummary := fmt.Sprintf("%s, %s, %s", estimate.FromCity, nullStringToStr(estimate.FromRegionCode), estimate.FromCountry)
	toSummary := fmt.Sprintf("%s, %s, %s", estimate.ToCity, nullStringToStr(estimate.ToRegionCode), estimate.ToCountry)

	// Insert invoice
	_, err = tx.Exec(`
		INSERT INTO invoices (
			id, invoice_number, organization_id, created_by_user_id,
			invoice_type, status, payment_status, payment_method,
			shipment_id, shipment_number,
			shipment_from_summary, shipment_to_summary,
			carrier_code, carrier_name, service_code, service_name,
			total_packages, currency,
			subtotal_amount, taxes_amount, discount_amount,
			total_amount, amount_paid, balance_due, refundable_amount,
			issued_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			'invoice', 'issued', 'unpaid', 'unassigned',
			$5, $6,
			$7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15, $16, 0,
			$17, 0, $17, 0,
			NOW(), NOW(), NOW()
		)
	`,
		invoiceID, invoiceNumber, organizationID, userID,
		shipmentID, shipmentNumber,
		fromSummary, toSummary,
		selectedRate.CarrierCode, selectedRate.CarrierName, selectedRate.ServiceCode, selectedRate.ServiceName,
		1, "CAD",
		subtotalAmount, taxesAmount,
		totalAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Insert invoice line items
	itemIndex := 1

	// Base rate item
	_, err = tx.Exec(`
		INSERT INTO invoice_items (
			invoice_id, item_index, item_type, section,
			description, quantity, unit_price, amount
		) VALUES ($1, $2, 'base_rate', 'charges', $3, 1, $4, $4)
	`, invoiceID, itemIndex, fmt.Sprintf("%s - %s", selectedRate.CarrierName, selectedRate.ServiceName), selectedRate.BaseRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice base rate item: %w", err)
	}
	itemIndex++

	// Fuel surcharge item (if applicable)
	if selectedRate.FuelSurcharge > 0 {
		_, err = tx.Exec(`
			INSERT INTO invoice_items (
				invoice_id, item_index, item_type, section,
				description, quantity, unit_price, amount
			) VALUES ($1, $2, 'fuel_surcharge', 'charges', 'Fuel Surcharge', 1, $3, $3)
		`, invoiceID, itemIndex, selectedRate.FuelSurcharge)
		if err != nil {
			return nil, fmt.Errorf("failed to create invoice fuel surcharge item: %w", err)
		}
		itemIndex++
	}

	// Other fees item (if applicable)
	if selectedRate.OtherFees > 0 {
		_, err = tx.Exec(`
			INSERT INTO invoice_items (
				invoice_id, item_index, item_type, section,
				description, quantity, unit_price, amount
			) VALUES ($1, $2, 'other', 'charges', 'Additional Fees', 1, $3, $3)
		`, invoiceID, itemIndex, selectedRate.OtherFees)
		if err != nil {
			return nil, fmt.Errorf("failed to create invoice other fees item: %w", err)
		}
		itemIndex++
	}

	// Tax item (if applicable)
	if selectedRate.Taxes > 0 {
		_, err = tx.Exec(`
			INSERT INTO invoice_items (
				invoice_id, item_index, item_type, section,
				description, quantity, unit_price, amount
			) VALUES ($1, $2, 'tax_hst', 'charges', 'HST (13%)', 1, $3, $3)
		`, invoiceID, itemIndex, selectedRate.Taxes)
		if err != nil {
			return nil, fmt.Errorf("failed to create invoice tax item: %w", err)
		}
	}

	// Create invoice event
	_, err = tx.Exec(`
		INSERT INTO invoice_events (
			invoice_id, event_type, description, actor_user_id
		) VALUES ($1, 'invoice.created', 'Invoice created for shipment', $2)
	`, invoiceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return shipment, nil
}

// ListShipments returns a paginated list of shipments
func (s *SimpleShipmentService) ListShipments(
	organizationID string,
	query ListShipmentsQuery,
) ([]*models.Shipment, int, error) {
	// Set defaults for pagination
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Count total records
	var total int
	var countQuery string
	var args []interface{}
	var argIndex int

	// If organizationID is empty, list all shipments (admin access)
	if organizationID == "" {
		countQuery = `SELECT COUNT(*) FROM shipments WHERE is_deleted = FALSE`
		args = []interface{}{}
		argIndex = 1
	} else {
		countQuery = `SELECT COUNT(*) FROM shipments WHERE organization_id = $1 AND is_deleted = FALSE`
		args = []interface{}{organizationID}
		argIndex = 2
	}

	// Add optional filters to count query
	if query.Status != nil && *query.Status != "" {
		countQuery += fmt.Sprintf(" AND shipment_status = $%d", argIndex)
		args = append(args, *query.Status)
		argIndex++
	}
	if query.IsArchived != nil {
		countQuery += fmt.Sprintf(" AND is_archived = $%d", argIndex)
		args = append(args, *query.IsArchived)
		argIndex++
	}

	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count shipments: %w", err)
	}

	// Build the main query
	var selectQuery string
	if organizationID == "" {
		selectQuery = `
			SELECT id, shipment_number, organization_id, created_by_user_id, shipment_estimate_id,
			       carrier_id, servicelevel_code, servicelevel_name,
			       from_country, from_region_code, from_city, from_postal_code,
			       to_country, to_region_code, to_city, to_postal_code,
			       total_packages, currency, shipment_status, payment_status,
			       is_archived, master_tracking_number, created_at, updated_at
			FROM shipments
			WHERE is_deleted = FALSE
		`
		args = []interface{}{}
		argIndex = 1
	} else {
		selectQuery = `
			SELECT id, shipment_number, organization_id, created_by_user_id, shipment_estimate_id,
			       carrier_id, servicelevel_code, servicelevel_name,
			       from_country, from_region_code, from_city, from_postal_code,
			       to_country, to_region_code, to_city, to_postal_code,
			       total_packages, currency, shipment_status, payment_status,
			       is_archived, master_tracking_number, created_at, updated_at
			FROM shipments
			WHERE organization_id = $1 AND is_deleted = FALSE
		`
		args = []interface{}{organizationID}
		argIndex = 2
	}

	// Add optional filters
	if query.Status != nil && *query.Status != "" {
		selectQuery += fmt.Sprintf(" AND shipment_status = $%d", argIndex)
		args = append(args, *query.Status)
		argIndex++
	}
	if query.IsArchived != nil {
		selectQuery += fmt.Sprintf(" AND is_archived = $%d", argIndex)
		args = append(args, *query.IsArchived)
		argIndex++
	}

	// Add sorting and pagination
	selectQuery += " ORDER BY created_at DESC"
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query shipments: %w", err)
	}
	defer rows.Close()

	// Parse results
	shipments := make([]*models.Shipment, 0)
	for rows.Next() {
		var shipment models.Shipment
		err := rows.Scan(
			&shipment.ID, &shipment.ShipmentNumber, &shipment.OrganizationID,
			&shipment.CreatedByUserID, &shipment.ShipmentEstimateID,
			&shipment.CarrierID, &shipment.ServicelevelCode, &shipment.ServicelevelName,
			&shipment.FromCountry, &shipment.FromRegionCode, &shipment.FromCity, &shipment.FromPostalCode,
			&shipment.ToCountry, &shipment.ToRegionCode, &shipment.ToCity, &shipment.ToPostalCode,
			&shipment.TotalPackages, &shipment.Currency, &shipment.ShipmentStatus, &shipment.PaymentStatus,
			&shipment.IsArchived, &shipment.MasterTrackingNumber, &shipment.CreatedAt, &shipment.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan shipment: %w", err)
		}
		shipments = append(shipments, &shipment)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating shipments: %w", err)
	}

	return shipments, total, nil
}

// GetShipmentByID returns a shipment
func (s *SimpleShipmentService) GetShipmentByID(
	shipmentID string,
	organizationID string,
) (*models.Shipment, []*models.ShipmentPackage, []*models.ShipmentEvent, error) {
	var shipment models.Shipment

	err := s.db.QueryRow(`
		SELECT id, shipment_number, organization_id, created_by_user_id, shipment_estimate_id,
		       carrier_id, from_country, to_country, shipment_status
		FROM shipments
		WHERE id = $1 AND organization_id = $2 AND is_deleted = FALSE
	`, shipmentID, organizationID).Scan(
		&shipment.ID, &shipment.ShipmentNumber, &shipment.OrganizationID, &shipment.CreatedByUserID, &shipment.ShipmentEstimateID,
		&shipment.CarrierID, &shipment.FromCountry, &shipment.ToCountry, &shipment.ShipmentStatus,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, errors.New("shipment not found or access denied")
		}
		return nil, nil, nil, fmt.Errorf("failed to fetch shipment: %w", err)
	}

	return &shipment, []*models.ShipmentPackage{}, []*models.ShipmentEvent{}, nil
}

// CancelShipment cancels a shipment
func (s *SimpleShipmentService) CancelShipment(shipmentID, organizationID, userID, reason string) error {
	_, err := s.db.Exec(`
		UPDATE shipments SET shipment_status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
	`, shipmentID, organizationID)
	return err
}

// VoidShipment voids a shipment
func (s *SimpleShipmentService) VoidShipment(shipmentID, organizationID, userID, reason string) error {
	_, err := s.db.Exec(`
		UPDATE shipments SET shipment_status = 'voided', updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
	`, shipmentID, organizationID)
	return err
}

// ArchiveShipment archives a shipment
func (s *SimpleShipmentService) ArchiveShipment(shipmentID, organizationID, userID string) error {
	_, err := s.db.Exec(`
		UPDATE shipments SET is_archived = NOT is_archived, updated_at = NOW()
		WHERE id = $1 AND organization_id = $2
	`, shipmentID, organizationID)
	return err
}

// GetShipmentStatus retrieves the current status of a shipment
func (s *SimpleShipmentService) GetShipmentStatus(shipmentID, organizationID string) (string, error) {
	var status string
	err := s.db.QueryRow(`
		SELECT shipment_status
		FROM shipments
		WHERE id = $1 AND organization_id = $2
	`, shipmentID, organizationID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("shipment not found or access denied")
		}
		return "", fmt.Errorf("failed to get shipment status: %w", err)
	}
	return status, nil
}

// UpdateShipmentStatus updates the status of a shipment and triggers commission creation when delivered
// This function should be called by webhook handlers or tracking services when shipment status changes
func (s *SimpleShipmentService) UpdateShipmentStatus(shipmentID, organizationID string, newStatus models.ShipmentStatus) error {
	// Get current shipment info for commission calculation (if needed)
	var currentStatus string
	var revenueAmount, costAmount float64
	var currency string
	err := s.db.QueryRow(`
		SELECT shipment_status, COALESCE(revenue_amount, 0), COALESCE(cost_amount, 0), COALESCE(currency, 'CAD')
		FROM shipments
		WHERE id = $1 AND organization_id = $2
	`, shipmentID, organizationID).Scan(&currentStatus, &revenueAmount, &costAmount, &currency)
	if err != nil {
		return fmt.Errorf("failed to get shipment: %w", err)
	}

	// Update shipment status
	_, err = s.db.Exec(`
		UPDATE shipments SET shipment_status = $1, updated_at = NOW()
		WHERE id = $2 AND organization_id = $3
	`, string(newStatus), shipmentID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update shipment status: %w", err)
	}

	// If shipment is now delivered, trigger commission creation
	if newStatus == models.ShipmentStatusDelivered && currentStatus != string(models.ShipmentStatusDelivered) {
		err = s.createCommissionOnDelivery(shipmentID, organizationID, revenueAmount, costAmount, currency)
		if err != nil {
			// Log error but don't fail the status update
			// In production, you'd want proper logging here
			fmt.Printf("Warning: failed to create commission for shipment %s: %v\n", shipmentID, err)
		}
	}

	return nil
}

// createCommissionOnDelivery creates a commission transaction when a shipment is delivered
// This is called internally when shipment status changes to "delivered"
func (s *SimpleShipmentService) createCommissionOnDelivery(shipmentID, organizationID string, revenueAmount, costAmount float64, currency string) error {
	// Find if the shipment creator was referred by someone
	var referredUserID string
	var referrerUserID sql.NullString

	// Get the user who created the shipment and check if they were referred
	err := s.db.QueryRow(`
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
	var commissionPlanID sql.NullString
	var commissionNumber, commissionBase, commissionValueType string
	var commissionValue float64

	err = s.db.QueryRow(`
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

	// Check if commission already exists for this shipment+beneficiary combination
	var existingCount int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM commission_transactions
		WHERE shipment_id = $1 AND beneficiary_user_id = $2
	`, shipmentID, referrerUserID.String).Scan(&existingCount)
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
	txNumber := fmt.Sprintf("CTX-%s", generateRandomCode(8))
	commissionID := uuid.New().String()
	now := time.Now()

	// Create the commission transaction (status: eligible since shipment is delivered)
	_, err = s.db.Exec(`
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

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// nullStringToStr returns string value or empty string for null
func nullStringToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Real carrier UUIDs from database (must match shipment_estimates module)
const (
	fedexCarrierID = "550e8400-e29b-41d4-a716-446655440001"
	upsCarrierID   = "550e8400-e29b-41d4-a716-446655440002"
)

// getMockRate returns mock rate data based on carrier and service code
// This matches the mock rates returned by the shipment_estimates module
func getMockRate(carrierID, serviceCode string) *models.CarrierRate {
	mockRates := []models.CarrierRate{
		{
			CarrierID:       fedexCarrierID,
			CarrierCode:     "fedex",
			CarrierName:     "FedEx",
			ServiceCode:     "FEDEX_GROUND",
			ServiceName:     "FedEx Ground",
			ServiceCategory: "ground",
			BaseRate:        22.50,
			FuelSurcharge:   2.25,
			OtherFees:       1.00,
			Taxes:           3.32,
			TotalAmount:     29.07,
			Currency:        "CAD",
		},
		{
			CarrierID:       upsCarrierID,
			CarrierCode:     "ups",
			CarrierName:     "UPS",
			ServiceCode:     "UPS_STANDARD",
			ServiceName:     "UPS Standard",
			ServiceCategory: "ground",
			BaseRate:        24.00,
			FuelSurcharge:   2.40,
			OtherFees:       1.20,
			Taxes:           3.55,
			TotalAmount:     31.15,
			Currency:        "CAD",
		},
		{
			CarrierID:       fedexCarrierID,
			CarrierCode:     "fedex",
			CarrierName:     "FedEx",
			ServiceCode:     "FEDEX_EXPRESS",
			ServiceName:     "FedEx Express Saver",
			ServiceCategory: "express",
			BaseRate:        35.00,
			FuelSurcharge:   3.50,
			OtherFees:       1.50,
			Taxes:           5.15,
			TotalAmount:     45.15,
			Currency:        "CAD",
		},
	}

	for _, rate := range mockRates {
		if rate.CarrierID == carrierID && rate.ServiceCode == serviceCode {
			return &rate
		}
	}
	return nil
}
