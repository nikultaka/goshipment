package addresses

import (
	"time"

	"gogin/internal/models"
)

// =====================================================
// Request DTOs
// =====================================================

// CreateAddressRequest represents a request to create an address
type CreateAddressRequest struct {
	Nickname     string `json:"nickname" binding:"required,min=1,max=100"`
	Company      string `json:"company" binding:"omitempty,max=255"`
	ContactName  string `json:"contact_name" binding:"required,min=1,max=255"`
	Phone        string `json:"phone" binding:"omitempty,max=20"`
	Email        string `json:"email" binding:"omitempty,email,max=255"`
	AddressLine1 string `json:"address_line_1" binding:"required,min=1,max=255"`
	AddressLine2 string `json:"address_line_2" binding:"omitempty,max=255"`
	City         string `json:"city" binding:"required,min=1,max=100"`
	ProvinceCode string `json:"province_code" binding:"omitempty,max=10"`
	PostalCode   string `json:"postal_code" binding:"required,min=1,max=20"`
	CountryCode  string `json:"country_code" binding:"required,len=2"`
	AddressType  string `json:"address_type" binding:"required,oneof=sender recipient both"`
	IsResidential bool  `json:"is_residential"`
	IsDefaultSender    bool `json:"is_default_sender"`
	IsDefaultRecipient bool `json:"is_default_recipient"`
}

// UpdateAddressRequest represents a request to update an address
type UpdateAddressRequest struct {
	Nickname     *string `json:"nickname" binding:"omitempty,min=1,max=100"`
	Company      *string `json:"company" binding:"omitempty,max=255"`
	ContactName  *string `json:"contact_name" binding:"omitempty,min=1,max=255"`
	Phone        *string `json:"phone" binding:"omitempty,max=20"`
	Email        *string `json:"email" binding:"omitempty,email,max=255"`
	AddressLine1 *string `json:"address_line_1" binding:"omitempty,min=1,max=255"`
	AddressLine2 *string `json:"address_line_2" binding:"omitempty,max=255"`
	City         *string `json:"city" binding:"omitempty,min=1,max=100"`
	ProvinceCode *string `json:"province_code" binding:"omitempty,max=10"`
	PostalCode   *string `json:"postal_code" binding:"omitempty,min=1,max=20"`
	CountryCode  *string `json:"country_code" binding:"omitempty,len=2"`
	AddressType  *string `json:"address_type" binding:"omitempty,oneof=sender recipient both"`
	IsResidential *bool  `json:"is_residential"`
}

// ListAddressesRequest represents query parameters for listing addresses
type ListAddressesRequest struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
	AddressType string `form:"address_type" binding:"omitempty,oneof=sender recipient both"`
	Search      string `form:"search" binding:"omitempty,max=100"`
	IsDefault   *bool  `form:"is_default"`
}

// SetDefaultRequest represents a request to set an address as default
type SetDefaultRequest struct {
	DefaultType string `json:"default_type" binding:"required,oneof=sender recipient"`
}

// =====================================================
// Response DTOs
// =====================================================

// AddressResponse represents an address in API responses
type AddressResponse struct {
	ID                 string    `json:"id"`
	AddressNumber      string    `json:"address_number"`
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
	FormattedAddress   string    `json:"formatted_address"`
	AddressType        string    `json:"address_type"`
	IsResidential      bool      `json:"is_residential"`
	IsDefaultSender    bool      `json:"is_default_sender"`
	IsDefaultRecipient bool      `json:"is_default_recipient"`
	IsVerified         bool      `json:"is_verified"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AddressListResponse represents a paginated list of addresses
type AddressListResponse struct {
	Addresses  []*AddressResponse `json:"addresses"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// =====================================================
// Conversion Functions
// =====================================================

// ToAddressResponse converts an Address model to AddressResponse DTO
func ToAddressResponse(address *models.Address) *AddressResponse {
	return &AddressResponse{
		ID:                 address.ID,
		AddressNumber:      address.AddressNumber,
		Nickname:           address.Nickname,
		Company:            address.Company,
		ContactName:        address.ContactName,
		Phone:              address.Phone,
		Email:              address.Email,
		AddressLine1:       address.AddressLine1,
		AddressLine2:       address.AddressLine2,
		City:               address.City,
		ProvinceCode:       address.ProvinceCode,
		PostalCode:         address.PostalCode,
		CountryCode:        address.CountryCode,
		FormattedAddress:   address.FormattedAddress(),
		AddressType:        string(address.AddressType),
		IsResidential:      address.IsResidential,
		IsDefaultSender:    address.IsDefaultSender,
		IsDefaultRecipient: address.IsDefaultRecipient,
		IsVerified:         address.IsVerified,
		CreatedAt:          address.CreatedAt,
		UpdatedAt:          address.UpdatedAt,
	}
}
