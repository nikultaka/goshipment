package shipment_estimates

import (
	"gogin/internal/models"
	"gogin/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EstimateHandlers contains shipment estimate HTTP handlers
type EstimateHandlers struct {
	service *EstimateService
}

// NewEstimateHandlers creates new estimate handlers
func NewEstimateHandlers(service *EstimateService) *EstimateHandlers {
	return &EstimateHandlers{
		service: service,
	}
}

// CreateEstimate godoc
// @Summary Create shipment estimate (rate quote)
// @Description Get shipping rate quotes from multiple carriers. No authentication required for guest quotes. Authenticated users get quotes linked to their organization.
// @Tags Shipment Estimates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateEstimateRequest true "Estimate request payload"
// @Success 201 {object} response.Response{data=EstimateResponse} "Estimate created successfully with carrier rates"
// @Failure 400 {object} response.Response "Invalid request payload or no carrier rates available"
// @Failure 422 {object} response.Response "Validation failed"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipment-estimates [post]
func (h *EstimateHandlers) CreateEstimate(c *gin.Context) {
	var req CreateEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	// Get user info from context (may be nil for guest requests)
	var organizationID *string
	var userID *string

	if orgID, exists := c.Get("organization_id"); exists && orgID != nil {
		orgIDStr := orgID.(string)
		organizationID = &orgIDStr
	}

	if uid, exists := c.Get("user_id"); exists && uid != nil {
		uidStr := uid.(string)
		userID = &uidStr
	}

	// Create estimate
	estimate, packages, carrierRates, err := h.service.CreateEstimate(req, organizationID, userID)
	if err != nil {
		if err.Error() == "no carrier rates available for this route" {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	// Build response
	resp := h.buildEstimateResponse(estimate, packages, carrierRates)

	response.Success(c, http.StatusCreated, "Shipment estimate created successfully", resp)
}

// GetEstimate godoc
// @Summary Get shipment estimate by ID
// @Description Retrieve a shipment estimate with all carrier rates. Guest estimates accessible by anyone. Organization estimates require authentication and membership.
// @Tags Shipment Estimates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Estimate ID (UUID)"
// @Success 200 {object} response.Response{data=EstimateResponse} "Estimate retrieved successfully"
// @Failure 403 {object} response.Response "Access denied - estimate belongs to different organization"
// @Failure 404 {object} response.Response "Estimate not found"
// @Failure 410 {object} response.Response "Estimate has expired"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /shipment-estimates/{id} [get]
func (h *EstimateHandlers) GetEstimate(c *gin.Context) {
	estimateID := c.Param("id")

	// Get estimate
	estimate, packages, carrierRates, err := h.service.GetEstimateByID(estimateID)
	if err != nil {
		if err.Error() == "estimate not found" {
			response.NotFound(c, "Estimate not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	// Check if expired
	if estimate.IsExpired() {
		response.Error(c, http.StatusGone, "Estimate has expired", "ESTIMATE_EXPIRED")
		return
	}

	// Authorization check for org-scoped estimates
	var organizationID string
	var isGuest = true

	if orgID, exists := c.Get("organization_id"); exists && orgID != nil {
		organizationID = orgID.(string)
		isGuest = false
	}

	// Check access permissions
	if !estimate.CanBeAccessedBy(organizationID, isGuest) {
		response.Forbidden(c, "Access denied - estimate belongs to different organization")
		return
	}

	// Build response
	resp := h.buildEstimateResponse(estimate, packages, carrierRates)

	response.Success(c, http.StatusOK, "Estimate retrieved successfully", resp)
}

// buildEstimateResponse constructs the API response from models
func (h *EstimateHandlers) buildEstimateResponse(
	estimate *models.ShipmentEstimate,
	packages []*models.ShipmentPackage,
	carrierRates []models.CarrierRate,
) EstimateResponse {
	// Build from address
	var fromRegionCode, fromCity *string
	if estimate.FromRegionCode.Valid {
		fromRegionCode = &estimate.FromRegionCode.String
	}
	if estimate.FromCity.Valid {
		fromCity = &estimate.FromCity.String
	}
	fromAddress := ToAddressResponse(
		estimate.FromCountry,
		estimate.FromPostalCode,
		fromRegionCode,
		fromCity,
	)

	// Build to address
	var toRegionCode, toCity *string
	if estimate.ToRegionCode.Valid {
		toRegionCode = &estimate.ToRegionCode.String
	}
	if estimate.ToCity.Valid {
		toCity = &estimate.ToCity.String
	}
	toAddress := ToAddressResponse(
		estimate.ToCountry,
		estimate.ToPostalCode,
		toRegionCode,
		toCity,
	)

	// Build packages response
	packagesResp := make([]PackageResponse, len(packages))
	for i, pkg := range packages {
		packagesResp[i] = ToPackageResponse(pkg)
	}

	// Build carrier rates response
	carrierRatesResp := make([]CarrierRateResponse, len(carrierRates))
	for i, rate := range carrierRates {
		carrierRatesResp[i] = CarrierRateResponse{
			CarrierID:                rate.CarrierID,
			CarrierCode:              rate.CarrierCode,
			CarrierName:              rate.CarrierName,
			ServiceCode:              rate.ServiceCode,
			ServiceName:              rate.ServiceName,
			ServiceCategory:          rate.ServiceCategory,
			BaseRate:                 rate.BaseRate,
			FuelSurcharge:            rate.FuelSurcharge,
			OtherFees:                rate.OtherFees,
			Taxes:                    rate.Taxes,
			TotalAmount:              rate.TotalAmount,
			Currency:                 rate.Currency,
			EstimatedDeliveryDate:    nullStringToPtr(rate.EstimatedDeliveryDate),
			TransitDays:              nullIntToPtr(rate.TransitDays),
			SupportsPickup:           rate.SupportsPickup,
			SupportsSignature:        rate.SupportsSignature,
			SupportsSaturdayDelivery: rate.SupportsSaturdayDelivery,
		}
	}

	// Build main response
	// Use ID as estimate reference (estimate_number column not available in DB schema)
	resp := EstimateResponse{
		ID:             estimate.ID,
		EstimateNumber: estimate.ID,
		Status:         GetEstimateStatus(estimate),
		FromAddress:    fromAddress,
		ToAddress:      toAddress,
		Packages:       packagesResp,
		CarrierRates:   carrierRatesResp,
		TotalPackages:  estimate.TotalPackages,
		TotalWeight:    estimate.TotalWeight,
		WeightUnit:     estimate.WeightUnit,
		Currency:       estimate.Currency,
		ExpiresAt:      estimate.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:      estimate.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add selected rate if any
	if estimate.SelectedServiceLevelCode.Valid {
		resp.SelectedServiceCode = &estimate.SelectedServiceLevelCode.String
	}
	if estimate.SelectedAmount.Valid {
		resp.SelectedAmount = &estimate.SelectedAmount.Float64
	}

	// Add total amount if calculated
	if estimate.TotalAmount.Valid {
		resp.TotalAmount = &estimate.TotalAmount.Float64
	}

	return resp
}
