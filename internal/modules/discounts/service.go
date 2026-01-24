package discounts

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
)

const (
	DiscountNumberPrefix   = "DSC-"
	RedemptionNumberPrefix = "RDM-"
)

// DiscountService handles discount business logic
type DiscountService struct {
	db *clients.Database
}

// NewDiscountService creates a new discount service
func NewDiscountService(db *clients.Database) *DiscountService {
	return &DiscountService{
		db: db,
	}
}

// GetOrganizationIDFromUser retrieves the organization ID for a user
func (s *DiscountService) GetOrganizationIDFromUser(userID string) (string, error) {
	query := `
		SELECT organization_id
		FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`

	var organizationID string
	err := s.db.DB.QueryRow(query, userID).Scan(&organizationID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user is not a member of any organization")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get organization: %w", err)
	}

	return organizationID, nil
}

// CreateDiscount creates a new discount
func (s *DiscountService) CreateDiscount(req CreateDiscountRequest, organizationID string, userID string) (*DiscountResponse, error) {
	// Generate discount number
	discountNumber, err := generateNumber(DiscountNumberPrefix, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate discount number: %w", err)
	}

	// Set defaults
	scope := "invoice"
	if req.Scope != nil {
		scope = *req.Scope
	}

	// Validate free_shipping must have shipment scope
	if req.DiscountType == "free_shipping" && scope != "shipment" {
		return nil, fmt.Errorf("free_shipping discount must have 'shipment' scope")
	}

	// Validate percent range
	if req.DiscountType == "percent" && (req.Value <= 0 || req.Value > 100) {
		return nil, fmt.Errorf("percent discount value must be between 0 and 100")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	firstTimeOnly := false
	if req.FirstTimeOnly != nil {
		firstTimeOnly = *req.FirstTimeOnly
	}

	minSubtotal := 0.0
	if req.MinSubtotalAmount != nil {
		minSubtotal = *req.MinSubtotalAmount
	}

	// Parse dates
	var startsAt, endsAt sql.NullTime
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err == nil {
			startsAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err == nil {
			endsAt = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Convert nullable fields to sql.Null types
	var maxDiscountAmount sql.NullFloat64
	if req.MaxDiscountAmount != nil {
		maxDiscountAmount = sql.NullFloat64{Float64: *req.MaxDiscountAmount, Valid: true}
	}

	var allowedCountry sql.NullString
	if req.AllowedCountry != nil {
		allowedCountry = sql.NullString{String: *req.AllowedCountry, Valid: true}
	}

	var maxUsesTotal sql.NullInt64
	if req.MaxUsesTotal != nil {
		maxUsesTotal = sql.NullInt64{Int64: int64(*req.MaxUsesTotal), Valid: true}
	}

	var maxUsesPerUser sql.NullInt64
	if req.MaxUsesPerUser != nil {
		maxUsesPerUser = sql.NullInt64{Int64: int64(*req.MaxUsesPerUser), Valid: true}
	}

	var maxTotalDiscountAmount sql.NullFloat64
	if req.MaxTotalDiscountAmount != nil {
		maxTotalDiscountAmount = sql.NullFloat64{Float64: *req.MaxTotalDiscountAmount, Valid: true}
	}

	var name sql.NullString
	if req.Name != nil {
		name = sql.NullString{String: *req.Name, Valid: true}
	}

	var description sql.NullString
	if req.Description != nil {
		description = sql.NullString{String: *req.Description, Valid: true}
	}

	query := `
		INSERT INTO discounts (
			discount_number, organization_id, created_by_user_id,
			code, name, description,
			discount_type, value, scope,
			min_subtotal_amount, max_discount_amount, allowed_country,
			max_uses_total, max_uses_per_user, max_total_discount_amount,
			starts_at, ends_at, is_active, first_time_only
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at, updated_at
	`

	var discount models.Discount
	discount.DiscountNumber = sql.NullString{String: discountNumber, Valid: true}
	discount.OrganizationID = organizationID
	discount.CreatedByUserID = sql.NullString{String: userID, Valid: true}
	discount.Code = req.Code
	discount.DiscountType = models.DiscountType(req.DiscountType)
	discount.Value = req.Value
	discount.Currency = "CAD"
	discount.Scope = models.DiscountScope(scope)
	discount.MinSubtotalAmount = minSubtotal
	discount.IsActive = isActive
	discount.FirstTimeOnly = firstTimeOnly

	err = s.db.DB.QueryRow(
		query,
		discountNumber, organizationID, userID,
		req.Code, name, description,
		req.DiscountType, req.Value, scope,
		minSubtotal, maxDiscountAmount, allowedCountry,
		maxUsesTotal, maxUsesPerUser, maxTotalDiscountAmount,
		startsAt, endsAt, isActive, firstTimeOnly,
	).Scan(&discount.ID, &discount.CreatedAt, &discount.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("discount code '%s' already exists", req.Code)
		}
		return nil, fmt.Errorf("failed to create discount: %w", err)
	}

	// Reload full discount
	return s.GetDiscountByID(discount.ID, organizationID)
}

// GetDiscountByID retrieves a discount by ID
func (s *DiscountService) GetDiscountByID(discountID string, organizationID string) (*DiscountResponse, error) {
	query := `
		SELECT d.id, d.discount_number, d.organization_id, d.created_by_user_id,
		       d.code, d.name, d.description,
		       d.discount_type, d.value, d.currency, d.scope,
		       d.min_subtotal_amount, d.max_discount_amount, d.allowed_country,
		       d.max_uses_total, d.max_uses_per_user, d.max_total_discount_amount,
		       d.starts_at, d.ends_at, d.is_active, d.first_time_only,
		       d.created_at, d.updated_at,
		       COALESCE((SELECT COUNT(*) FROM discount_redemptions WHERE discount_id = d.id AND status = 'applied'), 0) as usage_count
		FROM discounts d
		WHERE d.id = $1 AND d.organization_id = $2
	`

	var discount models.Discount
	err := s.db.DB.QueryRow(query, discountID, organizationID).Scan(
		&discount.ID, &discount.DiscountNumber, &discount.OrganizationID, &discount.CreatedByUserID,
		&discount.Code, &discount.Name, &discount.Description,
		&discount.DiscountType, &discount.Value, &discount.Currency, &discount.Scope,
		&discount.MinSubtotalAmount, &discount.MaxDiscountAmount, &discount.AllowedCountry,
		&discount.MaxUsesTotal, &discount.MaxUsesPerUser, &discount.MaxTotalDiscountAmount,
		&discount.StartsAt, &discount.EndsAt, &discount.IsActive, &discount.FirstTimeOnly,
		&discount.CreatedAt, &discount.UpdatedAt, &discount.UsageCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("discount not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get discount: %w", err)
	}

	return toDiscountResponse(&discount), nil
}

// ListDiscounts retrieves paginated discounts with filters
func (s *DiscountService) ListDiscounts(organizationID string, req ListDiscountsRequest) (*DiscountsListResponse, error) {
	// Set defaults
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build count query
	countQuery := `SELECT COUNT(*) FROM discounts WHERE organization_id = $1`
	countArgs := []interface{}{organizationID}
	argIndex := 2

	// Build select query
	selectQuery := `
		SELECT d.id, d.discount_number, d.organization_id, d.created_by_user_id,
		       d.code, d.name, d.description,
		       d.discount_type, d.value, d.currency, d.scope,
		       d.min_subtotal_amount, d.max_discount_amount, d.allowed_country,
		       d.max_uses_total, d.max_uses_per_user, d.max_total_discount_amount,
		       d.starts_at, d.ends_at, d.is_active, d.first_time_only,
		       d.created_at, d.updated_at,
		       COALESCE((SELECT COUNT(*) FROM discount_redemptions WHERE discount_id = d.id AND status = 'applied'), 0) as usage_count
		FROM discounts d
		WHERE d.organization_id = $1
	`
	selectArgs := []interface{}{organizationID}

	// Add filters
	if req.Code != nil && *req.Code != "" {
		countQuery += fmt.Sprintf(" AND code ILIKE $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND d.code ILIKE $%d", argIndex)
		searchTerm := "%" + *req.Code + "%"
		countArgs = append(countArgs, searchTerm)
		selectArgs = append(selectArgs, searchTerm)
		argIndex++
	}

	if req.DiscountType != nil && *req.DiscountType != "" {
		countQuery += fmt.Sprintf(" AND discount_type = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND d.discount_type = $%d", argIndex)
		countArgs = append(countArgs, *req.DiscountType)
		selectArgs = append(selectArgs, *req.DiscountType)
		argIndex++
	}

	if req.Scope != nil && *req.Scope != "" {
		countQuery += fmt.Sprintf(" AND scope = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND d.scope = $%d", argIndex)
		countArgs = append(countArgs, *req.Scope)
		selectArgs = append(selectArgs, *req.Scope)
		argIndex++
	}

	if req.IsActive != nil {
		countQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
		selectQuery += fmt.Sprintf(" AND d.is_active = $%d", argIndex)
		countArgs = append(countArgs, *req.IsActive)
		selectArgs = append(selectArgs, *req.IsActive)
		argIndex++
	}

	// Get total count
	var total int
	err := s.db.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count discounts: %w", err)
	}

	// Add sorting
	sortBy := "created_at"
	if req.SortBy != nil && *req.SortBy != "" {
		sortBy = "d." + *req.SortBy
	}
	sortOrder := "DESC"
	if req.SortOrder != nil && *req.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	selectQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	selectArgs = append(selectArgs, limit, offset)

	// Execute query
	rows, err := s.db.DB.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query discounts: %w", err)
	}
	defer rows.Close()

	// Parse results
	discounts := make([]DiscountResponse, 0)
	for rows.Next() {
		var d models.Discount
		err := rows.Scan(
			&d.ID, &d.DiscountNumber, &d.OrganizationID, &d.CreatedByUserID,
			&d.Code, &d.Name, &d.Description,
			&d.DiscountType, &d.Value, &d.Currency, &d.Scope,
			&d.MinSubtotalAmount, &d.MaxDiscountAmount, &d.AllowedCountry,
			&d.MaxUsesTotal, &d.MaxUsesPerUser, &d.MaxTotalDiscountAmount,
			&d.StartsAt, &d.EndsAt, &d.IsActive, &d.FirstTimeOnly,
			&d.CreatedAt, &d.UpdatedAt, &d.UsageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan discount: %w", err)
		}
		discounts = append(discounts, *toDiscountResponse(&d))
	}

	totalPages := (total + limit - 1) / limit

	return &DiscountsListResponse{
		Discounts: discounts,
		Pagination: PaginationMeta{
			Page:       page,
			PerPage:    limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateDiscount updates an existing discount
func (s *DiscountService) UpdateDiscount(discountID string, req UpdateDiscountRequest, organizationID string) (*DiscountResponse, error) {
	// Verify discount exists
	_, err := s.GetDiscountByID(discountID, organizationID)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	query := "UPDATE discounts SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIndex)
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *req.Description)
		argIndex++
	}
	if req.Value != nil {
		query += fmt.Sprintf(", value = $%d", argIndex)
		args = append(args, *req.Value)
		argIndex++
	}
	if req.MinSubtotalAmount != nil {
		query += fmt.Sprintf(", min_subtotal_amount = $%d", argIndex)
		args = append(args, *req.MinSubtotalAmount)
		argIndex++
	}
	if req.MaxDiscountAmount != nil {
		query += fmt.Sprintf(", max_discount_amount = $%d", argIndex)
		args = append(args, *req.MaxDiscountAmount)
		argIndex++
	}
	if req.AllowedCountry != nil {
		query += fmt.Sprintf(", allowed_country = $%d", argIndex)
		args = append(args, *req.AllowedCountry)
		argIndex++
	}
	if req.MaxUsesTotal != nil {
		query += fmt.Sprintf(", max_uses_total = $%d", argIndex)
		args = append(args, *req.MaxUsesTotal)
		argIndex++
	}
	if req.MaxUsesPerUser != nil {
		query += fmt.Sprintf(", max_uses_per_user = $%d", argIndex)
		args = append(args, *req.MaxUsesPerUser)
		argIndex++
	}
	if req.MaxTotalDiscountAmount != nil {
		query += fmt.Sprintf(", max_total_discount_amount = $%d", argIndex)
		args = append(args, *req.MaxTotalDiscountAmount)
		argIndex++
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err == nil {
			query += fmt.Sprintf(", starts_at = $%d", argIndex)
			args = append(args, t)
			argIndex++
		}
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err == nil {
			query += fmt.Sprintf(", ends_at = $%d", argIndex)
			args = append(args, t)
			argIndex++
		}
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIndex)
		args = append(args, *req.IsActive)
		argIndex++
	}
	if req.FirstTimeOnly != nil {
		query += fmt.Sprintf(", first_time_only = $%d", argIndex)
		args = append(args, *req.FirstTimeOnly)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND organization_id = $%d", argIndex, argIndex+1)
	args = append(args, discountID, organizationID)

	_, err = s.db.DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update discount: %w", err)
	}

	return s.GetDiscountByID(discountID, organizationID)
}

