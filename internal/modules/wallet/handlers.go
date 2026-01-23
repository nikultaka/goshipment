package wallet

import (
	"net/http"
	"strconv"

	"gogin/internal/middleware"
	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// WalletHandlers handles wallet HTTP requests
type WalletHandlers struct {
	service *WalletService
}

// NewWalletHandlers creates a new wallet handlers instance
func NewWalletHandlers(service *WalletService) *WalletHandlers {
	return &WalletHandlers{
		service: service,
	}
}

// GetWallet godoc
// @Summary Get organization wallet status
// @Description Retrieve current organization's wallet balance, held balance, and recent transactions
// @Tags Wallet
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number for transactions" default(1)
// @Param page_size query int false "Number of transactions per page" default(20)
// @Success 200 {object} response.Response{data=WalletResponse} "Wallet retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Wallet not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /wallet [get]
func (h *WalletHandlers) GetWallet(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access via query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		// Admin users must provide organization_id for wallet operations
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

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get wallet with transactions
	result, err := h.service.GetWalletWithTransactions(organizationID, page, pageSize)
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Wallet not found for organization")
			return
		}
		response.InternalError(c, "Failed to get wallet: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Wallet retrieved successfully", result)
}

// TopUp godoc
// @Summary Top up wallet via Stripe
// @Description Add funds to organization wallet using Stripe payment method
// @Tags Wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body TopUpRequest true "Top-up request"
// @Success 200 {object} response.Response{data=TopUpResponse} "Top-up successful"
// @Failure 400 {object} response.Response "Invalid request or amount constraints violated"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 402 {object} response.Response "Payment required - Stripe payment failed"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 409 {object} response.Response "Conflict - wallet is locked or disabled"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /wallet/topup [post]
func (h *WalletHandlers) TopUp(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access via query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		// Admin users must provide organization_id for wallet operations
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

	// Parse request
	var req TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Process top-up
	result, err := h.service.TopUp(req, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "wallet is") && contains(errMsg, "cannot accept"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "amount must be at least"), contains(errMsg, "amount cannot exceed"):
			response.BadRequest(c, err.Error())
		case contains(errMsg, "stripe payment failed"):
			response.Error(c, http.StatusPaymentRequired, err.Error(), "PAYMENT_REQUIRED")
		case contains(errMsg, "failed after"):
			response.Error(c, http.StatusConflict, "Concurrent update detected. Please retry.", "CONFLICT")
		default:
			response.InternalError(c, "Failed to process top-up: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Top-up successful", result)
}

// Refund godoc
// @Summary Refund shipment amount to wallet
// @Description Process a shipment refund to organization wallet (Manager/Admin only)
// @Tags Wallet
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefundRequest true "Refund request"
// @Success 200 {object} response.Response{data=RefundResponse} "Refund successful"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires Manager or Admin role"
// @Failure 404 {object} response.Response "Shipment not found"
// @Failure 409 {object} response.Response "Conflict - shipment already refunded"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /wallet/refund [post]
func (h *WalletHandlers) Refund(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access via query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		// Admin users must provide organization_id for wallet operations
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

	// Parse request
	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Process refund
	result, err := h.service.Refund(req, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, "Shipment not found or access denied")
		case contains(errMsg, "already been refunded"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "failed after"):
			response.Error(c, http.StatusConflict, "Concurrent update detected. Please retry.", "CONFLICT")
		default:
			response.InternalError(c, "Failed to process refund: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Refund successful", result)
}

// LockWallet godoc
// @Summary Lock organization wallet
// @Description Lock wallet to prevent all debit operations (Admin only)
// @Tags Wallet
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=WalletStatusResponse} "Wallet locked successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires Admin role"
// @Failure 404 {object} response.Response "Wallet not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /wallet/lock [post]
func (h *WalletHandlers) LockWallet(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access via query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		// Admin users must provide organization_id for wallet operations
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

	// Lock wallet
	result, err := h.service.LockWallet(organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Wallet not found for organization")
			return
		}
		response.InternalError(c, "Failed to lock wallet: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Wallet locked successfully", result)
}

// UnlockWallet godoc
// @Summary Unlock organization wallet
// @Description Unlock wallet to restore normal operations (Admin only)
// @Tags Wallet
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=WalletStatusResponse} "Wallet unlocked successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires Admin role"
// @Failure 404 {object} response.Response "Wallet not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /wallet/unlock [post]
func (h *WalletHandlers) UnlockWallet(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership (admin users can access via query param)
	organizationID, isAdminUser := middleware.GetOrganizationContext(c)
	if isAdminUser && organizationID == "" {
		// Admin users must provide organization_id for wallet operations
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

	// Unlock wallet
	result, err := h.service.UnlockWallet(organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "not found") {
			response.NotFound(c, "Wallet not found for organization")
			return
		}
		response.InternalError(c, "Failed to unlock wallet: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Wallet unlocked successfully", result)
}

// =====================================================
// UTILITY FUNCTIONS
// =====================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
