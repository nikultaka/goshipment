package shipments

import (
	"database/sql"
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

	// 1. Verify estimate exists and get basic info
	var estimate struct {
		ID             string
		FromCountry    string
		FromRegionCode sql.NullString
		FromCity       string
		FromPostalCode string
		ToCountry      string
		ToRegionCode   sql.NullString
		ToCity         string
		ToPostalCode   string
		ExpiresAt      time.Time
	}

	err = tx.QueryRow(`
		SELECT id, from_country, from_region_code, from_city, from_postal_code,
		       to_country, to_region_code, to_city, to_postal_code, expires_at
		FROM shipment_estimates
		WHERE id = $1 AND (organization_id = $2 OR organization_id IS NULL) AND is_deleted = FALSE
	`, req.EstimateID, organizationID).Scan(
		&estimate.ID, &estimate.FromCountry, &estimate.FromRegionCode, &estimate.FromCity, &estimate.FromPostalCode,
		&estimate.ToCountry, &estimate.ToRegionCode, &estimate.ToCity, &estimate.ToPostalCode,
		&estimate.ExpiresAt,
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
	// Since carrier_rates_snapshot is not stored in DB, use mock rate lookup
	selectedRate := getMockRate(req.SelectedCarrierID, req.SelectedServiceCode)
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

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return shipment, nil
}

// ListShipments returns a simplified list
func (s *SimpleShipmentService) ListShipments(
	organizationID string,
	query ListShipmentsQuery,
) ([]*models.Shipment, int, error) {
	// Simple implementation
	rows, err := s.db.Query(`
		SELECT COUNT(*) FROM shipments
		WHERE organization_id = $1 AND is_deleted = FALSE
	`, organizationID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var total int
	if rows.Next() {
		rows.Scan(&total)
	}

	return []*models.Shipment{}, total, nil
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

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// getMockRate returns mock rate data based on carrier and service code
// This matches the mock rates returned by the shipment_estimates module
func getMockRate(carrierID, serviceCode string) *models.CarrierRate {
	mockRates := []models.CarrierRate{
		{
			CarrierID:       "mock-fedex-id",
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
			CarrierID:       "mock-ups-id",
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
			CarrierID:       "mock-fedex-id",
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
