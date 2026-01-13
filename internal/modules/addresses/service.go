package addresses

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// AddressService handles address book business logic
type AddressService struct {
	db *clients.Database
}

// NewAddressService creates a new address service
func NewAddressService(db *clients.Database) *AddressService {
	return &AddressService{
		db: db,
	}
}

// ListAddresses retrieves all active addresses for an organization
func (s *AddressService) ListAddresses(organizationID string, activeOnly bool) ([]*AddressResponse, error) {
	query := `
		SELECT
			id, address_number, organization_id, nickname, company, contact_name,
			phone, email, address_line_1, address_line_2, city, province_code,
			postal_code, country_code, address_type, is_residential,
			is_default_sender, is_default_recipient, is_verified, is_active,
			created_at, updated_at
		FROM address_book
		WHERE organization_id = $1
	`

	args := []interface{}{organizationID}

	if activeOnly {
		query += " AND is_active = true"
	}

	query += " ORDER BY is_default_sender DESC, is_default_recipient DESC, nickname ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}
	defer rows.Close()

	addresses := []*AddressResponse{}
	for rows.Next() {
		var addr models.Address
		err := rows.Scan(
			&addr.ID, &addr.AddressNumber, &addr.OrganizationID, &addr.Nickname,
			&addr.Company, &addr.ContactName, &addr.Phone, &addr.Email,
			&addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.ProvinceCode,
			&addr.PostalCode, &addr.CountryCode, &addr.AddressType, &addr.IsResidential,
			&addr.IsDefaultSender, &addr.IsDefaultRecipient, &addr.IsVerified, &addr.IsActive,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}

		addresses = append(addresses, s.toAddressResponse(&addr))
	}

	return addresses, nil
}

