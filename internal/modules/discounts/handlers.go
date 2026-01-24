package discounts

import (
	"net/http"

	"gogin/internal/middleware"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// DiscountHandlers handles discount HTTP requests
type DiscountHandlers struct {
	service *DiscountService
}

// NewDiscountHandlers creates a new discount handlers instance
func NewDiscountHandlers(service *DiscountService) *DiscountHandlers {
	return &DiscountHandlers{
		service: service,
	}
}

// CreateDiscount godoc
// @Summary Create a new discount
// @Description Create a new discount/promo code for the organization
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateDiscountRequest true "Discount details"
// @Success 201 {object} response.Response{data=DiscountResponse} "Discount created successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 409 {object} response.Response "Discount code already exists"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts [post]
func (h *DiscountHandlers) CreateDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	var req CreateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.service.CreateDiscount(req, organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "already exists") {
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
			return
		}
		response.InternalError(c, "Failed to create discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Discount created successfully", result)
}

// GetDiscount godoc
// @Summary Get discount by ID
// @Description Retrieve a specific discount by its ID
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Discount ID (UUID)"
// @Success 200 {object} response.Response{data=DiscountResponse} "Discount retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Discount not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts/{id} [get]
func (h *DiscountHandlers) GetDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	discountID := c.Param("id")
	if discountID == "" {
		response.BadRequest(c, "Discount ID is required")
		return
	}

	result, err := h.service.GetDiscountByID(discountID, organizationID)
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Discount not found")
			return
		}
		response.InternalError(c, "Failed to get discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discount retrieved successfully", result)
}

// ListDiscounts godoc
// @Summary List discounts
// @Description Retrieve paginated list of discounts with optional filters
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param code query string false "Search by discount code"
// @Param discount_type query string false "Filter by type" Enums(percent, fixed_amount, free_shipping)
// @Param scope query string false "Filter by scope" Enums(invoice, shipment)
// @Param is_active query bool false "Filter by active status"
// @Param sort_by query string false "Sort by field" Enums(created_at, code, value)
// @Param sort_order query string false "Sort order" Enums(asc, desc)
// @Success 200 {object} response.Response{data=DiscountsListResponse} "Discounts retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts [get]
func (h *DiscountHandlers) ListDiscounts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	var req ListDiscountsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	result, err := h.service.ListDiscounts(organizationID, req)
	if err != nil {
		response.InternalError(c, "Failed to list discounts: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discounts retrieved successfully", result)
}

// UpdateDiscount godoc
// @Summary Update a discount
// @Description Update an existing discount
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Discount ID (UUID)"
// @Param request body UpdateDiscountRequest true "Discount update details"
// @Success 200 {object} response.Response{data=DiscountResponse} "Discount updated successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Discount not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts/{id} [put]
func (h *DiscountHandlers) UpdateDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	discountID := c.Param("id")
	if discountID == "" {
		response.BadRequest(c, "Discount ID is required")
		return
	}

	var req UpdateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.service.UpdateDiscount(discountID, req, organizationID)
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Discount not found")
			return
		}
		response.InternalError(c, "Failed to update discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discount updated successfully", result)
}

// DeleteDiscount godoc
// @Summary Delete a discount
// @Description Delete an existing discount (only if no active redemptions)
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Discount ID (UUID)"
// @Success 200 {object} response.Response "Discount deleted successfully"
// @Failure 400 {object} response.Response "Cannot delete discount with active redemptions"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Discount not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts/{id} [delete]
func (h *DiscountHandlers) DeleteDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	discountID := c.Param("id")
	if discountID == "" {
		response.BadRequest(c, "Discount ID is required")
		return
	}

	err := h.service.DeleteDiscount(discountID, organizationID)
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Discount not found")
			return
		}
		if contains(err.Error(), "active redemptions") {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Failed to delete discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discount deleted successfully", nil)
}

// ValidateDiscount godoc
// @Summary Validate a discount code
// @Description Validate if a discount code can be applied to an order
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ValidateDiscountRequest true "Validation request"
// @Success 200 {object} response.Response{data=ValidateDiscountResponse} "Discount validation result"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts/validate [post]
func (h *DiscountHandlers) ValidateDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	var req ValidateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.service.ValidateDiscount(req, organizationID, userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to validate discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discount validation completed", result)
}

// ApplyDiscount godoc
// @Summary Apply a discount code
// @Description Apply a discount code to a shipment or invoice
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ApplyDiscountRequest true "Apply discount request"
// @Success 200 {object} response.Response{data=ApplyDiscountResponse} "Discount applied successfully"
// @Failure 400 {object} response.Response "Invalid request or discount validation failed"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /discounts/apply [post]
func (h *DiscountHandlers) ApplyDiscount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		response.BadRequest(c, "organization_id query parameter is required for admin users")
		return
	} else if !isAdminUser {
		var err error
		organizationID, err = h.service.GetOrganizationIDFromUser(userID.(string))
		if err != nil {
			response.Forbidden(c, "User is not a member of any organization")
			return
		}
	}

	var req ApplyDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Validate that either shipment_id or invoice_id is provided
	if req.ShipmentID == nil && req.InvoiceID == nil {
		response.BadRequest(c, "Either shipment_id or invoice_id is required")
		return
	}

	result, err := h.service.ApplyDiscount(req, organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "validation failed") {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Failed to apply discount: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Discount applied successfully", result)
}
