package wallet

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// TopUpRequest represents a wallet top-up request
type TopUpRequest struct {
	Amount          float64 `json:"amount" binding:"required,gt=0" example:"100.00"`
	PaymentMethodID string  `json:"payment_method_id" binding:"required" example:"pm_1234567890abcdef"`
	IdempotencyKey  string  `json:"idempotency_key" binding:"omitempty,max=100" example:"topup_2026-01-11_abc123"`
}

// RefundRequest represents a wallet refund request (shipment refund to wallet)
type RefundRequest struct {
	ShipmentID     string  `json:"shipment_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount         float64 `json:"amount" binding:"required,gt=0" example:"45.50"`
	Reason         string  `json:"reason" binding:"required,min=1,max=255" example:"Shipment cancelled by customer"`
	IdempotencyKey string  `json:"idempotency_key" binding:"omitempty,max=100" example:"refund_2026-01-11_xyz789"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// WalletResponse represents the wallet status
type WalletResponse struct {
	ID                      string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WalletNumber            string                  `json:"wallet_number" example:"WAL-ABC12345"`
	OrganizationID          string                  `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`

	// Balance Information
	AvailableBalance        float64                 `json:"available_balance" example:"1250.75"`
	TotalInTransit          float64                 `json:"total_in_transit" example:"150.00"`
	Reserved                float64                 `json:"reserved" example:"50.00"`
	TotalBalance            float64                 `json:"total_balance" example:"1450.75"`

	// Financial Summary (This Month)
	TotalDeposits           float64                 `json:"total_deposits" example:"500.00"`
	TotalOutflow            float64                 `json:"total_outflow" example:"320.50"`
	NetMovement             float64                 `json:"net_movement" example:"179.50"`

	// Legacy fields (kept for backwards compatibility)
	Balance                 float64                 `json:"balance" example:"1250.75"`
	HeldBalance             float64                 `json:"held_balance" example:"50.00"`
	CarrierRefundInProgress float64                 `json:"carrier_refund_in_progress" example:"75.00"`

	Status                  string                  `json:"status" example:"active"`
	Currency                string                  `json:"currency" example:"CAD"`
	MinTopUpAmount          float64                 `json:"min_topup_amount" example:"10.00"`
	MaxTopUpAmount          *float64                `json:"max_topup_amount,omitempty" example:"10000.00"`
	MaxBalance              *float64                `json:"max_balance,omitempty" example:"50000.00"`
	CreatedAt               time.Time               `json:"created_at" example:"2026-01-01T10:00:00Z"`
	UpdatedAt               time.Time               `json:"updated_at" example:"2026-01-11T14:30:00Z"`
	RecentTransactions      []TransactionResponse   `json:"recent_transactions,omitempty"`
}

// TransactionResponse represents a single wallet transaction
type TransactionResponse struct {
	ID                string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	TransactionNumber string     `json:"transaction_number" example:"TRX-XYZ98765"`
	Direction         string     `json:"direction" example:"credit"`
	TxType            string     `json:"tx_type" example:"topup"`
	Amount            float64    `json:"amount" example:"100.00"`
	Currency          string     `json:"currency" example:"CAD"`
	BalanceBefore     float64    `json:"balance_before" example:"1150.75"`
	BalanceAfter      float64    `json:"balance_after" example:"1250.75"`
	Description       *string    `json:"description,omitempty" example:"Wallet top-up via Stripe"`
	ReferenceType     *string    `json:"reference_type,omitempty" example:"topup"`
	ReferenceID       *string    `json:"reference_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	PerformedByUserID *string    `json:"performed_by_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	IsSystemGenerated bool       `json:"is_system_generated" example:"false"`
	CreatedAt         time.Time  `json:"created_at" example:"2026-01-11T14:30:00Z"`
}

// TopUpResponse represents successful top-up result
type TopUpResponse struct {
	WalletID           string  `json:"wallet_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TransactionID      string  `json:"transaction_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	TransactionNumber  string  `json:"transaction_number" example:"TRX-XYZ98765"`
	TopUpNumber        string  `json:"topup_number" example:"TOP-ABC12345"`
	Amount             float64 `json:"amount" example:"100.00"`
	Currency           string  `json:"currency" example:"CAD"`
	NewBalance         float64 `json:"new_balance" example:"1250.75"`
	StripePaymentIntentID *string `json:"stripe_payment_intent_id,omitempty" example:"pi_1234567890abcdef"`
	StripeReceiptURL   *string `json:"stripe_receipt_url,omitempty" example:"https://pay.stripe.com/receipts/..."`
	Status             string  `json:"status" example:"succeeded"`
	CreatedAt          time.Time `json:"created_at" example:"2026-01-11T14:30:00Z"`
}

// RefundResponse represents successful refund result
type RefundResponse struct {
	WalletID          string  `json:"wallet_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TransactionID     string  `json:"transaction_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	TransactionNumber string  `json:"transaction_number" example:"TRX-XYZ98765"`
	ShipmentID        string  `json:"shipment_id" example:"550e8400-e29b-41d4-a716-446655440005"`
	Amount            float64 `json:"amount" example:"45.50"`
	Currency          string  `json:"currency" example:"CAD"`
	NewBalance        float64 `json:"new_balance" example:"1296.25"`
	Reason            string  `json:"reason" example:"Shipment cancelled by customer"`
	CreatedAt         time.Time `json:"created_at" example:"2026-01-11T14:35:00Z"`
}

// WalletStatusResponse represents wallet lock/unlock response
type WalletStatusResponse struct {
	WalletID     string    `json:"wallet_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WalletNumber string    `json:"wallet_number" example:"WAL-ABC12345"`
	Status       string    `json:"status" example:"locked"`
	UpdatedAt    time.Time `json:"updated_at" example:"2026-01-11T14:40:00Z"`
	UpdatedBy    string    `json:"updated_by" example:"550e8400-e29b-41d4-a716-446655440004"`
}

// TransactionsListResponse represents paginated transactions
type TransactionsListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Pagination   PaginationMeta        `json:"pagination"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page" example:"1"`
	PerPage    int `json:"per_page" example:"20"`
	Total      int `json:"total" example:"150"`
	TotalPages int `json:"total_pages" example:"8"`
}

// ListTransactionsRequest represents query parameters for listing transactions
type ListTransactionsRequest struct {
	Page              int     `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit             int     `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	TransactionNumber *string `form:"transaction_number" binding:"omitempty" example:"TRX-ABC12345"`
	TransactionID     *string `form:"transaction_id" binding:"omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Direction         *string `form:"direction" binding:"omitempty,oneof=credit debit" example:"credit"`
	TxType            *string `form:"tx_type" binding:"omitempty" example:"topup"`
	Status            *string `form:"status" binding:"omitempty" example:"completed"`
	MinAmount         *float64 `form:"min_amount" binding:"omitempty" example:"10.00"`
	MaxAmount         *float64 `form:"max_amount" binding:"omitempty" example:"1000.00"`
	FromDate          *string `form:"from_date" binding:"omitempty" example:"2026-01-01"`
	ToDate            *string `form:"to_date" binding:"omitempty" example:"2026-01-31"`
	SortBy            *string `form:"sort_by" binding:"omitempty,oneof=created_at amount" example:"created_at"`
	SortOrder         *string `form:"sort_order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// ExportTransactionsRequest represents filter options for exporting transactions
type ExportTransactionsRequest struct {
	Format    string `form:"format" binding:"omitempty,oneof=csv json" example:"csv"`
	FromDate  string `form:"from_date" binding:"omitempty" example:"2026-01-01"`
	ToDate    string `form:"to_date" binding:"omitempty" example:"2026-01-31"`
	Direction string `form:"direction" binding:"omitempty,oneof=credit debit" example:"credit"`
	TxType    string `form:"tx_type" binding:"omitempty" example:"topup"`
}

// ExportTransactionRow represents a single transaction row for export
type ExportTransactionRow struct {
	TransactionNumber string  `json:"transaction_number"`
	Date              string  `json:"date"`
	Time              string  `json:"time"`
	Direction         string  `json:"direction"`
	Type              string  `json:"type"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	BalanceBefore     float64 `json:"balance_before"`
	BalanceAfter      float64 `json:"balance_after"`
	Description       string  `json:"description"`
	ReferenceType     string  `json:"reference_type"`
	ReferenceID       string  `json:"reference_id"`
}

// ExportTransactionsResponse represents the export result for JSON format
type ExportTransactionsResponse struct {
	WalletNumber   string                 `json:"wallet_number" example:"WAL-ABC12345"`
	OrganizationID string                 `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ExportedAt     string                 `json:"exported_at" example:"2026-01-23T10:30:00Z"`
	FromDate       string                 `json:"from_date,omitempty" example:"2026-01-01"`
	ToDate         string                 `json:"to_date,omitempty" example:"2026-01-31"`
	TotalRecords   int                    `json:"total_records" example:"150"`
	Transactions   []ExportTransactionRow `json:"transactions"`
}

// =====================================================
// WALLET OVERVIEW DTOs
// =====================================================

// WalletOverviewRequest represents the request parameters for wallet overview
type WalletOverviewRequest struct {
	Period string `form:"period" binding:"omitempty,oneof=this_week this_month this_year" example:"this_month"`
}

// WalletOverviewResponse represents the wallet overview with financial summary
type WalletOverviewResponse struct {
	// Current balances
	AvailableBalance float64 `json:"available_balance" example:"1250.75"`
	TotalInTransit   float64 `json:"total_in_transit" example:"150.00"`
	Reserved         float64 `json:"reserved" example:"50.00"`

	// Period-based metrics
	TotalDeposits float64 `json:"total_deposits" example:"500.00"`
	TotalOutflow  float64 `json:"total_outflow" example:"320.50"`
	NetMovement   float64 `json:"net_movement" example:"179.50"`

	// Period info
	Period      WalletOverviewPeriod `json:"period"`
	Currency    string               `json:"currency" example:"CAD"`
	GeneratedAt time.Time            `json:"generated_at"`
}

// WalletOverviewPeriod represents the period information
type WalletOverviewPeriod struct {
	Type      string `json:"type" example:"this_month"`
	StartDate string `json:"start_date" example:"2026-01-01"`
	EndDate   string `json:"end_date" example:"2026-01-31"`
	Label     string `json:"label" example:"This Month"`
}

// =====================================================
// CARRIER REFUND DTOs
// =====================================================

// ListCarrierRefundsRequest represents query parameters for listing carrier refunds
type ListCarrierRefundsRequest struct {
	Page         int     `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit        int     `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	RefundNumber *string `form:"refund_number" binding:"omitempty" example:"CRF-ABC12345"`
	ShipmentID   *string `form:"shipment_id" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CarrierID    *string `form:"carrier_id" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	Status       *string `form:"status" binding:"omitempty,oneof=requested processing completed rejected failed cancelled" example:"requested"`
	RefundReason *string `form:"refund_reason" binding:"omitempty" example:"shipment_cancelled"`
	FromDate     *string `form:"from_date" binding:"omitempty" example:"2026-01-01"`
	ToDate       *string `form:"to_date" binding:"omitempty" example:"2026-01-31"`
	SortBy       *string `form:"sort_by" binding:"omitempty,oneof=requested_at requested_amount status" example:"requested_at"`
	SortOrder    *string `form:"sort_order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// CarrierRefundResponse represents a carrier refund in responses
type CarrierRefundResponse struct {
	ID                     string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RefundNumber           string     `json:"refund_number" example:"CRF-ABC12345"`
	OrganizationID         string     `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ShipmentID             string     `json:"shipment_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ShipmentNumber         *string    `json:"shipment_number,omitempty" example:"SHP-XYZ98765"`
	CarrierID              string     `json:"carrier_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	CarrierName            *string    `json:"carrier_name,omitempty" example:"Canada Post"`
	CarrierTrackingNumber  *string    `json:"carrier_tracking_number,omitempty" example:"1234567890"`
	CarrierRefundReference *string    `json:"carrier_refund_reference,omitempty" example:"REF-123456"`
	RequestedAmount        float64    `json:"requested_amount" example:"45.50"`
	ApprovedAmount         *float64   `json:"approved_amount,omitempty" example:"45.50"`
	Currency               string     `json:"currency" example:"CAD"`
	Status                 string     `json:"status" example:"requested"`
	RefundReason           string     `json:"refund_reason" example:"shipment_cancelled"`
	ReasonNotes            *string    `json:"reason_notes,omitempty" example:"Customer requested cancellation"`
	RequestedAt            time.Time  `json:"requested_at" example:"2026-01-15T10:00:00Z"`
	CarrierAcknowledgedAt  *time.Time `json:"carrier_acknowledged_at,omitempty" example:"2026-01-15T12:00:00Z"`
	CompletedAt            *time.Time `json:"completed_at,omitempty" example:"2026-01-20T15:00:00Z"`
	ExpectedCompletionDate *string    `json:"expected_completion_date,omitempty" example:"2026-01-25"`
	WalletTransactionID    *string    `json:"wallet_transaction_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	RequestedByUserID      *string    `json:"requested_by_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440005"`
	CarrierResponseCode    *string    `json:"carrier_response_code,omitempty" example:"APPROVED"`
	CarrierResponseMessage *string    `json:"carrier_response_message,omitempty" example:"Refund approved"`
	Notes                  *string    `json:"notes,omitempty" example:"Additional notes"`
	CreatedAt              time.Time  `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt              time.Time  `json:"updated_at" example:"2026-01-15T10:00:00Z"`
}

// CarrierRefundsListResponse represents paginated carrier refunds
type CarrierRefundsListResponse struct {
	Refunds    []CarrierRefundResponse `json:"refunds"`
	Summary    CarrierRefundSummary    `json:"summary"`
	Pagination PaginationMeta          `json:"pagination"`
}

// CarrierRefundSummary represents summary statistics for carrier refunds
type CarrierRefundSummary struct {
	TotalRequested  int     `json:"total_requested" example:"10"`
	TotalProcessing int     `json:"total_processing" example:"5"`
	TotalCompleted  int     `json:"total_completed" example:"20"`
	TotalRejected   int     `json:"total_rejected" example:"2"`
	TotalFailed     int     `json:"total_failed" example:"1"`
	TotalCancelled  int     `json:"total_cancelled" example:"0"`
	AmountPending   float64 `json:"amount_pending" example:"250.00"`
	AmountCompleted float64 `json:"amount_completed" example:"1500.00"`
	Currency        string  `json:"currency" example:"CAD"`
}
