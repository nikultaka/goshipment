package organizations

import (
	"database/sql"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
)

// OrganizationService handles organization business logic
type OrganizationService struct {
	db *clients.Database
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(db *clients.Database) *OrganizationService {
	return &OrganizationService{
		db: db,
	}
}

// GetOrganizationByUserID retrieves the organization for a given user
func (s *OrganizationService) GetOrganizationByUserID(userID string) (*models.Organization, *models.OrganizationMember, error) {
	query := `
		SELECT
			o.id, o.external_id, o.name, o.owner_user_id, o.status, o.plan,
			o.is_deleted, o.deleted_at, o.deleted_by_user_id, o.created_at, o.updated_at,
			om.id as member_id, om.organization_id, om.user_id, om.role, om.is_active, om.joined_at
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND om.is_active = true
		LIMIT 1
	`

	var org models.Organization
	var member models.OrganizationMember

	err := s.db.DB.QueryRow(query, userID).Scan(
		&org.ID, &org.ExternalID, &org.Name, &org.OwnerUserID, &org.Status, &org.Plan,
		&org.IsDeleted, &org.DeletedAt, &org.DeletedByUserID, &org.CreatedAt, &org.UpdatedAt,
		&member.ID, &member.OrganizationID, &member.UserID, &member.Role, &member.IsActive, &member.JoinedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("user not associated with any organization")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, &member, nil
}

// GetOrganizationByID retrieves an organization by its ID
func (s *OrganizationService) GetOrganizationByID(organizationID string) (*models.Organization, error) {
	query := `
		SELECT
			id, external_id, name, owner_user_id, status, plan,
			is_deleted, deleted_at, deleted_by_user_id, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	var org models.Organization

	err := s.db.DB.QueryRow(query, organizationID).Scan(
		&org.ID, &org.ExternalID, &org.Name, &org.OwnerUserID, &org.Status, &org.Plan,
		&org.IsDeleted, &org.DeletedAt, &org.DeletedByUserID, &org.CreatedAt, &org.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// GetOrganizationMembers retrieves all members of an organization
func (s *OrganizationService) GetOrganizationMembers(organizationID string) ([]models.OrganizationMember, error) {
	query := `
		SELECT id, organization_id, user_id, role, is_active, joined_at
		FROM organization_members
		WHERE organization_id = $1
		ORDER BY
			CASE role
				WHEN 'owner' THEN 1
				WHEN 'administrator' THEN 2
				WHEN 'manager' THEN 3
				WHEN 'operator' THEN 4
				WHEN 'viewer' THEN 5
			END,
			joined_at ASC
	`

	rows, err := s.db.DB.Query(query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization members: %w", err)
	}
	defer rows.Close()

	var members []models.OrganizationMember
	for rows.Next() {
		var member models.OrganizationMember
		err := rows.Scan(
			&member.ID, &member.OrganizationID, &member.UserID,
			&member.Role, &member.IsActive, &member.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating members: %w", err)
	}

	return members, nil
}

// GetOrganizationSettings retrieves organization settings
func (s *OrganizationService) GetOrganizationSettings(organizationID string) (*models.OrganizationSettings, error) {
	query := `
		SELECT id, organization_id, allow_same_day_pickup, allow_next_day_pickup, created_at, updated_at
		FROM organization_settings
		WHERE organization_id = $1
	`

	var settings models.OrganizationSettings
	err := s.db.DB.QueryRow(query, organizationID).Scan(
		&settings.ID, &settings.OrganizationID,
		&settings.AllowSameDayPickup, &settings.AllowNextDayPickup,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settings not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization settings: %w", err)
	}

	return &settings, nil
}

// UpdateOrganizationSettings updates organization settings
func (s *OrganizationService) UpdateOrganizationSettings(organizationID string, allowSameDayPickup, allowNextDayPickup *bool) (*models.OrganizationSettings, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current settings
	var settings models.OrganizationSettings
	query := `
		SELECT id, organization_id, allow_same_day_pickup, allow_next_day_pickup, created_at, updated_at
		FROM organization_settings
		WHERE organization_id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(query, organizationID).Scan(
		&settings.ID, &settings.OrganizationID,
		&settings.AllowSameDayPickup, &settings.AllowNextDayPickup,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settings not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization settings: %w", err)
	}

	// Update fields if provided
	if allowSameDayPickup != nil {
		settings.AllowSameDayPickup = *allowSameDayPickup
	}
	if allowNextDayPickup != nil {
		settings.AllowNextDayPickup = *allowNextDayPickup
	}
	settings.UpdatedAt = time.Now().UTC()

	// Save changes
	updateQuery := `
		UPDATE organization_settings
		SET allow_same_day_pickup = $1, allow_next_day_pickup = $2, updated_at = $3
		WHERE organization_id = $4
	`
	_, err = tx.Exec(updateQuery, settings.AllowSameDayPickup, settings.AllowNextDayPickup, settings.UpdatedAt, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization settings: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &settings, nil
}

// GetOrganizationPreferences retrieves organization preferences
func (s *OrganizationService) GetOrganizationPreferences(organizationID string) (*models.OrganizationPreferences, error) {
	query := `
		SELECT
			id, organization_id, default_sender_address_id, default_recipient_address_id,
			package_reference, signature_option, delivery_instructions,
			weight_unit, dimension_unit, timezone, label_format,
			hide_shipper_phone_on_label, show_usd_estimates, created_at, updated_at
		FROM organization_preferences
		WHERE organization_id = $1
	`

	var prefs models.OrganizationPreferences
	err := s.db.DB.QueryRow(query, organizationID).Scan(
		&prefs.ID, &prefs.OrganizationID,
		&prefs.DefaultSenderAddressID, &prefs.DefaultRecipientAddressID,
		&prefs.PackageReference, &prefs.SignatureOption, &prefs.DeliveryInstructions,
		&prefs.WeightUnit, &prefs.DimensionUnit, &prefs.Timezone, &prefs.LabelFormat,
		&prefs.HideShipperPhoneOnLabel, &prefs.ShowUSDEstimates,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("preferences not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization preferences: %w", err)
	}

	return &prefs, nil
}

// UpdateOrganizationPreferencesInput holds update parameters
type UpdateOrganizationPreferencesInput struct {
	DefaultSenderAddressID    *string
	DefaultRecipientAddressID *string
	PackageReference          *string
	SignatureOption           *string
	DeliveryInstructions      *string
	WeightUnit                *string
	DimensionUnit             *string
	Timezone                  *string
	LabelFormat               *string
	HideShipperPhoneOnLabel   *bool
	ShowUSDEstimates          *bool
}

// UpdateOrganizationPreferences updates organization preferences
func (s *OrganizationService) UpdateOrganizationPreferences(organizationID string, input UpdateOrganizationPreferencesInput) (*models.OrganizationPreferences, error) {
	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current preferences
	var prefs models.OrganizationPreferences
	query := `
		SELECT
			id, organization_id, default_sender_address_id, default_recipient_address_id,
			package_reference, signature_option, delivery_instructions,
			weight_unit, dimension_unit, timezone, label_format,
			hide_shipper_phone_on_label, show_usd_estimates, created_at, updated_at
		FROM organization_preferences
		WHERE organization_id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(query, organizationID).Scan(
		&prefs.ID, &prefs.OrganizationID,
		&prefs.DefaultSenderAddressID, &prefs.DefaultRecipientAddressID,
		&prefs.PackageReference, &prefs.SignatureOption, &prefs.DeliveryInstructions,
		&prefs.WeightUnit, &prefs.DimensionUnit, &prefs.Timezone, &prefs.LabelFormat,
		&prefs.HideShipperPhoneOnLabel, &prefs.ShowUSDEstimates,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("preferences not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization preferences: %w", err)
	}

	// Update fields if provided
	if input.DefaultSenderAddressID != nil {
		if *input.DefaultSenderAddressID == "" {
			prefs.DefaultSenderAddressID = sql.NullString{Valid: false}
		} else {
			prefs.DefaultSenderAddressID = sql.NullString{String: *input.DefaultSenderAddressID, Valid: true}
		}
	}
	if input.DefaultRecipientAddressID != nil {
		if *input.DefaultRecipientAddressID == "" {
			prefs.DefaultRecipientAddressID = sql.NullString{Valid: false}
		} else {
			prefs.DefaultRecipientAddressID = sql.NullString{String: *input.DefaultRecipientAddressID, Valid: true}
		}
	}
	if input.PackageReference != nil {
		if *input.PackageReference == "" {
			prefs.PackageReference = sql.NullString{Valid: false}
		} else {
			prefs.PackageReference = sql.NullString{String: *input.PackageReference, Valid: true}
		}
	}
	if input.SignatureOption != nil {
		prefs.SignatureOption = *input.SignatureOption
	}
	if input.DeliveryInstructions != nil {
		if *input.DeliveryInstructions == "" {
			prefs.DeliveryInstructions = sql.NullString{Valid: false}
		} else {
			prefs.DeliveryInstructions = sql.NullString{String: *input.DeliveryInstructions, Valid: true}
		}
	}
	if input.WeightUnit != nil {
		prefs.WeightUnit = *input.WeightUnit
	}
	if input.DimensionUnit != nil {
		prefs.DimensionUnit = *input.DimensionUnit
	}
	if input.Timezone != nil {
		prefs.Timezone = *input.Timezone
	}
	if input.LabelFormat != nil {
		prefs.LabelFormat = *input.LabelFormat
	}
	if input.HideShipperPhoneOnLabel != nil {
		prefs.HideShipperPhoneOnLabel = *input.HideShipperPhoneOnLabel
	}
	if input.ShowUSDEstimates != nil {
		prefs.ShowUSDEstimates = *input.ShowUSDEstimates
	}

	prefs.UpdatedAt = time.Now().UTC()

	// Save changes
	updateQuery := `
		UPDATE organization_preferences
		SET
			default_sender_address_id = $1,
			default_recipient_address_id = $2,
			package_reference = $3,
			signature_option = $4,
			delivery_instructions = $5,
			weight_unit = $6,
			dimension_unit = $7,
			timezone = $8,
			label_format = $9,
			hide_shipper_phone_on_label = $10,
			show_usd_estimates = $11,
			updated_at = $12
		WHERE organization_id = $13
	`
	_, err = tx.Exec(updateQuery,
		prefs.DefaultSenderAddressID, prefs.DefaultRecipientAddressID,
		prefs.PackageReference, prefs.SignatureOption, prefs.DeliveryInstructions,
		prefs.WeightUnit, prefs.DimensionUnit, prefs.Timezone, prefs.LabelFormat,
		prefs.HideShipperPhoneOnLabel, prefs.ShowUSDEstimates,
		prefs.UpdatedAt, organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization preferences: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &prefs, nil
}

// ValidateUserInOrganization checks if user belongs to organization
func (s *OrganizationService) ValidateUserInOrganization(userID, organizationID string) (*models.OrganizationMember, error) {
	query := `
		SELECT id, organization_id, user_id, role, is_active, joined_at
		FROM organization_members
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
	`

	var member models.OrganizationMember
	err := s.db.DB.QueryRow(query, userID, organizationID).Scan(
		&member.ID, &member.OrganizationID, &member.UserID,
		&member.Role, &member.IsActive, &member.JoinedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not authorized for this organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate user membership: %w", err)
	}

	return &member, nil
}

// CreateOrganization creates a new organization with the user as owner
func (s *OrganizationService) CreateOrganization(name string, plan models.OrganizationPlan, userID string) (*models.Organization, error) {
	// Check if user already belongs to an organization
	existingOrg, _, err := s.GetOrganizationByUserID(userID)
	if err == nil && existingOrg != nil {
		return nil, fmt.Errorf("user already belongs to an organization: %s", existingOrg.Name)
	}

	// Start transaction
	tx, err := s.db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate external_id (org_XXXXXXXX)
	externalID, err := generateOrganizationExternalID(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate external ID: %w", err)
	}

	// Create organization
	var org models.Organization
	orgQuery := `
		INSERT INTO organizations (external_id, name, owner_user_id, status, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, external_id, name, owner_user_id, status, plan, is_deleted, deleted_at, deleted_by_user_id, created_at, updated_at
	`
	err = tx.QueryRow(orgQuery, externalID, name, userID, models.OrganizationStatusActive, plan).Scan(
		&org.ID, &org.ExternalID, &org.Name, &org.OwnerUserID, &org.Status, &org.Plan,
		&org.IsDeleted, &org.DeletedAt, &org.DeletedByUserID, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add creator as owner in organization_members
	memberQuery := `
		INSERT INTO organization_members (organization_id, user_id, role, is_active, joined_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err = tx.Exec(memberQuery, org.ID, userID, models.OrganizationRoleOwner, true)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner to organization members: %w", err)
	}

	// Create default organization_settings
	settingsQuery := `
		INSERT INTO organization_settings (organization_id, allow_same_day_pickup, allow_next_day_pickup, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`
	_, err = tx.Exec(settingsQuery, org.ID, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization settings: %w", err)
	}

	// Create default organization_preferences
	prefsQuery := `
		INSERT INTO organization_preferences (
			organization_id, signature_option, weight_unit, dimension_unit,
			timezone, label_format, hide_shipper_phone_on_label, show_usd_estimates,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err = tx.Exec(prefsQuery, org.ID, "no_signature", "kg", "cm", "America/Toronto", "pdf", false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization preferences: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &org, nil
}

// generateOrganizationExternalID generates a unique external ID for organization
func generateOrganizationExternalID(tx *sql.Tx) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8

	for attempts := 0; attempts < 10; attempts++ {
		// Generate random string
		randomStr := make([]byte, length)
		for i := range randomStr {
			randomStr[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		}
		externalID := "org_" + string(randomStr)

		// Check if exists
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM organizations WHERE external_id = $1)", externalID).Scan(&exists)
		if err != nil {
			return "", err
		}

		if !exists {
			return externalID, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique external ID after 10 attempts")
}
