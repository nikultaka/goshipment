package shipment_estimates

import "gogin/internal/models"

// AddressInput represents address information for estimate
type AddressInput struct {
	Country    string  `json:"country" binding:"required,len=2" example:"CA"`
	PostalCode string  `json:"postal_code" binding:"required" example:"M5V3A8"`
	RegionCode *string `json:"region_code,omitempty" example:"ON"`
	City       *string `json:"city,omitempty" example:"Toronto"`
}

// PackageInput represents a package for estimate
type PackageInput struct {
	Length        float64  `json:"length" binding:"required,gt=0" example:"30"`
	Width         float64  `json:"width" binding:"required,gt=0" example:"20"`
	Height        float64  `json:"height" binding:"required,gt=0" example:"15"`
	DimensionUnit string   `json:"dimension_unit" binding:"required,oneof=cm in" example:"cm"`
	Weight        float64  `json:"weight" binding:"required,gt=0" example:"2.5"`
	WeightUnit    string   `json:"weight_unit" binding:"required,oneof=kg lb" example:"kg"`
	PackageName   *string  `json:"package_name,omitempty" example:"Box 1"`
	DeclaredValue *float64 `json:"declared_value,omitempty" example:"100.00"`
}

// CreateEstimateRequest represents the request to create a shipment estimate
type CreateEstimateRequest struct {
	FromAddress AddressInput   `json:"from_address" binding:"required"`
	ToAddress   AddressInput   `json:"to_address" binding:"required"`
	Packages    []PackageInput `json:"packages" binding:"required,min=1,max=10"`
	Source      *string        `json:"source,omitempty" example:"frontend"`

	// Optional: link to address book entries (for authenticated users)
	FromAddressBookID *string `json:"from_address_book_id,omitempty"`
	ToAddressBookID   *string `json:"to_address_book_id,omitempty"`
}

// CarrierRateResponse represents a single rate quote
type CarrierRateResponse struct {
	CarrierID                string  `json:"carrier_id"`
	CarrierCode              string  `json:"carrier_code"`
	CarrierName              string  `json:"carrier_name"`
	ServiceCode              string  `json:"service_code"`
	ServiceName              string  `json:"service_name"`
	ServiceCategory          string  `json:"service_category"`
	BaseRate                 float64 `json:"base_rate"`
	FuelSurcharge            float64 `json:"fuel_surcharge"`
	OtherFees                float64 `json:"other_fees"`
	Taxes                    float64 `json:"taxes"`
	TotalAmount              float64 `json:"total_amount"`
	Currency                 string  `json:"currency"`
	EstimatedDeliveryDate    *string `json:"estimated_delivery_date,omitempty"`
	TransitDays              *int    `json:"transit_days,omitempty"`
	SupportsPickup           bool    `json:"supports_pickup"`
	SupportsSignature        bool    `json:"supports_signature"`
	SupportsSaturdayDelivery bool    `json:"supports_saturday_delivery,omitempty"`
}

// PackageResponse represents package details in response
type PackageResponse struct {
	ID            string   `json:"id"`
	PackageIndex  int      `json:"package_index"`
	PackageName   *string  `json:"package_name,omitempty"`
	Length        float64  `json:"length"`
	Width         float64  `json:"width"`
	Height        float64  `json:"height"`
	DimensionUnit string   `json:"dimension_unit"`
	Weight        float64  `json:"weight"`
	WeightUnit    string   `json:"weight_unit"`
	DeclaredValue *float64 `json:"declared_value,omitempty"`
}

// EstimateResponse represents the full estimate response
type EstimateResponse struct {
	ID             string  `json:"id"`
	EstimateNumber string  `json:"estimate_number"`
	Status         string  `json:"status"` // "active", "expired", "locked"

	// Origin & destination
	FromAddress AddressResponse `json:"from_address"`
	ToAddress   AddressResponse `json:"to_address"`

	// Packages
	Packages []PackageResponse `json:"packages"`

	// Carrier rates
	CarrierRates []CarrierRateResponse `json:"carrier_rates"`

	// Selected rate (if any)
	SelectedCarrierCode *string  `json:"selected_carrier_code,omitempty"`
	SelectedServiceCode *string  `json:"selected_service_code,omitempty"`
	SelectedAmount      *float64 `json:"selected_amount,omitempty"`

	// Totals
	TotalPackages int      `json:"total_packages"`
	TotalWeight   float64  `json:"total_weight"`
	WeightUnit    string   `json:"weight_unit"`
	Currency      string   `json:"currency"`
	TotalAmount   *float64 `json:"total_amount,omitempty"`

	// Validity
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// AddressResponse represents address in response
type AddressResponse struct {
	Country    string  `json:"country"`
	PostalCode string  `json:"postal_code"`
	RegionCode *string `json:"region_code,omitempty"`
	City       *string `json:"city,omitempty"`
}

// Helper functions to convert models to DTOs

func nullStringToPtr(ns string) *string {
	if ns == "" {
		return nil
	}
	return &ns
}

func nullFloatToPtr(nf float64) *float64 {
	if nf == 0 {
		return nil
	}
	return &nf
}

func nullIntToPtr(ni int) *int {
	if ni == 0 {
		return nil
	}
	return &ni
}

// ToAddressResponse converts DB address fields to response DTO
func ToAddressResponse(country, postalCode string, regionCode, city *string) AddressResponse {
	return AddressResponse{
		Country:    country,
		PostalCode: postalCode,
		RegionCode: regionCode,
		City:       city,
	}
}

// ToPackageResponse converts ShipmentPackage model to response DTO
func ToPackageResponse(pkg *models.ShipmentPackage) PackageResponse {
	resp := PackageResponse{
		ID:            pkg.ID,
		PackageIndex:  pkg.PackageIndex,
		Length:        pkg.Length,
		Width:         pkg.Width,
		Height:        pkg.Height,
		DimensionUnit: pkg.DimensionUnit,
		Weight:        pkg.Weight,
		WeightUnit:    pkg.WeightUnit,
	}

	if pkg.PackageName.Valid {
		resp.PackageName = &pkg.PackageName.String
	}

	if pkg.DeclaredValue.Valid {
		resp.DeclaredValue = &pkg.DeclaredValue.Float64
	}

	return resp
}

// GetEstimateStatus determines the status of an estimate
func GetEstimateStatus(estimate *models.ShipmentEstimate) string {
	if estimate.IsExpired() {
		return "expired"
	}
	if estimate.IsLocked {
		return "locked"
	}
	return "active"
}
