package labels

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// PurchaseLabelsRequest represents the request to purchase labels
type PurchaseLabelsRequest struct {
	ShipmentID      string `json:"shipment_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	IdempotencyKey  string `json:"idempotency_key" binding:"required,min=1,max=100" example:"purchase_2026-01-11_abc123"`
	UseCarrierBilling *bool `json:"use_carrier_billing" example:"false"` // If true, skip wallet payment
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// PackageLabelInfo represents label and tracking info for a single package
type PackageLabelInfo struct {
	PackageID       string     `json:"package_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PackageNumber   string     `json:"package_number" example:"PKG-K9R2B7X4-01"`
	PackageIndex    int        `json:"package_index" example:"1"`
	PackageName     *string    `json:"package_name,omitempty" example:"Box 1"`

	// Label Information
	LabelURL        *string    `json:"label_url,omitempty" example:"https://cdn.shippo.com/labels/abc123.pdf"`
	LabelFileType   *string    `json:"label_file_type,omitempty" example:"pdf"`
	TrackingNumber  *string    `json:"tracking_number,omitempty" example:"1Z999AA10123456784"`
	TrackingURL     *string    `json:"tracking_url,omitempty" example:"https://track.ups.com/track?tracknum=1Z999AA10123456784"`

	// Carrier Information
	CarrierParcelID          *string `json:"carrier_parcel_id,omitempty" example:"abc123def456"`
	CarrierLabelTransactionID *string `json:"carrier_label_transaction_id,omitempty" example:"txn_123456"`

	// Status
	PackageStatus   string     `json:"package_status" example:"label_purchased"`
	LabelPurchasedAt *time.Time `json:"label_purchased_at,omitempty" example:"2026-01-11T10:30:00Z"`
}

// PurchaseLabelsResponse represents the successful label purchase response
type PurchaseLabelsResponse struct {
	ShipmentID       string             `json:"shipment_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShipmentNumber   string             `json:"shipment_number" example:"SIF-8F3K9A2Q"`
	ShipmentStatus   string             `json:"shipment_status" example:"label_purchased"`

	// Payment Information
	WalletTransactionID *string `json:"wallet_transaction_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	TransactionNumber   *string `json:"transaction_number,omitempty" example:"TRX-ABC12345"`
	PaymentStatus       string  `json:"payment_status" example:"paid"`
	TotalAmount         float64 `json:"total_amount" example:"45.50"`
	Currency            string  `json:"currency" example:"CAD"`

	// Master Tracking (if available from carrier)
	MasterTrackingNumber *string `json:"master_tracking_number,omitempty" example:"1Z999AA10123456784"`
	MasterTrackingURL    *string `json:"master_tracking_url,omitempty" example:"https://track.ups.com/track?tracknum=1Z999AA10123456784"`

	// Package Labels
	Packages []PackageLabelInfo `json:"packages"`

	// Metadata
	PurchasedAt time.Time `json:"purchased_at" example:"2026-01-11T10:30:00Z"`
	PurchasedBy string    `json:"purchased_by" example:"550e8400-e29b-41d4-a716-446655440002"`
}

// GetLabelResponse represents the response for getting label info
type GetLabelResponse struct {
	PackageID       string     `json:"package_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PackageNumber   string     `json:"package_number" example:"PKG-K9R2B7X4-01"`
	PackageIndex    int        `json:"package_index" example:"1"`
	PackageName     *string    `json:"package_name,omitempty" example:"Box 1"`

	// Shipment Context
	ShipmentID      string     `json:"shipment_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShipmentNumber  string     `json:"shipment_number" example:"SIF-8F3K9A2Q"`
	ShipmentStatus  string     `json:"shipment_status" example:"label_purchased"`

	// Label Information
	LabelURL        *string    `json:"label_url,omitempty" example:"https://cdn.shippo.com/labels/abc123.pdf"`
	LabelFileType   *string    `json:"label_file_type,omitempty" example:"pdf"`
	TrackingNumber  *string    `json:"tracking_number,omitempty" example:"1Z999AA10123456784"`
	TrackingURL     *string    `json:"tracking_url,omitempty" example:"https://track.ups.com/track?tracknum=1Z999AA10123456784"`

	// Carrier Information
	CarrierParcelID          *string `json:"carrier_parcel_id,omitempty" example:"abc123def456"`
	CarrierLabelTransactionID *string `json:"carrier_label_transaction_id,omitempty" example:"txn_123456"`

	// Package Details
	Length         float64 `json:"length" example:"10.5"`
	Width          float64 `json:"width" example:"8.0"`
	Height         float64 `json:"height" example:"6.0"`
	DimensionUnit  string  `json:"dimension_unit" example:"in"`
	Weight         float64 `json:"weight" example:"2.5"`
	WeightUnit     string  `json:"weight_unit" example:"lb"`
	DeclaredValue  *float64 `json:"declared_value,omitempty" example:"100.00"`
	DeclaredCurrency *string `json:"declared_currency,omitempty" example:"CAD"`

	// Status
	PackageStatus   string     `json:"package_status" example:"label_purchased"`
	LabelPurchasedAt *time.Time `json:"label_purchased_at,omitempty" example:"2026-01-11T10:30:00Z"`
	FailureReason   *string    `json:"failure_reason,omitempty"`
}
