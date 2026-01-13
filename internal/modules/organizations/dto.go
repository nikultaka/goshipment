package organizations

import (
	"database/sql"
	"gogin/internal/models"
)

// GetOrganizationResponse represents the organization details response
type GetOrganizationResponse struct {
	ID          string                     `json:"id"`
	ExternalID  string                     `json:"external_id"`
	Name        string                     `json:"name"`
	OwnerUserID string                     `json:"owner_user_id"`
	Status      models.OrganizationStatus  `json:"status"`
	Plan        models.OrganizationPlan    `json:"plan"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
	MyRole      models.OrganizationRole    `json:"my_role,omitempty"`
}

// OrganizationMemberResponse represents a member in the organization
type OrganizationMemberResponse struct {
	ID             string                  `json:"id"`
	OrganizationID string                  `json:"organization_id"`
	UserID         string                  `json:"user_id"`
	Role           models.OrganizationRole `json:"role"`
	IsActive       bool                    `json:"is_active"`
	JoinedAt       string                  `json:"joined_at"`
	User           *MemberUserInfo         `json:"user,omitempty"`
}

// MemberUserInfo represents basic user info for organization member
type MemberUserInfo struct {
	ID        string  `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
}

// GetOrganizationSettingsResponse represents organization settings
type GetOrganizationSettingsResponse struct {
	AllowSameDayPickup bool   `json:"allow_same_day_pickup"`
	AllowNextDayPickup bool   `json:"allow_next_day_pickup"`
	UpdatedAt          string `json:"updated_at"`
}

// UpdateOrganizationSettingsRequest represents update settings payload
type UpdateOrganizationSettingsRequest struct {
	AllowSameDayPickup *bool `json:"allow_same_day_pickup"`
	AllowNextDayPickup *bool `json:"allow_next_day_pickup"`
}

// GetOrganizationPreferencesResponse represents organization preferences
type GetOrganizationPreferencesResponse struct {
	DefaultSenderAddressID    *string `json:"default_sender_address_id,omitempty"`
	DefaultRecipientAddressID *string `json:"default_recipient_address_id,omitempty"`
	PackageReference          *string `json:"package_reference,omitempty"`
	SignatureOption           string  `json:"signature_option"`
	DeliveryInstructions      *string `json:"delivery_instructions,omitempty"`
	WeightUnit                string  `json:"weight_unit"`
	DimensionUnit             string  `json:"dimension_unit"`
	Timezone                  string  `json:"timezone"`
	LabelFormat               string  `json:"label_format"`
	HideShipperPhoneOnLabel   bool    `json:"hide_shipper_phone_on_label"`
	ShowUSDEstimates          bool    `json:"show_usd_estimates"`
	UpdatedAt                 string  `json:"updated_at"`
}

// UpdateOrganizationPreferencesRequest represents update preferences payload
type UpdateOrganizationPreferencesRequest struct {
	DefaultSenderAddressID    *string `json:"default_sender_address_id"`
	DefaultRecipientAddressID *string `json:"default_recipient_address_id"`
	PackageReference          *string `json:"package_reference"`
	SignatureOption           *string `json:"signature_option"`
	DeliveryInstructions      *string `json:"delivery_instructions"`
	WeightUnit                *string `json:"weight_unit"`
	DimensionUnit             *string `json:"dimension_unit"`
	Timezone                  *string `json:"timezone"`
	LabelFormat               *string `json:"label_format"`
	HideShipperPhoneOnLabel   *bool   `json:"hide_shipper_phone_on_label"`
	ShowUSDEstimates          *bool   `json:"show_usd_estimates"`
}

// AddOrganizationMemberRequest represents request to add member
type AddOrganizationMemberRequest struct {
	UserID string                  `json:"user_id" binding:"required"`
	Role   models.OrganizationRole `json:"role" binding:"required"`
}

// UpdateMemberRoleRequest represents request to update member role
type UpdateMemberRoleRequest struct {
	Role models.OrganizationRole `json:"role" binding:"required"`
}

// Helper function to convert sql.NullString to *string
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// ToPreferencesResponse converts model to response DTO
func ToPreferencesResponse(prefs *models.OrganizationPreferences) *GetOrganizationPreferencesResponse {
	return &GetOrganizationPreferencesResponse{
		DefaultSenderAddressID:    nullStringToPtr(prefs.DefaultSenderAddressID),
		DefaultRecipientAddressID: nullStringToPtr(prefs.DefaultRecipientAddressID),
		PackageReference:          nullStringToPtr(prefs.PackageReference),
		SignatureOption:           prefs.SignatureOption,
		DeliveryInstructions:      nullStringToPtr(prefs.DeliveryInstructions),
		WeightUnit:                prefs.WeightUnit,
		DimensionUnit:             prefs.DimensionUnit,
		Timezone:                  prefs.Timezone,
		LabelFormat:               prefs.LabelFormat,
		HideShipperPhoneOnLabel:   prefs.HideShipperPhoneOnLabel,
		ShowUSDEstimates:          prefs.ShowUSDEstimates,
		UpdatedAt:                 prefs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