// GetAddress retrieves a single address by ID (must belong to organization)
func (s *AddressService) GetAddress(addressID, organizationID string) (*AddressResponse, error) {
	query := `
		SELECT
			id, address_number, organization_id, nickname, company, contact_name,
			phone, email, address_line_1, address_line_2, city, province_code,
			postal_code, country_code, address_type, is_residential,
			is_default_sender, is_default_recipient, is_verified, is_active,
			created_at, updated_at
		FROM address_book
		WHERE id = $1 AND organization_id = $2
	`

	var addr models.Address
	err := s.db.QueryRow(query, addressID, organizationID).Scan(
		&addr.ID, &addr.AddressNumber, &addr.OrganizationID, &addr.Nickname,
		&addr.Company, &addr.ContactName, &addr.Phone, &addr.Email,
		&addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.ProvinceCode,
		&addr.PostalCode, &addr.CountryCode, &addr.AddressType, &addr.IsResidential,
		&addr.IsDefaultSender, &addr.IsDefaultRecipient, &addr.IsVerified, &addr.IsActive,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("address not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	return s.toAddressResponse(&addr), nil
}

// CreateAddress creates a new address book entry
func (s *AddressService) CreateAddress(organizationID, userID string, req *CreateAddressRequest) (*AddressResponse, error) {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate address number (ADR-XXXXXXXX)
	addressNumber := s.generateAddressNumber()

	// If setting as default sender, clear previous default
	if req.IsDefaultSender {
		_, err = tx.Exec(`
			UPDATE address_book
			SET is_default_sender = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_sender = true
		`, time.Now().UTC(), organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear previous default sender: %w", err)
		}
	}

	// If setting as default recipient, clear previous default
	if req.IsDefaultRecipient {
		_, err = tx.Exec(`
			UPDATE address_book
			SET is_default_recipient = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_recipient = true
		`, time.Now().UTC(), organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear previous default recipient: %w", err)
		}
	}

	// Insert new address
	now := time.Now().UTC()
	addressID := uuid.New().String()

	query := `
		INSERT INTO address_book (
			id, address_number, organization_id, created_by_user_id,
			nickname, company, contact_name, phone, email,
			address_line_1, address_line_2, city, province_code, postal_code, country_code,
			address_type, is_residential,
			is_default_sender, is_default_recipient, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15,
			$16::address_type_enum, $17,
			$18, $19, true,
			$20, $21
		)
		RETURNING
			id, address_number, organization_id, nickname, company, contact_name,
			phone, email, address_line_1, address_line_2, city, province_code,
			postal_code, country_code, address_type, is_residential,
			is_default_sender, is_default_recipient, is_verified, is_active,
			created_at, updated_at
	`

	var addr models.Address
	err = tx.QueryRow(query,
		addressID, addressNumber, organizationID, userID,
		req.Nickname, req.Company, req.ContactName, req.Phone, req.Email,
		req.AddressLine1, req.AddressLine2, req.City, req.ProvinceCode, req.PostalCode, req.CountryCode,
		req.AddressType, req.IsResidential,
		req.IsDefaultSender, req.IsDefaultRecipient,
		now, now,
	).Scan(
		&addr.ID, &addr.AddressNumber, &addr.OrganizationID, &addr.Nickname,
		&addr.Company, &addr.ContactName, &addr.Phone, &addr.Email,
		&addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.ProvinceCode,
		&addr.PostalCode, &addr.CountryCode, &addr.AddressType, &addr.IsResidential,
		&addr.IsDefaultSender, &addr.IsDefaultRecipient, &addr.IsVerified, &addr.IsActive,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.toAddressResponse(&addr), nil
}

// UpdateAddress updates an existing address book entry
func (s *AddressService) UpdateAddress(addressID, organizationID string, req *UpdateAddressRequest) (*AddressResponse, error) {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify address belongs to organization
	var exists bool
	err = tx.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM address_book WHERE id = $1 AND organization_id = $2)
	`, addressID, organizationID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to verify address: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("address not found")
	}

	// Handle default flags
	if req.IsDefaultSender != nil && *req.IsDefaultSender {
		_, err = tx.Exec(`
			UPDATE address_book
			SET is_default_sender = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_sender = true AND id != $3
		`, time.Now().UTC(), organizationID, addressID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear previous default sender: %w", err)
		}
	}

	if req.IsDefaultRecipient != nil && *req.IsDefaultRecipient {
		_, err = tx.Exec(`
			UPDATE address_book
			SET is_default_recipient = false, updated_at = $1
			WHERE organization_id = $2 AND is_default_recipient = true AND id != $3
		`, time.Now().UTC(), organizationID, addressID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear previous default recipient: %w", err)
		}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Nickname != nil {
		updates = append(updates, fmt.Sprintf("nickname = $%d", argPos))
		args = append(args, *req.Nickname)
		argPos++
	}

	if req.Company != nil {
		updates = append(updates, fmt.Sprintf("company = $%d", argPos))
		args = append(args, *req.Company)
		argPos++
	}

	if req.ContactName != nil {
		updates = append(updates, fmt.Sprintf("contact_name = $%d", argPos))
		args = append(args, *req.ContactName)
		argPos++
	}

	if req.Phone != nil {
		updates = append(updates, fmt.Sprintf("phone = $%d", argPos))
		args = append(args, *req.Phone)
		argPos++
	}

	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, *req.Email)
		argPos++
	}

	if req.AddressLine1 != nil {
		updates = append(updates, fmt.Sprintf("address_line_1 = $%d", argPos))
		args = append(args, *req.AddressLine1)
		argPos++
	}

	if req.AddressLine2 != nil {
		updates = append(updates, fmt.Sprintf("address_line_2 = $%d", argPos))
		args = append(args, *req.AddressLine2)
		argPos++
	}

	if req.City != nil {
		updates = append(updates, fmt.Sprintf("city = $%d", argPos))
		args = append(args, *req.City)
		argPos++
	}

	if req.ProvinceCode != nil {
		updates = append(updates, fmt.Sprintf("province_code = $%d", argPos))
		args = append(args, *req.ProvinceCode)
		argPos++
	}

	if req.PostalCode != nil {
		updates = append(updates, fmt.Sprintf("postal_code = $%d", argPos))
		args = append(args, *req.PostalCode)
		argPos++
	}

	if req.CountryCode != nil {
		updates = append(updates, fmt.Sprintf("country_code = $%d", argPos))
		args = append(args, *req.CountryCode)
		argPos++
	}

	if req.AddressType != nil {
		updates = append(updates, fmt.Sprintf("address_type = $%d::address_type_enum", argPos))
		args = append(args, *req.AddressType)
		argPos++
	}

	if req.IsResidential != nil {
		updates = append(updates, fmt.Sprintf("is_residential = $%d", argPos))
		args = append(args, *req.IsResidential)
		argPos++
	}

	if req.IsDefaultSender != nil {
		updates = append(updates, fmt.Sprintf("is_default_sender = $%d", argPos))
		args = append(args, *req.IsDefaultSender)
		argPos++
	}

	if req.IsDefaultRecipient != nil {
		updates = append(updates, fmt.Sprintf("is_default_recipient = $%d", argPos))
		args = append(args, *req.IsDefaultRecipient)
		argPos++
	}

	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Always update updated_at
	now := time.Now().UTC()
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, now)
	argPos++

	// Add WHERE clause
	args = append(args, addressID)
	args = append(args, organizationID)

	updateQuery := fmt.Sprintf(`
		UPDATE address_book
		SET %s
		WHERE id = $%d AND organization_id = $%d
		RETURNING
			id, address_number, organization_id, nickname, company, contact_name,
			phone, email, address_line_1, address_line_2, city, province_code,
			postal_code, country_code, address_type, is_residential,
			is_default_sender, is_default_recipient, is_verified, is_active,
			created_at, updated_at
	`, strings.Join(updates, ", "), argPos, argPos+1)

	var addr models.Address
	err = tx.QueryRow(updateQuery, args...).Scan(
		&addr.ID, &addr.AddressNumber, &addr.OrganizationID, &addr.Nickname,
		&addr.Company, &addr.ContactName, &addr.Phone, &addr.Email,
		&addr.AddressLine1, &addr.AddressLine2, &addr.City, &addr.ProvinceCode,
		&addr.PostalCode, &addr.CountryCode, &addr.AddressType, &addr.IsResidential,
		&addr.IsDefaultSender, &addr.IsDefaultRecipient, &addr.IsVerified, &addr.IsActive,
		&addr.CreatedAt, &addr.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("address not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.toAddressResponse(&addr), nil
}

// Helper methods

// generateAddressNumber generates a unique address number (ADR-XXXXXXXX)
func (s *AddressService) generateAddressNumber() string {
	// Generate 8 uppercase alphanumeric characters
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[uuid.New().ID()%uint32(len(chars))]
	}
	return fmt.Sprintf("ADR-%s", string(result))
}

// toAddressResponse converts a model address to a response DTO
func (s *AddressService) toAddressResponse(addr *models.Address) *AddressResponse {
	return &AddressResponse{
		ID:                 addr.ID,
		AddressNumber:      addr.AddressNumber,
		OrganizationID:     addr.OrganizationID,
		Nickname:           addr.Nickname,
		Company:            addr.Company,
		ContactName:        addr.ContactName,
		Phone:              addr.Phone,
		Email:              addr.Email,
		AddressLine1:       addr.AddressLine1,
		AddressLine2:       addr.AddressLine2,
		City:               addr.City,
		ProvinceCode:       addr.ProvinceCode,
		PostalCode:         addr.PostalCode,
		CountryCode:        addr.CountryCode,
		AddressType:        string(addr.AddressType),
		IsResidential:      addr.IsResidential,
		IsDefaultSender:    addr.IsDefaultSender,
		IsDefaultRecipient: addr.IsDefaultRecipient,
		IsVerified:         addr.IsVerified,
		IsActive:           addr.IsActive,
		FormattedAddress:   addr.FormattedAddress(),
		CreatedAt:          addr.CreatedAt,
		UpdatedAt:          addr.UpdatedAt,
	}
}
