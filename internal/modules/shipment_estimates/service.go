package shipment_estimates

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
)

const (
	// DefaultExpirationMinutes is the default estimate validity period
	DefaultExpirationMinutes = 15
)

// EstimateService handles shipment estimate business logic
type EstimateService struct {
	db *clients.Database
}

// NewEstimateService creates a new estimate service
func NewEstimateService(db *clients.Database) *EstimateService {
	return &EstimateService{
		db: db,
	}
}

// CreateEstimate creates a new shipment estimate with carrier rates
func (s *EstimateService) CreateEstimate(
	req CreateEstimateRequest,
	organizationID *string,
	userID *string,
) (*models.ShipmentEstimate, []*models.ShipmentPackage, []models.CarrierRate, error) {

	// Calculate expiration time
	expiresAt := time.Now().UTC().Add(DefaultExpirationMinutes * time.Minute)

	// Get carrier rates (this will call carrier APIs in production)
	carrierRates, err := s.getCarrierRates(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get carrier rates: %w", err)
	}

	if len(carrierRates) == 0 {
		return nil, nil, nil, fmt.Errorf("no carrier rates available for this route")
	}

	// Calculate totals from packages
	totalPackages := len(req.Packages)
	totalWeight := 0.0
	weightUnit := req.Packages[0].WeightUnit // Use first package's unit

	for _, pkg := range req.Packages {
		totalWeight += pkg.Weight
	}

	// Determine source
	source := models.EstimateSourceFrontend
	if req.Source != nil {
		source = models.EstimateSource(*req.Source)
	}

	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Normalize country codes to uppercase (DB stores ISO 3166-1 alpha-2 in uppercase)
	fromCountry := strings.ToUpper(req.FromAddress.Country)
	toCountry := strings.ToUpper(req.ToAddress.Country)

	// Create estimate record
	estimate := &models.ShipmentEstimate{
		Source:         source,
		FromCountry:    fromCountry,
		FromPostalCode: req.FromAddress.PostalCode,
		ToCountry:      toCountry,
		ToPostalCode:   req.ToAddress.PostalCode,
		TotalPackages:  totalPackages,
		TotalWeight:    totalWeight,
		WeightUnit:     weightUnit,
		Currency:       "CAD",
		ExpiresAt:      expiresAt,
		IsLocked:       false,
		IsDeleted:      false,
	}

	// Set optional fields
	if req.FromAddress.RegionCode != nil {
		estimate.FromRegionCode = sql.NullString{String: *req.FromAddress.RegionCode, Valid: true}
	}
	if req.FromAddress.City != nil {
		estimate.FromCity = sql.NullString{String: *req.FromAddress.City, Valid: true}
	}
	if req.ToAddress.RegionCode != nil {
		estimate.ToRegionCode = sql.NullString{String: *req.ToAddress.RegionCode, Valid: true}
	}
	if req.ToAddress.City != nil {
		estimate.ToCity = sql.NullString{String: *req.ToAddress.City, Valid: true}
	}

	// Set ownership (nullable for guest estimates)
	// IMPORTANT: Database constraint requires: (org_id IS NULL AND user_id IS NULL) OR (org_id IS NOT NULL)
	// So if user is authenticated but has no organization, treat as guest estimate (both NULL)
	if organizationID != nil && *organizationID != "" {
		estimate.OrganizationID = sql.NullString{String: *organizationID, Valid: true}
		// Only set user_id if organization_id is also set (satisfies DB constraint)
		if userID != nil && *userID != "" {
			estimate.UserID = sql.NullString{String: *userID, Valid: true}
		}
	}
	// If no organization_id, leave both as NULL (guest estimate)

	// Set address book links if provided (validate UUID format)
	if req.FromAddressBookID != nil && *req.FromAddressBookID != "" && *req.FromAddressBookID != "string" {
		// Basic UUID validation (36 characters with hyphens)
		if len(*req.FromAddressBookID) == 36 {
			estimate.FromAddressBookID = sql.NullString{String: *req.FromAddressBookID, Valid: true}
		}
	}
	if req.ToAddressBookID != nil && *req.ToAddressBookID != "" && *req.ToAddressBookID != "string" {
		// Basic UUID validation (36 characters with hyphens)
		if len(*req.ToAddressBookID) == 36 {
			estimate.ToAddressBookID = sql.NullString{String: *req.ToAddressBookID, Valid: true}
		}
	}

	// Insert estimate
	query := `
		INSERT INTO shipment_estimates (
			organization_id, user_id, source,
			from_address_book_id, to_address_book_id,
			from_country, from_region_code, from_city, from_postal_code,
			to_country, to_region_code, to_city, to_postal_code,
			total_packages, total_weight, weight_unit,
			currency, expires_at, is_locked,
			is_deleted, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW()
		) RETURNING id, created_at
	`

	err = tx.QueryRow(query,
		estimate.OrganizationID, estimate.UserID, estimate.Source,
		estimate.FromAddressBookID, estimate.ToAddressBookID,
		estimate.FromCountry, estimate.FromRegionCode, estimate.FromCity, estimate.FromPostalCode,
		estimate.ToCountry, estimate.ToRegionCode, estimate.ToCity, estimate.ToPostalCode,
		estimate.TotalPackages, estimate.TotalWeight, estimate.WeightUnit,
		estimate.Currency, estimate.ExpiresAt, estimate.IsLocked,
		estimate.IsDeleted,
	).Scan(&estimate.ID, &estimate.CreatedAt)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create estimate: %w", err)
	}

	// Insert packages
	packages := make([]*models.ShipmentPackage, 0, len(req.Packages))
	packageQuery := `
		INSERT INTO shipment_packages (
			shipment_estimate_id, package_index, package_name,
			length, width, height, dimension_unit,
			weight, weight_unit,
			declared_value, declared_currency,
			package_status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending', NOW()
		) RETURNING id, created_at
	`

	for i, pkg := range req.Packages {
		shipmentPkg := &models.ShipmentPackage{
			ShipmentEstimateID: sql.NullString{String: estimate.ID, Valid: true},
			PackageIndex:       i + 1,
			Length:             pkg.Length,
			Width:              pkg.Width,
			Height:             pkg.Height,
			DimensionUnit:      pkg.DimensionUnit,
			Weight:             pkg.Weight,
			WeightUnit:         pkg.WeightUnit,
			PackageStatus:      "pending",
		}

		if pkg.PackageName != nil {
			shipmentPkg.PackageName = sql.NullString{String: *pkg.PackageName, Valid: true}
		}

		if pkg.DeclaredValue != nil {
			shipmentPkg.DeclaredValue = sql.NullFloat64{Float64: *pkg.DeclaredValue, Valid: true}
			shipmentPkg.DeclaredCurrency = sql.NullString{String: "CAD", Valid: true}
		}

		err = tx.QueryRow(packageQuery,
			shipmentPkg.ShipmentEstimateID, shipmentPkg.PackageIndex, shipmentPkg.PackageName,
			shipmentPkg.Length, shipmentPkg.Width, shipmentPkg.Height, shipmentPkg.DimensionUnit,
			shipmentPkg.Weight, shipmentPkg.WeightUnit,
			shipmentPkg.DeclaredValue, shipmentPkg.DeclaredCurrency,
		).Scan(&shipmentPkg.ID, &shipmentPkg.CreatedAt)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create package %d: %w", i+1, err)
		}

		packages = append(packages, shipmentPkg)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return estimate, packages, carrierRates, nil
}

