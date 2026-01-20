package invoices

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"gogin/internal/clients"
)

// InvoiceService handles invoice business logic
type InvoiceService struct {
	db *clients.Database
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(db *clients.Database) *InvoiceService {
	return &InvoiceService{
		db: db,
	}
}

// =====================================================
// ORGANIZATION SCOPING
// =====================================================

// GetOrganizationIDFromUser gets organization ID from user membership
func (s *InvoiceService) GetOrganizationIDFromUser(userID string) (string, error) {
	query := `
		SELECT organization_id
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`
	var organizationID string
	err := s.db.DB.QueryRow(query, userID).Scan(&organizationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user is not a member of any organization")
		}
		return "", fmt.Errorf("failed to get organization: %w", err)
	}
	return organizationID, nil
}

// GetUserRole gets user's role in their organization
func (s *InvoiceService) GetUserRole(userID string) (string, error) {
	query := `
		SELECT role
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`
	var role string
	err := s.db.DB.QueryRow(query, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user is not a member of any organization")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}
	return role, nil
}

// =====================================================
// LIST INVOICES
// =====================================================

// GetInvoices retrieves invoices with filtering and pagination
func (s *InvoiceService) GetInvoices(req ListInvoicesRequest, organizationID, userID string) (*InvoiceListResponse, error) {
	// Get user role to determine access level
	role, err := s.GetUserRole(userID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Build query based on role
	// Users see only invoices for their shipments
	// Managers and admins see all organization invoices
	var query string
	var countQuery string
	args := []interface{}{organizationID}
	argIndex := 2

	if role == "user" {
		// Users see only their own shipments' invoices
		query = `
			SELECT
				i.id, i.invoice_number, i.organization_id, o.name as organization_name,
				i.shipment_id, s.shipment_number,
				i.invoice_type, i.status, i.payment_status, i.currency,
				i.subtotal_amount, i.taxes_amount, i.discount_amount, i.total_amount,
				i.amount_paid, i.balance_due,
				i.issued_at, i.due_at, i.voided_at,
				i.parent_invoice_id, i.credit_note_reason, i.notes AS credit_note_reason_notes,
				i.created_at, i.updated_at
			FROM invoices i
			JOIN organizations o ON i.organization_id = o.id
			LEFT JOIN shipments s ON i.shipment_id = s.id
			WHERE i.organization_id = $1
			AND (i.shipment_id IN (SELECT id FROM shipments WHERE created_by_user_id = $` + fmt.Sprint(argIndex) + `)
			     OR i.invoice_type = 'credit_note')
		`
		countQuery = `
			SELECT COUNT(*)
			FROM invoices i
			WHERE i.organization_id = $1
			AND (i.shipment_id IN (SELECT id FROM shipments WHERE created_by_user_id = $` + fmt.Sprint(argIndex) + `)
			     OR i.invoice_type = 'credit_note')
		`
		args = append(args, userID)
		argIndex++
	} else {
		// Managers and admins see all organization invoices
		query = `
			SELECT
				i.id, i.invoice_number, i.organization_id, o.name as organization_name,
				i.shipment_id, s.shipment_number,
				i.invoice_type, i.status, i.payment_status, i.currency,
				i.subtotal_amount, i.taxes_amount, i.discount_amount, i.total_amount,
				i.amount_paid, i.balance_due,
				i.issued_at, i.due_at, i.voided_at,
				i.parent_invoice_id, i.credit_note_reason, i.notes AS credit_note_reason_notes,
				i.created_at, i.updated_at
			FROM invoices i
			JOIN organizations o ON i.organization_id = o.id
			LEFT JOIN shipments s ON i.shipment_id = s.id
			WHERE i.organization_id = $1
		`
		countQuery = `
			SELECT COUNT(*)
			FROM invoices i
			WHERE i.organization_id = $1
		`
	}

	// Add filters
	if req.InvoiceType != "" {
		query += fmt.Sprintf(" AND i.invoice_type = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND i.invoice_type = $%d", argIndex)
		args = append(args, req.InvoiceType)
		argIndex++
	}
	if req.Status != "" {
		query += fmt.Sprintf(" AND i.status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND i.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}
	if req.PaymentStatus != "" {
		query += fmt.Sprintf(" AND i.payment_status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND i.payment_status = $%d", argIndex)
		args = append(args, req.PaymentStatus)
		argIndex++
	}
	if !req.FromDate.IsZero() {
		query += fmt.Sprintf(" AND i.created_at >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND i.created_at >= $%d", argIndex)
		args = append(args, req.FromDate)
		argIndex++
	}
	if !req.ToDate.IsZero() {
		query += fmt.Sprintf(" AND i.created_at <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND i.created_at <= $%d", argIndex)
		args = append(args, req.ToDate)
		argIndex++
	}

	// Get total count
	var totalCount int
	err = s.db.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count invoices: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY i.created_at DESC"
	offset := (req.Page - 1) * req.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.PageSize, offset)

	// Execute query
	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	invoices := []InvoiceResponse{}
	for rows.Next() {
		var inv InvoiceResponse
		err := rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.OrganizationID, &inv.OrganizationName,
			&inv.ShipmentID, &inv.ShipmentNumber,
			&inv.InvoiceType, &inv.Status, &inv.PaymentStatus, &inv.Currency,
			&inv.SubtotalAmount, &inv.TaxAmount, &inv.DiscountAmount, &inv.TotalAmount,
			&inv.AmountPaid, &inv.BalanceDue,
			&inv.IssuedAt, &inv.DueDate, &inv.VoidedAt,
			&inv.ParentInvoiceID, &inv.CreditNoteReason, &inv.CreditNoteReasonNotes,
			&inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(req.PageSize)))

	return &InvoiceListResponse{
		Invoices:   invoices,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// =====================================================
// GET INVOICE DETAILS
// =====================================================

// GetInvoiceByID retrieves complete invoice details with line items, payments, and events
func (s *InvoiceService) GetInvoiceByID(invoiceID, organizationID, userID string) (*InvoiceDetailResponse, error) {
	// Get user role for access control
	role, err := s.GetUserRole(userID)
	if err != nil {
		return nil, err
	}

	// Get invoice header
	var query string
	var args []interface{}

	if role == "user" {
		// Users can only see invoices for their own shipments
		query = `
			SELECT
				i.id, i.invoice_number, i.organization_id, o.name as organization_name,
				i.shipment_id, s.shipment_number,
				i.invoice_type, i.status, i.payment_status, i.currency,
				i.subtotal_amount, i.taxes_amount, i.discount_amount, i.total_amount,
				i.amount_paid, i.balance_due,
				i.issued_at, i.due_at, i.voided_at,
				i.parent_invoice_id, i.credit_note_reason, i.notes AS credit_note_reason_notes,
				i.bill_to_name, i.bill_to_company, i.bill_to_email, i.bill_to_phone,
				i.bill_to_address_line1, i.bill_to_address_line2, i.bill_to_city,
				i.bill_to_region_code, i.bill_to_postal_code, i.bill_to_country,
				i.shipment_from_summary, i.shipment_to_summary,
				i.carrier_code, i.carrier_name, i.service_name, i.total_packages,
				i.created_at, i.updated_at
			FROM invoices i
			JOIN organizations o ON i.organization_id = o.id
			LEFT JOIN shipments s ON i.shipment_id = s.id
			WHERE i.id = $1 AND i.organization_id = $2
			AND (i.shipment_id IN (SELECT id FROM shipments WHERE created_by_user_id = $3)
			     OR i.invoice_type = 'credit_note')
		`
		args = []interface{}{invoiceID, organizationID, userID}
	} else {
		// Managers and admins see all organization invoices
		query = `
			SELECT
				i.id, i.invoice_number, i.organization_id, o.name as organization_name,
				i.shipment_id, s.shipment_number,
				i.invoice_type, i.status, i.payment_status, i.currency,
				i.subtotal_amount, i.taxes_amount, i.discount_amount, i.total_amount,
				i.amount_paid, i.balance_due,
				i.issued_at, i.due_at, i.voided_at,
				i.parent_invoice_id, i.credit_note_reason, i.notes AS credit_note_reason_notes,
				i.bill_to_name, i.bill_to_company, i.bill_to_email, i.bill_to_phone,
				i.bill_to_address_line1, i.bill_to_address_line2, i.bill_to_city,
				i.bill_to_region_code, i.bill_to_postal_code, i.bill_to_country,
				i.shipment_from_summary, i.shipment_to_summary,
				i.carrier_code, i.carrier_name, i.service_name, i.total_packages,
				i.created_at, i.updated_at
			FROM invoices i
			JOIN organizations o ON i.organization_id = o.id
			LEFT JOIN shipments s ON i.shipment_id = s.id
			WHERE i.id = $1 AND i.organization_id = $2
		`
		args = []interface{}{invoiceID, organizationID}
	}

	var inv InvoiceDetailResponse
	// Billing address fields
	var billToName, billToCompany, billToEmail, billToPhone sql.NullString
	var billToAddr1, billToAddr2, billToCity, billToRegion, billToPostal, billToCountry sql.NullString
	// Shipment snapshot fields
	var shipFromSummary, shipToSummary sql.NullString
	var carrierCode, carrierName, serviceName sql.NullString
	var totalPackages int

	err = s.db.DB.QueryRow(query, args...).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.OrganizationID, &inv.OrganizationName,
		&inv.ShipmentID, &inv.ShipmentNumber,
		&inv.InvoiceType, &inv.Status, &inv.PaymentStatus, &inv.Currency,
		&inv.SubtotalAmount, &inv.TaxAmount, &inv.DiscountAmount, &inv.TotalAmount,
		&inv.AmountPaid, &inv.BalanceDue,
		&inv.IssuedAt, &inv.DueDate, &inv.VoidedAt,
		&inv.ParentInvoiceID, &inv.CreditNoteReason, &inv.CreditNoteReasonNotes,
		&billToName, &billToCompany, &billToEmail, &billToPhone,
		&billToAddr1, &billToAddr2, &billToCity,
		&billToRegion, &billToPostal, &billToCountry,
		&shipFromSummary, &shipToSummary,
		&carrierCode, &carrierName, &serviceName, &totalPackages,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found or access denied")
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Build billing address snapshot from individual columns
	if billToName.Valid || billToCompany.Valid || billToAddr1.Valid {
		billing := BillingAddressSnapshot{
			ContactName:  nullStringValue(billToName),
			ContactEmail: nullStringValue(billToEmail),
			AddressLine1: nullStringValue(billToAddr1),
			City:         nullStringValue(billToCity),
			StateProvince: nullStringValue(billToRegion),
			PostalCode:   nullStringValue(billToPostal),
			CountryCode:  nullStringValue(billToCountry),
		}
		if billToCompany.Valid {
			billing.CompanyName = &billToCompany.String
		}
		if billToPhone.Valid {
			billing.ContactPhone = &billToPhone.String
		}
		if billToAddr2.Valid {
			billing.AddressLine2 = &billToAddr2.String
		}
		inv.BillingAddress = &billing
	}

	// Build shipment snapshot from individual columns
	if inv.ShipmentNumber != nil || shipFromSummary.Valid {
		shipmentNum := ""
		if inv.ShipmentNumber != nil {
			shipmentNum = *inv.ShipmentNumber
		}
		snapshot := ShipmentSnapshot{
			ShipmentNumber: shipmentNum,
			CarrierName:    nullStringValue(carrierName),
			ServiceType:    nullStringValue(serviceName),
			TotalPackages:  totalPackages,
		}
		// Parse origin/destination from summaries (format: "City, Province, Country")
		if shipFromSummary.Valid {
			snapshot.OriginCity = shipFromSummary.String
		}
		if shipToSummary.Valid {
			snapshot.DestinationCity = shipToSummary.String
		}
		inv.ShipmentSnapshot = &snapshot
	}

	// Get invoice items
	items, err := s.getInvoiceItems(invoiceID)
	if err != nil {
		return nil, err
	}
	inv.Items = items

	// Get payments
	payments, err := s.getInvoicePayments(invoiceID)
	if err != nil {
		return nil, err
	}
	inv.Payments = payments

	// Get events
	events, err := s.getInvoiceEvents(invoiceID)
	if err != nil {
		return nil, err
	}
	inv.Events = events

	// Get refund summary if applicable (only for issued invoices)
	if inv.InvoiceType == "invoice" && inv.Status == "issued" {
		summary, err := s.getRefundSummary(invoiceID)
		if err == nil {
			inv.RefundSummary = summary
		}
	}

	return &inv, nil
}

// getInvoiceItems retrieves line items for an invoice
func (s *InvoiceService) getInvoiceItems(invoiceID string) ([]InvoiceItemResponse, error) {
	query := `
		SELECT
			ii.id, ii.item_type::text, ii.description,
			ii.package_id, ii.package_number,
			COALESCE(ii.quantity, 1), ii.unit_price, ii.amount,
			ii.tax_rate,
			ii.item_index
		FROM invoice_items ii
		WHERE ii.invoice_id = $1
		ORDER BY ii.item_index
	`

	rows, err := s.db.DB.Query(query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice items: %w", err)
	}
	defer rows.Close()

	items := []InvoiceItemResponse{}
	for rows.Next() {
		var item InvoiceItemResponse
		var unitPrice, amount sql.NullFloat64
		var taxRate sql.NullFloat64
		err := rows.Scan(
			&item.ID, &item.ItemType, &item.Description,
			&item.PackageID, &item.PackageNumber,
			&item.Quantity, &unitPrice, &amount,
			&taxRate,
			&item.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice item: %w", err)
		}
		if unitPrice.Valid {
			item.UnitPrice = unitPrice.Float64
		}
		if amount.Valid {
			item.TotalPrice = amount.Float64
			item.NetAmount = amount.Float64
		}
		if taxRate.Valid {
			item.TaxRate = &taxRate.Float64
		}
		item.Currency = "CAD" // Default currency
		items = append(items, item)
	}

	return items, nil
}

// getInvoicePayments retrieves payment records for an invoice
func (s *InvoiceService) getInvoicePayments(invoiceID string) ([]InvoicePaymentResponse, error) {
	query := `
		SELECT
			ip.id, ip.method, ip.amount, ip.currency, ip.processed_at,
			ip.wallet_transaction_id, wt.transaction_number,
			ip.stripe_payment_intent_id, ip.provider_reference, ip.note
		FROM invoice_payments ip
		LEFT JOIN wallet_transactions wt ON ip.wallet_transaction_id = wt.id
		WHERE ip.invoice_id = $1
		ORDER BY ip.processed_at DESC
	`

	rows, err := s.db.DB.Query(query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice payments: %w", err)
	}
	defer rows.Close()

	payments := []InvoicePaymentResponse{}
	for rows.Next() {
		var payment InvoicePaymentResponse
		err := rows.Scan(
			&payment.ID, &payment.PaymentMethod, &payment.Amount, &payment.Currency, &payment.PaidAt,
			&payment.WalletTransactionID, &payment.WalletTransactionNumber,
			&payment.StripePaymentIntentID, &payment.PaymentReference, &payment.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice payment: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

// getInvoiceEvents retrieves audit events for an invoice
func (s *InvoiceService) getInvoiceEvents(invoiceID string) ([]InvoiceEventResponse, error) {
	query := `
		SELECT
			ie.id, ie.event_type, ie.description,
			ie.actor_user_id, COALESCE(u.display_name, CONCAT(u.first_name, ' ', u.last_name)) as user_name,
			ie.metadata, ie.created_at
		FROM invoice_events ie
		LEFT JOIN users u ON ie.actor_user_id = u.id
		WHERE ie.invoice_id = $1
		ORDER BY ie.created_at DESC
	`

	rows, err := s.db.DB.Query(query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoice events: %w", err)
	}
	defer rows.Close()

	events := []InvoiceEventResponse{}
	for rows.Next() {
		var event InvoiceEventResponse
		err := rows.Scan(
			&event.ID, &event.EventType, &event.Description,
			&event.PerformedByUserID, &event.PerformedByUserName,
			&event.Metadata, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// getRefundSummary retrieves refund eligibility information
func (s *InvoiceService) getRefundSummary(invoiceID string) (*RefundSummary, error) {
	query := `
		SELECT
			r.original_amount, r.total_refunded, r.refundable_remaining,
			(SELECT COUNT(*) FROM invoices cn WHERE cn.parent_invoice_id = $1 AND cn.invoice_type = 'credit_note' AND cn.status <> 'void') AS refund_count
		FROM v_invoice_refund_summary r
		WHERE r.invoice_id = $1
	`

	var summary RefundSummary
	err := s.db.DB.QueryRow(query, invoiceID).Scan(
		&summary.OriginalAmount,
		&summary.TotalRefunded,
		&summary.RefundableRemaining,
		&summary.RefundCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// No refund summary means invoice is not eligible
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refund summary: %w", err)
	}

	summary.CanRefund = summary.RefundableRemaining > 0

	return &summary, nil
}

// =====================================================
// GENERATE PDF
// =====================================================

// GeneratePDF generates invoice PDF from snapshot data
func (s *InvoiceService) GeneratePDF(invoiceID, organizationID, userID string) ([]byte, error) {
	// Get complete invoice details
	invoice, err := s.GetInvoiceByID(invoiceID, organizationID, userID)
	if err != nil {
		return nil, err
	}

	// TODO: Replace with actual PDF generation library (e.g., go-pdf, wkhtmltopdf)
	// For now, return mock PDF content
	mockPDF := fmt.Sprintf(`%%PDF-1.4
MOCK INVOICE PDF

Invoice Number: %s
Organization: %s
Type: %s
Status: %s
Payment Status: %s

Total Amount: %.2f %s
Amount Paid: %.2f %s
Balance Due: %.2f %s

Issued At: %v
Due Date: %v

This is a mock PDF. Replace with actual PDF generation using go-pdf or similar library.
`,
		invoice.InvoiceNumber,
		invoice.OrganizationName,
		invoice.InvoiceType,
		invoice.Status,
		invoice.PaymentStatus,
		invoice.TotalAmount, invoice.Currency,
		invoice.AmountPaid, invoice.Currency,
		invoice.BalanceDue, invoice.Currency,
		invoice.IssuedAt,
		invoice.DueDate,
	)

	return []byte(mockPDF), nil
}

// =====================================================
// ISSUE REFUND (CREDIT NOTE)
// =====================================================

// IssueRefund creates a credit note and refunds to wallet
func (s *InvoiceService) IssueRefund(req RefundInvoiceRequest, invoiceID, organizationID, userID string) (*RefundResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Lock and verify parent invoice
	query := `
		SELECT
			i.id, i.invoice_number, i.organization_id, i.shipment_id,
			i.invoice_type, i.status, i.payment_status, i.currency, i.total_amount,
			r.refundable_remaining
		FROM invoices i
		LEFT JOIN v_invoice_refund_summary r ON i.id = r.invoice_id
		WHERE i.id = $1 AND i.organization_id = $2
		FOR UPDATE OF i
	`

	var parentInvoice struct {
		ID                  string
		InvoiceNumber       string
		OrganizationID      string
		ShipmentID          sql.NullString
		InvoiceType         string
		Status              string
		PaymentStatus       string
		Currency            string
		TotalAmount         float64
		RefundableRemaining sql.NullFloat64
	}

	err = tx.QueryRow(query, invoiceID, organizationID).Scan(
		&parentInvoice.ID,
		&parentInvoice.InvoiceNumber,
		&parentInvoice.OrganizationID,
		&parentInvoice.ShipmentID,
		&parentInvoice.InvoiceType,
		&parentInvoice.Status,
		&parentInvoice.PaymentStatus,
		&parentInvoice.Currency,
		&parentInvoice.TotalAmount,
		&parentInvoice.RefundableRemaining,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found or access denied")
		}
		return nil, fmt.Errorf("failed to lock invoice: %w", err)
	}

	// Validate parent invoice
	if parentInvoice.InvoiceType != "invoice" {
		return nil, fmt.Errorf("can only refund standard invoices, not %s", parentInvoice.InvoiceType)
	}
	if parentInvoice.Status == "void" {
		return nil, fmt.Errorf("cannot refund voided invoice")
	}
	if !parentInvoice.RefundableRemaining.Valid || parentInvoice.RefundableRemaining.Float64 <= 0 {
		return nil, fmt.Errorf("invoice has no refundable balance remaining")
	}
	if req.Amount > parentInvoice.RefundableRemaining.Float64 {
		return nil, fmt.Errorf("refund amount %.2f exceeds refundable remaining %.2f",
			req.Amount, parentInvoice.RefundableRemaining.Float64)
	}

	// Check idempotency
	var existingCreditNoteID string
	err = tx.QueryRow(`
		SELECT id FROM invoices
		WHERE idempotency_key = $1
	`, req.IdempotencyKey).Scan(&existingCreditNoteID)
	if err == nil {
		// Idempotency key already used - return existing credit note
		var resp RefundResponse
		err = tx.QueryRow(`
			SELECT id, invoice_number, parent_invoice_id, total_amount, currency,
				   credit_note_reason, credit_note_reason_notes, issued_at
			FROM invoices
			WHERE id = $1
		`, existingCreditNoteID).Scan(
			&resp.CreditNoteID, &resp.CreditNoteNumber, &resp.ParentInvoiceID,
			&resp.RefundAmount, &resp.Currency,
			&resp.Reason, &resp.ReasonNotes, &resp.IssuedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve existing credit note: %w", err)
		}
		resp.RefundAmount = -resp.RefundAmount // Make positive for response
		resp.ParentInvoiceNumber = parentInvoice.InvoiceNumber
		tx.Commit()
		return &resp, nil
	}

	// Generate credit note number
	var creditNoteNumber string
	err = tx.QueryRow(`
		SELECT 'CN-' || TO_CHAR(NOW(), 'YYYY') || '-' ||
		       LPAD((COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM 9) AS INTEGER)), 0) + 1)::TEXT, 5, '0')
		FROM invoices
		WHERE invoice_number LIKE 'CN-' || TO_CHAR(NOW(), 'YYYY') || '-%'
	`).Scan(&creditNoteNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to generate credit note number: %w", err)
	}

	// Create credit note (negative invoice)
	var creditNoteID string
	refundAmount := -req.Amount // Negative for credit note
	now := time.Now()

	err = tx.QueryRow(`
		INSERT INTO invoices (
			invoice_number, organization_id, shipment_id,
			invoice_type, status, payment_status,
			currency, subtotal_amount, tax_amount, discount_amount, total_amount,
			amount_paid, balance_due,
			parent_invoice_id, credit_note_reason, credit_note_reason_notes,
			issued_at, idempotency_key
		) VALUES (
			$1, $2, $3, 'credit_note', 'issued', 'paid',
			$4, $5, 0, 0, $5, 0, 0,
			$6, $7, $8, $9, $10
		) RETURNING id
	`,
		creditNoteNumber, parentInvoice.OrganizationID, parentInvoice.ShipmentID,
		parentInvoice.Currency, refundAmount,
		parentInvoice.ID, req.Reason, req.ReasonNotes, now, req.IdempotencyKey,
	).Scan(&creditNoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to create credit note: %w", err)
	}

	// Create credit note event
	_, err = tx.Exec(`
		INSERT INTO invoice_events (
			invoice_id, event_type, description,
			actor_user_id, metadata
		) VALUES ($1, $2, $3, $4, $5)
	`,
		creditNoteID,
		"invoice.credit_note_issued",
		fmt.Sprintf("Credit note issued for %.2f %s (reason: %s)", req.Amount, parentInvoice.Currency, req.Reason),
		userID,
		fmt.Sprintf(`{"parent_invoice_id":"%s","refund_amount":%.2f,"reason":"%s"}`,
			parentInvoice.ID, req.Amount, req.Reason),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credit note event: %w", err)
	}

	// Get wallet for organization
	var walletID string
	var walletBalance float64
	err = tx.QueryRow(`
		SELECT id, balance FROM wallets
		WHERE organization_id = $1 AND status = 'active'
		FOR UPDATE
	`, organizationID).Scan(&walletID, &walletBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create wallet refund transaction
	var txnID, txnNumber string
	newBalance := walletBalance + req.Amount

	err = tx.QueryRow(`
		INSERT INTO wallet_transactions (
			transaction_number, wallet_id, direction, tx_type,
			amount, currency, balance_before, balance_after,
			description, reference_type, reference_id,
			performed_by_user_id, is_system_generated, idempotency_key
		) VALUES (
			'TXN-' || TO_CHAR(NOW(), 'YYYY') || '-' ||
			LPAD((COALESCE((SELECT MAX(CAST(SUBSTRING(transaction_number FROM 10) AS INTEGER)) FROM wallet_transactions WHERE transaction_number LIKE 'TXN-' || TO_CHAR(NOW(), 'YYYY') || '-%'), 0) + 1)::TEXT, 5, '0'),
			$1, 'credit', 'refund',
			$2, $3, $4, $5,
			$6, 'invoice', $7,
			$8, false, $9
		) RETURNING id, transaction_number
	`,
		walletID, req.Amount, parentInvoice.Currency, walletBalance, newBalance,
		fmt.Sprintf("Refund from credit note %s", creditNoteNumber),
		creditNoteID, userID, "refund_"+req.IdempotencyKey,
	).Scan(&txnID, &txnNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet transaction: %w", err)
	}

	// Update wallet balance
	_, err = tx.Exec(`
		UPDATE wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2
	`, req.Amount, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Create invoice payment record for credit note
	_, err = tx.Exec(`
		INSERT INTO invoice_payments (
			invoice_id, payment_method, amount, currency, paid_at,
			wallet_transaction_id, payment_reference
		) VALUES ($1, 'wallet', $2, $3, $4, $5, $6)
	`,
		creditNoteID, refundAmount, parentInvoice.Currency, now,
		txnID, fmt.Sprintf("Refund to wallet via %s", txnNumber),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice payment: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &RefundResponse{
		CreditNoteID:            creditNoteID,
		CreditNoteNumber:        creditNoteNumber,
		ParentInvoiceID:         parentInvoice.ID,
		ParentInvoiceNumber:     parentInvoice.InvoiceNumber,
		RefundAmount:            req.Amount,
		Currency:                parentInvoice.Currency,
		Reason:                  req.Reason,
		ReasonNotes:             &req.ReasonNotes,
		WalletTransactionID:     &txnID,
		WalletTransactionNumber: &txnNumber,
		IssuedAt:                now,
	}, nil
}

// =====================================================
// VOID INVOICE
// =====================================================

// VoidInvoice marks an invoice as void (admin only)
func (s *InvoiceService) VoidInvoice(req VoidInvoiceRequest, invoiceID, organizationID, userID string) (*VoidResponse, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Lock and verify invoice
	query := `
		SELECT
			i.id, i.invoice_number, i.invoice_type, i.status, i.balance_due
		FROM invoices i
		WHERE i.id = $1 AND i.organization_id = $2
		FOR UPDATE
	`

	var invoice struct {
		ID             string
		InvoiceNumber  string
		InvoiceType    string
		Status         string
		BalanceDue     float64
	}

	err = tx.QueryRow(query, invoiceID, organizationID).Scan(
		&invoice.ID, &invoice.InvoiceNumber, &invoice.InvoiceType,
		&invoice.Status, &invoice.BalanceDue,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found or access denied")
		}
		return nil, fmt.Errorf("failed to lock invoice: %w", err)
	}

	// Validate invoice
	if invoice.InvoiceType != "invoice" {
		return nil, fmt.Errorf("can only void standard invoices, not %s", invoice.InvoiceType)
	}
	if invoice.Status == "void" {
		// Already voided - return current state (idempotent)
		return &VoidResponse{
			InvoiceID:       invoice.ID,
			InvoiceNumber:   invoice.InvoiceNumber,
			Status:          "void",
			VoidReason:      req.Reason,
			BalanceRefunded: invoice.BalanceDue,
		}, nil
	}

	// Void the invoice
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE invoices
		SET status = 'void',
		    voided_at = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, now, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to void invoice: %w", err)
	}

	// Create void event
	_, err = tx.Exec(`
		INSERT INTO invoice_events (
			invoice_id, event_type, description,
			actor_user_id, metadata
		) VALUES ($1, $2, $3, $4, $5)
	`,
		invoiceID,
		"invoice.voided",
		fmt.Sprintf("Invoice voided: %s", req.Reason),
		userID,
		fmt.Sprintf(`{"void_reason":"%s"}`, req.Reason),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create void event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &VoidResponse{
		InvoiceID:       invoiceID,
		InvoiceNumber:   invoice.InvoiceNumber,
		Status:          "void",
		VoidedAt:        now,
		VoidReason:      req.Reason,
		BalanceRefunded: invoice.BalanceDue,
	}, nil
}

// =====================================================
// HELPER FUNCTIONS
// =====================================================

// nullStringValue returns the string value or empty string if null
func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
