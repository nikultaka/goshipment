package addresses

import (
	"time"
)

// =====================================================
// Request DTOs
// =====================================================

// CreateAddressRequest represents a request to create a new address
type CreateAddressRequest struct {
	Nickname           string `json:"nickname" binding:"required,min=1,max=100"`
	Company            string `json:"company" binding:"omitempty,max=150"`
	ContactName        string `json:"contact_name" binding:"required,min=1,max=150"`
	Phone              string `json:"phone" binding:"omitempty,max=30"`
	Email              string `json:"email" binding:"omitempty,email,max=255"`
	AddressLine1       string `json:"address_line_1" binding:"required,min=1,max=255"`
	AddressLine2       string `json:"address_line_2" binding:"omitempty,max=255"`
	City               string `json:"city" binding:"required,min=1,max=100"`
	ProvinceCode       string `json:"province_code" binding:"omitempty,max=50"`
	PostalCode         string `json:"postal_code" binding:"required,min=1,max=20"`
	CountryCode        string `json:"country_code" binding:"required,len=2"` // ISO 3166-1 alpha-2
	AddressType        string `json:"address_type" binding:"required,oneof=sender recipient both"`
	IsResidential      bool   `json:"is_residential"`
	IsDefaultSender    bool   `json:"is_default_sender"`
	IsDefaultRecipient bool   `json:"is_default_recipient"`
}

// UpdateAddressRequest represents a request to update an existing address
type UpdateAddressRequest struct {
	Nickname           *string `json:"nickname" binding:"omitempty,min=1,max=100"`
	Company            *string `json:"company" binding:"omitempty,max=150"`
	ContactName        *string `json:"contact_name" binding:"omitempty,min=1,max=150"`
	Phone              *string `json:"phone" binding:"omitempty,max=30"`
	Email              *string `json:"email" binding:"omitempty,email,max=255"`
	AddressLine1       *string `json:"address_line_1" binding:"omitempty,min=1,max=255"`
	AddressLine2       *string `json:"address_line_2" binding:"omitempty,max=255"`
	City               *string `json:"city" binding:"omitempty,min=1,max=100"`
	ProvinceCode       *string `json:"province_code" binding:"omitempty,max=50"`
	PostalCode         *string `json:"postal_code" binding:"omitempty,min=1,max=20"`
	CountryCode        *string `json:"country_code" binding:"omitempty,len=2"`
	AddressType        *string `json:"address_type" binding:"omitempty,oneof=sender recipient both"`
	IsResidential      *bool   `json:"is_residential"`
	IsDefaultSender    *bool   `json:"is_default_sender"`
	IsDefaultRecipient *bool   `json:"is_default_recipient"`
	IsActive           *bool   `json:"is_active"` // Allow deactivation
}

// =====================================================
// Response DTOs
// =====================================================

// AddressResponse represents an address in API responses
type AddressResponse struct {
	ID                 string    `json:"id"`
	AddressNumber      string    `json:"address_number"`
	OrganizationID     string    `json:"organization_id"`
	Nickname           string    `json:"nickname"`
	Company            string    `json:"company,omitempty"`
	ContactName        string    `json:"contact_name"`
	Phone              string    `json:"phone,omitempty"`
	Email              string    `json:"email,omitempty"`
	AddressLine1       string    `json:"address_line_1"`
	AddressLine2       string    `json:"address_line_2,omitempty"`
	City               string    `json:"city"`
	ProvinceCode       string    `json:"province_code,omitempty"`
	PostalCode         string    `json:"postal_code"`
	CountryCode        string    `json:"country_code"`
	AddressType        string    `json:"address_type"`
	IsResidential      bool      `json:"is_residential"`
	IsDefaultSender    bool      `json:"is_default_sender"`
	IsDefaultRecipient bool      `json:"is_default_recipient"`
	IsVerified         bool      `json:"is_verified"`
	IsActive           bool      `json:"is_active"`
	FormattedAddress   string    `json:"formatted_address"` // Computed field
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AddressListResponse represents a paginated list of addresses
type AddressListResponse struct {
	Addresses []*AddressResponse `json:"addresses"`
	Total     int                `json:"total"`
	Page      int                `json:"page,omitempty"`
	Limit     int                `json:"limit,omitempty"`
}
