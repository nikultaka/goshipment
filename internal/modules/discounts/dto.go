package discounts

import "time"

// =====================================================
// REQUEST DTOs
// =====================================================

// CreateDiscountRequest represents the request to create a discount
type CreateDiscountRequest struct {
	Code                   string   `json:"code" binding:"required,min=3,max=50" example:"SAVE10"`
	Name                   *string  `json:"name" binding:"omitempty,max=120" example:"10% Off Summer Sale"`
	Description            *string  `json:"description" binding:"omitempty" example:"Get 10% off on all shipments"`
	DiscountType           string   `json:"discount_type" binding:"required,oneof=percent fixed_amount free_shipping" example:"percent"`
	Value                  float64  `json:"value" binding:"required,gt=0" example:"10"`
	Scope                  *string  `json:"scope" binding:"omitempty,oneof=invoice shipment" example:"invoice"`
	MinSubtotalAmount      *float64 `json:"min_subtotal_amount" binding:"omitempty,gte=0" example:"50.00"`
	MaxDiscountAmount      *float64 `json:"max_discount_amount" binding:"omitempty,gt=0" example:"100.00"`
	AllowedCountry         *string  `json:"allowed_country" binding:"omitempty,len=2" example:"CA"`
	MaxUsesTotal           *int     `json:"max_uses_total" binding:"omitempty,gt=0" example:"1000"`
	MaxUsesPerUser         *int     `json:"max_uses_per_user" binding:"omitempty,gt=0" example:"1"`
	MaxTotalDiscountAmount *float64 `json:"max_total_discount_amount" binding:"omitempty,gt=0" example:"10000.00"`
	StartsAt               *string  `json:"starts_at" binding:"omitempty" example:"2026-01-01T00:00:00Z"`
	EndsAt                 *string  `json:"ends_at" binding:"omitempty" example:"2026-12-31T23:59:59Z"`
	IsActive               *bool    `json:"is_active" example:"true"`
	FirstTimeOnly          *bool    `json:"first_time_only" example:"false"`
}

// UpdateDiscountRequest represents the request to update a discount
type UpdateDiscountRequest struct {
	Name                   *string  `json:"name" binding:"omitempty,max=120" example:"Updated Discount Name"`
	Description            *string  `json:"description" binding:"omitempty" example:"Updated description"`
	Value                  *float64 `json:"value" binding:"omitempty,gt=0" example:"15"`
	MinSubtotalAmount      *float64 `json:"min_subtotal_amount" binding:"omitempty,gte=0" example:"75.00"`
	MaxDiscountAmount      *float64 `json:"max_discount_amount" binding:"omitempty,gt=0" example:"150.00"`
	AllowedCountry         *string  `json:"allowed_country" binding:"omitempty,len=2" example:"US"`
	MaxUsesTotal           *int     `json:"max_uses_total" binding:"omitempty,gt=0" example:"2000"`
	MaxUsesPerUser         *int     `json:"max_uses_per_user" binding:"omitempty,gt=0" example:"2"`
	MaxTotalDiscountAmount *float64 `json:"max_total_discount_amount" binding:"omitempty,gt=0" example:"20000.00"`
	StartsAt               *string  `json:"starts_at" binding:"omitempty" example:"2026-02-01T00:00:00Z"`
	EndsAt                 *string  `json:"ends_at" binding:"omitempty" example:"2026-12-31T23:59:59Z"`
	IsActive               *bool    `json:"is_active" example:"true"`
	FirstTimeOnly          *bool    `json:"first_time_only" example:"false"`
}

