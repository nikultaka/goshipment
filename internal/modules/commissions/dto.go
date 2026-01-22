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
		CommissionAmount:            tx.CommissionAmount,
		Currency:                    tx.Currency,
		Status:                      string(tx.Status),
		EarnedAt:                    tx.EarnedAt,
		CreatedAt:                   tx.CreatedAt,
	}

	if tx.ShipmentID.Valid {
		resp.ShipmentID = tx.ShipmentID.String
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

// =====================================================
// COMMISSION PLANS DTOs
// =====================================================

// ListPlansRequest represents query parameters for listing commission plans
type ListPlansRequest struct {
	Status   string `form:"status" binding:"omitempty,oneof=active paused"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// CreatePlanRequest represents a request to create a commission plan
type CreatePlanRequest struct {
	Name                     string  `json:"name" binding:"required,min=1,max=120"`
	Description              string  `json:"description,omitempty" binding:"omitempty,max=1000"`
	CommissionBase           string  `json:"commission_base" binding:"required,oneof=profit revenue per_shipment"`
	CommissionValueType      string  `json:"commission_value_type" binding:"required,oneof=percent fixed_amount"`
	Value                    float64 `json:"value" binding:"required,gt=0"`
	ApplyWhen                string  `json:"apply_when,omitempty" binding:"omitempty,oneof=shipment_completed include_refunded_shipments"`
	IncludeRefundedShipments bool    `json:"include_refunded_shipments,omitempty"`
	Status                   string  `json:"status,omitempty" binding:"omitempty,oneof=active paused"`
}

// UpdatePlanRequest represents a request to update a commission plan
type UpdatePlanRequest struct {
	Name                     *string  `json:"name,omitempty" binding:"omitempty,min=1,max=120"`
	Description              *string  `json:"description,omitempty" binding:"omitempty,max=1000"`
	CommissionBase           *string  `json:"commission_base,omitempty" binding:"omitempty,oneof=profit revenue per_shipment"`
	CommissionValueType      *string  `json:"commission_value_type,omitempty" binding:"omitempty,oneof=percent fixed_amount"`
	Value                    *float64 `json:"value,omitempty" binding:"omitempty,gt=0"`
	ApplyWhen                *string  `json:"apply_when,omitempty" binding:"omitempty,oneof=shipment_completed include_refunded_shipments"`
	IncludeRefundedShipments *bool    `json:"include_refunded_shipments,omitempty"`
	Status                   *string  `json:"status,omitempty" binding:"omitempty,oneof=active paused"`
}

// CommissionPlanResponse represents a commission plan in API responses
type CommissionPlanResponse struct {
	ID                       string    `json:"id"`
	CommissionNumber         string    `json:"commission_number"`
	Name                     string    `json:"name"`
	Description              string    `json:"description,omitempty"`
	CommissionBase           string    `json:"commission_base"`
	CommissionValueType      string    `json:"commission_value_type"`
	Value                    float64   `json:"value"`
	ApplyWhen                string    `json:"apply_when"`
	IncludeRefundedShipments bool      `json:"include_refunded_shipments"`
	Status                   string    `json:"status"`
	CreatedByUserID          string    `json:"created_by_user_id,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// ListPlansResponse represents the response for listing commission plans
type ListPlansResponse struct {
	Plans      []*CommissionPlanResponse `json:"plans"`
	TotalCount int                       `json:"total_count"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
}

// ToCommissionPlanResponse converts a CommissionPlan model to response DTO
func ToCommissionPlanResponse(plan *models.CommissionPlan) *CommissionPlanResponse {
	resp := &CommissionPlanResponse{
		ID:                       plan.ID,
		CommissionNumber:         plan.CommissionNumber,
		Name:                     plan.Name,
		CommissionBase:           string(plan.CommissionBase),
		CommissionValueType:      string(plan.CommissionValueType),
		Value:                    plan.Value,
		ApplyWhen:                string(plan.ApplyWhen),
		IncludeRefundedShipments: plan.IncludeRefundedShipments,
		Status:                   string(plan.Status),
		CreatedAt:                plan.CreatedAt,
		UpdatedAt:                plan.UpdatedAt,
	}

	if plan.Description.Valid {
		resp.Description = plan.Description.String
	}

	if plan.CreatedByUserID.Valid {
		resp.CreatedByUserID = plan.CreatedByUserID.String
	}

	return resp
}

// =====================================================
// MANUAL COMMISSION TRANSACTION DTOs
// =====================================================

// CreateCommissionTransactionRequest represents a request to manually create a commission transaction
type CreateCommissionTransactionRequest struct {
	BeneficiaryUserID string  `json:"beneficiary_user_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceUserID      string  `json:"source_user_id,omitempty" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	ShipmentID        string  `json:"shipment_id,omitempty" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	CommissionPlanID  string  `json:"commission_plan_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440003"`
	RevenueAmount     float64 `json:"revenue_amount" binding:"required,gte=0" example:"100.00"`
	CostAmount        float64 `json:"cost_amount" binding:"required,gte=0" example:"60.00"`
	Currency          string  `json:"currency,omitempty" binding:"omitempty,len=3" example:"CAD"`
	Status            string  `json:"status,omitempty" binding:"omitempty,oneof=pending eligible" example:"eligible"`
	Note              string  `json:"note,omitempty" binding:"omitempty,max=500" example:"Manual commission for Q1 bonus"`
}

// CreateCommissionTransactionResponse represents the response after creating a commission transaction
type CreateCommissionTransactionResponse struct {
	ID                          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CommissionTransactionNumber string    `json:"commission_transaction_number" example:"CTX-a1b2c3d4"`
	BeneficiaryUserID           string    `json:"beneficiary_user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SourceUserID                string    `json:"source_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	ShipmentID                  string    `json:"shipment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	CommissionPlanID            string    `json:"commission_plan_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	CommissionNumber            string    `json:"commission_number" example:"COM-12345678"`
	CommissionBase              string    `json:"commission_base" example:"profit"`
	CommissionValueType         string    `json:"commission_value_type" example:"percent"`
	CommissionValue             float64   `json:"commission_value" example:"10"`
	Currency                    string    `json:"currency" example:"CAD"`
	RevenueAmount               float64   `json:"revenue_amount" example:"100.00"`
	CostAmount                  float64   `json:"cost_amount" example:"60.00"`
	ProfitAmount                float64   `json:"profit_amount" example:"40.00"`
	CommissionAmount            float64   `json:"commission_amount" example:"4.00"`
	Status                      string    `json:"status" example:"eligible"`
	EarnedAt                    time.Time `json:"earned_at"`
	EligibleAt                  time.Time `json:"eligible_at,omitempty"`
	CreatedAt                   time.Time `json:"created_at"`
	Message                     string    `json:"message" example:"Commission transaction created successfully"`
}
