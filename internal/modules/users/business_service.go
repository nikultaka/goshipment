package users

import (
	"database/sql"
	"fmt"
	"strings"

	"gogin/internal/models"
)

// GetBusiness retrieves a business profile for a user
func (s *UserService) GetBusiness(userID string) (*models.Business, *models.BusinessEmail, *models.BusinessPhone, error) {
	query := `
		SELECT
			b.id, b.user_id, b.business_name, b.legal_name, b.business_type,
			b.tax_id, b.avatar_file_id, b.website, b.description,
			b.address_line1, b.address_line2, b.city, b.province, b.postal_code, b.country,
			b.is_active, b.created_at, b.updated_at,
			be.id AS email_id, be.email, be.email_type, be.is_primary AS email_is_primary,
			bp.id AS phone_id, bp.phone, bp.phone_type, bp.is_primary AS phone_is_primary
		FROM businesses b
		LEFT JOIN business_emails be ON b.id = be.business_id AND be.is_primary = true AND be.is_active = true
		LEFT JOIN business_phones bp ON b.id = bp.business_id AND bp.is_primary = true AND bp.is_active = true
		WHERE b.user_id = $1 AND b.is_active = true
	`

	business := &models.Business{}
	var businessEmail *models.BusinessEmail
	var businessPhone *models.BusinessPhone

	var emailID, emailAddr, emailType sql.NullString
	var emailIsPrimary sql.NullBool
	var phoneID, phoneNum, phoneType sql.NullString
	var phoneIsPrimary sql.NullBool

	err := s.db.QueryRow(query, userID).Scan(
		&business.ID, &business.UserID, &business.BusinessName, &business.LegalName, &business.BusinessType,
		&business.TaxID, &business.AvatarFileID, &business.Website, &business.Description,
		&business.AddressLine1, &business.AddressLine2, &business.City, &business.Province, &business.PostalCode, &business.Country,
		&business.IsActive, &business.CreatedAt, &business.UpdatedAt,
		&emailID, &emailAddr, &emailType, &emailIsPrimary,
		&phoneID, &phoneNum, &phoneType, &phoneIsPrimary,
	)

	if err == sql.ErrNoRows {
		return nil, nil, nil, fmt.Errorf("business profile not found")
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get business: %w", err)
	}

	// Populate email if exists
	if emailID.Valid {
		businessEmail = &models.BusinessEmail{
			ID:         emailID.String,
			BusinessID: business.ID,
			Email:      emailAddr.String,
			EmailType:  emailType.String,
			IsPrimary:  emailIsPrimary.Bool,
		}
	}

	// Populate phone if exists
	if phoneID.Valid {
		businessPhone = &models.BusinessPhone{
			ID:         phoneID.String,
			BusinessID: business.ID,
			Phone:      phoneNum.String,
			PhoneType:  phoneType.String,
			IsPrimary:  phoneIsPrimary.Bool,
		}
	}

	return business, businessEmail, businessPhone, nil
}

