package models

import (
	"database/sql"
	"time"
)

// ShipmentStatus represents the lifecycle state of a shipment
type ShipmentStatus string

const (
	// Pre-Label States
	ShipmentStatusDraft           ShipmentStatus = "draft"
	ShipmentStatusAwaitingPayment ShipmentStatus = "awaiting_payment"
	ShipmentStatusPaymentFailed   ShipmentStatus = "payment_failed"

	// Label & Pickup States
	ShipmentStatusLabelPurchased  ShipmentStatus = "label_purchased"
	ShipmentStatusReadyForPickup  ShipmentStatus = "ready_for_pickup"
	ShipmentStatusPickupScheduled ShipmentStatus = "pickup_scheduled"

	// In-Transit States
	ShipmentStatusInTransit      ShipmentStatus = "in_transit"
	ShipmentStatusOutForDelivery ShipmentStatus = "out_for_delivery"

	// Final States
	ShipmentStatusDelivered      ShipmentStatus = "delivered"
	ShipmentStatusDeliveryFailed ShipmentStatus = "delivery_failed"
	ShipmentStatusReturned       ShipmentStatus = "returned"

	// Terminal / Exception States
	ShipmentStatusCancelled ShipmentStatus = "cancelled"
	ShipmentStatusRefunded  ShipmentStatus = "refunded"
	ShipmentStatusVoided    ShipmentStatus = "voided"
)

