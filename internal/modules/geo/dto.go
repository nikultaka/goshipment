package geo

import (
	"time"
)

// =====================================================
// Response DTOs
// =====================================================

// CountryResponse represents a country in API responses
type CountryResponse struct {
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	CurrencyCode string    `json:"currency_code"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CountryListResponse represents a list of countries
type CountryListResponse struct {
	Countries []*CountryResponse `json:"countries"`
	Total     int                `json:"total"`
}

// RegionResponse represents a region/state/province in API responses
type RegionResponse struct {
	ID          string    `json:"id"`
	CountryCode string    `json:"country_code"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RegionListResponse represents a list of regions
type RegionListResponse struct {
	Regions []*RegionResponse `json:"regions"`
	Total   int               `json:"total"`
}