// GetEstimateByID retrieves an estimate by ID
func (s *EstimateService) GetEstimateByID(estimateID string) (*models.ShipmentEstimate, []*models.ShipmentPackage, []models.CarrierRate, error) {
	// Get estimate
	query := `
		SELECT
			id, organization_id, user_id, source,
			from_address_book_id, to_address_book_id,
			from_country, from_region_code, from_city, from_postal_code,
			to_country, to_region_code, to_city, to_postal_code,
			total_packages, total_weight, weight_unit,
			subtotal_amount, taxes_amount, total_amount, currency,
			expires_at, is_locked, locked_at,
			selected_carrier_id, selected_servicelevel_name, selected_servicelevel_code,
			selected_amount, rate_selected_at,
			is_deleted, created_at
		FROM shipment_estimates
		WHERE id = $1 AND is_deleted = FALSE
	`

	estimate := &models.ShipmentEstimate{}
	err := s.db.DB.QueryRow(query, estimateID).Scan(
		&estimate.ID, &estimate.OrganizationID, &estimate.UserID, &estimate.Source,
		&estimate.FromAddressBookID, &estimate.ToAddressBookID,
		&estimate.FromCountry, &estimate.FromRegionCode, &estimate.FromCity, &estimate.FromPostalCode,
		&estimate.ToCountry, &estimate.ToRegionCode, &estimate.ToCity, &estimate.ToPostalCode,
		&estimate.TotalPackages, &estimate.TotalWeight, &estimate.WeightUnit,
		&estimate.SubtotalAmount, &estimate.TaxesAmount, &estimate.TotalAmount, &estimate.Currency,
		&estimate.ExpiresAt, &estimate.IsLocked, &estimate.LockedAt,
		&estimate.SelectedCarrierID, &estimate.SelectedServiceLevelName, &estimate.SelectedServiceLevelCode,
		&estimate.SelectedAmount, &estimate.RateSelectedAt,
		&estimate.IsDeleted, &estimate.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil, nil, fmt.Errorf("estimate not found")
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get estimate: %w", err)
	}

	// Get packages
	packages, err := s.getEstimatePackages(estimateID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get packages: %w", err)
	}

	// Carrier rates are not stored in database, return empty array
	// Rates are only available at creation time and expire after 15 minutes
	var carrierRates []models.CarrierRate

	return estimate, packages, carrierRates, nil
}