// DeleteDiscount deletes a discount
func (s *DiscountService) DeleteDiscount(discountID string, organizationID string) error {
	// Check if discount has redemptions
	var count int
	err := s.db.DB.QueryRow(`
		SELECT COUNT(*) FROM discount_redemptions
		WHERE discount_id = $1 AND status = 'applied'
	`, discountID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check redemptions: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete discount with active redemptions")
	}

	result, err := s.db.DB.Exec(`
		DELETE FROM discounts WHERE id = $1 AND organization_id = $2
	`, discountID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete discount: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("discount not found")
	}

	return nil
}

// ValidateDiscount validates a discount code
func (s *DiscountService) ValidateDiscount(req ValidateDiscountRequest, organizationID string, userID string) (*ValidateDiscountResponse, error) {
	// Find discount by code
	query := `
		SELECT d.id, d.discount_number, d.code, d.name,
		       d.discount_type, d.value, d.currency, d.scope,
		       d.min_subtotal_amount, d.max_discount_amount, d.allowed_country,
		       d.max_uses_total, d.max_uses_per_user, d.max_total_discount_amount,
		       d.starts_at, d.ends_at, d.is_active, d.first_time_only,
		       COALESCE((SELECT COUNT(*) FROM discount_redemptions WHERE discount_id = d.id AND status = 'applied'), 0) as usage_count,
		       COALESCE((SELECT COUNT(*) FROM discount_redemptions WHERE discount_id = d.id AND redeemed_by_user_id = $3 AND status = 'applied'), 0) as user_usage_count
		FROM discounts d
		WHERE d.code = $1 AND d.organization_id = $2
	`

	var d models.Discount
	var userUsageCount int
	err := s.db.DB.QueryRow(query, req.Code, organizationID, userID).Scan(
		&d.ID, &d.DiscountNumber, &d.Code, &d.Name,
		&d.DiscountType, &d.Value, &d.Currency, &d.Scope,
		&d.MinSubtotalAmount, &d.MaxDiscountAmount, &d.AllowedCountry,
		&d.MaxUsesTotal, &d.MaxUsesPerUser, &d.MaxTotalDiscountAmount,
		&d.StartsAt, &d.EndsAt, &d.IsActive, &d.FirstTimeOnly,
		&d.UsageCount, &userUsageCount,
	)

	if err == sql.ErrNoRows {
		reason := "Invalid discount code"
		return &ValidateDiscountResponse{
			Valid:       false,
			Code:        req.Code,
			ErrorReason: &reason,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get discount: %w", err)
	}

	// Validate discount
	reason := s.validateDiscountEligibility(&d, req.Subtotal, req.Country, userUsageCount)
	if reason != "" {
		return &ValidateDiscountResponse{
			Valid:        false,
			Code:         d.Code,
			DiscountType: string(d.DiscountType),
			Value:        d.Value,
			ErrorReason:  &reason,
		}, nil
	}

	// Calculate discount amount
	discountAmount := d.CalculateDiscount(req.Subtotal)
	finalAmount := req.Subtotal - discountAmount

	message := "Discount code is valid"
	return &ValidateDiscountResponse{
		Valid:          true,
		Code:           d.Code,
		DiscountType:   string(d.DiscountType),
		Value:          d.Value,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		Message:        &message,
	}, nil
}

// ApplyDiscount applies a discount to a shipment or invoice
func (s *DiscountService) ApplyDiscount(req ApplyDiscountRequest, organizationID string, userID string) (*ApplyDiscountResponse, error) {
	// Validate first
	validateReq := ValidateDiscountRequest{
		Code:     req.Code,
		Subtotal: req.Subtotal,
	}
	validateResp, err := s.ValidateDiscount(validateReq, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if !validateResp.Valid {
		return nil, fmt.Errorf("discount validation failed: %s", *validateResp.ErrorReason)
	}

	// Get discount ID
	var discountID string
	err = s.db.DB.QueryRow(`SELECT id FROM discounts WHERE code = $1 AND organization_id = $2`, req.Code, organizationID).Scan(&discountID)
	if err != nil {
		return nil, fmt.Errorf("discount not found: %w", err)
	}

	// Generate redemption number
	usageNumber, err := generateNumber(RedemptionNumberPrefix, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate redemption number: %w", err)
	}

	// Create redemption
	query := `
		INSERT INTO discount_redemptions (
			usage_number, organization_id, discount_id, redeemed_by_user_id,
			invoice_id, shipment_id, status, currency, discount_amount
		) VALUES ($1, $2, $3, $4, $5, $6, 'applied', 'CAD', $7)
		RETURNING id, redeemed_at
	`

	var redemptionID string
	var redeemedAt time.Time
	err = s.db.DB.QueryRow(
		query,
		usageNumber, organizationID, discountID, userID,
		req.InvoiceID, req.ShipmentID, validateResp.DiscountAmount,
	).Scan(&redemptionID, &redeemedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create redemption: %w", err)
	}

	return &ApplyDiscountResponse{
		RedemptionID:   redemptionID,
		UsageNumber:    usageNumber,
		DiscountID:     discountID,
		Code:           req.Code,
		DiscountType:   validateResp.DiscountType,
		DiscountAmount: validateResp.DiscountAmount,
		Currency:       "CAD",
		AppliedAt:      redeemedAt,
	}, nil
}

// =====================================================
// HELPER FUNCTIONS
// =====================================================

func (s *DiscountService) validateDiscountEligibility(d *models.Discount, subtotal float64, country *string, userUsageCount int) string {
	// Check if active
	if !d.IsActive {
		return "Discount code is not active"
	}

	// Check date validity
	now := time.Now()
	if d.StartsAt.Valid && now.Before(d.StartsAt.Time) {
		return "Discount code is not yet valid"
	}
	if d.EndsAt.Valid && now.After(d.EndsAt.Time) {
		return "Discount code has expired"
	}

	// Check minimum subtotal
	if subtotal < d.MinSubtotalAmount {
		return fmt.Sprintf("Minimum order amount of %.2f required", d.MinSubtotalAmount)
	}

	// Check max uses total
	if d.MaxUsesTotal.Valid && d.UsageCount >= int(d.MaxUsesTotal.Int64) {
		return "Discount code usage limit reached"
	}

	// Check max uses per user
	if d.MaxUsesPerUser.Valid && userUsageCount >= int(d.MaxUsesPerUser.Int64) {
		return "You have already used this discount code"
	}

	// Check allowed country
	if d.AllowedCountry.Valid && country != nil && *country != d.AllowedCountry.String {
		return fmt.Sprintf("Discount code is only valid for %s", d.AllowedCountry.String)
	}

	return ""
}

func toDiscountResponse(d *models.Discount) *DiscountResponse {
	resp := &DiscountResponse{
		ID:                d.ID,
		Code:              d.Code,
		DiscountType:      string(d.DiscountType),
		Value:             d.Value,
		Currency:          d.Currency,
		Scope:             string(d.Scope),
		MinSubtotalAmount: d.MinSubtotalAmount,
		IsActive:          d.IsActive,
		FirstTimeOnly:     d.FirstTimeOnly,
		UsageCount:        d.UsageCount,
		IsValid:           d.IsValid(),
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}

	if d.DiscountNumber.Valid {
		resp.DiscountNumber = &d.DiscountNumber.String
	}
	if d.Name.Valid {
		resp.Name = &d.Name.String
	}
	if d.Description.Valid {
		resp.Description = &d.Description.String
	}
	if d.MaxDiscountAmount.Valid {
		resp.MaxDiscountAmount = &d.MaxDiscountAmount.Float64
	}
	if d.AllowedCountry.Valid {
		resp.AllowedCountry = &d.AllowedCountry.String
	}
	if d.MaxUsesTotal.Valid {
		v := int(d.MaxUsesTotal.Int64)
		resp.MaxUsesTotal = &v
	}
	if d.MaxUsesPerUser.Valid {
		v := int(d.MaxUsesPerUser.Int64)
		resp.MaxUsesPerUser = &v
	}
	if d.MaxTotalDiscountAmount.Valid {
		resp.MaxTotalDiscountAmount = &d.MaxTotalDiscountAmount.Float64
	}
	if d.StartsAt.Valid {
		resp.StartsAt = &d.StartsAt.Time
	}
	if d.EndsAt.Valid {
		resp.EndsAt = &d.EndsAt.Time
	}

	return resp
}

func generateNumber(prefix string, length int) (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return prefix + string(b), nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "unique") || contains(err.Error(), "duplicate"))
}

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
