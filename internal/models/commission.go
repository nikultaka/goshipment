package models

import (
	"database/sql"
	"time"
)

// CommissionBase represents the base calculation for commission
type CommissionBase string

const (
	CommissionBaseProfit      CommissionBase = "profit"        // (revenue - cost) * %
	CommissionBaseRevenue     CommissionBase = "revenue"       // revenue * %
	CommissionBasePerShipment CommissionBase = "per_shipment" // fixed amount per completed shipment
)

// CommissionValueType represents the type of commission value
type CommissionValueType string

const (
	CommissionValueTypePercent    CommissionValueType = "percent"
	CommissionValueTypeFixedAmount CommissionValueType = "fixed_amount"
)

// CommissionTxStatus represents the status of a commission transaction
type CommissionTxStatus string

const (
	CommissionTxStatusPending    CommissionTxStatus = "pending"    // not completed yet (not payable)
	CommissionTxStatusEligible   CommissionTxStatus = "eligible"   // completed and qualifies (payable)
	CommissionTxStatusProcessing CommissionTxStatus = "processing" // locked for payout processing
	CommissionTxStatusPaid       CommissionTxStatus = "paid"       // included in a completed payout
	CommissionTxStatusVoided     CommissionTxStatus = "voided"     // invalidated
)

// CommissionPayoutRequestStatus represents the status of a payout request
type CommissionPayoutRequestStatus string

const (
	PayoutRequestStatusPending   CommissionPayoutRequestStatus = "pending"   // requested, awaiting review
	PayoutRequestStatusApproved  CommissionPayoutRequestStatus = "approved"  // approved, ready to process
	PayoutRequestStatusRejected  CommissionPayoutRequestStatus = "rejected"  // rejected (final)
	PayoutRequestStatusCancelled CommissionPayoutRequestStatus = "cancelled" // cancelled before processing
)

// CommissionPayoutStatus represents the status of a payout
type CommissionPayoutStatus string

const (
	PayoutStatusPending    CommissionPayoutStatus = "pending"    // created
	PayoutStatusProcessing CommissionPayoutStatus = "processing" // sending money
	PayoutStatusPaid       CommissionPayoutStatus = "paid"       // paid successfully
	PayoutStatusRejected   CommissionPayoutStatus = "rejected"   // rejected/blocked
	PayoutStatusFailed     CommissionPayoutStatus = "failed"     // attempted but failed
	PayoutStatusCancelled  CommissionPayoutStatus = "cancelled"  // cancelled before payment
)

// CommissionPayoutMethod represents the payout method
type CommissionPayoutMethod string

const (
	PayoutMethodStripe CommissionPayoutMethod = "stripe"
	PayoutMethodPayPal CommissionPayoutMethod = "paypal"
	PayoutMethodManual CommissionPayoutMethod = "manual"
)

// CommissionTransaction represents a commission earnings ledger entry
type CommissionTransaction struct {
	ID                           string              `json:"id" db:"id"`
	CommissionTransactionNumber  string              `json:"commission_transaction_number" db:"commission_transaction_number"`
	OrganizationID               string              `json:"organization_id" db:"organization_id"`
	BeneficiaryUserID            string              `json:"beneficiary_user_id" db:"beneficiary_user_id"`
	SourceUserID                 sql.NullString      `json:"source_user_id,omitempty" db:"source_user_id"`
	ShipmentID                   string              `json:"shipment_id" db:"shipment_id"`
	InvoiceID                    sql.NullString      `json:"invoice_id,omitempty" db:"invoice_id"`
	CommissionPlanID             string              `json:"commission_plan_id" db:"commission_plan_id"`
	CommissionNumber             string              `json:"commission_number" db:"commission_number"`
	CommissionBase               CommissionBase      `json:"commission_base" db:"commission_base"`
	CommissionValueType          CommissionValueType `json:"commission_value_type" db:"commission_value_type"`
	CommissionValue              float64             `json:"commission_value" db:"commission_value"`
	Currency                     string              `json:"currency" db:"currency"`
	RevenueAmount                float64             `json:"revenue_amount" db:"revenue_amount"`
	CostAmount                   float64             `json:"cost_amount" db:"cost_amount"`
	ProfitAmount                 float64             `json:"profit_amount" db:"profit_amount"`
	CommissionAmount             float64             `json:"commission_amount" db:"commission_amount"`
	Status                       CommissionTxStatus  `json:"status" db:"status"`
	EarnedAt                     time.Time           `json:"earned_at" db:"earned_at"`
	EligibleAt                   sql.NullTime        `json:"eligible_at,omitempty" db:"eligible_at"`
	ProcessingAt                 sql.NullTime        `json:"processing_at,omitempty" db:"processing_at"`
	PaidAt                       sql.NullTime        `json:"paid_at,omitempty" db:"paid_at"`
	PayoutID                     sql.NullString      `json:"payout_id,omitempty" db:"payout_id"`
	Metadata                     sql.NullString      `json:"metadata,omitempty" db:"metadata"` // JSONB
	CreatedAt                    time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time           `json:"updated_at" db:"updated_at"`
}

