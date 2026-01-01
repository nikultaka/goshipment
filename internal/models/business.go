package models

import (
	"database/sql"
	"time"
)

// Business represents a business profile associated with a user
type Business struct {
	ID           string         `json:"id" db:"id"`
	UserID       string         `json:"user_id" db:"user_id"`
	BusinessName string         `json:"business_name" db:"business_name"`
	LegalName    sql.NullString `json:"legal_name,omitempty" db:"legal_name"`
	BusinessType string         `json:"business_type" db:"business_type"`
	TaxID        sql.NullString `json:"tax_id,omitempty" db:"tax_id"`
	AvatarFileID sql.NullString `json:"avatar_file_id,omitempty" db:"avatar_file_id"`
	Website      sql.NullString `json:"website,omitempty" db:"website"`
	Description  sql.NullString `json:"description,omitempty" db:"description"`

	// Structured address
	AddressLine1 sql.NullString `json:"address_line1,omitempty" db:"address_line1"`
	AddressLine2 sql.NullString `json:"address_line2,omitempty" db:"address_line2"`
	City         sql.NullString `json:"city,omitempty" db:"city"`
	Province     sql.NullString `json:"province,omitempty" db:"province"`
	PostalCode   sql.NullString `json:"postal_code,omitempty" db:"postal_code"`
	Country      sql.NullString `json:"country,omitempty" db:"country"`

	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// BusinessEmail represents an email address associated with a business
type BusinessEmail struct {
	ID         string         `json:"id" db:"id"`
	BusinessID string         `json:"business_id" db:"business_id"`
	Email      string         `json:"email" db:"email"`
	EmailLower string         `json:"email_lower" db:"email_lower"`
	EmailType  string         `json:"email_type" db:"email_type"`
	IsPrimary  bool           `json:"is_primary" db:"is_primary"`
	IsVerified bool           `json:"is_verified" db:"is_verified"`
	VerifiedAt sql.NullTime   `json:"verified_at,omitempty" db:"verified_at"`
	IsActive   bool           `json:"is_active" db:"is_active"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
}

// BusinessPhone represents a phone number associated with a business
type BusinessPhone struct {
	ID         string         `json:"id" db:"id"`
	BusinessID string         `json:"business_id" db:"business_id"`
	Phone      string         `json:"phone" db:"phone"`
	PhoneE164  sql.NullString `json:"phone_e164,omitempty" db:"phone_e164"`
	PhoneType  string         `json:"phone_type" db:"phone_type"`
	IsPrimary  bool           `json:"is_primary" db:"is_primary"`
	IsVerified bool           `json:"is_verified" db:"is_verified"`
	VerifiedAt sql.NullTime   `json:"verified_at,omitempty" db:"verified_at"`
	IsActive   bool           `json:"is_active" db:"is_active"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
}
