package addresses

import (
	"net/http"

	"gogin/internal/middleware"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// getOrganizationID helper to get organization ID from context or user membership
func (m *AddressModule) getOrganizationID(c *gin.Context) (string, error) {
	// Check if admin user (can pass organization_id as query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser {
		if organizationID == "" {
			// Admin without org_id param - this is okay for some operations
			return "", nil
		}
		return organizationID, nil
	}

	// Regular user - get organization from membership
	userID, exists := c.Get("user_id")
	if !exists {
		return "", nil
	}

	orgID, err := m.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		return "", err
	}
	return orgID, nil
}

// listAddresses lists all addresses in the organization's address book
// @Summary List addresses
// @Description Get a paginated list of addresses in the organization's address book
// @Tags Address Book
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param address_type query string false "Filter by address type" Enums(sender, recipient, both)
// @Param search query string false "Search by nickname, contact name, company, or city"
// @Param is_default query bool false "Filter to show only default addresses"
// @Param organization_id query string false "Organization ID (admin only)"
// @Success 200 {object} response.Response{data=AddressListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /addresses [get]
func (m *AddressModule) listAddresses(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	var req ListAddressesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	addresses, total, err := m.service.ListAddresses(organizationID, &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Set defaults for pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	totalPages := (total + limit - 1) / limit

	// Convert to response DTOs
	addressResponses := make([]*AddressResponse, len(addresses))
	for i, addr := range addresses {
		addressResponses[i] = ToAddressResponse(addr)
	}

	response.Success(c, http.StatusOK, "Addresses retrieved successfully", AddressListResponse{
		Addresses:  addressResponses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// createAddress creates a new address in the address book
// @Summary Create address
// @Description Create a new address in the organization's address book
// @Tags Address Book
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id query string false "Organization ID (admin only)"
// @Param request body CreateAddressRequest true "Address details"
// @Success 201 {object} response.Response{data=AddressResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /addresses [post]
func (m *AddressModule) createAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	userID, _ := c.Get("user_id")

	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	address, err := m.service.CreateAddress(organizationID, userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Address created successfully", ToAddressResponse(address))
}

// getAddress retrieves a specific address by ID
// @Summary Get address
// @Description Get a specific address from the organization's address book
// @Tags Address Book
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param organization_id query string false "Organization ID (admin only)"
// @Success 200 {object} response.Response{data=AddressResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/{id} [get]
func (m *AddressModule) getAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	address, err := m.service.GetAddressByID(addressID, organizationID)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Address retrieved successfully", ToAddressResponse(address))
}

// updateAddress updates an existing address
// @Summary Update address
// @Description Update an existing address in the organization's address book
// @Tags Address Book
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param organization_id query string false "Organization ID (admin only)"
// @Param request body UpdateAddressRequest true "Address updates"
// @Success 200 {object} response.Response{data=AddressResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/{id} [patch]
func (m *AddressModule) updateAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	address, err := m.service.UpdateAddress(addressID, organizationID, &req)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Address updated successfully", ToAddressResponse(address))
}

// deleteAddress soft-deletes an address
// @Summary Delete address
// @Description Delete an address from the organization's address book
// @Tags Address Book
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param organization_id query string false "Organization ID (admin only)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/{id} [delete]
func (m *AddressModule) deleteAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	err = m.service.DeleteAddress(addressID, organizationID)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// setDefaultAddress sets an address as the default sender or recipient
// @Summary Set default address
// @Description Set an address as the default sender or recipient for the organization
// @Tags Address Book
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param organization_id query string false "Organization ID (admin only)"
// @Param request body SetDefaultRequest true "Default type (sender or recipient)"
// @Success 200 {object} response.Response{data=AddressResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/{id}/set-default [post]
func (m *AddressModule) setDefaultAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	address, err := m.service.SetDefaultAddress(addressID, organizationID, req.DefaultType)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Default address set successfully", ToAddressResponse(address))
}

// clearDefaultAddress removes the default status from an address
// @Summary Clear default address
// @Description Remove the default sender or recipient status from an address
// @Tags Address Book
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Param organization_id query string false "Organization ID (admin only)"
// @Param request body SetDefaultRequest true "Default type to clear (sender or recipient)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/{id}/clear-default [post]
func (m *AddressModule) clearDefaultAddress(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err = m.service.ClearDefault(addressID, organizationID, req.DefaultType)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Default status cleared successfully", nil)
}

// getDefaultSender retrieves the default sender address
// @Summary Get default sender
// @Description Get the default sender address for the organization
// @Tags Address Book
// @Produce json
// @Security BearerAuth
// @Param organization_id query string false "Organization ID (admin only)"
// @Success 200 {object} response.Response{data=AddressResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/default/sender [get]
func (m *AddressModule) getDefaultSender(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	address, err := m.service.GetDefaultAddress(organizationID, "sender")
	if err != nil {
		if err.Error() == "no default sender address found" {
			response.NotFound(c, "No default sender address found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Default sender address retrieved successfully", ToAddressResponse(address))
}

// getDefaultRecipient retrieves the default recipient address
// @Summary Get default recipient
// @Description Get the default recipient address for the organization
// @Tags Address Book
// @Produce json
// @Security BearerAuth
// @Param organization_id query string false "Organization ID (admin only)"
// @Success 200 {object} response.Response{data=AddressResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /addresses/default/recipient [get]
func (m *AddressModule) getDefaultRecipient(c *gin.Context) {
	organizationID, err := m.getOrganizationID(c)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	if organizationID == "" {
		response.Forbidden(c, "Organization ID is required")
		return
	}

	address, err := m.service.GetDefaultAddress(organizationID, "recipient")
	if err != nil {
		if err.Error() == "no default recipient address found" {
			response.NotFound(c, "No default recipient address found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Default recipient address retrieved successfully", ToAddressResponse(address))
}
