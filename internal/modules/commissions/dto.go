package commissions

import (
	"time"

	"gogin/internal/models"
)

// ListCommissionsRequest represents query parameters for listing commission transactions
type ListCommissionsRequest struct {
	Status       string `form:"status" binding:"omitempty,oneof=pending eligible processing paid voided"`
	ShipmentID   string `form:"shipment_id" binding:"omitempty,uuid"`
	FromDate     string `form:"from_date" binding:"omitempty"` // YYYY-MM-DD
	ToDate       string `form:"to_date" binding:"omitempty"`   // YYYY-MM-DD
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// CommissionTransactionResponse represents a commission transaction in API responses
type CommissionTransactionResponse struct {
	ID                          string     `json:"id"`
	CommissionTransactionNumber string     `json:"commission_transaction_number"`
	ShipmentID                  string     `json:"shipment_id"`
	CommissionAmount            float64    `json:"commission_amount"`
	Currency                    string     `json:"currency"`
	Status                      string     `json:"status"`
	EarnedAt                    time.Time  `json:"earned_at"`
	EligibleAt                  *time.Time `json:"eligible_at,omitempty"`
	PaidAt                      *time.Time `json:"paid_at,omitempty"`
	PayoutID                    string     `json:"payout_id,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
}

// ListCommissionsResponse represents the response for listing commissions
type ListCommissionsResponse struct {
	Commissions     []*CommissionTransactionResponse `json:"commissions"`
	TotalCount      int                              `json:"total_count"`
	Page            int                              `json:"page"`
	PageSize        int                              `json:"page_size"`
	TotalPages      int                              `json:"total_pages"`
	TotalEarnings   float64                          `json:"total_earnings"`
	EligibleBalance float64                          `json:"eligible_balance"`
	PaidBalance     float64                          `json:"paid_balance"`
}

// ListPayoutsRequest represents query parameters for listing payouts
type ListPayoutsRequest struct {
	Status   string `form:"status" binding:"omitempty,oneof=pending processing paid rejected failed cancelled"`
	FromDate string `form:"from_date" binding:"omitempty"` // YYYY-MM-DD
	ToDate   string `form:"to_date" binding:"omitempty"`   // YYYY-MM-DD
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// PayoutResponse represents a payout in API responses
type PayoutResponse struct {
	ID                string     `json:"id"`
	PayoutNumber      string     `json:"payout_number"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	Method            string     `json:"method"`
	Status            string     `json:"status"`
	UserVisibleNote   string     `json:"user_visible_note,omitempty"`
	RequestedAt       time.Time  `json:"requested_at"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	TransactionCount  int        `json:"transaction_count,omitempty"`
}

// ListPayoutsResponse represents the response for listing payouts
type ListPayoutsResponse struct {
	Payouts    []*PayoutResponse `json:"payouts"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// RequestPayoutRequest represents a request to create a payout
type RequestPayoutRequest struct {
	PayoutMethod              string   `json:"payout_method" binding:"required,oneof=stripe paypal manual"`
	CommissionTransactionIDs  []string `json:"commission_transaction_ids" binding:"required,min=1,dive,uuid"`
	Note                      string   `json:"note,omitempty"`
}

// RequestPayoutResponse represents the response after requesting a payout
type RequestPayoutResponse struct {
	PayoutRequestID     string    `json:"payout_request_id"`
	PayoutRequestNumber string    `json:"payout_request_number"`
	AmountRequested     float64   `json:"amount_requested"`
	Currency            string    `json:"currency"`
	Status              string    `json:"status"`
	TransactionCount    int       `json:"transaction_count"`
	Message             string    `json:"message"`
	CreatedAt           time.Time `json:"created_at"`
}

// ProcessPayoutRequest represents an admin request to process a payout
type ProcessPayoutRequest struct {
	PayoutRequestID string `json:"payout_request_id" binding:"required,uuid"`
	Action          string `json:"action" binding:"required,oneof=approve reject"`
	ReviewNote      string `json:"review_note,omitempty"`
	ManualReference string `json:"manual_reference,omitempty"` // For manual payouts
}

// ProcessPayoutResponse represents the response after processing a payout
type ProcessPayoutResponse struct {
	PayoutID        string    `json:"payout_id,omitempty"`
	PayoutNumber    string    `json:"payout_number,omitempty"`
	Status          string    `json:"status"`
	Amount          float64   `json:"amount,omitempty"`
	Message         string    `json:"message"`
	ProcessedAt     time.Time `json:"processed_at"`
}

// ToCommissionTransactionResponse converts a CommissionTransaction model to response DTO
func ToCommissionTransactionResponse(tx *models.CommissionTransaction) *CommissionTransactionResponse {
	resp := &CommissionTransactionResponse{
		ID:                          tx.ID,
		CommissionTransactionNumber: tx.CommissionTransactionNumber,
		ShipmentID:                  tx.ShipmentID,
		CommissionAmount:            tx.CommissionAmount,
		Currency:                    tx.Currency,
		Status:                      string(tx.Status),
		EarnedAt:                    tx.EarnedAt,
		CreatedAt:                   tx.CreatedAt,
	}

	if tx.EligibleAt.Valid {
		resp.EligibleAt = &tx.EligibleAt.Time
	}

	if tx.PaidAt.Valid {
		resp.PaidAt = &tx.PaidAt.Time
	}

	if tx.PayoutID.Valid {
		resp.PayoutID = tx.PayoutID.String
	}

	return resp
}

// ToPayoutResponse converts a CommissionPayout model to response DTO
func ToPayoutResponse(payout *models.CommissionPayout) *PayoutResponse {
	resp := &PayoutResponse{
		ID:           payout.ID,
		PayoutNumber: payout.PayoutNumber,
		Amount:       payout.Amount,
		Currency:     payout.Currency,
		Method:       string(payout.Method),
		Status:       string(payout.Status),
		RequestedAt:  payout.RequestedAt,
		CreatedAt:    payout.CreatedAt,
	}

	if payout.UserVisibleNote.Valid {
		resp.UserVisibleNote = payout.UserVisibleNote.String
	}

	if payout.ProcessedAt.Valid {
		resp.ProcessedAt = &payout.ProcessedAt.Time
	}

	if payout.PaidAt.Valid {
		resp.PaidAt = &payout.PaidAt.Time
	}

	return resp
}
