package models

import (
	"database/sql"
	"time"
)

// DiscountType represents the type of discount
type DiscountType string

const (
	DiscountTypePercent      DiscountType = "percent"
	DiscountTypeFixedAmount  DiscountType = "fixed_amount"
	DiscountTypeFreeShipping DiscountType = "free_shipping"
)

// DiscountScope represents what the discount applies to
type DiscountScope string

const (
	DiscountScopeInvoice  DiscountScope = "invoice"
	DiscountScopeShipment DiscountScope = "shipment"
)

// DiscountRedemptionStatus represents the status of a discount redemption
type DiscountRedemptionStatus string

const (
	DiscountRedemptionStatusApplied DiscountRedemptionStatus = "applied"
	DiscountRedemptionStatusVoided  DiscountRedemptionStatus = "voided"
)

// Discount represents a discount/promo code
type Discount struct {
	ID                     string         `json:"id" db:"id"`
	DiscountNumber         sql.NullString `json:"discount_number" db:"discount_number"`
	OrganizationID         string         `json:"organization_id" db:"organization_id"`
	CreatedByUserID        sql.NullString `json:"created_by_user_id" db:"created_by_user_id"`
	Code                   string         `json:"code" db:"code"`
	Name                   sql.NullString `json:"name" db:"name"`
	Description            sql.NullString `json:"description" db:"description"`
	DiscountType           DiscountType   `json:"discount_type" db:"discount_type"`
	Value                  float64        `json:"value" db:"value"`
	Currency               string         `json:"currency" db:"currency"`
	Scope                  DiscountScope  `json:"scope" db:"scope"`
	MinSubtotalAmount      float64        `json:"min_subtotal_amount" db:"min_subtotal_amount"`
	MaxDiscountAmount      sql.NullFloat64 `json:"max_discount_amount" db:"max_discount_amount"`
	AllowedCountry         sql.NullString `json:"allowed_country" db:"allowed_country"`
	MaxUsesTotal           sql.NullInt64  `json:"max_uses_total" db:"max_uses_total"`
	MaxUsesPerUser         sql.NullInt64  `json:"max_uses_per_user" db:"max_uses_per_user"`
	MaxTotalDiscountAmount sql.NullFloat64 `json:"max_total_discount_amount" db:"max_total_discount_amount"`
	StartsAt               sql.NullTime   `json:"starts_at" db:"starts_at"`
	EndsAt                 sql.NullTime   `json:"ends_at" db:"ends_at"`
	IsActive               bool           `json:"is_active" db:"is_active"`
	FirstTimeOnly          bool           `json:"first_time_only" db:"first_time_only"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`

	// Computed fields (not in DB)
	UsageCount int `json:"usage_count" db:"-"`
}

// DiscountRedemption represents a discount redemption record
type DiscountRedemption struct {
	ID               string                   `json:"id" db:"id"`
	UsageNumber      sql.NullString           `json:"usage_number" db:"usage_number"`
	OrganizationID   string                   `json:"organization_id" db:"organization_id"`
	DiscountID       string                   `json:"discount_id" db:"discount_id"`
	RedeemedByUserID string                   `json:"redeemed_by_user_id" db:"redeemed_by_user_id"`
	InvoiceID        sql.NullString           `json:"invoice_id" db:"invoice_id"`
	ShipmentID       sql.NullString           `json:"shipment_id" db:"shipment_id"`
	Status           DiscountRedemptionStatus `json:"status" db:"status"`
	Currency         string                   `json:"currency" db:"currency"`
	DiscountAmount   float64                  `json:"discount_amount" db:"discount_amount"`
	RedeemedAt       time.Time                `json:"redeemed_at" db:"redeemed_at"`
	CreatedAt        time.Time                `json:"created_at" db:"created_at"`
}

// IsValid checks if the discount is currently valid
func (d *Discount) IsValid() bool {
	if !d.IsActive {
		return false
	}

	now := time.Now()

	if d.StartsAt.Valid && now.Before(d.StartsAt.Time) {
		return false
	}

	if d.EndsAt.Valid && now.After(d.EndsAt.Time) {
		return false
	}

	return true
}

// CalculateDiscount calculates the discount amount for a given subtotal
func (d *Discount) CalculateDiscount(subtotal float64) float64 {
	if subtotal < d.MinSubtotalAmount {
		return 0
	}

	var discountAmount float64

	switch d.DiscountType {
	case DiscountTypePercent:
		discountAmount = subtotal * (d.Value / 100)
	case DiscountTypeFixedAmount:
		discountAmount = d.Value
	case DiscountTypeFreeShipping:
		// Free shipping - the value represents shipping cost to waive
		discountAmount = d.Value
	}

	// Apply max discount cap if set
	if d.MaxDiscountAmount.Valid && discountAmount > d.MaxDiscountAmount.Float64 {
		discountAmount = d.MaxDiscountAmount.Float64
	}

	// Don't exceed the subtotal
	if discountAmount > subtotal {
		discountAmount = subtotal
	}

	return discountAmount
}
