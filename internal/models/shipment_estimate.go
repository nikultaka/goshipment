package models

import (
	"database/sql"
	"time"
)

// EstimateSource represents the source of the estimate
type EstimateSource string

const (
	EstimateSourceFrontend  EstimateSource = "frontend"
	EstimateSourceDashboard EstimateSource = "dashboard"
	EstimateSourceAPI       EstimateSource = "api"
	EstimateSourceAdmin     EstimateSource = "admin"
)

// WeightUnit represents weight measurement unit
type WeightUnit string

const (
	WeightUnitKg WeightUnit = "kg"
	WeightUnitLb WeightUnit = "lb"
)

// DimensionUnit represents dimension measurement unit
type DimensionUnit string

const (
	DimensionUnitCm DimensionUnit = "cm"
	DimensionUnitIn DimensionUnit = "in"
)

// ShipmentEstimate represents a shipment rate quote
// Uses ACTUAL database schema from migration 026_shipment_estimates.sql
type ShipmentEstimate struct {
	ID             string         `json:"id" db:"id"`
	EstimateNumber sql.NullString `json:"estimate_number,omitempty" db:"estimate_number"`

	// Ownership & Origin
	OrganizationID sql.NullString `json:"organization_id,omitempty" db:"organization_id"`
	UserID         sql.NullString `json:"user_id,omitempty" db:"user_id"`
	Source         EstimateSource `json:"source" db:"source"`

	// Optional address book links (for UX convenience)
	FromAddressBookID sql.NullString `json:"from_address_book_id,omitempty" db:"from_address_book_id"`
	ToAddressBookID   sql.NullString `json:"to_address_book_id,omitempty" db:"to_address_book_id"`

	// From address snapshot (individual columns, NOT JSONB)
	FromCountry    string         `json:"from_country" db:"from_country"`
	FromRegionCode sql.NullString `json:"from_region_code,omitempty" db:"from_region_code"`
	FromCity       sql.NullString `json:"from_city,omitempty" db:"from_city"`
	FromPostalCode string         `json:"from_postal_code" db:"from_postal_code"`

	// To address snapshot (individual columns, NOT JSONB)
	ToCountry    string         `json:"to_country" db:"to_country"`
	ToRegionCode sql.NullString `json:"to_region_code,omitempty" db:"to_region_code"`
	ToCity       sql.NullString `json:"to_city,omitempty" db:"to_city"`
	ToPostalCode string         `json:"to_postal_code" db:"to_postal_code"`

	// Package aggregates (packages stored in separate shipment_packages table)
	TotalPackages int     `json:"total_packages" db:"total_packages"`
	TotalWeight   float64 `json:"total_weight" db:"total_weight"`
	WeightUnit    string  `json:"weight_unit" db:"weight_unit"`

	// Pricing totals
	SubtotalAmount sql.NullFloat64 `json:"subtotal_amount,omitempty" db:"subtotal_amount"`
	TaxesAmount    sql.NullFloat64 `json:"taxes_amount,omitempty" db:"taxes_amount"`
	TotalAmount    sql.NullFloat64 `json:"total_amount,omitempty" db:"total_amount"`
	Currency       string          `json:"currency" db:"currency"`

	// Validity & locking
	ExpiresAt time.Time    `json:"expires_at" db:"expires_at"`
	IsLocked  bool         `json:"is_locked" db:"is_locked"`
	LockedAt  sql.NullTime `json:"locked_at,omitempty" db:"locked_at"`

	// Selected rate (when user chooses a specific carrier/service)
	SelectedCarrierID        sql.NullString  `json:"selected_carrier_id,omitempty" db:"selected_carrier_id"`
	SelectedServiceLevelName sql.NullString  `json:"selected_servicelevel_name,omitempty" db:"selected_servicelevel_name"`
	SelectedServiceLevelCode sql.NullString  `json:"selected_servicelevel_code,omitempty" db:"selected_servicelevel_code"`
	SelectedAmount           sql.NullFloat64 `json:"selected_amount,omitempty" db:"selected_amount"`
	RateSelectedAt           sql.NullTime    `json:"rate_selected_at,omitempty" db:"rate_selected_at"`

	// Carrier rates snapshot (JSONB - stores all rate quotes)
	CarrierRatesSnapshot sql.NullString `json:"carrier_rates_snapshot,omitempty" db:"carrier_rates_snapshot"`

	// Audit
	IsDeleted bool      `json:"is_deleted" db:"is_deleted"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ShipmentPackage represents a package in a shipment or estimate
// Uses ACTUAL database schema from migration 027_shipment_packages.sql
type ShipmentPackage struct {
	ID            string         `json:"id" db:"id"`
	PackageNumber sql.NullString `json:"package_number,omitempty" db:"package_number"`

	// Parent linkage (either estimate OR shipment, not both)
	ShipmentEstimateID sql.NullString `json:"shipment_estimate_id,omitempty" db:"shipment_estimate_id"`
	ShipmentID         sql.NullString `json:"shipment_id,omitempty" db:"shipment_id"`

	// Package ordering (1-based index)
	PackageIndex int            `json:"package_index" db:"package_index"`
	PackageName  sql.NullString `json:"package_name,omitempty" db:"package_name"`

	// Dimensions
	Length        float64 `json:"length" db:"length"`
	Width         float64 `json:"width" db:"width"`
	Height        float64 `json:"height" db:"height"`
	DimensionUnit string  `json:"dimension_unit" db:"dimension_unit"`

	// Weight
	Weight     float64 `json:"weight" db:"weight"`
	WeightUnit string  `json:"weight_unit" db:"weight_unit"`

	// Declared value (optional, for insurance)
	DeclaredValue    sql.NullFloat64 `json:"declared_value,omitempty" db:"declared_value"`
	DeclaredCurrency sql.NullString  `json:"declared_currency,omitempty" db:"declared_currency"`

	// Carrier object links (populated when interacting with carrier APIs)
	CarrierParcelID sql.NullString `json:"carrier_parcel_id,omitempty" db:"carrier_parcel_id"`

	// Label & tracking (per package/parcel)
	CarrierLabelTransactionID sql.NullString `json:"carrier_label_transaction_id,omitempty" db:"carrier_label_transaction_id"`
	LabelURL                  sql.NullString `json:"label_url,omitempty" db:"label_url"`
	LabelFileType             sql.NullString `json:"label_file_type,omitempty" db:"label_file_type"`
	TrackingNumber            sql.NullString `json:"tracking_number,omitempty" db:"tracking_number"`
	TrackingURL               sql.NullString `json:"tracking_url,omitempty" db:"tracking_url"`
	LabelPurchasedAt          sql.NullTime   `json:"label_purchased_at,omitempty" db:"label_purchased_at"`

	// Package delivery status
	PackageStatus string         `json:"package_status" db:"package_status"`
	FailureReason sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`
	CarrierError  sql.NullString `json:"carrier_error,omitempty" db:"carrier_error"`

	// Audit
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CarrierRate represents a single rate quote from a carrier
// This is the structure stored in carrier_rates_snapshot JSONB
type CarrierRate struct {
	CarrierID              string  `json:"carrier_id"`
	CarrierCode            string  `json:"carrier_code"`
	CarrierName            string  `json:"carrier_name"`
	ServiceCode            string  `json:"service_code"`
	ServiceName            string  `json:"service_name"`
	ServiceCategory        string  `json:"service_category"`
	BaseRate               float64 `json:"base_rate"`
	FuelSurcharge          float64 `json:"fuel_surcharge"`
	OtherFees              float64 `json:"other_fees"`
	Taxes                  float64 `json:"taxes"`
	TotalAmount            float64 `json:"total_amount"`
	Currency               string  `json:"currency"`
	EstimatedDeliveryDate  string  `json:"estimated_delivery_date,omitempty"`
	TransitDays            int     `json:"transit_days,omitempty"`
	SupportsPickup         bool    `json:"supports_pickup"`
	SupportsSignature      bool    `json:"supports_signature"`
	SupportsSaturdayDelivery bool  `json:"supports_saturday_delivery,omitempty"`
}

// CarrierRatesSnapshot represents the complete rates snapshot structure
type CarrierRatesSnapshot struct {
	Rates     []CarrierRate `json:"rates"`
	QuotedAt  time.Time     `json:"quoted_at"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// Helper methods for ShipmentEstimate

// IsExpired checks if the estimate has expired
func (e *ShipmentEstimate) IsExpired() bool {
	return time.Now().UTC().After(e.ExpiresAt)
}

// IsGuest checks if estimate was created by a guest (no authentication)
func (e *ShipmentEstimate) IsGuest() bool {
	return !e.OrganizationID.Valid && !e.UserID.Valid
}

// GetEstimateNumber returns the estimate number or empty string
func (e *ShipmentEstimate) GetEstimateNumber() string {
	if e.EstimateNumber.Valid {
		return e.EstimateNumber.String
	}
	return ""
}

// BelongsToOrganization checks if estimate belongs to given organization
func (e *ShipmentEstimate) BelongsToOrganization(organizationID string) bool {
	return e.OrganizationID.Valid && e.OrganizationID.String == organizationID
}

// CanBeAccessedBy checks if estimate can be accessed by user/org
func (e *ShipmentEstimate) CanBeAccessedBy(organizationID string, isGuest bool) bool {
	// Guest estimates can only be accessed by ID (no org check)
	if e.IsGuest() {
		return true
	}

	// Org estimates can only be accessed by same org
	if !isGuest && e.BelongsToOrganization(organizationID) {
		return true
	}

	return false
}