// CommissionPayoutRequest represents a user's request for commission payout
type CommissionPayoutRequest struct {
	ID                    string                        `json:"id" db:"id"`
	PayoutRequestNumber   string                        `json:"payout_request_number" db:"payout_request_number"`
	OrganizationID        string                        `json:"organization_id" db:"organization_id"`
	RequestedByUserID     string                        `json:"requested_by_user_id" db:"requested_by_user_id"`
	BeneficiaryUserID     string                        `json:"beneficiary_user_id" db:"beneficiary_user_id"`
	AmountRequested       float64                       `json:"amount_requested" db:"amount_requested"`
	Currency              string                        `json:"currency" db:"currency"`
	Method                CommissionPayoutMethod        `json:"method" db:"method"`
	Status                CommissionPayoutRequestStatus `json:"status" db:"status"`
	RequestNote           sql.NullString                `json:"request_note,omitempty" db:"request_note"`
	ReviewedByUserID      sql.NullString                `json:"reviewed_by_user_id,omitempty" db:"reviewed_by_user_id"`
	ReviewedAt            sql.NullTime                  `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewNote            sql.NullString                `json:"review_note,omitempty" db:"review_note"`
	CreatedAt             time.Time                     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time                     `json:"updated_at" db:"updated_at"`
}

// CommissionPayout represents an actual payout made to a beneficiary
type CommissionPayout struct {
	ID                 string                 `json:"id" db:"id"`
	PayoutNumber       string                 `json:"payout_number" db:"payout_number"`
	OrganizationID     string                 `json:"organization_id" db:"organization_id"`
	BeneficiaryUserID  string                 `json:"beneficiary_user_id" db:"beneficiary_user_id"`
	PayoutRequestID    sql.NullString         `json:"payout_request_id,omitempty" db:"payout_request_id"`
	Method             CommissionPayoutMethod `json:"method" db:"method"`
	Amount             float64                `json:"amount" db:"amount"`
	Currency           string                 `json:"currency" db:"currency"`
	Status             CommissionPayoutStatus `json:"status" db:"status"`
	PayoutNote         sql.NullString         `json:"payout_note,omitempty" db:"payout_note"`
	UserVisibleNote    sql.NullString         `json:"user_visible_note,omitempty" db:"user_visible_note"`
	StripeTransferID   sql.NullString         `json:"stripe_transfer_id,omitempty" db:"stripe_transfer_id"`
	StripePayoutID     sql.NullString         `json:"stripe_payout_id,omitempty" db:"stripe_payout_id"`
	PayPalPayoutBatchID sql.NullString        `json:"paypal_payout_batch_id,omitempty" db:"paypal_payout_batch_id"`
	PayPalTransactionID sql.NullString        `json:"paypal_transaction_id,omitempty" db:"paypal_transaction_id"`
	ManualReference    sql.NullString         `json:"manual_reference,omitempty" db:"manual_reference"`
	CreatedByUserID    sql.NullString         `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	ProcessedByUserID  sql.NullString         `json:"processed_by_user_id,omitempty" db:"processed_by_user_id"`
	RequestedAt        time.Time              `json:"requested_at" db:"requested_at"`
	ProcessedAt        sql.NullTime           `json:"processed_at,omitempty" db:"processed_at"`
	PaidAt             sql.NullTime           `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// Helper methods

// IsPayable returns true if the commission transaction is eligible for payout
func (c *CommissionTransaction) IsPayable() bool {
	return c.Status == CommissionTxStatusEligible
}

// IsPaid returns true if the commission has been paid
func (c *CommissionTransaction) IsPaid() bool {
	return c.Status == CommissionTxStatusPaid
}

// IsProcessing returns true if the commission is being processed for payout
func (c *CommissionTransaction) IsProcessing() bool {
	return c.Status == CommissionTxStatusProcessing
}

// CanBeProcessed returns true if the payout request can be processed
func (p *CommissionPayoutRequest) CanBeProcessed() bool {
	return p.Status == PayoutRequestStatusApproved
}

// IsPending returns true if the payout request is pending review
func (p *CommissionPayoutRequest) IsPending() bool {
	return p.Status == PayoutRequestStatusPending
}
