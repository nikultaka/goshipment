package pickups

import (
	"time"

	"gogin/internal/models"
)

// SchedulePickupRequest represents the request to schedule a new pickup
type SchedulePickupRequest struct {
	ShipmentID        string `json:"shipment_id" binding:"required,uuid"`
	PickupDate        string `json:"pickup_date" binding:"required"` // YYYY-MM-DD format
	PickupWindowStart string `json:"pickup_window_start" binding:"required"` // HH:MM format
	PickupWindowEnd   string `json:"pickup_window_end" binding:"required"` // HH:MM format
	BuildingType      string `json:"building_type" binding:"required,oneof=house apartment condo office warehouse retail industrial medical government other"`
	PickupLocation    string `json:"pickup_location" binding:"required,oneof=front_door basement between_doors counter desk front_desk back_door front_porch garage gate_house kiosk lab loading_dock lobby mailroom mailbox outside_door parts_department pharmacy pro_shop receiving security service_counter shipping side_door switchboard vault other"`
	Instructions      string `json:"instructions,omitempty"`
}

// PickupResponse represents a pickup in API responses
type PickupResponse struct {
	ID                  string    `json:"id"`
	PickupNumber        string    `json:"pickup_number"`
	OrganizationID      string    `json:"organization_id"`
	ShipmentID          string    `json:"shipment_id"`
	CarrierID           string    `json:"carrier_id"`
	PickupAddressBookID string    `json:"pickup_address_book_id"`

	// Address Snapshot
	PickupName         string `json:"pickup_name,omitempty"`
	PickupCompany      string `json:"pickup_company,omitempty"`
	PickupPhone        string `json:"pickup_phone,omitempty"`
	PickupEmail        string `json:"pickup_email,omitempty"`
	PickupCountry      string `json:"pickup_country"`
	PickupRegionCode   string `json:"pickup_region_code,omitempty"`
	PickupCity         string `json:"pickup_city"`
	PickupPostalCode   string `json:"pickup_postal_code"`
	PickupAddressLine1 string `json:"pickup_address_line1"`
	PickupAddressLine2 string `json:"pickup_address_line2,omitempty"`
	PickupIsResidential bool  `json:"pickup_is_residential"`

	// Location Details
	BuildingType       string `json:"building_type"`
	PickupLocation     string `json:"pickup_location"`
	PickupInstructions string `json:"pickup_instructions,omitempty"`

	// Scheduling
	PickupDate        string `json:"pickup_date"` // YYYY-MM-DD
	PickupWindowStart string `json:"pickup_window_start"` // HH:MM
	PickupWindowEnd   string `json:"pickup_window_end"` // HH:MM
	Timezone          string `json:"timezone"`

	// Provider
	ProviderPickupID  string `json:"provider_pickup_id,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`

	// Status
	PickupStatus  string     `json:"pickup_status"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CancelPickupResponse represents the response after cancelling a pickup
type CancelPickupResponse struct {
	ID           string    `json:"id"`
	PickupNumber string    `json:"pickup_number"`
	PickupStatus string    `json:"pickup_status"`
	CancelledAt  time.Time `json:"cancelled_at"`
	Message      string    `json:"message"`
}

// ListPickupsRequest represents query parameters for listing pickups
type ListPickupsRequest struct {
	Status     string `form:"status" binding:"omitempty,oneof=draft scheduled rescheduled cancelled completed failed"`
	ShipmentID string `form:"shipment_id" binding:"omitempty,uuid"`
	FromDate   string `form:"from_date" binding:"omitempty"` // YYYY-MM-DD
	ToDate     string `form:"to_date" binding:"omitempty"`   // YYYY-MM-DD
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ListPickupsResponse represents the response for listing pickups
type ListPickupsResponse struct {
	Pickups    []PickupResponse `json:"pickups"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// ToPickupResponse converts a Pickup model to PickupResponse
func ToPickupResponse(pickup *models.Pickup) *PickupResponse {
	resp := &PickupResponse{
		ID:                  pickup.ID,
		PickupNumber:        pickup.PickupNumber,
		OrganizationID:      pickup.OrganizationID,
		ShipmentID:          pickup.ShipmentID,
		CarrierID:           pickup.CarrierID,
		PickupAddressBookID: pickup.PickupAddressBookID,

		// Address
		PickupCountry:      pickup.PickupCountry,
		PickupCity:         pickup.PickupCity,
		PickupPostalCode:   pickup.PickupPostalCode,
		PickupAddressLine1: pickup.PickupAddressLine1,
		PickupIsResidential: pickup.PickupIsResidential,

		// Location
		BuildingType:   string(pickup.BuildingType),
		PickupLocation: string(pickup.PickupLocation),
		Timezone:       pickup.Timezone,

		// Status
		PickupStatus: string(pickup.PickupStatus),

		// Audit
		CreatedAt: pickup.CreatedAt,
		UpdatedAt: pickup.UpdatedAt,
	}

	// Optional fields
	if pickup.PickupName.Valid {
		resp.PickupName = pickup.PickupName.String
	}
	if pickup.PickupCompany.Valid {
		resp.PickupCompany = pickup.PickupCompany.String
	}
	if pickup.PickupPhone.Valid {
		resp.PickupPhone = pickup.PickupPhone.String
	}
	if pickup.PickupEmail.Valid {
		resp.PickupEmail = pickup.PickupEmail.String
	}
	if pickup.PickupRegionCode.Valid {
		resp.PickupRegionCode = pickup.PickupRegionCode.String
	}
	if pickup.PickupAddressLine2.Valid {
		resp.PickupAddressLine2 = pickup.PickupAddressLine2.String
	}
	if pickup.PickupInstructions.Valid {
		resp.PickupInstructions = pickup.PickupInstructions.String
	}
	if pickup.ProviderPickupID.Valid {
		resp.ProviderPickupID = pickup.ProviderPickupID.String
	}
	if pickup.ProviderReference.Valid {
		resp.ProviderReference = pickup.ProviderReference.String
	}
	if pickup.ScheduledAt.Valid {
		resp.ScheduledAt = &pickup.ScheduledAt.Time
	}
	if pickup.CancelledAt.Valid {
		resp.CancelledAt = &pickup.CancelledAt.Time
	}
	if pickup.FailureReason.Valid {
		resp.FailureReason = pickup.FailureReason.String
	}

	// Format dates and times
	resp.PickupDate = pickup.PickupDate.Format("2006-01-02")
	resp.PickupWindowStart = pickup.PickupWindowStart.Format("15:04")
	resp.PickupWindowEnd = pickup.PickupWindowEnd.Format("15:04")

	return resp
}
