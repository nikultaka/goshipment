package addresses

import (
	"gogin/internal/modules/organizations"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Address Book Handlers
// =====================================================

// listAddresses handles GET /addresses
// Lists all active address book entries for the organization
// Access: Any authenticated user with addresses.read scope
func (m *AddressesModule) listAddresses(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, _, err := orgService.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// List addresses (active only by default)
	addresses, err := m.service.ListAddresses(org.ID, true)
	if err != nil {
		response.InternalError(c, "Failed to list addresses")
		return
	}

	resp := &AddressListResponse{
		Addresses: addresses,
		Total:     len(addresses),
	}

	response.Success(c, 200, "Addresses retrieved successfully", resp)
}

// createAddress handles POST /addresses
// Creates a new address book entry
// Access: Any authenticated user with addresses.write scope
func (m *AddressesModule) createAddress(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, _, err := orgService.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Bind request
	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Create address
	address, err := m.service.CreateAddress(org.ID, userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 201, "Address created successfully", address)
}

// updateAddress handles PATCH /addresses/:id
// Updates an existing address book entry
// Access: Any authenticated user with addresses.write scope
func (m *AddressesModule) updateAddress(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get address ID from URL
	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, _, err := orgService.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Bind request
	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{
			{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Update address
	address, err := m.service.UpdateAddress(addressID, org.ID, &req)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, 200, "Address updated successfully", address)
}

// getAddress handles GET /addresses/:id
// Retrieves a single address by ID
// Access: Any authenticated user with addresses.read scope
func (m *AddressesModule) getAddress(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get address ID from URL
	addressID := c.Param("id")
	if addressID == "" {
		response.BadRequest(c, "Address ID is required")
		return
	}

	// Get user's organization
	orgService := organizations.NewOrganizationService(m.service.db)
	org, _, err := orgService.GetOrganizationByUserID(userID.(string))
	if err != nil {
		response.Forbidden(c, "User not associated with any organization")
		return
	}

	// Get address
	address, err := m.service.GetAddress(addressID, org.ID)
	if err != nil {
		if err.Error() == "address not found" {
			response.NotFound(c, "Address not found")
			return
		}
		response.InternalError(c, "Failed to get address")
		return
	}

	response.Success(c, 200, "Address retrieved successfully", address)
}
