package geo

import (
	"strconv"
	"strings"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Geography Handlers
// =====================================================

// listCountries handles GET /geo/countries
// Lists all countries, optionally filtered by is_active query param
// Access: Any authenticated user with geo.read scope
func (m *GeoModule) listCountries(c *gin.Context) {
	// Parse is_active query param
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		parsed, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			response.BadRequest(c, "Invalid is_active parameter: must be true or false")
			return
		}
		isActive = &parsed
	}

	// List countries
	countries, err := m.service.ListCountries(isActive)
	if err != nil {
		response.InternalError(c, "Failed to list countries")
		return
	}

	resp := &CountryListResponse{
		Countries: countries,
		Total:     len(countries),
	}

	response.Success(c, 200, "Countries retrieved successfully", resp)
}

// listRegions handles GET /geo/countries/:code/regions
// Lists all regions for a specific country, optionally filtered by is_active
// Access: Any authenticated user with geo.read scope
func (m *GeoModule) listRegions(c *gin.Context) {
	// Get country code from URL (convert to uppercase for consistency)
	countryCode := strings.ToUpper(c.Param("code"))
	if countryCode == "" || len(countryCode) != 2 {
		response.BadRequest(c, "Country code is required and must be 2 characters (ISO 3166-1 alpha-2)")
		return
	}

	// Check if country exists
	exists, err := m.service.CountryExists(countryCode)
	if err != nil {
		response.InternalError(c, "Failed to verify country")
		return
	}
	if !exists {
		response.NotFound(c, "Country not found")
		return
	}

	// Parse is_active query param
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		parsed, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			response.BadRequest(c, "Invalid is_active parameter: must be true or false")
			return
		}
		isActive = &parsed
	}

	// List regions
	regions, err := m.service.ListRegionsByCountry(countryCode, isActive)
	if err != nil {
		response.InternalError(c, "Failed to list regions")
		return
	}

	resp := &RegionListResponse{
		Regions: regions,
		Total:   len(regions),
	}

	response.Success(c, 200, "Regions retrieved successfully", resp)
}
