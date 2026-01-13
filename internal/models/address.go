package models

import (
	"database/sql"
	"time"
)

// AddressType represents the purpose/role of an address
type AddressType string

const (
	AddressTypeSender    AddressType = "sender"
	AddressTypeRecipient AddressType = "recipient"
	AddressTypeBoth      AddressType = "both"
)

// Address represents an address book entry
type Address struct {
	ID               string      `json:"id" db:"id"`
	AddressNumber    string      `json:"address_number" db:"address_number"`
	OrganizationID   string      `json:"organization_id" db:"organization_id"`
	CreatedByUserID  sql.NullString `json:"created_by_user_id,omitempty" db:"created_by_user_id"`

	// Identity/Labeling
	Nickname      string `json:"nickname" db:"nickname"`
	Company       string `json:"company,omitempty" db:"company"`
	ContactName   string `json:"contact_name" db:"contact_name"`
	Phone         string `json:"phone,omitempty" db:"phone"`
	Email         string `json:"email,omitempty" db:"email"`

	// Address Lines
	AddressLine1  string `json:"address_line_1" db:"address_line_1"`
	AddressLine2  string `json:"address_line_2,omitempty" db:"address_line_2"`
	City          string `json:"city" db:"city"`
	ProvinceCode  string `json:"province_code,omitempty" db:"province_code"`
	PostalCode    string `json:"postal_code" db:"postal_code"`
	CountryCode   string `json:"country_code" db:"country_code"`

	// Shipping Semantics
	AddressType     AddressType `json:"address_type" db:"address_type"`
	IsResidential   bool        `json:"is_residential" db:"is_residential"`

	// Defaults & Status
	IsDefaultSender    bool `json:"is_default_sender" db:"is_default_sender"`
	IsDefaultRecipient bool `json:"is_default_recipient" db:"is_default_recipient"`
	IsVerified         bool `json:"is_verified" db:"is_verified"`
	IsActive           bool `json:"is_active" db:"is_active"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsSender checks if the address can be used as a sender
func (a *Address) IsSender() bool {
	return a.AddressType == AddressTypeSender || a.AddressType == AddressTypeBoth
}

// IsRecipient checks if the address can be used as a recipient
func (a *Address) IsRecipient() bool {
	return a.AddressType == AddressTypeRecipient || a.AddressType == AddressTypeBoth
}

// FormattedAddress returns the full formatted address
func (a *Address) FormattedAddress() string {
	address := a.AddressLine1
	if a.AddressLine2 != "" {
		address += ", " + a.AddressLine2
	}
	address += ", " + a.City
	if a.ProvinceCode != "" {
		address += ", " + a.ProvinceCode
	}
	address += " " + a.PostalCode
	address += ", " + a.CountryCode
	return address
}