// getEstimatePackages retrieves packages for an estimate
func (s *EstimateService) getEstimatePackages(estimateID string) ([]*models.ShipmentPackage, error) {
	query := `
		SELECT
			id, package_number, shipment_estimate_id, shipment_id,
			package_index, package_name,
			length, width, height, dimension_unit,
			weight, weight_unit,
			declared_value, declared_currency,
			carrier_parcel_id,
			carrier_label_transaction_id, label_url, label_file_type,
			tracking_number, tracking_url, label_purchased_at,
			package_status, failure_reason, carrier_error,
			created_at
		FROM shipment_packages
		WHERE shipment_estimate_id = $1
		ORDER BY package_index ASC
	`

	rows, err := s.db.DB.Query(query, estimateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packages := make([]*models.ShipmentPackage, 0)
	for rows.Next() {
		pkg := &models.ShipmentPackage{}
		err := rows.Scan(
			&pkg.ID, &pkg.PackageNumber, &pkg.ShipmentEstimateID, &pkg.ShipmentID,
			&pkg.PackageIndex, &pkg.PackageName,
			&pkg.Length, &pkg.Width, &pkg.Height, &pkg.DimensionUnit,
			&pkg.Weight, &pkg.WeightUnit,
			&pkg.DeclaredValue, &pkg.DeclaredCurrency,
			&pkg.CarrierParcelID,
			&pkg.CarrierLabelTransactionID, &pkg.LabelURL, &pkg.LabelFileType,
			&pkg.TrackingNumber, &pkg.TrackingURL, &pkg.LabelPurchasedAt,
			&pkg.PackageStatus, &pkg.FailureReason, &pkg.CarrierError,
			&pkg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}

// getCarrierRates fetches carrier rates for the given request
// TODO: In production, this will call real carrier APIs (FedEx, UPS, etc.)
// For now, returns mock data
func (s *EstimateService) getCarrierRates(req CreateEstimateRequest) ([]models.CarrierRate, error) {
	// Mock carrier rates for development
	// In production, this will:
	// 1. Query enabled carriers from database
	// 2. Call each carrier's rate API
	// 3. Apply platform markups
	// 4. Return combined results

	isDomestic := strings.ToUpper(req.FromAddress.Country) == strings.ToUpper(req.ToAddress.Country)

	rates := []models.CarrierRate{
		{
			CarrierID:              "mock-fedex-id",
			CarrierCode:            "fedex",
			CarrierName:            "FedEx",
			ServiceCode:            "FEDEX_GROUND",
			ServiceName:            "FedEx Ground",
			ServiceCategory:        "ground",
			BaseRate:               22.50,
			FuelSurcharge:          2.25,
			OtherFees:              1.00,
			Taxes:                  3.32,
			TotalAmount:            29.07,
			Currency:               "CAD",
			EstimatedDeliveryDate:  time.Now().AddDate(0, 0, 3).Format("2006-01-02"),
			TransitDays:            3,
			SupportsPickup:         true,
			SupportsSignature:      true,
			SupportsSaturdayDelivery: false,
		},
		{
			CarrierID:              "mock-ups-id",
			CarrierCode:            "ups",
			CarrierName:            "UPS",
			ServiceCode:            "UPS_STANDARD",
			ServiceName:            "UPS Standard",
			ServiceCategory:        "ground",
			BaseRate:               24.00,
			FuelSurcharge:          2.40,
			OtherFees:              1.20,
			Taxes:                  3.55,
			TotalAmount:            31.15,
			Currency:               "CAD",
			EstimatedDeliveryDate:  time.Now().AddDate(0, 0, 3).Format("2006-01-02"),
			TransitDays:            3,
			SupportsPickup:         true,
			SupportsSignature:      true,
			SupportsSaturdayDelivery: false,
		},
	}

	if isDomestic {
		// Add express options for domestic
		rates = append(rates, models.CarrierRate{
			CarrierID:              "mock-fedex-id",
			CarrierCode:            "fedex",
			CarrierName:            "FedEx",
			ServiceCode:            "FEDEX_EXPRESS",
			ServiceName:            "FedEx Express Saver",
			ServiceCategory:        "express",
			BaseRate:               35.00,
			FuelSurcharge:          3.50,
			OtherFees:              1.50,
			Taxes:                  5.15,
			TotalAmount:            45.15,
			Currency:               "CAD",
			EstimatedDeliveryDate:  time.Now().AddDate(0, 0, 2).Format("2006-01-02"),
			TransitDays:            2,
			SupportsPickup:         true,
			SupportsSignature:      true,
			SupportsSaturdayDelivery: true,
		})
	}

	return rates, nil
}
