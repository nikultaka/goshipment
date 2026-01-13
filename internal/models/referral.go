package models

import (
	"database/sql"
	"time"
)

// ReferralRedemptionStatus represents the status of a referral redemption
type ReferralRedemptionStatus string

const (
	ReferralStatusActive  ReferralRedemptionStatus = "active"  // valid, reward eligible
	ReferralStatusRevoked ReferralRedemptionStatus = "revoked" // revoked by admin (fraud/abuse)
	ReferralStatusExpired ReferralRedemptionStatus = "expired" // no longer eligible (policy-based)
)

// ReferralRedemption represents a referral redemption record
type ReferralRedemption struct {
	ID                   string                   `json:"id" db:"id"`
	ReferenceNumber      string                   `json:"reference_number" db:"reference_number"` // REF-XXXXXXXX
	OrganizationID       sql.NullString           `json:"organization_id,omitempty" db:"organization_id"`
	ReferrerUserID       string                   `json:"referrer_user_id" db:"referrer_user_id"`
	ReferredUserID       string                   `json:"referred_user_id" db:"referred_user_id"`
	Status               ReferralRedemptionStatus `json:"status" db:"status"`
	RedeemedAt           time.Time                `json:"redeemed_at" db:"redeemed_at"`
	CommissionPlanID     sql.NullString           `json:"commission_plan_id,omitempty" db:"commission_plan_id"`
	CommissionNumber     string                   `json:"commission_number" db:"commission_number"`
	CommissionBase       CommissionBase           `json:"commission_base" db:"commission_base"`
	CommissionValueType  CommissionValueType      `json:"commission_value_type" db:"commission_value_type"`
	CommissionValue      float64                  `json:"commission_value" db:"commission_value"`
	SnapshotTakenAt      time.Time                `json:"snapshot_taken_at" db:"snapshot_taken_at"`
	Metadata             sql.NullString           `json:"metadata,omitempty" db:"metadata"` // JSONB
	CreatedAt            time.Time                `json:"created_at" db:"created_at"`
}

// IsActive returns true if referral is active and eligible for rewards
func (r *ReferralRedemption) IsActive() bool {
	return r.Status == ReferralStatusActive
}

// IsRevoked returns true if referral was revoked by admin
func (r *ReferralRedemption) IsRevoked() bool {
	return r.Status == ReferralStatusRevoked
}

// IsExpired returns true if referral has expired
func (r *ReferralRedemption) IsExpired() bool {
	return r.Status == ReferralStatusExpired
}
