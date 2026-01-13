package models

import (
	"database/sql"
	"time"
)

// PickupStatus represents the lifecycle state of a pickup
type PickupStatus string

const (
	PickupStatusDraft       PickupStatus = "draft"       // created but not scheduled yet
	PickupStatusScheduled   PickupStatus = "scheduled"   // scheduled successfully with carrier
	PickupStatusRescheduled PickupStatus = "rescheduled" // schedule changed after scheduling
	PickupStatusCancelled   PickupStatus = "cancelled"   // cancelled by user/admin or provider
	PickupStatusCompleted   PickupStatus = "completed"   // pickup completed successfully
	PickupStatusFailed      PickupStatus = "failed"      // scheduling or pickup failed
)

// PickupBuildingType represents the type of building for pickup
type PickupBuildingType string

const (
	PickupBuildingTypeHouse      PickupBuildingType = "house"
	PickupBuildingTypeApartment  PickupBuildingType = "apartment"
	PickupBuildingTypeCondo      PickupBuildingType = "condo"
	PickupBuildingTypeOffice     PickupBuildingType = "office"
	PickupBuildingTypeWarehouse  PickupBuildingType = "warehouse"
	PickupBuildingTypeRetail     PickupBuildingType = "retail"
	PickupBuildingTypeIndustrial PickupBuildingType = "industrial"
	PickupBuildingTypeMedical    PickupBuildingType = "medical"
	PickupBuildingTypeGovernment PickupBuildingType = "government"
	PickupBuildingTypeOther      PickupBuildingType = "other"
)

// PickupLocation represents the pickup location type
type PickupLocation string

const (
	PickupLocationFrontDoor        PickupLocation = "front_door"
	PickupLocationBasement         PickupLocation = "basement"
	PickupLocationBetweenDoors     PickupLocation = "between_doors"
	PickupLocationCounter          PickupLocation = "counter"
	PickupLocationDesk             PickupLocation = "desk"
	PickupLocationFrontDesk        PickupLocation = "front_desk"
	PickupLocationBackDoor         PickupLocation = "back_door"
	PickupLocationFrontPorch       PickupLocation = "front_porch"
	PickupLocationGarage           PickupLocation = "garage"
	PickupLocationGateHouse        PickupLocation = "gate_house"
	PickupLocationKiosk            PickupLocation = "kiosk"
	PickupLocationLab              PickupLocation = "lab"
	PickupLocationLoadingDock      PickupLocation = "loading_dock"
	PickupLocationLobby            PickupLocation = "lobby"
	PickupLocationMailroom         PickupLocation = "mailroom"
	PickupLocationMailbox          PickupLocation = "mailbox"
	PickupLocationOutsideDoor      PickupLocation = "outside_door"
	PickupLocationPartsDepartment  PickupLocation = "parts_department"
	PickupLocationPharmacy         PickupLocation = "pharmacy"
	PickupLocationProShop          PickupLocation = "pro_shop"
	PickupLocationReceiving        PickupLocation = "receiving"
	PickupLocationSecurity         PickupLocation = "security"
	PickupLocationServiceCounter   PickupLocation = "service_counter"
	PickupLocationShipping         PickupLocation = "shipping"
	PickupLocationSideDoor         PickupLocation = "side_door"
	PickupLocationSwitchboard      PickupLocation = "switchboard"
	PickupLocationVault            PickupLocation = "vault"
	PickupLocationOther            PickupLocation = "other"
)