// PaymentStatus represents the payment state of a shipment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// Shipment represents a shipment record in the database
type Shipment struct {
	// Primary Key
	ID string `json:"id" db:"id"`

	// Identifiers
	ShipmentNumber     string         `json:"shipment_number" db:"shipment_number"`
	OrganizationID     string         `json:"organization_id" db:"organization_id"`
	CreatedByUserID    string         `json:"created_by_user_id" db:"created_by_user_id"`
	ShipmentEstimateID string         `json:"shipment_estimate_id" db:"shipment_estimate_id"`

	// Carrier & Service
	CarrierID           string         `json:"carrier_id" db:"carrier_id"`
	ServicelevelCode    sql.NullString `json:"servicelevel_code,omitempty" db:"servicelevel_code"`
	ServicelevelName    sql.NullString `json:"servicelevel_name,omitempty" db:"servicelevel_name"`
	CarrierShipmentID   sql.NullString `json:"carrier_shipment_id,omitempty" db:"carrier_shipment_id"`
	CarrierPayload      sql.NullString `json:"carrier_payload,omitempty" db:"carrier_payload"` // JSONB
	CarrierRateID       sql.NullString `json:"carrier_rate_id,omitempty" db:"carrier_rate_id"`
	CarrierPurchaseID   sql.NullString `json:"carrier_purchase_id,omitempty" db:"carrier_purchase_id"`

	// Master Tracking
	MasterTrackingNumber sql.NullString `json:"master_tracking_number,omitempty" db:"master_tracking_number"`
	MasterTrackingURL    sql.NullString `json:"master_tracking_url,omitempty" db:"master_tracking_url"`

	// Address Book Links (optional)
	FromAddressBookID sql.NullString `json:"from_address_book_id,omitempty" db:"from_address_book_id"`
	ToAddressBookID   sql.NullString `json:"to_address_book_id,omitempty" db:"to_address_book_id"`

	// FROM Address Snapshot
	FromName         sql.NullString `json:"from_name,omitempty" db:"from_name"`
	FromCompany      sql.NullString `json:"from_company,omitempty" db:"from_company"`
	FromPhone        sql.NullString `json:"from_phone,omitempty" db:"from_phone"`
	FromEmail        sql.NullString `json:"from_email,omitempty" db:"from_email"`
	FromCountry      string         `json:"from_country" db:"from_country"`
	FromRegionCode   sql.NullString `json:"from_region_code,omitempty" db:"from_region_code"`
	FromCity         string         `json:"from_city" db:"from_city"`
	FromPostalCode   string         `json:"from_postal_code" db:"from_postal_code"`
	FromAddressLine1 string         `json:"from_address_line1" db:"from_address_line1"`
	FromAddressLine2 sql.NullString `json:"from_address_line2,omitempty" db:"from_address_line2"`

	// TO Address Snapshot
	ToName         sql.NullString `json:"to_name,omitempty" db:"to_name"`
	ToCompany      sql.NullString `json:"to_company,omitempty" db:"to_company"`
	ToPhone        sql.NullString `json:"to_phone,omitempty" db:"to_phone"`
	ToEmail        sql.NullString `json:"to_email,omitempty" db:"to_email"`
	ToCountry      string         `json:"to_country" db:"to_country"`
	ToRegionCode   sql.NullString `json:"to_region_code,omitempty" db:"to_region_code"`
	ToCity         string         `json:"to_city" db:"to_city"`
	ToPostalCode   string         `json:"to_postal_code" db:"to_postal_code"`
	ToAddressLine1 string         `json:"to_address_line1" db:"to_address_line1"`
	ToAddressLine2 sql.NullString `json:"to_address_line2,omitempty" db:"to_address_line2"`

	// Package Aggregates
	TotalPackages int     `json:"total_packages" db:"total_packages"`
	TotalWeight   float64 `json:"total_weight" db:"total_weight"`
	WeightUnit    string  `json:"weight_unit" db:"weight_unit"`

	// Delivery Options
	RequireSignature     bool           `json:"require_signature" db:"require_signature"`
	EmailNotifications   bool           `json:"email_notifications" db:"email_notifications"`
	DeliveryInstructions sql.NullString `json:"delivery_instructions,omitempty" db:"delivery_instructions"`

	// Pricing
	BaseAmount            float64        `json:"base_amount" db:"base_amount"`
	Currency              string         `json:"currency" db:"currency"`
	InsuranceAmount       sql.NullFloat64 `json:"insurance_amount,omitempty" db:"insurance_amount"`
	PickupFeeAmount       sql.NullFloat64 `json:"pickup_fee_amount,omitempty" db:"pickup_fee_amount"`
	TaxesAmount           sql.NullFloat64 `json:"taxes_amount,omitempty" db:"taxes_amount"`
	RetailRate            sql.NullFloat64 `json:"retail_rate,omitempty" db:"retail_rate"`
	RetailCurrency        sql.NullString  `json:"retail_currency,omitempty" db:"retail_currency"`
	AdjustmentsTotal      float64         `json:"adjustments_total" db:"adjustments_total"`
	AdjustmentsCurrency   sql.NullString  `json:"adjustments_currency,omitempty" db:"adjustments_currency"`
	HasPendingAdjustments bool            `json:"has_pending_adjustments" db:"has_pending_adjustments"`
	AdjustmentsBreakdown  sql.NullString  `json:"adjustments_breakdown,omitempty" db:"adjustments_breakdown"` // JSONB

	// Payment
	WalletTransactionID    sql.NullString `json:"wallet_transaction_id,omitempty" db:"wallet_transaction_id"`
	PaymentStatus          PaymentStatus  `json:"payment_status" db:"payment_status"`
	StripePaymentIntentID  sql.NullString `json:"stripe_payment_intent_id,omitempty" db:"stripe_payment_intent_id"`

	// Status
	ShipmentStatus ShipmentStatus `json:"shipment_status" db:"shipment_status"`
	FailureReason  sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`

	// Pickup
	PickupRequired bool           `json:"pickup_required" db:"pickup_required"`
	PickupID       sql.NullString `json:"pickup_id,omitempty" db:"pickup_id"`

	// UI Archival
	IsArchived       bool           `json:"is_archived" db:"is_archived"`
	ArchivedAt       sql.NullTime   `json:"archived_at,omitempty" db:"archived_at"`
	ArchivedByUserID sql.NullString `json:"archived_by_user_id,omitempty" db:"archived_by_user_id"`

	// Audit
	IdempotencyKey sql.NullString `json:"idempotency_key,omitempty" db:"idempotency_key"`
	Revision       int            `json:"revision" db:"revision"`
	IsDeleted      bool           `json:"is_deleted" db:"is_deleted"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// ShipmentEvent represents an event in the shipment timeline
type ShipmentEvent struct {
	ID             string         `json:"id" db:"id"`
	ShipmentID     string         `json:"shipment_id" db:"shipment_id"`
	OrganizationID string         `json:"organization_id" db:"organization_id"`
	PackageID      sql.NullString `json:"package_id,omitempty" db:"package_id"`
	PickupID       sql.NullString `json:"pickup_id,omitempty" db:"pickup_id"`
	EntityType     string         `json:"entity_type" db:"entity_type"` // shipment | package | pickup
	EventKey       string         `json:"event_key" db:"event_key"`     // e.g., "shipment.created"
	Source         string         `json:"source" db:"source"`           // system | user | admin | provider
	Severity       string         `json:"severity" db:"severity"`       // info | warning | error
	Title          sql.NullString `json:"title,omitempty" db:"title"`
	Message        sql.NullString `json:"message,omitempty" db:"message"`
	Provider       sql.NullString `json:"provider,omitempty" db:"provider"`
	ProviderEventID sql.NullString `json:"provider_event_id,omitempty" db:"provider_event_id"`
	ProviderReference sql.NullString `json:"provider_reference,omitempty" db:"provider_reference"`
	RawPayload     sql.NullString `json:"raw_payload,omitempty" db:"raw_payload"` // JSONB
	OccurredAt     time.Time      `json:"occurred_at" db:"occurred_at"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	ActorUserID    sql.NullString `json:"actor_user_id,omitempty" db:"actor_user_id"`
	ActorIP        sql.NullString `json:"actor_ip,omitempty" db:"actor_ip"`
	ActorUserAgent sql.NullString `json:"actor_user_agent,omitempty" db:"actor_user_agent"`
}

// Helper methods for Shipment

// CanBeCancelled returns true if the shipment can be cancelled (pre-label only)
func (s *Shipment) CanBeCancelled() bool {
	preLabelStatuses := []ShipmentStatus{
		ShipmentStatusDraft,
		ShipmentStatusAwaitingPayment,
		ShipmentStatusPaymentFailed,
	}
	for _, status := range preLabelStatuses {
		if s.ShipmentStatus == status {
			return true
		}
	}
	return false
}

// CanBeVoided returns true if the shipment can be voided (post-label, not yet delivered)
func (s *Shipment) CanBeVoided() bool {
	voidableStatuses := []ShipmentStatus{
		ShipmentStatusLabelPurchased,
		ShipmentStatusReadyForPickup,
		ShipmentStatusPickupScheduled,
	}
	for _, status := range voidableStatuses {
		if s.ShipmentStatus == status {
			return true
		}
	}
	return false
}

// CanBeArchived returns true if the shipment can be archived
func (s *Shipment) CanBeArchived() bool {
	// Can archive any shipment except if already archived
	return !s.IsArchived
}

// IsTerminal returns true if the shipment is in a terminal state
func (s *Shipment) IsTerminal() bool {
	terminalStatuses := []ShipmentStatus{
		ShipmentStatusDelivered,
		ShipmentStatusReturned,
		ShipmentStatusCancelled,
		ShipmentStatusVoided,
		ShipmentStatusRefunded,
	}
	for _, status := range terminalStatuses {
		if s.ShipmentStatus == status {
			return true
		}
	}
	return false
}

// GetTotalAmount calculates the total amount including base, fees, and taxes
func (s *Shipment) GetTotalAmount() float64 {
	total := s.BaseAmount

	if s.InsuranceAmount.Valid {
		total += s.InsuranceAmount.Float64
	}

	if s.PickupFeeAmount.Valid {
		total += s.PickupFeeAmount.Float64
	}

	if s.TaxesAmount.Valid {
		total += s.TaxesAmount.Float64
	}

	total += s.AdjustmentsTotal

	return total
}
