package labels

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// LabelHandlers handles label HTTP requests
type LabelHandlers struct {
	service *LabelService
}

// NewLabelHandlers creates a new label handlers instance
func NewLabelHandlers(service *LabelService) *LabelHandlers {
	return &LabelHandlers{
		service: service,
	}
}

// PurchaseLabels godoc
// @Summary Purchase shipping labels for packages
// @Description Purchase shipping labels for all packages in a shipment. This is the monetization point - it debits the wallet or uses carrier billing. Label purchase is IRREVERSIBLE.
// @Tags Labels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PurchaseLabelsRequest true "Label purchase request"
// @Success 200 {object} response.Response{data=PurchaseLabelsResponse} "Labels purchased successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - insufficient permissions"
// @Failure 404 {object} response.Response "Shipment not found"
// @Failure 409 {object} response.Response "Shipment not in valid state for label purchase"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /labels/purchase [post]
func (h *LabelHandlers) PurchaseLabels(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Parse request
	var req PurchaseLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Purchase labels
	result, err := h.service.PurchaseLabels(req, organizationID, userID.(string))
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, err.Error())
		case contains(errMsg, "not in pre-label state"), contains(errMsg, "not in pending state"),
			contains(errMsg, "not in valid state"), contains(errMsg, "wallet hold not found"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "wallet is not active"):
			response.Forbidden(c, err.Error())
		default:
			response.InternalError(c, "Failed to purchase labels: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Labels purchased successfully", result)
}

// GetLabel godoc
// @Summary Get label and tracking information
// @Description Retrieve label URL, tracking number, and other details for a specific package
// @Tags Labels
// @Produce json
// @Security BearerAuth
// @Param package_id path string true "Package ID (UUID)" format(uuid)
// @Success 200 {object} response.Response{data=GetLabelResponse} "Label info retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Package not found or access denied"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /labels/{package_id} [get]
func (h *LabelHandlers) GetLabel(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		response.Unauthorized(c, "User context required")
		return
	}

	// Get organization ID from user membership
	organizationID, err := h.service.GetOrganizationIDFromUser(userID.(string))
	if err != nil {
		response.Forbidden(c, "User is not a member of any organization")
		return
	}

	// Get package ID from path
	packageID := c.Param("package_id")
	if packageID == "" {
		response.BadRequest(c, "Package ID is required")
		return
	}

	// Get label info
	result, err := h.service.GetLabelInfo(packageID, organizationID)
	if err != nil {
		if contains(err.Error(), "not found") || contains(err.Error(), "access denied") {
			response.NotFound(c, "Package not found or access denied")
			return
		}
		response.InternalError(c, "Failed to get label info: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Label info retrieved successfully", result)
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
