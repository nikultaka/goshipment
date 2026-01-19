package pickups

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
)

const (
	// PickupNumberPrefix is the prefix for pickup numbers
	PickupNumberPrefix = "PKU-"
)

// PickupService handles pickup business logic
type PickupService struct {
	db *clients.Database
}

// NewPickupService creates a new pickup service
func NewPickupService(db *clients.Database) *PickupService {
	return &PickupService{db: db}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *PickupService) GetOrganizationIDFromUser(userID string) (string, error) {
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

// generatePickupNumber generates a unique pickup number
func (s *PickupService) generatePickupNumber() (string, error) {
	// Generate 8 random characters
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return PickupNumberPrefix + hex.EncodeToString(bytes), nil
}

// SchedulePickup schedules a new pickup for a shipment
func (s *PickupService) SchedulePickup(req SchedulePickupRequest, organizationID, userID string) (*PickupResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify shipment exists and belongs to organization
	var shipment models.Shipment
	err = tx.QueryRow(`
		SELECT id, organization_id, shipment_status, carrier_id, from_address_book_id,
			from_name, from_company, from_phone, from_email,
			from_country, from_region_code, from_city, from_postal_code,
			from_address_line1, from_address_line2
		FROM shipments
		WHERE id = $1 AND organization_id = $2 AND is_deleted = FALSE
	`, req.ShipmentID, organizationID).Scan(
		&shipment.ID, &shipment.OrganizationID, &shipment.ShipmentStatus, &shipment.CarrierID,
		&shipment.FromAddressBookID,
		&shipment.FromName, &shipment.FromCompany, &shipment.FromPhone, &shipment.FromEmail,
		&shipment.FromCountry, &shipment.FromRegionCode, &shipment.FromCity, &shipment.FromPostalCode,
		&shipment.FromAddressLine1, &shipment.FromAddressLine2,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("shipment not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shipment: %w", err)
	}

	// 2. Verify shipment status is valid for pickup (label_purchased or ready_for_pickup)
	validStatuses := []models.ShipmentStatus{
		models.ShipmentStatusLabelPurchased,
		models.ShipmentStatusReadyForPickup,
	}
	isValidStatus := false
	for _, status := range validStatuses {
		if shipment.ShipmentStatus == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return nil, fmt.Errorf("shipment must be in 'label_purchased' or 'ready_for_pickup' status to schedule pickup")
	}

	// 3. Parse pickup date
	pickupDate, err := time.Parse("2006-01-02", req.PickupDate)
	if err != nil {
		return nil, fmt.Errorf("invalid pickup date format (use YYYY-MM-DD): %w", err)
	}

	// 4. Parse pickup time windows
	pickupWindowStart, err := time.Parse("15:04", req.PickupWindowStart)
	if err != nil {
		return nil, fmt.Errorf("invalid pickup window start format (use HH:MM): %w", err)
	}

	pickupWindowEnd, err := time.Parse("15:04", req.PickupWindowEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid pickup window end format (use HH:MM): %w", err)
	}

	// 5. Validate pickup window
	if !pickupWindowEnd.After(pickupWindowStart) {
		return nil, fmt.Errorf("pickup window end must be after start")
	}

	// 6. Check same-day pickup rules
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)

	if pickupDate.Before(tomorrow) {
		// This is a same-day or past pickup request

		// Check system_settings.allow_same_day_pickups
		var systemAllowSameDayPickup bool
		err = tx.QueryRow(`
			SELECT COALESCE(value::boolean, false)
			FROM settings
			WHERE category = 'system' AND key = 'allow_same_day_pickups'
		`).Scan(&systemAllowSameDayPickup)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check system settings: %w", err)
		}

		// Check organization_settings.allow_same_day_pickup
		var orgAllowSameDayPickup bool
		err = tx.QueryRow(`
			SELECT allow_same_day_pickup
			FROM organization_settings
			WHERE organization_id = $1
		`, organizationID).Scan(&orgAllowSameDayPickup)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check organization settings: %w", err)
		}

		if !systemAllowSameDayPickup || !orgAllowSameDayPickup {
			return nil, fmt.Errorf("same-day pickup is not allowed")
		}
	}

	// 7. Validate pickup location instructions for 'other'
	if req.PickupLocation == "other" && req.Instructions == "" {
		return nil, fmt.Errorf("instructions are required when pickup location is 'other'")
	}

	// 8. Get or create address book entry for pickup
	var addressBookID string
	if shipment.FromAddressBookID.Valid {
		addressBookID = shipment.FromAddressBookID.String
	} else {
		// Create address book entry from shipment's from address
		newAddressID, err := s.createAddressBookEntryFromShipment(tx, &shipment, organizationID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to create address book entry: %w", err)
		}
		addressBookID = newAddressID
	}

	// 9. Generate pickup number
	pickupNumber, err := s.generatePickupNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pickup number: %w", err)
	}

	// 10. Create pickup record
	pickup := &models.Pickup{}
	now = time.Now()

	// Create full datetime for window start/end
	windowStartDateTime := time.Date(pickupDate.Year(), pickupDate.Month(), pickupDate.Day(),
		pickupWindowStart.Hour(), pickupWindowStart.Minute(), 0, 0, pickupDate.Location())
	windowEndDateTime := time.Date(pickupDate.Year(), pickupDate.Month(), pickupDate.Day(),
		pickupWindowEnd.Hour(), pickupWindowEnd.Minute(), 0, 0, pickupDate.Location())

	err = tx.QueryRow(`
		INSERT INTO pickups (
			pickup_number, organization_id, created_by_user_id, shipment_id, carrier_id,
			pickup_address_book_id,
			pickup_name, pickup_company, pickup_phone, pickup_email,
			pickup_country, pickup_region_code, pickup_city, pickup_postal_code,
			pickup_address_line1, pickup_address_line2, pickup_is_residential,
			building_type, pickup_location, pickup_instructions,
			pickup_date, pickup_window_start, pickup_window_end, timezone,
			pickup_status, scheduled_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, false,
			$17, $18, $19,
			$20, $21, $22, 'America/Toronto',
			'scheduled', $23, $24, $25
		)
		RETURNING id, pickup_number, organization_id, created_by_user_id, shipment_id, carrier_id,
			pickup_address_book_id,
			pickup_name, pickup_company, pickup_phone, pickup_email,
			pickup_country, pickup_region_code, pickup_city, pickup_postal_code,
			pickup_address_line1, pickup_address_line2, pickup_is_residential,
			building_type, pickup_location, pickup_instructions,
			pickup_date, pickup_window_start, pickup_window_end, timezone,
			pickup_status, scheduled_at, created_at, updated_at
	`,
		pickupNumber, organizationID, userID, req.ShipmentID, shipment.CarrierID,
		addressBookID,
		shipment.FromName, shipment.FromCompany, shipment.FromPhone, shipment.FromEmail,
		shipment.FromCountry, shipment.FromRegionCode, shipment.FromCity, shipment.FromPostalCode,
		shipment.FromAddressLine1, shipment.FromAddressLine2,
		req.BuildingType, req.PickupLocation, sql.NullString{String: req.Instructions, Valid: req.Instructions != ""},
		pickupDate, windowStartDateTime, windowEndDateTime,
		now, now, now,
	).Scan(
		&pickup.ID, &pickup.PickupNumber, &pickup.OrganizationID, &pickup.CreatedByUserID,
		&pickup.ShipmentID, &pickup.CarrierID, &pickup.PickupAddressBookID,
		&pickup.PickupName, &pickup.PickupCompany, &pickup.PickupPhone, &pickup.PickupEmail,
		&pickup.PickupCountry, &pickup.PickupRegionCode, &pickup.PickupCity, &pickup.PickupPostalCode,
		&pickup.PickupAddressLine1, &pickup.PickupAddressLine2, &pickup.PickupIsResidential,
		&pickup.BuildingType, &pickup.PickupLocation, &pickup.PickupInstructions,
		&pickup.PickupDate, &pickup.PickupWindowStart, &pickup.PickupWindowEnd, &pickup.Timezone,
		&pickup.PickupStatus, &pickup.ScheduledAt, &pickup.CreatedAt, &pickup.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pickup: %w", err)
	}

	// 11. Create shipment event for pickup.scheduled
	_, err = tx.Exec(`
		INSERT INTO shipment_events (
			shipment_id, organization_id, pickup_id,
			entity_type, event_key, source, severity,
			title, message,
			occurred_at, actor_user_id
		) VALUES (
			$1, $2, $3,
			'pickup', 'pickup.scheduled', 'user', 'info',
			'Pickup Scheduled', 'Pickup ' || $4 || ' has been scheduled for ' || $5,
			$6, $7
		)
	`, req.ShipmentID, organizationID, pickup.ID,
		pickup.PickupNumber, pickupDate.Format("2006-01-02"),
		now, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment event: %w", err)
	}

	// 12. Update shipment status to pickup_scheduled
	_, err = tx.Exec(`
		UPDATE shipments
		SET shipment_status = 'pickup_scheduled', pickup_id = $1, updated_at = $2
		WHERE id = $3
	`, pickup.ID, now, req.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update shipment status: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ToPickupResponse(pickup), nil
}

