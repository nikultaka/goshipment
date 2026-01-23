package wallet

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// TopUpRequest represents a wallet top-up request
type TopUpRequest struct {
	Amount          float64 `json:"amount" binding:"required,gt=0" example:"100.00"`
	PaymentMethodID string  `json:"payment_method_id" binding:"required" example:"pm_1234567890abcdef"`
	IdempotencyKey  string  `json:"idempotency_key" binding:"required,min=1,max=100" example:"topup_2026-01-11_abc123"`
}

// RefundRequest represents a wallet refund request (shipment refund to wallet)
type RefundRequest struct {
	ShipmentID     string  `json:"shipment_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount         float64 `json:"amount" binding:"required,gt=0" example:"45.50"`
	Reason         string  `json:"reason" binding:"required,min=1,max=255" example:"Shipment cancelled by customer"`
	IdempotencyKey string  `json:"idempotency_key" binding:"required,min=1,max=100" example:"refund_2026-01-11_xyz789"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// WalletResponse represents the wallet status
type WalletResponse struct {
	ID                      string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WalletNumber            string                  `json:"wallet_number" example:"WAL-ABC12345"`
	OrganizationID          string                  `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Balance                 float64                 `json:"balance" example:"1250.75"`
	HeldBalance             float64                 `json:"held_balance" example:"50.00"`
	CarrierRefundInProgress float64                 `json:"carrier_refund_in_progress" example:"75.00"`
	TotalBalance            float64                 `json:"total_balance" example:"1300.75"`
	Status                  string                  `json:"status" example:"active"`
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
	Total        int                   `json:"total" example:"150"`
	Page         int                   `json:"page" example:"1"`
	PageSize     int                   `json:"page_size" example:"20"`
	TotalPages   int                   `json:"total_pages" example:"8"`
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
