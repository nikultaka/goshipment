package carriers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
)

// CarrierService handles carrier business logic
type CarrierService struct {
	db *clients.Database
}

// NewCarrierService creates a new carrier service
func NewCarrierService(db *clients.Database) *CarrierService {
	return &CarrierService{db: db}
}

// ListCarriers retrieves all enabled carriers
func (s *CarrierService) ListCarriers(activeOnly bool) ([]*CarrierResponse, error) {
	query := `
		SELECT
			id, code, name, COALESCE(display_name, name), carrier_type,
			supports_domestic, supports_international, supports_pickup, supports_drop_off,
			supports_returns, supports_labels, supports_tracking, supports_insurance,
			supports_signature, supports_cod, is_enabled, is_active,
			COALESCE(color_hex, ''), sort_order, created_at, updated_at
		FROM carriers
		WHERE 1=1
	`

	args := []interface{}{}
	if activeOnly {
		query += " AND is_enabled = true AND is_active = true"
	}
	query += " ORDER BY sort_order ASC, name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list carriers: %w", err)
	}
	defer rows.Close()

	carriers := []*CarrierResponse{}
	for rows.Next() {
		var c CarrierResponse
		err := rows.Scan(
			&c.ID, &c.Code, &c.Name, &c.DisplayName, &c.CarrierType,
			&c.SupportsDomestic, &c.SupportsInternational, &c.SupportsPickup, &c.SupportsDropOff,
			&c.SupportsReturns, &c.SupportsLabels, &c.SupportsTracking, &c.SupportsInsurance,
			&c.SupportsSignature, &c.SupportsCOD, &c.IsEnabled, &c.IsActive,
			&c.ColorHex, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan carrier: %w", err)
		}
		carriers = append(carriers, &c)
	}

	return carriers, nil
}

// GetCarrierByID retrieves a carrier by its UUID
func (s *CarrierService) GetCarrierByID(id string) (*CarrierResponse, error) {
	query := `
		SELECT
			id, code, name, COALESCE(display_name, name), carrier_type,
			supports_domestic, supports_international, supports_pickup, supports_drop_off,
			supports_returns, supports_labels, supports_tracking, supports_insurance,
			supports_signature, supports_cod, is_enabled, is_active,
			COALESCE(color_hex, ''), sort_order, created_at, updated_at
		FROM carriers
		WHERE id = $1
	`

	var c CarrierResponse
	err := s.db.QueryRow(query, id).Scan(
		&c.ID, &c.Code, &c.Name, &c.DisplayName, &c.CarrierType,
		&c.SupportsDomestic, &c.SupportsInternational, &c.SupportsPickup, &c.SupportsDropOff,
		&c.SupportsReturns, &c.SupportsLabels, &c.SupportsTracking, &c.SupportsInsurance,
		&c.SupportsSignature, &c.SupportsCOD, &c.IsEnabled, &c.IsActive,
		&c.ColorHex, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("carrier not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get carrier: %w", err)
	}

	return &c, nil
}

// GetCarrierByCode retrieves a carrier by its code (e.g., "fedex", "ups")
func (s *CarrierService) GetCarrierByCode(code string) (*CarrierResponse, error) {
	query := `
		SELECT
			id, code, name, COALESCE(display_name, name), carrier_type,
			supports_domestic, supports_international, supports_pickup, supports_drop_off,
			supports_returns, supports_labels, supports_tracking, supports_insurance,
			supports_signature, supports_cod, is_enabled, is_active,
			COALESCE(color_hex, ''), sort_order, created_at, updated_at
		FROM carriers
		WHERE LOWER(code) = LOWER($1)
	`

	var c CarrierResponse
	err := s.db.QueryRow(query, code).Scan(
		&c.ID, &c.Code, &c.Name, &c.DisplayName, &c.CarrierType,
		&c.SupportsDomestic, &c.SupportsInternational, &c.SupportsPickup, &c.SupportsDropOff,
		&c.SupportsReturns, &c.SupportsLabels, &c.SupportsTracking, &c.SupportsInsurance,
		&c.SupportsSignature, &c.SupportsCOD, &c.IsEnabled, &c.IsActive,
		&c.ColorHex, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("carrier not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get carrier: %w", err)
	}

	return &c, nil
}