// CancelPickup cancels a scheduled pickup
func (s *PickupService) CancelPickup(pickupID, organizationID, userID string) (*CancelPickupResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch pickup
	var pickup models.Pickup
	err = tx.QueryRow(`
		SELECT id, pickup_number, organization_id, shipment_id, pickup_status
		FROM pickups
		WHERE id = $1 AND organization_id = $2 AND is_deleted = FALSE
	`, pickupID, organizationID).Scan(
		&pickup.ID, &pickup.PickupNumber, &pickup.OrganizationID,
		&pickup.ShipmentID, &pickup.PickupStatus,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pickup not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pickup: %w", err)
	}

	// 2. Verify pickup can be cancelled
	if pickup.PickupStatus == models.PickupStatusCompleted {
		return nil, fmt.Errorf("cannot cancel a completed pickup")
	}
	if pickup.PickupStatus == models.PickupStatusCancelled {
		return nil, fmt.Errorf("pickup is already cancelled")
	}

	// 3. Update pickup status to cancelled
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE pickups
		SET pickup_status = 'cancelled', cancelled_at = $1, updated_at = $2
		WHERE id = $3
	`, now, now, pickupID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel pickup: %w", err)
	}

	// 4. Create shipment event for pickup.cancelled
	_, err = tx.Exec(`
		INSERT INTO shipment_events (
			shipment_id, organization_id, pickup_id,
			entity_type, event_key, source, severity,
			title, message,
			occurred_at, actor_user_id
		) VALUES (
			$1, $2, $3,
			'pickup', 'pickup.cancelled', 'user', 'info',
			'Pickup Cancelled', 'Pickup ' || $4 || ' has been cancelled',
			$5, $6
		)
	`, pickup.ShipmentID, organizationID, pickupID,
		pickup.PickupNumber, now, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment event: %w", err)
	}

	// 5. Update shipment status back to ready_for_pickup
	_, err = tx.Exec(`
		UPDATE shipments
		SET shipment_status = 'ready_for_pickup', updated_at = $1
		WHERE id = $2
	`, now, pickup.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update shipment status: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CancelPickupResponse{
		ID:           pickup.ID,
		PickupNumber: pickup.PickupNumber,
		PickupStatus: string(models.PickupStatusCancelled),
		CancelledAt:  now,
		Message:      "Pickup cancelled successfully",
	}, nil
}

// ListPickups retrieves a list of pickups for an organization
func (s *PickupService) ListPickups(req ListPickupsRequest, organizationID, userID string) (*ListPickupsResponse, error) {
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
		SELECT id, pickup_number, organization_id, created_by_user_id, shipment_id, carrier_id,
			pickup_address_book_id,
			pickup_name, pickup_company, pickup_phone, pickup_email,
			pickup_country, pickup_region_code, pickup_city, pickup_postal_code,
			pickup_address_line1, pickup_address_line2, pickup_is_residential,
			building_type, pickup_location, pickup_instructions,
			pickup_date, pickup_window_start, pickup_window_end, timezone,
			provider_pickup_id, provider_reference,
			pickup_status, scheduled_at, cancelled_at, failure_reason,
			created_at, updated_at
		FROM pickups
		WHERE organization_id = $1 AND is_deleted = FALSE
	`
	args := []interface{}{organizationID}
	argIndex := 2

	// Add filters
	if req.Status != "" {
		query += fmt.Sprintf(" AND pickup_status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.ShipmentID != "" {
		query += fmt.Sprintf(" AND shipment_id = $%d", argIndex)
		args = append(args, req.ShipmentID)
		argIndex++
	}

	if req.FromDate != "" {
		query += fmt.Sprintf(" AND pickup_date >= $%d", argIndex)
		args = append(args, req.FromDate)
		argIndex++
	}

	if req.ToDate != "" {
		query += fmt.Sprintf(" AND pickup_date <= $%d", argIndex)
		args = append(args, req.ToDate)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	var totalCount int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count pickups: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pickups: %w", err)
	}
	defer rows.Close()

	pickups := []PickupResponse{}
	for rows.Next() {
		var pickup models.Pickup
		err := rows.Scan(
			&pickup.ID, &pickup.PickupNumber, &pickup.OrganizationID, &pickup.CreatedByUserID,
			&pickup.ShipmentID, &pickup.CarrierID, &pickup.PickupAddressBookID,
			&pickup.PickupName, &pickup.PickupCompany, &pickup.PickupPhone, &pickup.PickupEmail,
			&pickup.PickupCountry, &pickup.PickupRegionCode, &pickup.PickupCity, &pickup.PickupPostalCode,
			&pickup.PickupAddressLine1, &pickup.PickupAddressLine2, &pickup.PickupIsResidential,
			&pickup.BuildingType, &pickup.PickupLocation, &pickup.PickupInstructions,
			&pickup.PickupDate, &pickup.PickupWindowStart, &pickup.PickupWindowEnd, &pickup.Timezone,
			&pickup.ProviderPickupID, &pickup.ProviderReference,
			&pickup.PickupStatus, &pickup.ScheduledAt, &pickup.CancelledAt, &pickup.FailureReason,
			&pickup.CreatedAt, &pickup.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pickup: %w", err)
		}

		pickups = append(pickups, *ToPickupResponse(&pickup))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pickups: %w", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	return &ListPickupsResponse{
		Pickups:    pickups,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// createAddressBookEntryFromShipment creates an address book entry from shipment's from address
func (s *PickupService) createAddressBookEntryFromShipment(tx *sql.Tx, shipment *models.Shipment, organizationID, userID string) (string, error) {
	// Generate address number
	addressNumber, err := s.generateAddressNumber()
	if err != nil {
		return "", fmt.Errorf("failed to generate address number: %w", err)
	}

	// Use shipment data for address, with defaults for required fields
	contactName := "Pickup Contact"
	if shipment.FromName.Valid && shipment.FromName.String != "" {
		contactName = shipment.FromName.String
	}

	addressLine1 := "Address"
	if shipment.FromAddressLine1 != "" {
		addressLine1 = shipment.FromAddressLine1
	}

	nickname := fmt.Sprintf("Pickup Address - %s", shipment.ID[:8])

	var addressID string
	err = tx.QueryRow(`
		INSERT INTO address_book (
			address_number, organization_id, created_by_user_id,
			nickname, company, contact_name, phone, email,
			address_line_1, address_line_2, city, province_code, postal_code, country_code,
			address_type, is_residential, is_default_sender, is_default_recipient, is_verified, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			'sender', false, false, false, false, true,
			NOW(), NOW()
		)
		RETURNING id
	`,
		addressNumber, organizationID, userID,
		nickname, shipment.FromCompany, contactName, shipment.FromPhone, shipment.FromEmail,
		addressLine1, shipment.FromAddressLine2, shipment.FromCity, shipment.FromRegionCode, shipment.FromPostalCode, shipment.FromCountry,
	).Scan(&addressID)

	if err != nil {
		return "", fmt.Errorf("failed to insert address book entry: %w", err)
	}

	// Update shipment to link the new address book entry
	_, err = tx.Exec(`
		UPDATE shipments SET from_address_book_id = $1, updated_at = NOW() WHERE id = $2
	`, addressID, shipment.ID)
	if err != nil {
		return "", fmt.Errorf("failed to update shipment with address book: %w", err)
	}

	return addressID, nil
}

// generateAddressNumber generates a unique address number in format ADR-[A-Z0-9]{8}
func (s *PickupService) generateAddressNumber() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}
	return "ADR-" + string(bytes), nil
}
