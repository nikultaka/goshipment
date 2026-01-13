package models

import (
	"database/sql"
	"time"
)

// CarrierType represents the type of carrier
type CarrierType string

const (
	CarrierTypeParcel     CarrierType = "parcel"
	CarrierTypeFreightLTL CarrierType = "freight_ltl"
	CarrierTypeFreightFTL CarrierType = "freight_ftl"
)

// ServiceLevelCategory represents the service category
type ServiceLevelCategory string

const (
	ServiceLevelGround        ServiceLevelCategory = "ground"
	ServiceLevelExpress       ServiceLevelCategory = "express"
	ServiceLevelPriority      ServiceLevelCategory = "priority"
	ServiceLevelSameDay       ServiceLevelCategory = "same_day"
	ServiceLevelEconomy       ServiceLevelCategory = "economy"
	ServiceLevelInternational ServiceLevelCategory = "international"
)

// Carrier represents a shipping carrier (FedEx, UPS, etc.)
type Carrier struct {
	ID          string      `json:"id" db:"id"`
	Code        string      `json:"code" db:"code"`
	Name        string      `json:"name" db:"name"`
	DisplayName sql.NullString `json:"display_name,omitempty" db:"display_name"`

	// Classification
	CarrierType CarrierType `json:"carrier_type" db:"carrier_type"`

	// Capabilities
	SupportsDomestic      bool `json:"supports_domestic" db:"supports_domestic"`
	SupportsInternational bool `json:"supports_international" db:"supports_international"`
	SupportsPickup        bool `json:"supports_pickup" db:"supports_pickup"`
	SupportsDropOff       bool `json:"supports_drop_off" db:"supports_drop_off"`
	SupportsReturns       bool `json:"supports_returns" db:"supports_returns"`
	SupportsLabels        bool `json:"supports_labels" db:"supports_labels"`
	SupportsTracking      bool `json:"supports_tracking" db:"supports_tracking"`
	SupportsInsurance     bool `json:"supports_insurance" db:"supports_insurance"`
	SupportsSignature     bool `json:"supports_signature" db:"supports_signature"`
	SupportsCOD           bool `json:"supports_cod" db:"supports_cod"`

	// API Integration
	APIVersion   sql.NullString `json:"api_version,omitempty" db:"api_version"`
	APIBaseURL   sql.NullString `json:"api_base_url,omitempty" db:"api_base_url"`

	// Status
	IsEnabled bool `json:"is_enabled" db:"is_enabled"`
	IsActive  bool `json:"is_active" db:"is_active"`

	// UI
	LogoFileID sql.NullString `json:"logo_file_id,omitempty" db:"logo_file_id"`
	SortOrder  int            `json:"sort_order" db:"sort_order"`
	ColorHex   sql.NullString `json:"color_hex,omitempty" db:"color_hex"`

	// Metadata
	Metadata  sql.NullString `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// CarrierSettings represents carrier operational settings
type CarrierSettings struct {
	ID         string    `json:"id" db:"id"`
	CarrierID  string    `json:"carrier_id" db:"carrier_id"`

	// Pickup Configuration
	MinPickupLeadHours  sql.NullInt32 `json:"min_pickup_lead_hours,omitempty" db:"min_pickup_lead_hours"`
	MaxPickupDaysAhead  sql.NullInt32 `json:"max_pickup_days_ahead,omitempty" db:"max_pickup_days_ahead"`

	// Package Limits
	MaxPackagesPerShipment sql.NullInt32   `json:"max_packages_per_shipment,omitempty" db:"max_packages_per_shipment"`
	MaxPackageWeightKg     sql.NullFloat64 `json:"max_package_weight_kg,omitempty" db:"max_package_weight_kg"`
	MaxPackageLengthCm     sql.NullFloat64 `json:"max_package_length_cm,omitempty" db:"max_package_length_cm"`
	MaxDeclaredValue       sql.NullFloat64 `json:"max_declared_value,omitempty" db:"max_declared_value"`

	// Pricing (Platform markup)
	RateMultiplier    float64 `json:"rate_multiplier" db:"rate_multiplier"`
	FlatMarkupAmount  float64 `json:"flat_markup_amount" db:"flat_markup_amount"`
	PickupFeeAmount   float64 `json:"pickup_fee_amount" db:"pickup_fee_amount"`

	// Discounts
	CarrierDiscountPercent     sql.NullFloat64 `json:"carrier_discount_percent,omitempty" db:"carrier_discount_percent"`
	PassThroughDiscountPercent sql.NullFloat64 `json:"pass_through_discount_percent,omitempty" db:"pass_through_discount_percent"`

	// Fuel Surcharge
	FuelSurchargePercent   sql.NullFloat64 `json:"fuel_surcharge_percent,omitempty" db:"fuel_surcharge_percent"`
	FuelSurchargeUpdatedAt sql.NullTime    `json:"fuel_surcharge_updated_at,omitempty" db:"fuel_surcharge_updated_at"`

	// Status
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CarrierServiceLevel represents a service level option (Ground, Express, etc.)
type CarrierServiceLevel struct {
	ID         string    `json:"id" db:"id"`
	CarrierID  string    `json:"carrier_id" db:"carrier_id"`

	// Identity
	Code         string         `json:"code" db:"code"`
	CarrierCode  string         `json:"carrier_code" db:"carrier_code"`
	Name         string         `json:"name" db:"name"`
	DisplayName  sql.NullString `json:"display_name,omitempty" db:"display_name"`

	// Classification
	Category ServiceLevelCategory `json:"category" db:"category"`

	// Delivery Estimates
	MinTransitDays         sql.NullInt32  `json:"min_transit_days,omitempty" db:"min_transit_days"`
	MaxTransitDays         sql.NullInt32  `json:"max_transit_days,omitempty" db:"max_transit_days"`
	DeliveryTimeDescription sql.NullString `json:"delivery_time_description,omitempty" db:"delivery_time_description"`

	// Scope
	IsDomestic           bool              `json:"is_domestic" db:"is_domestic"`
	IsInternational      bool              `json:"is_international" db:"is_international"`
	OriginCountries      []string          `json:"origin_countries,omitempty" db:"origin_countries"`
	DestinationCountries []string          `json:"destination_countries,omitempty" db:"destination_countries"`

	// Capabilities
	SupportsPickup            bool `json:"supports_pickup" db:"supports_pickup"`
	SupportsSaturdayDelivery  bool `json:"supports_saturday_delivery" db:"supports_saturday_delivery"`
	SupportsSignature         bool `json:"supports_signature" db:"supports_signature"`
	SupportsInsurance         bool `json:"supports_insurance" db:"supports_insurance"`
	RequiresDimensions        bool `json:"requires_dimensions" db:"requires_dimensions"`

	// Restrictions
	MaxWeightKg      sql.NullFloat64 `json:"max_weight_kg,omitempty" db:"max_weight_kg"`
	MaxLengthCm      sql.NullFloat64 `json:"max_length_cm,omitempty" db:"max_length_cm"`
	MaxDeclaredValue sql.NullFloat64 `json:"max_declared_value,omitempty" db:"max_declared_value"`

	// Status
	IsEnabled bool      `json:"is_enabled" db:"is_enabled"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CarrierPackageType represents a package type option
type CarrierPackageType struct {
	ID               string         `json:"id" db:"id"`
	CarrierID        string         `json:"carrier_id" db:"carrier_id"`
	Code             string         `json:"code" db:"code"`
	CarrierCode      string         `json:"carrier_code" db:"carrier_code"`
	Name             string         `json:"name" db:"name"`
	DisplayName      sql.NullString `json:"display_name,omitempty" db:"display_name"`
	IsCustom         bool           `json:"is_custom" db:"is_custom"`
	IsCarrierProvided bool          `json:"is_carrier_provided" db:"is_carrier_provided"`

	// Dimensions (for carrier-provided packaging)
	LengthCm    sql.NullFloat64 `json:"length_cm,omitempty" db:"length_cm"`
	WidthCm     sql.NullFloat64 `json:"width_cm,omitempty" db:"width_cm"`
	HeightCm    sql.NullFloat64 `json:"height_cm,omitempty" db:"height_cm"`
	MaxWeightKg sql.NullFloat64 `json:"max_weight_kg,omitempty" db:"max_weight_kg"`

	// Status
	IsEnabled bool      `json:"is_enabled" db:"is_enabled"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsAvailable checks if carrier is available for use
func (c *Carrier) IsAvailable() bool {
	return c.IsEnabled && c.IsActive
}

// GetDisplayName returns display name or falls back to name
func (c *Carrier) GetDisplayName() string {
	if c.DisplayName.Valid && c.DisplayName.String != "" {
		return c.DisplayName.String
	}
	return c.Name
}

// GetDisplayName returns display name or falls back to name for service level
func (s *CarrierServiceLevel) GetDisplayName() string {
	if s.DisplayName.Valid && s.DisplayName.String != "" {
		return s.DisplayName.String
	}
	return s.Name
}

// IsAvailable checks if service level is available for use
func (s *CarrierServiceLevel) IsAvailable() bool {
	return s.IsEnabled && s.IsActive
}
