package models

import (
	"database/sql"
	"time"
)

// CarrierRefundStatus represents the status of a carrier refund
type CarrierRefundStatus string

const (
	CarrierRefundStatusRequested  CarrierRefundStatus = "requested"
	CarrierRefundStatusProcessing CarrierRefundStatus = "processing"
	CarrierRefundStatusCompleted  CarrierRefundStatus = "completed"
	CarrierRefundStatusRejected   CarrierRefundStatus = "rejected"
	CarrierRefundStatusFailed     CarrierRefundStatus = "failed"
	CarrierRefundStatusCancelled  CarrierRefundStatus = "cancelled"
)

// CarrierRefund represents a refund request from a carrier
type CarrierRefund struct {
	ID                     string              `json:"id" db:"id"`
	RefundNumber           string              `json:"refund_number" db:"refund_number"`
	OrganizationID         string              `json:"organization_id" db:"organization_id"`
	WalletID               string              `json:"wallet_id" db:"wallet_id"`
	ShipmentID             string              `json:"shipment_id" db:"shipment_id"`
	InvoiceID              sql.NullString      `json:"invoice_id" db:"invoice_id"`
	CarrierID              string              `json:"carrier_id" db:"carrier_id"`
	CarrierTrackingNumber  sql.NullString      `json:"carrier_tracking_number" db:"carrier_tracking_number"`
	CarrierRefundReference sql.NullString      `json:"carrier_refund_reference" db:"carrier_refund_reference"`
	RequestedAmount        float64             `json:"requested_amount" db:"requested_amount"`
	ApprovedAmount         sql.NullFloat64     `json:"approved_amount" db:"approved_amount"`
	Currency               string              `json:"currency" db:"currency"`
	Status                 CarrierRefundStatus `json:"status" db:"status"`
	RefundReason           string              `json:"refund_reason" db:"refund_reason"`
	ReasonNotes            sql.NullString      `json:"reason_notes" db:"reason_notes"`
	RequestedAt            time.Time           `json:"requested_at" db:"requested_at"`
	CarrierAcknowledgedAt  sql.NullTime        `json:"carrier_acknowledged_at" db:"carrier_acknowledged_at"`
	CompletedAt            sql.NullTime        `json:"completed_at" db:"completed_at"`
	ExpectedCompletionDate sql.NullTime        `json:"expected_completion_date" db:"expected_completion_date"`
	WalletTransactionID    sql.NullString      `json:"wallet_transaction_id" db:"wallet_transaction_id"`
	RequestedByUserID      sql.NullString      `json:"requested_by_user_id" db:"requested_by_user_id"`
	ProcessedByUserID      sql.NullString      `json:"processed_by_user_id" db:"processed_by_user_id"`
	CarrierResponseCode    sql.NullString      `json:"carrier_response_code" db:"carrier_response_code"`
	CarrierResponseMessage sql.NullString      `json:"carrier_response_message" db:"carrier_response_message"`
	Notes                  sql.NullString      `json:"notes" db:"notes"`
	CreatedAt              time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time           `json:"updated_at" db:"updated_at"`

	// Joined fields (not in DB)
	CarrierName    string `json:"carrier_name" db:"-"`
	ShipmentNumber string `json:"shipment_number" db:"-"`
}

// IsInProgress checks if the refund is still in progress
func (r *CarrierRefund) IsInProgress() bool {
	return r.Status == CarrierRefundStatusRequested || r.Status == CarrierRefundStatusProcessing
}

// IsFinal checks if the refund has reached a final state
func (r *CarrierRefund) IsFinal() bool {
	return r.Status == CarrierRefundStatusCompleted ||
		r.Status == CarrierRefundStatusRejected ||
		r.Status == CarrierRefundStatusFailed ||
		r.Status == CarrierRefundStatusCancelled
}
