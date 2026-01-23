package addresses

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// AddressService handles address book operations
type AddressService struct {
	db *clients.Database
}

// NewAddressService creates a new address service
func NewAddressService(db *clients.Database) *AddressService {
	return &AddressService{db: db}
}

// GetOrganizationIDFromUser gets the organization ID for a user
func (s *AddressService) GetOrganizationIDFromUser(userID string) (string, error) {
	var orgID string
	err := s.db.DB.QueryRow(`
		SELECT organization_id FROM organization_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`, userID).Scan(&orgID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("user is not a member of any organization")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get organization: %w", err)
	}
	return orgID, nil
}

// generateAddressNumber generates a unique address number (ADR-XXXXXXXX)
func generateAddressNumber() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "ADR-" + strings.ToUpper(hex.EncodeToString(bytes))
}

// CreateAddress creates a new address in the address book
func (s *AddressService) CreateAddress(organizationID, userID string, req *CreateAddressRequest) (*models.Address, error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// If setting as default sender, clear existing default sender
	if req.IsDefaultSender {
		_, err = tx.Exec(`
			UPDATE address_book SET is_default_sender = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_sender = true
		`, time.Now().UTC(), organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear default sender: %w", err)
		}
	}

	// If setting as default recipient, clear existing default recipient
	if req.IsDefaultRecipient {
		_, err = tx.Exec(`
			UPDATE address_book SET is_default_recipient = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_recipient = true
		`, time.Now().UTC(), organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear default recipient: %w", err)
		}
	}

	// Create the address
	address := &models.Address{
		ID:                 uuid.New().String(),
		AddressNumber:      generateAddressNumber(),
		OrganizationID:     organizationID,
		Nickname:           req.Nickname,
		Company:            req.Company,
		ContactName:        req.ContactName,
		Phone:              req.Phone,
		Email:              req.Email,
		AddressLine1:       req.AddressLine1,
		AddressLine2:       req.AddressLine2,
		City:               req.City,
		ProvinceCode:       req.ProvinceCode,
		PostalCode:         req.PostalCode,
		CountryCode:        strings.ToUpper(req.CountryCode),
		AddressType:        models.AddressType(req.AddressType),
		IsResidential:      req.IsResidential,
		IsDefaultSender:    req.IsDefaultSender,
		IsDefaultRecipient: req.IsDefaultRecipient,
		IsVerified:         false,
		IsActive:           true,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	query := `
		INSERT INTO address_book (
			id, address_number, organization_id, created_by_user_id,
			nickname, company, contact_name, phone, email,
			address_line_1, address_line_2, city, province_code, postal_code, country_code,
			address_type, is_residential, is_default_sender, is_default_recipient,
			is_verified, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	_, err = tx.Exec(query,
		address.ID, address.AddressNumber, address.OrganizationID, userID,
		address.Nickname, address.Company, address.ContactName, address.Phone, address.Email,
		address.AddressLine1, address.AddressLine2, address.City, address.ProvinceCode, address.PostalCode, address.CountryCode,
		address.AddressType, address.IsResidential, address.IsDefaultSender, address.IsDefaultRecipient,
		address.IsVerified, address.IsActive, address.CreatedAt, address.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return address, nil
}

// GetAddressByID retrieves an address by ID
func (s *AddressService) GetAddressByID(addressID, organizationID string) (*models.Address, error) {
	query := `
		SELECT id, address_number, organization_id, created_by_user_id,
		       nickname, company, contact_name, phone, email,
		       address_line_1, address_line_2, city, province_code, postal_code, country_code,
		       address_type, is_residential, is_default_sender, is_default_recipient,
		       is_verified, is_active, created_at, updated_at
		FROM address_book
		WHERE id = $1 AND organization_id = $2 AND is_active = true
	`

	address := &models.Address{}
	var createdByUserID, company, phone, email, addressLine2, provinceCode sql.NullString

	err := s.db.DB.QueryRow(query, addressID, organizationID).Scan(
		&address.ID, &address.AddressNumber, &address.OrganizationID, &createdByUserID,
		&address.Nickname, &company, &address.ContactName, &phone, &email,
		&address.AddressLine1, &addressLine2, &address.City, &provinceCode, &address.PostalCode, &address.CountryCode,
		&address.AddressType, &address.IsResidential, &address.IsDefaultSender, &address.IsDefaultRecipient,
		&address.IsVerified, &address.IsActive, &address.CreatedAt, &address.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("address not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	address.CreatedByUserID = createdByUserID
	address.Company = company.String
	address.Phone = phone.String
	address.Email = email.String
	address.AddressLine2 = addressLine2.String
	address.ProvinceCode = provinceCode.String
	return address, nil
}

// ListAddresses retrieves a paginated list of addresses
func (s *AddressService) ListAddresses(organizationID string, req *ListAddressesRequest) ([]*models.Address, int, error) {
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
	countQuery := `SELECT COUNT(*) FROM address_book WHERE organization_id = $1 AND is_active = true`
	args := []interface{}{organizationID}
	argIndex := 2

	// Add filters
	if req.AddressType != "" {
		countQuery += fmt.Sprintf(" AND address_type = $%d", argIndex)
		args = append(args, req.AddressType)
		argIndex++
	}
	if req.Search != "" {
		countQuery += fmt.Sprintf(" AND (nickname ILIKE $%d OR contact_name ILIKE $%d OR company ILIKE $%d OR city ILIKE $%d)", argIndex, argIndex, argIndex, argIndex)
		args = append(args, "%"+req.Search+"%")
		argIndex++
	}
	if req.IsDefault != nil && *req.IsDefault {
		countQuery += " AND (is_default_sender = true OR is_default_recipient = true)"
	}

	// Get total count
	var total int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count addresses: %w", err)
	}

	// Build select query
	selectQuery := `
		SELECT id, address_number, organization_id, created_by_user_id,
		       nickname, company, contact_name, phone, email,
		       address_line_1, address_line_2, city, province_code, postal_code, country_code,
		       address_type, is_residential, is_default_sender, is_default_recipient,
		       is_verified, is_active, created_at, updated_at
		FROM address_book
		WHERE organization_id = $1 AND is_active = true
	`
	args = []interface{}{organizationID}
	argIndex = 2

	// Add filters
	if req.AddressType != "" {
		selectQuery += fmt.Sprintf(" AND address_type = $%d", argIndex)
		args = append(args, req.AddressType)
		argIndex++
	}
	if req.Search != "" {
		selectQuery += fmt.Sprintf(" AND (nickname ILIKE $%d OR contact_name ILIKE $%d OR company ILIKE $%d OR city ILIKE $%d)", argIndex, argIndex, argIndex, argIndex)
		args = append(args, "%"+req.Search+"%")
		argIndex++
	}
	if req.IsDefault != nil && *req.IsDefault {
		selectQuery += " AND (is_default_sender = true OR is_default_recipient = true)"
	}

	// Add ordering and pagination
	selectQuery += " ORDER BY is_default_sender DESC, is_default_recipient DESC, nickname ASC"
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := s.db.DB.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query addresses: %w", err)
	}
	defer rows.Close()

	// Parse results
	addresses := make([]*models.Address, 0)
	for rows.Next() {
		address := &models.Address{}
		var createdByUserID, company, phone, email, addressLine2, provinceCode sql.NullString

		err := rows.Scan(
			&address.ID, &address.AddressNumber, &address.OrganizationID, &createdByUserID,
			&address.Nickname, &company, &address.ContactName, &phone, &email,
			&address.AddressLine1, &addressLine2, &address.City, &provinceCode, &address.PostalCode, &address.CountryCode,
			&address.AddressType, &address.IsResidential, &address.IsDefaultSender, &address.IsDefaultRecipient,
			&address.IsVerified, &address.IsActive, &address.CreatedAt, &address.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan address: %w", err)
		}

		address.CreatedByUserID = createdByUserID
		address.Company = company.String
		address.Phone = phone.String
		address.Email = email.String
		address.AddressLine2 = addressLine2.String
		address.ProvinceCode = provinceCode.String
		addresses = append(addresses, address)
	}

	return addresses, total, nil
}

// UpdateAddress updates an existing address
func (s *AddressService) UpdateAddress(addressID, organizationID string, req *UpdateAddressRequest) (*models.Address, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Nickname != nil {
		updates = append(updates, fmt.Sprintf("nickname = $%d", argIndex))
		args = append(args, *req.Nickname)
		argIndex++
	}
	if req.Company != nil {
		updates = append(updates, fmt.Sprintf("company = $%d", argIndex))
		args = append(args, *req.Company)
		argIndex++
	}
	if req.ContactName != nil {
		updates = append(updates, fmt.Sprintf("contact_name = $%d", argIndex))
		args = append(args, *req.ContactName)
		argIndex++
	}
	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argIndex))
		args = append(args, *req.Phone)
		argIndex++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}
	if req.AddressLine1 != nil {
		updates = append(updates, fmt.Sprintf("address_line_1 = $%d", argIndex))
		args = append(args, *req.AddressLine1)
		argIndex++
	}
	if req.AddressLine2 != nil {
		updates = append(updates, fmt.Sprintf("address_line_2 = $%d", argIndex))
		args = append(args, *req.AddressLine2)
		argIndex++
	}
	if req.City != nil {
		updates = append(updates, fmt.Sprintf("city = $%d", argIndex))
		args = append(args, *req.City)
		argIndex++
	}
	if req.ProvinceCode != nil {
		updates = append(updates, fmt.Sprintf("province_code = $%d", argIndex))
		args = append(args, *req.ProvinceCode)
		argIndex++
	}
	if req.PostalCode != nil {
		updates = append(updates, fmt.Sprintf("postal_code = $%d", argIndex))
		args = append(args, *req.PostalCode)
		argIndex++
	}
	if req.CountryCode != nil {
		updates = append(updates, fmt.Sprintf("country_code = $%d", argIndex))
		args = append(args, strings.ToUpper(*req.CountryCode))
		argIndex++
	}
	if req.AddressType != nil {
		updates = append(updates, fmt.Sprintf("address_type = $%d", argIndex))
		args = append(args, *req.AddressType)
		argIndex++
	}
	if req.IsResidential != nil {
		updates = append(updates, fmt.Sprintf("is_residential = $%d", argIndex))
		args = append(args, *req.IsResidential)
		argIndex++
	}

	// If no updates, just return the existing address
	if len(updates) == 0 {
		return s.GetAddressByID(addressID, organizationID)
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now().UTC())
	argIndex++

	// Add WHERE clause parameters
	args = append(args, addressID, organizationID)

	query := fmt.Sprintf(
		"UPDATE address_book SET %s WHERE id = $%d AND organization_id = $%d AND is_active = true",
		strings.Join(updates, ", "),
		argIndex,
		argIndex+1,
	)

	result, err := s.db.DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("address not found")
	}

	return s.GetAddressByID(addressID, organizationID)
}

// DeleteAddress soft-deletes an address
func (s *AddressService) DeleteAddress(addressID, organizationID string) error {
	query := `
		UPDATE address_book
		SET is_active = false, is_default_sender = false, is_default_recipient = false, updated_at = $1
		WHERE id = $2 AND organization_id = $3 AND is_active = true
	`

	result, err := s.db.DB.Exec(query, time.Now().UTC(), addressID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}

	return nil
}

// SetDefaultAddress sets an address as the default sender or recipient
func (s *AddressService) SetDefaultAddress(addressID, organizationID, defaultType string) (*models.Address, error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify address exists and belongs to organization
	var addressType string
	err = tx.QueryRow(`
		SELECT address_type FROM address_book
		WHERE id = $1 AND organization_id = $2 AND is_active = true
	`, addressID, organizationID).Scan(&addressType)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("address not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	// Validate address type compatibility
	if defaultType == "sender" && addressType == "recipient" {
		return nil, fmt.Errorf("cannot set a recipient-only address as default sender")
	}
	if defaultType == "recipient" && addressType == "sender" {
		return nil, fmt.Errorf("cannot set a sender-only address as default recipient")
	}

	// Clear existing default
	if defaultType == "sender" {
		_, err = tx.Exec(`
			UPDATE address_book SET is_default_sender = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_sender = true
		`, time.Now().UTC(), organizationID)
	} else {
		_, err = tx.Exec(`
			UPDATE address_book SET is_default_recipient = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_recipient = true
		`, time.Now().UTC(), organizationID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing default: %w", err)
	}

	// Set new default
	var updateQuery string
	if defaultType == "sender" {
		updateQuery = `UPDATE address_book SET is_default_sender = true, updated_at = $1 WHERE id = $2`
	} else {
		updateQuery = `UPDATE address_book SET is_default_recipient = true, updated_at = $1 WHERE id = $2`
	}
	_, err = tx.Exec(updateQuery, time.Now().UTC(), addressID)
	if err != nil {
		return nil, fmt.Errorf("failed to set default: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetAddressByID(addressID, organizationID)
}

// GetDefaultAddress gets the default sender or recipient address
func (s *AddressService) GetDefaultAddress(organizationID, defaultType string) (*models.Address, error) {
	var query string
	if defaultType == "sender" {
		query = `
			SELECT id, address_number, organization_id, created_by_user_id,
			       nickname, company, contact_name, phone, email,
			       address_line_1, address_line_2, city, province_code, postal_code, country_code,
			       address_type, is_residential, is_default_sender, is_default_recipient,
			       is_verified, is_active, created_at, updated_at
			FROM address_book
			WHERE organization_id = $1 AND is_default_sender = true AND is_active = true
		`
	} else {
		query = `
			SELECT id, address_number, organization_id, created_by_user_id,
			       nickname, company, contact_name, phone, email,
			       address_line_1, address_line_2, city, province_code, postal_code, country_code,
			       address_type, is_residential, is_default_sender, is_default_recipient,
			       is_verified, is_active, created_at, updated_at
			FROM address_book
			WHERE organization_id = $1 AND is_default_recipient = true AND is_active = true
		`
	}

	address := &models.Address{}
	var createdByUserID, company, phone, email, addressLine2, provinceCode sql.NullString

	err := s.db.DB.QueryRow(query, organizationID).Scan(
		&address.ID, &address.AddressNumber, &address.OrganizationID, &createdByUserID,
		&address.Nickname, &company, &address.ContactName, &phone, &email,
		&address.AddressLine1, &addressLine2, &address.City, &provinceCode, &address.PostalCode, &address.CountryCode,
		&address.AddressType, &address.IsResidential, &address.IsDefaultSender, &address.IsDefaultRecipient,
		&address.IsVerified, &address.IsActive, &address.CreatedAt, &address.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no default %s address found", defaultType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default address: %w", err)
	}

	address.CreatedByUserID = createdByUserID
	address.Company = company.String
	address.Phone = phone.String
	address.Email = email.String
	address.AddressLine2 = addressLine2.String
	address.ProvinceCode = provinceCode.String
	return address, nil
}

// ClearDefault removes the default status from an address
func (s *AddressService) ClearDefault(addressID, organizationID, defaultType string) error {
	var query string
	if defaultType == "sender" {
		query = `UPDATE address_book SET is_default_sender = false, updated_at = $1 WHERE id = $2 AND organization_id = $3`
	} else {
		query = `UPDATE address_book SET is_default_recipient = false, updated_at = $1 WHERE id = $2 AND organization_id = $3`
	}

	result, err := s.db.DB.Exec(query, time.Now().UTC(), addressID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to clear default: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}

	return nil
}