// CreateBusiness creates a new business profile for a user
func (s *UserService) CreateBusiness(userID string, req *CreateBusinessRequest) (*models.Business, error) {
	// Check if user already has a business
	var existingID string
	err := s.db.QueryRow("SELECT id FROM businesses WHERE user_id = $1 AND is_active = true", userID).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("user already has a business profile")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing business: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create business
	query := `
		INSERT INTO businesses (
			user_id, business_name, business_type, description,
			address_line1, address_line2, city, province, postal_code, country,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, NOW(), NOW())
		RETURNING id, user_id, business_name, business_type, description,
			address_line1, address_line2, city, province, postal_code, country,
			is_active, created_at, updated_at
	`

	business := &models.Business{}
	err = tx.QueryRow(
		query,
		userID,
		req.Name,
		"individual", // Default business type
		nullString(req.Description),
		nullString(req.AddressLine1),
		nullString(req.AddressLine2),
		nullString(req.City),
		nullString(req.Province),
		nullString(req.PostalCode),
		nullString(req.Country),
	).Scan(
		&business.ID, &business.UserID, &business.BusinessName, &business.BusinessType, &business.Description,
		&business.AddressLine1, &business.AddressLine2, &business.City, &business.Province, &business.PostalCode, &business.Country,
		&business.IsActive, &business.CreatedAt, &business.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create business: %w", err)
	}

	// Create email if provided
	if req.Email != "" {
		emailQuery := `
			INSERT INTO business_emails (business_id, email, email_type, is_primary, is_active, created_at, updated_at)
			VALUES ($1, $2, 'general', true, true, NOW(), NOW())
		`
		_, err = tx.Exec(emailQuery, business.ID, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to create business email: %w", err)
		}
	}

	// Create phone if provided
	if req.Phone != "" {
		phoneQuery := `
			INSERT INTO business_phones (business_id, phone, phone_type, is_primary, is_active, created_at, updated_at)
			VALUES ($1, $2, 'general', true, true, NOW(), NOW())
		`
		_, err = tx.Exec(phoneQuery, business.ID, req.Phone)
		if err != nil {
			return nil, fmt.Errorf("failed to create business phone: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return business, nil
}

// UpdateBusiness updates a business profile (partial updates)
func (s *UserService) UpdateBusiness(userID string, req *UpdateBusinessRequest) error {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("business_name = $%d", argCount))
		args = append(args, *req.Name)
		argCount++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, nullString(*req.Description))
		argCount++
	}

	if req.AddressLine1 != nil {
		updates = append(updates, fmt.Sprintf("address_line1 = $%d", argCount))
		args = append(args, nullString(*req.AddressLine1))
		argCount++
	}

	if req.AddressLine2 != nil {
		updates = append(updates, fmt.Sprintf("address_line2 = $%d", argCount))
		args = append(args, nullString(*req.AddressLine2))
		argCount++
	}

	if req.City != nil {
		updates = append(updates, fmt.Sprintf("city = $%d", argCount))
		args = append(args, nullString(*req.City))
		argCount++
	}

	if req.Province != nil {
		updates = append(updates, fmt.Sprintf("province = $%d", argCount))
		args = append(args, nullString(*req.Province))
		argCount++
	}

	if req.PostalCode != nil {
		updates = append(updates, fmt.Sprintf("postal_code = $%d", argCount))
		args = append(args, nullString(*req.PostalCode))
		argCount++
	}

	if req.Country != nil {
		updates = append(updates, fmt.Sprintf("country = $%d", argCount))
		args = append(args, nullString(*req.Country))
		argCount++
	}

	// If no updates provided, return early
	if len(updates) == 0 {
		return nil
	}

	// Add updated_at
	updates = append(updates, "updated_at = NOW()")

	// Add user_id to args
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE businesses SET %s WHERE user_id = $%d AND is_active = true",
		strings.Join(updates, ", "),
		argCount,
	)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update business: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("business profile not found")
	}

	// Handle email update if provided
	if req.Email != nil {
		// Start transaction for email update
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		// Get business ID
		var businessID string
		err = tx.QueryRow("SELECT id FROM businesses WHERE user_id = $1 AND is_active = true", userID).Scan(&businessID)
		if err != nil {
			return fmt.Errorf("failed to get business ID: %w", err)
		}

		// Check if primary email exists
		var emailID string
		err = tx.QueryRow("SELECT id FROM business_emails WHERE business_id = $1 AND is_primary = true AND is_active = true", businessID).Scan(&emailID)

		if err == sql.ErrNoRows {
			// Create new email
			_, err = tx.Exec(
				"INSERT INTO business_emails (business_id, email, email_type, is_primary, is_active, created_at, updated_at) VALUES ($1, $2, 'general', true, true, NOW(), NOW())",
				businessID, *req.Email,
			)
		} else if err == nil {
			// Update existing email
			_, err = tx.Exec(
				"UPDATE business_emails SET email = $1, updated_at = NOW() WHERE id = $2",
				*req.Email, emailID,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to update business email: %w", err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit email transaction: %w", err)
		}
	}

	// Handle phone update if provided
	if req.Phone != nil {
		// Start transaction for phone update
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		// Get business ID
		var businessID string
		err = tx.QueryRow("SELECT id FROM businesses WHERE user_id = $1 AND is_active = true", userID).Scan(&businessID)
		if err != nil {
			return fmt.Errorf("failed to get business ID: %w", err)
		}

		// Check if primary phone exists
		var phoneID string
		err = tx.QueryRow("SELECT id FROM business_phones WHERE business_id = $1 AND is_primary = true AND is_active = true", businessID).Scan(&phoneID)

		if err == sql.ErrNoRows {
			// Create new phone
			_, err = tx.Exec(
				"INSERT INTO business_phones (business_id, phone, phone_type, is_primary, is_active, created_at, updated_at) VALUES ($1, $2, 'general', true, true, NOW(), NOW())",
				businessID, *req.Phone,
			)
		} else if err == nil {
			// Update existing phone
			_, err = tx.Exec(
				"UPDATE business_phones SET phone = $1, updated_at = NOW() WHERE id = $2",
				*req.Phone, phoneID,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to update business phone: %w", err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit phone transaction: %w", err)
		}
	}

	return nil
}

