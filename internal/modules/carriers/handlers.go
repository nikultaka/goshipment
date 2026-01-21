package carriers

import (
	"time"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Carrier Handlers
// =====================================================

// listCarriers handles GET /carriers
// Lists all available carriers
func (m *CarrierModule) listCarriers(c *gin.Context) {
	// Parse is_active query param (default: true)
	activeOnly := true
	if c.Query("all") == "true" {
		activeOnly = false
	}

	carriers, err := m.service.ListCarriers(activeOnly)
	if err != nil {
		response.InternalError(c, "Failed to list carriers")
		return
	}

	resp := &CarrierListResponse{
		Carriers: carriers,
		Total:    len(carriers),
	}

	response.Success(c, 200, "Carriers retrieved successfully", resp)
}

// getCarrier handles GET /carriers/:id
// Gets a carrier by ID
func (m *CarrierModule) getCarrier(c *gin.Context) {
	carrierID := c.Param("id")
	if carrierID == "" {
		response.BadRequest(c, "Carrier ID is required")
		return
	}

	carrier, err := m.service.GetCarrierByID(carrierID)
	if err != nil {
		if err.Error() == "carrier not found" {
			response.NotFound(c, "Carrier not found")
			return
		}
		response.InternalError(c, "Failed to get carrier")
		return
	}

	response.Success(c, 200, "Carrier retrieved successfully", carrier)
}

// getCarrierAvailability handles GET /carriers/:code/availability
// Gets carrier availability for a specific date
func (m *CarrierModule) getCarrierAvailability(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "Carrier code is required")
		return
	}

	// Parse date query param (default: today)
	dateStr := c.Query("date")
	var date time.Time
	if dateStr == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(c, "Invalid date format. Use YYYY-MM-DD")
			return
		}
	}

	availability, err := m.service.GetCarrierAvailability(code, date)
	if err != nil {
		if err.Error() == "carrier not found" {
			response.NotFound(c, "Carrier not found")
			return
		}
		response.InternalError(c, "Failed to get carrier availability: "+err.Error())
		return
	}

	response.Success(c, 200, "Carrier availability retrieved successfully", availability)
}

// getCarrierServiceLevels handles GET /carriers/:id/services
// Gets service levels for a carrier
func (m *CarrierModule) getCarrierServiceLevels(c *gin.Context) {
	carrierID := c.Param("id")
	if carrierID == "" {
		response.BadRequest(c, "Carrier ID is required")
		return
	}

	// Verify carrier exists
	_, err := m.service.GetCarrierByID(carrierID)
	if err != nil {
		if err.Error() == "carrier not found" {
			response.NotFound(c, "Carrier not found")
			return
		}
		response.InternalError(c, "Failed to verify carrier")
		return
	}

	levels, err := m.service.GetServiceLevels(carrierID)
	if err != nil {
		response.InternalError(c, "Failed to get service levels")
		return
	}

	response.Success(c, 200, "Service levels retrieved successfully", gin.H{
		"service_levels": levels,
		"total":          len(levels),
	})
}
