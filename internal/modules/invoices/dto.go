package invoices

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// ListInvoicesRequest represents query parameters for listing invoices
type ListInvoicesRequest struct {
	Page          int       `form:"page" binding:"omitempty,gte=1"`
	PageSize      int       `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	InvoiceType   string    `form:"invoice_type" binding:"omitempty,oneof=invoice credit_note adjustment"`
	Status        string    `form:"status" binding:"omitempty,oneof=draft issued void"`
	PaymentStatus string    `form:"payment_status" binding:"omitempty,oneof=unpaid partial paid overpaid"`
	FromDate      time.Time `form:"from_date" binding:"omitempty" time_format:"2006-01-02"`
	ToDate        time.Time `form:"to_date" binding:"omitempty" time_format:"2006-01-02"`
}

// RefundInvoiceRequest represents a request to issue a refund via credit note
type RefundInvoiceRequest struct {
	Amount         float64 `json:"amount" binding:"required,gt=0" example:"125.50"`
	Reason         string  `json:"reason" binding:"required,oneof=full_refund partial_refund rate_adjustment customer_dispute damaged_goods service_failure duplicate_charge other" example:"rate_adjustment"`
	ReasonNotes    string  `json:"reason_notes" binding:"omitempty,max=500" example:"Shipment was delayed by 2 days"`
	IdempotencyKey string  `json:"idempotency_key" binding:"required,min=1,max=100" example:"refund_20260111_abc123"`
}

// VoidInvoiceRequest represents a request to void an invoice
type VoidInvoiceRequest struct {
	Reason         string `json:"reason" binding:"required,min=1,max=255" example:"Invoice issued in error - duplicate shipment"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,min=1,max=100" example:"void_20260111_xyz789"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// InvoiceResponse represents an invoice header (list view)
type InvoiceResponse struct {
	ID                 string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvoiceNumber      string     `json:"invoice_number" example:"INV-2026-00001"`
	OrganizationID     string     `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationName   string     `json:"organization_name" example:"Acme Logistics Inc."`
	ShipmentID         *string    `json:"shipment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShipmentNumber     *string    `json:"shipment_number,omitempty" example:"SH-2026-00042"`
	InvoiceType        string     `json:"invoice_type" example:"invoice"`
	Status             string     `json:"status" example:"issued"`
	PaymentStatus      string     `json:"payment_status" example:"paid"`
	Currency           string     `json:"currency" example:"CAD"`
	SubtotalAmount     float64    `json:"subtotal_amount" example:"100.00"`
	TaxAmount          float64    `json:"tax_amount" example:"13.00"`
	DiscountAmount     float64    `json:"discount_amount" example:"0.00"`
	TotalAmount        float64    `json:"total_amount" example:"113.00"`
	AmountPaid         float64    `json:"amount_paid" example:"113.00"`
	BalanceDue         float64    `json:"balance_due" example:"0.00"`
	IssuedAt           *time.Time `json:"issued_at,omitempty" example:"2026-01-11T10:30:00Z"`
	DueDate            *time.Time `json:"due_date,omitempty" example:"2026-01-25T23:59:59Z"`
	VoidedAt           *time.Time `json:"voided_at,omitempty"`
	ParentInvoiceID    *string    `json:"parent_invoice_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreditNoteReason   *string    `json:"credit_note_reason,omitempty" example:"rate_adjustment"`
	CreditNoteReasonNotes *string `json:"credit_note_reason_notes,omitempty" example:"Shipment delayed"`
	CreatedAt          time.Time  `json:"created_at" example:"2026-01-11T10:25:00Z"`
	UpdatedAt          time.Time  `json:"updated_at" example:"2026-01-11T10:30:00Z"`
}

// InvoiceDetailResponse represents complete invoice details with line items
type InvoiceDetailResponse struct {
	InvoiceResponse
	// Billing snapshot data
	BillingAddress     *BillingAddressSnapshot     `json:"billing_address,omitempty"`
	ShipmentSnapshot   *ShipmentSnapshot           `json:"shipment_snapshot,omitempty"`
	// Line items
	Items              []InvoiceItemResponse       `json:"items"`
	// Payment records
	Payments           []InvoicePaymentResponse    `json:"payments"`
	// Audit trail
	Events             []InvoiceEventResponse      `json:"events"`
	// Refund summary (for invoices that can be refunded)
	RefundSummary      *RefundSummary              `json:"refund_summary,omitempty"`
}

// BillingAddressSnapshot represents the snapshot of billing address at invoice time
type BillingAddressSnapshot struct {
	CompanyName    *string `json:"company_name,omitempty" example:"Acme Logistics Inc."`
	ContactName    string  `json:"contact_name" example:"John Doe"`
	ContactEmail   string  `json:"contact_email" example:"billing@acme.com"`
	ContactPhone   *string `json:"contact_phone,omitempty" example:"+1-416-555-0100"`
	AddressLine1   string  `json:"address_line1" example:"123 Main Street"`
	AddressLine2   *string `json:"address_line2,omitempty" example:"Suite 100"`
	City           string  `json:"city" example:"Toronto"`
	StateProvince  string  `json:"state_province" example:"ON"`
	PostalCode     string  `json:"postal_code" example:"M5H 2N2"`
	CountryCode    string  `json:"country_code" example:"CA"`
}

// ShipmentSnapshot represents key shipment details at invoice time
type ShipmentSnapshot struct {
	ShipmentNumber      string     `json:"shipment_number" example:"SH-2026-00042"`
	TrackingNumber      *string    `json:"tracking_number,omitempty" example:"1Z999AA10123456789"`
	ServiceType         string     `json:"service_type" example:"standard"`
	CarrierName         string     `json:"carrier_name" example:"UPS"`
	OriginCity          string     `json:"origin_city" example:"Toronto"`
	OriginProvince      string     `json:"origin_province" example:"ON"`
	OriginCountryCode   string     `json:"origin_country_code" example:"CA"`
	DestinationCity     string     `json:"destination_city" example:"Vancouver"`
	DestinationProvince string     `json:"destination_province" example:"BC"`
	DestinationCountryCode string  `json:"destination_country_code" example:"CA"`
	TotalPackages       int        `json:"total_packages" example:"2"`
	TotalWeight         float64    `json:"total_weight" example:"15.5"`
	WeightUnit          string     `json:"weight_unit" example:"kg"`
	ShipDate            *time.Time `json:"ship_date,omitempty" example:"2026-01-11T00:00:00Z"`
}

// InvoiceItemResponse represents an invoice line item
type InvoiceItemResponse struct {
	ID              string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ItemType        string   `json:"item_type" example:"shipping"`
	Description     string   `json:"description" example:"Standard Shipping - Package 1"`
	PackageID       *string  `json:"package_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	PackageNumber   *string  `json:"package_number,omitempty" example:"PKG-001"`
	Quantity        int      `json:"quantity" example:"1"`
	UnitPrice       float64  `json:"unit_price" example:"50.00"`
	TotalPrice      float64  `json:"total_price" example:"50.00"`
	Currency        string   `json:"currency" example:"CAD"`
	TaxRate         *float64 `json:"tax_rate,omitempty" example:"0.13"`
	TaxAmount       *float64 `json:"tax_amount,omitempty" example:"6.50"`
	DiscountRate    *float64 `json:"discount_rate,omitempty" example:"0.00"`
	DiscountAmount  *float64 `json:"discount_amount,omitempty" example:"0.00"`
	NetAmount       float64  `json:"net_amount" example:"56.50"`
	SortOrder       int      `json:"sort_order" example:"1"`
}

// InvoicePaymentResponse represents a payment record
type InvoicePaymentResponse struct {
	ID                      string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PaymentMethod           string     `json:"payment_method" example:"wallet"`
	Amount                  float64    `json:"amount" example:"113.00"`
	Currency                string     `json:"currency" example:"CAD"`
	PaidAt                  time.Time  `json:"paid_at" example:"2026-01-11T10:30:00Z"`
	WalletTransactionID     *string    `json:"wallet_transaction_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	WalletTransactionNumber *string    `json:"wallet_transaction_number,omitempty" example:"TXN-2026-00123"`
	StripePaymentIntentID   *string    `json:"stripe_payment_intent_id,omitempty" example:"pi_1234567890abcdef"`
	PaymentReference        *string    `json:"payment_reference,omitempty" example:"Payment for shipment SH-2026-00042"`
	Notes                   *string    `json:"notes,omitempty"`
}

// InvoiceEventResponse represents an audit event
type InvoiceEventResponse struct {
	ID              string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType       string     `json:"event_type" example:"invoice.issued"`
	Description     string     `json:"description" example:"Invoice issued for shipment SH-2026-00042"`
	PerformedByUserID *string  `json:"performed_by_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	PerformedByUserName *string `json:"performed_by_user_name,omitempty" example:"John Doe"`
	IsSystemGenerated bool     `json:"is_system_generated" example:"false"`
	Metadata        *string    `json:"metadata,omitempty" example:"{\"reason\":\"rate_adjustment\"}"`
	CreatedAt       time.Time  `json:"created_at" example:"2026-01-11T10:30:00Z"`
}

// RefundSummary represents refund eligibility and history
type RefundSummary struct {
	OriginalAmount      float64 `json:"original_amount" example:"113.00"`
	TotalRefunded       float64 `json:"total_refunded" example:"0.00"`
	RefundableRemaining float64 `json:"refundable_remaining" example:"113.00"`
	RefundCount         int     `json:"refund_count" example:"0"`
	CanRefund           bool    `json:"can_refund" example:"true"`
}

// InvoiceListResponse represents paginated invoice list
type InvoiceListResponse struct {
	Invoices     []InvoiceResponse `json:"invoices"`
	TotalCount   int               `json:"total_count" example:"42"`
	Page         int               `json:"page" example:"1"`
	PageSize     int               `json:"page_size" example:"20"`
	TotalPages   int               `json:"total_pages" example:"3"`
}

// RefundResponse represents the result of a refund operation
type RefundResponse struct {
	CreditNoteID         string    `json:"credit_note_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreditNoteNumber     string    `json:"credit_note_number" example:"CN-2026-00001"`
	ParentInvoiceID      string    `json:"parent_invoice_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParentInvoiceNumber  string    `json:"parent_invoice_number" example:"INV-2026-00001"`
	RefundAmount         float64   `json:"refund_amount" example:"125.50"`
	Currency             string    `json:"currency" example:"CAD"`
	Reason               string    `json:"reason" example:"rate_adjustment"`
	ReasonNotes          *string   `json:"reason_notes,omitempty" example:"Shipment delayed"`
	WalletTransactionID  *string   `json:"wallet_transaction_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	WalletTransactionNumber *string `json:"wallet_transaction_number,omitempty" example:"TXN-2026-00124"`
	IssuedAt             time.Time `json:"issued_at" example:"2026-01-11T11:00:00Z"`
}

// VoidResponse represents the result of a void operation
type VoidResponse struct {
	InvoiceID        string    `json:"invoice_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvoiceNumber    string    `json:"invoice_number" example:"INV-2026-00001"`
	Status           string    `json:"status" example:"void"`
	VoidedAt         time.Time `json:"voided_at" example:"2026-01-11T11:15:00Z"`
	VoidReason       string    `json:"void_reason" example:"Invoice issued in error"`
	BalanceRefunded  float64   `json:"balance_refunded" example:"113.00"`
}