// ListDiscountsRequest represents query parameters for listing discounts
type ListDiscountsRequest struct {
	Page         int     `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit        int     `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Code         *string `form:"code" binding:"omitempty" example:"SAVE10"`
	DiscountType *string `form:"discount_type" binding:"omitempty,oneof=percent fixed_amount free_shipping" example:"percent"`
	Scope        *string `form:"scope" binding:"omitempty,oneof=invoice shipment" example:"invoice"`
	IsActive     *bool   `form:"is_active" example:"true"`
	SortBy       *string `form:"sort_by" binding:"omitempty,oneof=created_at code value" example:"created_at"`
	SortOrder    *string `form:"sort_order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// ValidateDiscountRequest represents the request to validate a discount code
type ValidateDiscountRequest struct {
	Code     string  `json:"code" binding:"required" example:"SAVE10"`
	Subtotal float64 `json:"subtotal" binding:"required,gt=0" example:"100.00"`
	Country  *string `json:"country" binding:"omitempty,len=2" example:"CA"`
}

// ApplyDiscountRequest represents the request to apply a discount
type ApplyDiscountRequest struct {
	Code       string  `json:"code" binding:"required" example:"SAVE10"`
	ShipmentID *string `json:"shipment_id" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvoiceID  *string `json:"invoice_id" binding:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	Subtotal   float64 `json:"subtotal" binding:"required,gt=0" example:"100.00"`
}

// =====================================================
// RESPONSE DTOs
// =====================================================

// DiscountResponse represents a discount in responses
type DiscountResponse struct {
	ID                     string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DiscountNumber         *string    `json:"discount_number,omitempty" example:"DSC-ABC12345"`
	Code                   string     `json:"code" example:"SAVE10"`
	Name                   *string    `json:"name,omitempty" example:"10% Off Summer Sale"`
	Description            *string    `json:"description,omitempty" example:"Get 10% off on all shipments"`
	DiscountType           string     `json:"discount_type" example:"percent"`
	Value                  float64    `json:"value" example:"10"`
	Currency               string     `json:"currency" example:"CAD"`
	Scope                  string     `json:"scope" example:"invoice"`
	MinSubtotalAmount      float64    `json:"min_subtotal_amount" example:"50.00"`
	MaxDiscountAmount      *float64   `json:"max_discount_amount,omitempty" example:"100.00"`
	AllowedCountry         *string    `json:"allowed_country,omitempty" example:"CA"`
	MaxUsesTotal           *int       `json:"max_uses_total,omitempty" example:"1000"`
	MaxUsesPerUser         *int       `json:"max_uses_per_user,omitempty" example:"1"`
	MaxTotalDiscountAmount *float64   `json:"max_total_discount_amount,omitempty" example:"10000.00"`
	StartsAt               *time.Time `json:"starts_at,omitempty" example:"2026-01-01T00:00:00Z"`
	EndsAt                 *time.Time `json:"ends_at,omitempty" example:"2026-12-31T23:59:59Z"`
	IsActive               bool       `json:"is_active" example:"true"`
	FirstTimeOnly          bool       `json:"first_time_only" example:"false"`
	UsageCount             int        `json:"usage_count" example:"150"`
	IsValid                bool       `json:"is_valid" example:"true"`
	CreatedAt              time.Time  `json:"created_at" example:"2026-01-01T10:00:00Z"`
	UpdatedAt              time.Time  `json:"updated_at" example:"2026-01-15T14:30:00Z"`
}

// DiscountsListResponse represents paginated discounts
type DiscountsListResponse struct {
	Discounts  []DiscountResponse `json:"discounts"`
	Pagination PaginationMeta     `json:"pagination"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page" example:"1"`
	PerPage    int `json:"per_page" example:"20"`
	Total      int `json:"total" example:"150"`
	TotalPages int `json:"total_pages" example:"8"`
}

// ValidateDiscountResponse represents the response for discount validation
type ValidateDiscountResponse struct {
	Valid          bool     `json:"valid" example:"true"`
	Code           string   `json:"code" example:"SAVE10"`
	DiscountType   string   `json:"discount_type" example:"percent"`
	Value          float64  `json:"value" example:"10"`
	DiscountAmount float64  `json:"discount_amount" example:"10.00"`
	FinalAmount    float64  `json:"final_amount" example:"90.00"`
	Message        *string  `json:"message,omitempty" example:"Discount applied successfully"`
	ErrorReason    *string  `json:"error_reason,omitempty" example:"Discount code has expired"`
}

// ApplyDiscountResponse represents the response for applying a discount
type ApplyDiscountResponse struct {
	RedemptionID   string    `json:"redemption_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	UsageNumber    string    `json:"usage_number" example:"RDM-ABC12345"`
	DiscountID     string    `json:"discount_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code           string    `json:"code" example:"SAVE10"`
	DiscountType   string    `json:"discount_type" example:"percent"`
	DiscountAmount float64   `json:"discount_amount" example:"10.00"`
	Currency       string    `json:"currency" example:"CAD"`
	AppliedAt      time.Time `json:"applied_at" example:"2026-01-15T14:30:00Z"`
}

// RedemptionResponse represents a discount redemption in responses
type RedemptionResponse struct {
	ID               string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`
	UsageNumber      *string   `json:"usage_number,omitempty" example:"RDM-ABC12345"`
	DiscountID       string    `json:"discount_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DiscountCode     string    `json:"discount_code" example:"SAVE10"`
	RedeemedByUserID string    `json:"redeemed_by_user_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	InvoiceID        *string   `json:"invoice_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	ShipmentID       *string   `json:"shipment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	Status           string    `json:"status" example:"applied"`
	Currency         string    `json:"currency" example:"CAD"`
	DiscountAmount   float64   `json:"discount_amount" example:"10.00"`
	RedeemedAt       time.Time `json:"redeemed_at" example:"2026-01-15T14:30:00Z"`
}

// RedemptionsListResponse represents paginated redemptions
type RedemptionsListResponse struct {
	Redemptions []RedemptionResponse `json:"redemptions"`
	Pagination  PaginationMeta       `json:"pagination"`
}
