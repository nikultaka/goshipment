package invoices

import (
	"net/http"
	"strconv"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// InvoiceHandlers handles invoice HTTP requests
type InvoiceHandlers struct {
	service *InvoiceService
}

// NewInvoiceHandlers creates a new invoice handlers instance
func NewInvoiceHandlers(service *InvoiceService) *InvoiceHandlers {
	return &InvoiceHandlers{
		service: service,
	}
}

// ListInvoices godoc
// @Summary List organization invoices
// @Description Retrieve paginated list of invoices for the organization. Users see only their own shipments' invoices. Managers and admins see all organization invoices.
// @Tags Invoices
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Number of invoices per page" default(20) minimum(1) maximum(100)
// @Param invoice_type query string false "Filter by invoice type" Enums(invoice, credit_note, adjustment)
// @Param status query string false "Filter by status" Enums(draft, issued, void)
// @Param payment_status query string false "Filter by payment status" Enums(unpaid, partial, paid, overpaid)
// @Param from_date query string false "Filter by created date (from)" format(date) example(2026-01-01)
// @Param to_date query string false "Filter by created date (to)" format(date) example(2026-01-31)
// @Success 200 {object} response.Response{data=InvoiceListResponse} "Invoices retrieved successfully"
// @Failure 400 {object} response.Response "Invalid request parameters"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /invoices [get]
func (h *InvoiceHandlers) ListInvoices(c *gin.Context) {
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

	// Parse query parameters
	var req ListInvoicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	// Get invoices
	result, err := h.service.GetInvoices(req, organizationID, userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get invoices: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Invoices retrieved successfully", result)
}

// GetInvoice godoc
// @Summary Get invoice details
// @Description Retrieve complete invoice details including line items, payments, and audit events
// @Tags Invoices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invoice ID" format(uuid)
// @Success 200 {object} response.Response{data=InvoiceDetailResponse} "Invoice retrieved successfully"
// @Failure 400 {object} response.Response "Invalid invoice ID"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Invoice not found or access denied"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /invoices/{id} [get]
func (h *InvoiceHandlers) GetInvoice(c *gin.Context) {
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

	// Get invoice ID from path
	invoiceID := c.Param("id")
	if invoiceID == "" {
		response.BadRequest(c, "Invoice ID is required")
		return
	}

	// Get invoice details
	result, err := h.service.GetInvoiceByID(invoiceID, organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "not found") || contains(err.Error(), "access denied") {
			response.NotFound(c, "Invoice not found or access denied")
			return
		}
		response.InternalError(c, "Failed to get invoice: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Invoice retrieved successfully", result)
}

// DownloadInvoice godoc
// @Summary Download invoice PDF
// @Description Download invoice as PDF file. The PDF is generated from snapshot data stored at invoice creation time.
// @Tags Invoices
// @Produce application/pdf
// @Security BearerAuth
// @Param id path string true "Invoice ID" format(uuid)
// @Success 200 {file} binary "Invoice PDF file"
// @Failure 400 {object} response.Response "Invalid invoice ID"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - user not in organization"
// @Failure 404 {object} response.Response "Invoice not found or access denied"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /invoices/{id}/download [get]
func (h *InvoiceHandlers) DownloadInvoice(c *gin.Context) {
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

	// Get invoice ID from path
	invoiceID := c.Param("id")
	if invoiceID == "" {
		response.BadRequest(c, "Invoice ID is required")
		return
	}

	// Generate PDF
	pdfBytes, err := h.service.GeneratePDF(invoiceID, organizationID, userID.(string))
	if err != nil {
		if contains(err.Error(), "not found") || contains(err.Error(), "access denied") {
			response.NotFound(c, "Invoice not found or access denied")
			return
		}
		response.InternalError(c, "Failed to generate PDF: "+err.Error())
		return
	}

	// Get invoice to determine filename
	invoice, err := h.service.GetInvoiceByID(invoiceID, organizationID, userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to get invoice details: "+err.Error())
		return
	}

	// Set headers for PDF download
	filename := invoice.InvoiceNumber + ".pdf"
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))

	// Write PDF bytes
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// RefundInvoice godoc
// @Summary Issue refund via credit note
// @Description Issue a refund for an invoice by creating a credit note and refunding to wallet. Requires Manager or Admin role. Creates a negative invoice (credit note) linked to the parent invoice.
// @Tags Invoices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invoice ID" format(uuid)
// @Param request body RefundInvoiceRequest true "Refund request"
// @Success 200 {object} response.Response{data=RefundResponse} "Refund successful"
// @Failure 400 {object} response.Response "Invalid request or refund amount exceeds refundable balance"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires Manager or Admin role"
// @Failure 404 {object} response.Response "Invoice not found or access denied"
// @Failure 409 {object} response.Response "Conflict - invoice cannot be refunded (voided, no balance, etc.)"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /invoices/{id}/refund [post]
func (h *InvoiceHandlers) RefundInvoice(c *gin.Context) {
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

	// Get invoice ID from path
	invoiceID := c.Param("id")
	if invoiceID == "" {
		response.BadRequest(c, "Invoice ID is required")
		return
	}

	// Parse request
	var req RefundInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Issue refund
	result, err := h.service.IssueRefund(req, invoiceID, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, "Invoice not found or access denied")
		case contains(errMsg, "cannot refund"), contains(errMsg, "no refundable balance"), contains(errMsg, "exceeds refundable"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		case contains(errMsg, "can only refund standard invoices"):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to issue refund: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Refund successful", result)
}

// VoidInvoice godoc
// @Summary Void invoice
// @Description Mark an invoice as void. Requires Admin role. Voided invoices cannot be used for any financial operations. This action is irreversible.
// @Tags Invoices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invoice ID" format(uuid)
// @Param request body VoidInvoiceRequest true "Void request"
// @Success 200 {object} response.Response{data=VoidResponse} "Invoice voided successfully"
// @Failure 400 {object} response.Response "Invalid request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - requires Admin role"
// @Failure 404 {object} response.Response "Invoice not found or access denied"
// @Failure 409 {object} response.Response "Conflict - invoice already voided or cannot be voided"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /invoices/{id}/void [post]
func (h *InvoiceHandlers) VoidInvoice(c *gin.Context) {
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

	// Get invoice ID from path
	invoiceID := c.Param("id")
	if invoiceID == "" {
		response.BadRequest(c, "Invoice ID is required")
		return
	}

	// Parse request
	var req VoidInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Void invoice
	result, err := h.service.VoidInvoice(req, invoiceID, organizationID, userID.(string))
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"), contains(errMsg, "access denied"):
			response.NotFound(c, "Invoice not found or access denied")
		case contains(errMsg, "already voided"), contains(errMsg, "can only void"):
			response.Error(c, http.StatusConflict, err.Error(), "CONFLICT")
		default:
			response.InternalError(c, "Failed to void invoice: "+err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Invoice voided successfully", result)
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