// UpdateBusinessAvatar updates the avatar for a business
func (s *UserService) UpdateBusinessAvatar(userID, fileID string) error {
	query := "UPDATE businesses SET avatar_file_id = $1, updated_at = NOW() WHERE user_id = $2 AND is_active = true"
	result, err := s.db.Exec(query, fileID, userID)
	if err != nil {
		return fmt.Errorf("failed to update business avatar: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("business profile not found")
	}

	return nil
}

// DeleteBusinessAvatar removes the avatar from a business and returns the old file ID for cleanup
func (s *UserService) DeleteBusinessAvatar(userID string) (string, error) {
	// Get current avatar file ID
	var oldFileID sql.NullString
	err := s.db.QueryRow(
		"SELECT avatar_file_id FROM businesses WHERE user_id = $1 AND is_active = true",
		userID,
	).Scan(&oldFileID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("business profile not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get business avatar: %w", err)
	}

	// Clear avatar file ID
	_, err = s.db.Exec(
		"UPDATE businesses SET avatar_file_id = NULL, updated_at = NOW() WHERE user_id = $1 AND is_active = true",
		userID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to delete business avatar: %w", err)
	}

	if oldFileID.Valid {
		return oldFileID.String, nil
	}

	return "", nil
}

// sanitizeBusiness converts business data to response DTO
func (s *UserService) sanitizeBusiness(business *models.Business, email *models.BusinessEmail, phone *models.BusinessPhone, baseURL string) *BusinessResponse {
	response := &BusinessResponse{
		ID:          business.ID,
		Name:        business.BusinessName,
		CreatedAt:   business.CreatedAt,
		UpdatedAt:   business.UpdatedAt,
	}

	if business.Description.Valid {
		response.Description = business.Description.String
	}

	// Build address components
	if business.AddressLine1.Valid {
		response.AddressLine1 = business.AddressLine1.String
	}
	if business.AddressLine2.Valid {
		response.AddressLine2 = business.AddressLine2.String
	}
	if business.City.Valid {
		response.City = business.City.String
	}
	if business.Province.Valid {
		response.Province = business.Province.String
	}
	if business.PostalCode.Valid {
		response.PostalCode = business.PostalCode.String
	}
	if business.Country.Valid {
		response.Country = business.Country.String
	}

	// Build formatted address
	addressParts := []string{}
	if business.AddressLine1.Valid && business.AddressLine1.String != "" {
		addressParts = append(addressParts, business.AddressLine1.String)
	}
	if business.AddressLine2.Valid && business.AddressLine2.String != "" {
		addressParts = append(addressParts, business.AddressLine2.String)
	}
	if business.City.Valid && business.City.String != "" {
		addressParts = append(addressParts, business.City.String)
	}
	if business.Province.Valid && business.Province.String != "" {
		addressParts = append(addressParts, business.Province.String)
	}
	if business.PostalCode.Valid && business.PostalCode.String != "" {
		addressParts = append(addressParts, business.PostalCode.String)
	}
	if business.Country.Valid && business.Country.String != "" {
		addressParts = append(addressParts, business.Country.String)
	}
	if len(addressParts) > 0 {
		response.Address = strings.Join(addressParts, ", ")
	}

	// Add email if exists
	if email != nil {
		response.Email = email.Email
	}

	// Add phone if exists
	if phone != nil {
		response.Phone = phone.Phone
	}

	// Add avatar URL if exists
	if business.AvatarFileID.Valid {
		response.AvatarURL = fmt.Sprintf("%s/api/v1/storage/files/%s/download", baseURL, business.AvatarFileID.String)
	}

	return response
}

// nullString converts an empty string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