// GetCarrierAvailability retrieves availability for a carrier on a specific date
func (s *CarrierService) GetCarrierAvailability(code string, date time.Time) (*AvailabilityResponse, error) {
	// First get the carrier
	carrier, err := s.GetCarrierByCode(code)
	if err != nil {
		return nil, err
	}

	// Database enum uses lowercase (monday, tuesday, etc.)
	dayOfWeek := strings.ToLower(date.Weekday().String())

	// Get operating schedule for this day
	scheduleQuery := `
		SELECT
			COALESCE(is_operating, false),
			pickup_start_time,
			pickup_end_time,
			same_day_cutoff_time,
			COALESCE(notes, '')
		FROM carrier_operating_schedule
		WHERE carrier_id = $1 AND day_of_week = $2::day_of_week_enum
	`

	var isOperating bool
	var pickupStart, pickupEnd, sameDayCutoff sql.NullString
	var notes string

	err = s.db.QueryRow(scheduleQuery, carrier.ID, dayOfWeek).Scan(
		&isOperating, &pickupStart, &pickupEnd, &sameDayCutoff, &notes,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Check for holidays
	holidayQuery := `
		SELECT
			is_closed,
			COALESCE(holiday_name, ''),
			modified_pickup_start,
			modified_pickup_end,
			modified_cutoff,
			COALESCE(notes, '')
		FROM carrier_holidays
		WHERE carrier_id = $1 AND holiday_date = $2
		LIMIT 1
	`

	var isHoliday, isClosed bool
	var holidayName string
	var modPickupStart, modPickupEnd, modCutoff sql.NullString
	var holidayNotes string

	err = s.db.QueryRow(holidayQuery, carrier.ID, date).Scan(
		&isClosed, &holidayName, &modPickupStart, &modPickupEnd, &modCutoff, &holidayNotes,
	)
	if err == nil {
		isHoliday = true
		if isClosed {
			isOperating = false
		}
		if modPickupStart.Valid {
			pickupStart = modPickupStart
		}
		if modPickupEnd.Valid {
			pickupEnd = modPickupEnd
		}
		if modCutoff.Valid {
			sameDayCutoff = modCutoff
		}
		if holidayNotes != "" {
			notes = holidayNotes
		}
	}

	// Get service levels
	serviceLevels, _ := s.GetServiceLevels(carrier.ID)

	resp := &AvailabilityResponse{
		CarrierID:   carrier.ID,
		CarrierCode: carrier.Code,
		CarrierName: carrier.Name,
		Date:        date.Format("2006-01-02"),
		DayOfWeek:   date.Weekday().String(), // Display as "Tuesday" not "tuesday"
		IsAvailable: isOperating && carrier.IsEnabled && carrier.IsActive,
		IsHoliday:   isHoliday,
		HolidayName: holidayName,
		Notes:       notes,
		ServiceLevels: serviceLevels,
	}

	if pickupStart.Valid {
		resp.PickupStartTime = pickupStart.String
	}
	if pickupEnd.Valid {
		resp.PickupEndTime = pickupEnd.String
	}
	if sameDayCutoff.Valid {
		resp.SameDayCutoffTime = sameDayCutoff.String
	}

	return resp, nil
}

// GetServiceLevels retrieves service levels for a carrier
func (s *CarrierService) GetServiceLevels(carrierID string) ([]*ServiceLevelResponse, error) {
	query := `
		SELECT
			id, code, carrier_code, name, COALESCE(display_name, name),
			category, min_transit_days, max_transit_days,
			COALESCE(delivery_time_description, ''),
			is_domestic, is_international, supports_pickup,
			supports_saturday_delivery, is_enabled, sort_order
		FROM carrier_service_levels
		WHERE carrier_id = $1 AND is_enabled = true AND is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := s.db.Query(query, carrierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service levels: %w", err)
	}
	defer rows.Close()

	levels := []*ServiceLevelResponse{}
	for rows.Next() {
		var sl ServiceLevelResponse
		var minDays, maxDays sql.NullInt64

		err := rows.Scan(
			&sl.ID, &sl.Code, &sl.CarrierCode, &sl.Name, &sl.DisplayName,
			&sl.Category, &minDays, &maxDays, &sl.DeliveryTimeDescription,
			&sl.IsDomestic, &sl.IsInternational, &sl.SupportsPickup,
			&sl.SupportsSaturdayDelivery, &sl.IsEnabled, &sl.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service level: %w", err)
		}

		if minDays.Valid {
			v := int(minDays.Int64)
			sl.MinTransitDays = &v
		}
		if maxDays.Valid {
			v := int(maxDays.Int64)
			sl.MaxTransitDays = &v
		}

		levels = append(levels, &sl)
	}

	return levels, nil
}
