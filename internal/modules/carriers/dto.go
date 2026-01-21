package carriers

import (
	"time"
)

// =====================================================
// Response DTOs
// =====================================================

// CarrierResponse represents a carrier in API responses
type CarrierResponse struct {
	ID                    string    `json:"id"`
	Code                  string    `json:"code"`
	Name                  string    `json:"name"`
	DisplayName           string    `json:"display_name,omitempty"`
	CarrierType           string    `json:"carrier_type"`
	SupportsDomestic      bool      `json:"supports_domestic"`
	SupportsInternational bool      `json:"supports_international"`
	SupportsPickup        bool      `json:"supports_pickup"`
	SupportsDropOff       bool      `json:"supports_drop_off"`
	SupportsReturns       bool      `json:"supports_returns"`
	SupportsLabels        bool      `json:"supports_labels"`
	SupportsTracking      bool      `json:"supports_tracking"`
	SupportsInsurance     bool      `json:"supports_insurance"`
	SupportsSignature     bool      `json:"supports_signature"`
	SupportsCOD           bool      `json:"supports_cod"`
	IsEnabled             bool      `json:"is_enabled"`
	IsActive              bool      `json:"is_active"`
	LogoURL               string    `json:"logo_url,omitempty"`
	ColorHex              string    `json:"color_hex,omitempty"`
	SortOrder             int       `json:"sort_order"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// CarrierListResponse represents a list of carriers
type CarrierListResponse struct {
	Carriers []*CarrierResponse `json:"carriers"`
	Total    int                `json:"total"`
}

// ServiceLevelResponse represents a carrier service level
type ServiceLevelResponse struct {
	ID                      string `json:"id"`
	Code                    string `json:"code"`
	CarrierCode             string `json:"carrier_code"`
	Name                    string `json:"name"`
	DisplayName             string `json:"display_name,omitempty"`
	Category                string `json:"category"`
	MinTransitDays          *int   `json:"min_transit_days,omitempty"`
	MaxTransitDays          *int   `json:"max_transit_days,omitempty"`
	DeliveryTimeDescription string `json:"delivery_time_description,omitempty"`
	IsDomestic              bool   `json:"is_domestic"`
	IsInternational         bool   `json:"is_international"`
	SupportsPickup          bool   `json:"supports_pickup"`
	SupportsSaturdayDelivery bool  `json:"supports_saturday_delivery"`
	IsEnabled               bool   `json:"is_enabled"`
	SortOrder               int    `json:"sort_order"`
}

// AvailabilityResponse represents carrier availability for a date
type AvailabilityResponse struct {
	CarrierID         string  `json:"carrier_id"`
	CarrierCode       string  `json:"carrier_code"`
	CarrierName       string  `json:"carrier_name"`
	Date              string  `json:"date"`
	DayOfWeek         string  `json:"day_of_week"`
	IsAvailable       bool    `json:"is_available"`
	IsHoliday         bool    `json:"is_holiday"`
	HolidayName       string  `json:"holiday_name,omitempty"`
	PickupStartTime   string  `json:"pickup_start_time,omitempty"`
	PickupEndTime     string  `json:"pickup_end_time,omitempty"`
	SameDayCutoffTime string  `json:"same_day_cutoff_time,omitempty"`
	Notes             string  `json:"notes,omitempty"`
	ServiceLevels     []*ServiceLevelResponse `json:"service_levels,omitempty"`
}
