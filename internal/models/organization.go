package models

import (
	"database/sql"
	"time"
)

// OrganizationStatus represents the organization lifecycle status
type OrganizationStatus string

const (
	OrganizationStatusActive    OrganizationStatus = "active"
	OrganizationStatusSuspended OrganizationStatus = "suspended"
	OrganizationStatusClosed    OrganizationStatus = "closed"
)

// OrganizationPlan represents the organization subscription tier
type OrganizationPlan string

const (
	OrganizationPlanTier1 OrganizationPlan = "tier_1" // Entry / Starter
	OrganizationPlanTier2 OrganizationPlan = "tier_2" // Growth / Pro
	OrganizationPlanTier3 OrganizationPlan = "tier_3" // Business / Scale
	OrganizationPlanTier4 OrganizationPlan = "tier_4" // Enterprise / Custom
)

// OrganizationRole represents member authority roles
type OrganizationRole string

const (
	OrganizationRoleOwner         OrganizationRole = "owner"
	OrganizationRoleAdministrator OrganizationRole = "administrator"
	OrganizationRoleManager       OrganizationRole = "manager"
	OrganizationRoleOperator      OrganizationRole = "operator"
	OrganizationRoleViewer        OrganizationRole = "viewer"
)

// Organization represents an organization entity
type Organization struct {
	ID             string             `json:"id" db:"id"`
	ExternalID     string             `json:"external_id" db:"external_id"`
	Name           string             `json:"name" db:"name"`
	OwnerUserID    string             `json:"owner_user_id" db:"owner_user_id"`
	Status         OrganizationStatus `json:"status" db:"status"`
	Plan           OrganizationPlan   `json:"plan" db:"plan"`
	IsDeleted      bool               `json:"is_deleted" db:"is_deleted"`
	DeletedAt      sql.NullTime       `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedByUserID sql.NullString    `json:"deleted_by_user_id,omitempty" db:"deleted_by_user_id"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	ID             string           `json:"id" db:"id"`
	OrganizationID string           `json:"organization_id" db:"organization_id"`
	UserID         string           `json:"user_id" db:"user_id"`
	Role           OrganizationRole `json:"role" db:"role"`
	IsActive       bool             `json:"is_active" db:"is_active"`
	JoinedAt       time.Time        `json:"joined_at" db:"joined_at"`
}

// OrganizationSettings represents admin-controlled feature flags
type OrganizationSettings struct {
	ID                   string    `json:"id" db:"id"`
	OrganizationID       string    `json:"organization_id" db:"organization_id"`
	AllowSameDayPickup   bool      `json:"allow_same_day_pickup" db:"allow_same_day_pickup"`
	AllowNextDayPickup   bool      `json:"allow_next_day_pickup" db:"allow_next_day_pickup"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationPreferences represents user-controlled org defaults
type OrganizationPreferences struct {
	ID                         string         `json:"id" db:"id"`
	OrganizationID             string         `json:"organization_id" db:"organization_id"`
	DefaultSenderAddressID     sql.NullString `json:"default_sender_address_id,omitempty" db:"default_sender_address_id"`
	DefaultRecipientAddressID  sql.NullString `json:"default_recipient_address_id,omitempty" db:"default_recipient_address_id"`
	PackageReference           sql.NullString `json:"package_reference,omitempty" db:"package_reference"`
	SignatureOption            string         `json:"signature_option" db:"signature_option"`
	DeliveryInstructions       sql.NullString `json:"delivery_instructions,omitempty" db:"delivery_instructions"`
	WeightUnit                 string         `json:"weight_unit" db:"weight_unit"`
	DimensionUnit              string         `json:"dimension_unit" db:"dimension_unit"`
	Timezone                   string         `json:"timezone" db:"timezone"`
	LabelFormat                string         `json:"label_format" db:"label_format"`
	HideShipperPhoneOnLabel    bool           `json:"hide_shipper_phone_on_label" db:"hide_shipper_phone_on_label"`
	ShowUSDEstimates           bool           `json:"show_usd_estimates" db:"show_usd_estimates"`
	CreatedAt                  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at" db:"updated_at"`
}

// IsOwner checks if the role is owner
func (r OrganizationRole) IsOwner() bool {
	return r == OrganizationRoleOwner
}

// IsAdminOrHigher checks if the role is admin or owner
func (r OrganizationRole) IsAdminOrHigher() bool {
	return r == OrganizationRoleOwner || r == OrganizationRoleAdministrator
}

// IsManagerOrHigher checks if the role is manager, admin, or owner
func (r OrganizationRole) IsManagerOrHigher() bool {
	return r == OrganizationRoleOwner || r == OrganizationRoleAdministrator || r == OrganizationRoleManager
}

// IsActive checks if organization is active
func (o *Organization) IsActive() bool {
	return o.Status == OrganizationStatusActive && !o.IsDeleted
}

// IsReadOnly checks if organization is in read-only mode (suspended)
func (o *Organization) IsReadOnly() bool {
	return o.Status == OrganizationStatusSuspended
}

// IsClosed checks if organization is closed
func (o *Organization) IsClosed() bool {
	return o.Status == OrganizationStatusClosed
}