// Pickup represents a pickup record in the database
type Pickup struct {
	// Primary Key
	ID string `json:"id" db:"id"`

	// Identifiers
	PickupNumber   string `json:"pickup_number" db:"pickup_number"`
	OrganizationID string `json:"organization_id" db:"organization_id"`
	CreatedByUserID string `json:"created_by_user_id" db:"created_by_user_id"`

	// Relationship
	ShipmentID string `json:"shipment_id" db:"shipment_id"`

	// Carrier
	CarrierID string `json:"carrier_id" db:"carrier_id"`

	// Pickup Address Book Reference
	PickupAddressBookID string `json:"pickup_address_book_id" db:"pickup_address_book_id"`

	// Pickup Address Snapshot (IMMUTABLE)
	PickupName         sql.NullString `json:"pickup_name,omitempty" db:"pickup_name"`
	PickupCompany      sql.NullString `json:"pickup_company,omitempty" db:"pickup_company"`
	PickupPhone        sql.NullString `json:"pickup_phone,omitempty" db:"pickup_phone"`
	PickupEmail        sql.NullString `json:"pickup_email,omitempty" db:"pickup_email"`
	PickupCountry      string         `json:"pickup_country" db:"pickup_country"`
	PickupRegionCode   sql.NullString `json:"pickup_region_code,omitempty" db:"pickup_region_code"`
	PickupCity         string         `json:"pickup_city" db:"pickup_city"`
	PickupPostalCode   string         `json:"pickup_postal_code" db:"pickup_postal_code"`
	PickupAddressLine1 string         `json:"pickup_address_line1" db:"pickup_address_line1"`
	PickupAddressLine2 sql.NullString `json:"pickup_address_line2,omitempty" db:"pickup_address_line2"`
	PickupIsResidential bool          `json:"pickup_is_residential" db:"pickup_is_residential"`

	// Location Details
	BuildingType        PickupBuildingType `json:"building_type" db:"building_type"`
	PickupLocation      PickupLocation     `json:"pickup_location" db:"pickup_location"`
	PickupInstructions  sql.NullString     `json:"pickup_instructions,omitempty" db:"pickup_instructions"`

	// Scheduling
	PickupDate        time.Time `json:"pickup_date" db:"pickup_date"`
	PickupWindowStart time.Time `json:"pickup_window_start" db:"pickup_window_start"`
	PickupWindowEnd   time.Time `json:"pickup_window_end" db:"pickup_window_end"`
	Timezone          string    `json:"timezone" db:"timezone"`

	// Provider Response
	ProviderPickupID  sql.NullString `json:"provider_pickup_id,omitempty" db:"provider_pickup_id"`
	ProviderReference sql.NullString `json:"provider_reference,omitempty" db:"provider_reference"`
	ProviderResponse  sql.NullString `json:"provider_response,omitempty" db:"provider_response"` // JSONB

	// Post-Pickup Adjustments
	AdjustmentsTotal        float64        `json:"adjustments_total" db:"adjustments_total"`
	HasPendingAdjustments   bool           `json:"has_pending_adjustments" db:"has_pending_adjustments"`
	AdjustmentsBreakdown    sql.NullString `json:"adjustments_breakdown,omitempty" db:"adjustments_breakdown"` // JSONB
	InvoiceID               sql.NullString `json:"invoice_id,omitempty" db:"invoice_id"`

	// Status
	PickupStatus  PickupStatus   `json:"pickup_status" db:"pickup_status"`
	ScheduledAt   sql.NullTime   `json:"scheduled_at,omitempty" db:"scheduled_at"`
	CancelledAt   sql.NullTime   `json:"cancelled_at,omitempty" db:"cancelled_at"`
	FailureReason sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`

	// Safety / Audit
	IdempotencyKey sql.NullString `json:"idempotency_key,omitempty" db:"idempotency_key"`
	Revision       int            `json:"revision" db:"revision"`
	IsDeleted      bool           `json:"is_deleted" db:"is_deleted"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// Helper methods for Pickup

// CanBeCancelled returns true if the pickup can be cancelled
func (p *Pickup) CanBeCancelled() bool {
	cancellableStatuses := []PickupStatus{
		PickupStatusDraft,
		PickupStatusScheduled,
		PickupStatusRescheduled,
	}
	for _, status := range cancellableStatuses {
		if p.PickupStatus == status {
			return true
		}
	}
	return false
}

// IsActive returns true if the pickup is in an active state
func (p *Pickup) IsActive() bool {
	activeStatuses := []PickupStatus{
		PickupStatusDraft,
		PickupStatusScheduled,
		PickupStatusRescheduled,
	}
	for _, status := range activeStatuses {
		if p.PickupStatus == status {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the pickup is in a terminal state
func (p *Pickup) IsTerminal() bool {
	terminalStatuses := []PickupStatus{
		PickupStatusCancelled,
		PickupStatusCompleted,
		PickupStatusFailed,
	}
	for _, status := range terminalStatuses {
		if p.PickupStatus == status {
			return true
		}
	}
	return false
}
