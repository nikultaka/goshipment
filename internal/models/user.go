package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// User represents a user in the system (updated for new schema)
type User struct {
	ID                 string         `json:"id" db:"id"`
	ExternalID         string         `json:"external_id" db:"external_id"` // usr_xxx
	PrimaryEmailID     sql.NullString `json:"primary_email_id,omitempty" db:"primary_email_id"`
	PrimaryPhoneID     sql.NullString `json:"primary_phone_id,omitempty" db:"primary_phone_id"`
	PasswordHash       string         `json:"-" db:"password_hash"`
	ClientNumber       sql.NullString `json:"client_number,omitempty" db:"client_number"` // CL-00001234
	ClientNumberGenAt  sql.NullTime   `json:"client_number_generated_at,omitempty" db:"client_number_generated_at"`
	Username           sql.NullString `json:"username,omitempty" db:"username"`

	// Profile
	FirstName     string         `json:"first_name" db:"first_name"`
	LastName      string         `json:"last_name" db:"last_name"`
	DisplayName   sql.NullString `json:"display_name,omitempty" db:"display_name"`
	AvatarURL     sql.NullString `json:"avatar_url,omitempty" db:"avatar_file_id"` // Note: DB column is avatar_file_id (UUID)

	// Preferences
	Timezone   string `json:"timezone" db:"timezone"`
	Locale     string `json:"locale" db:"locale"`
	DateFormat string `json:"date_format" db:"date_format"`
	TimeFormat string `json:"time_format" db:"time_format"`

	// Account Status
	IsActive          bool `json:"is_active" db:"is_active"`
	IsVerified        bool `json:"is_verified" db:"is_verified"` // KYC verified
	IsSalesRep        bool `json:"is_sales_rep" db:"is_sales_rep"` // Sales representative flag
	VerificationLevel int  `json:"verification_level" db:"verification_level"`

	// Login Security
	LastLoginAt   sql.NullTime   `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP   sql.NullString `json:"last_login_ip,omitempty" db:"last_login_ip"`
	LoginAttempts int            `json:"login_attempts" db:"login_attempts"`
	LockedUntil   sql.NullTime   `json:"locked_until,omitempty" db:"locked_until"`
	LockoutReason sql.NullString `json:"lockout_reason,omitempty" db:"lockout_reason"`

	// Token Security
	TokenVersion int `json:"token_version" db:"token_version"`

	// 2FA (TOTP)
	TOTPSecret       sql.NullString `json:"-" db:"totp_secret"`
	TOTPEnabled      bool           `json:"totp_enabled" db:"has_2fa_enabled"`
	TOTPBackupCodes  []string       `json:"-" db:"totp_backup_codes"` // TEXT[]

	// Privacy & Consent
	PrivacyConsent              json.RawMessage `json:"privacy_consent" db:"privacy_consent"`
	MarketingConsent            bool            `json:"marketing_consent" db:"marketing_consent"`
	MarketingConsentDate        sql.NullTime    `json:"marketing_consent_date,omitempty" db:"marketing_consent_date"`
	LastPrivacyNoticeAccepted   sql.NullTime    `json:"last_privacy_notice_accepted,omitempty" db:"last_privacy_notice_accepted"`

	// Account Lifecycle
	AccountDeletionRequestedAt sql.NullTime   `json:"account_deletion_requested_at,omitempty" db:"account_deletion_requested_at"`
	AccountDeletionReason      sql.NullString `json:"account_deletion_reason,omitempty" db:"account_deletion_reason"`
	AccountAnonymizedAt        sql.NullTime   `json:"account_anonymized_at,omitempty" db:"account_anonymized_at"`

	// Referral System
	ReferralCode      sql.NullString `json:"referral_code,omitempty" db:"referral_code"`
	ReferredByUserID  sql.NullString `json:"referred_by_user_id,omitempty" db:"referred_by_user_id"`

	// Registration Metadata
	RegistrationIP            sql.NullString  `json:"registration_ip,omitempty" db:"registration_ip"`
	RegistrationUserAgentHash sql.NullString  `json:"registration_user_agent_hash,omitempty" db:"registration_user_agent_hash"`
	UTMSource                 sql.NullString  `json:"utm_source,omitempty" db:"utm_source"`
	UTMMedium                 sql.NullString  `json:"utm_medium,omitempty" db:"utm_medium"`
	UTMCampaign               sql.NullString  `json:"utm_campaign,omitempty" db:"utm_campaign"`
	Metadata                  json.RawMessage `json:"metadata" db:"metadata"`

	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// UserEmail represents a user's email address (multi-email support)
type UserEmail struct {
	ID                    string       `json:"id" db:"id"`
	UserID                string       `json:"user_id" db:"user_id"`
	Email                 string       `json:"email" db:"email"`
	IsPrimary             bool         `json:"is_primary" db:"is_primary"`
	IsRecovery            bool         `json:"is_recovery" db:"is_recovery"`
	IsVerified            bool         `json:"is_verified" db:"is_verified"`
	VerifiedAt            sql.NullTime `json:"verified_at,omitempty" db:"verified_at"`
	VerificationToken     sql.NullString `json:"-" db:"verification_token"`
	VerificationExpiresAt sql.NullTime `json:"verification_expires_at,omitempty" db:"verification_expires_at"`
	IsActive              bool         `json:"is_active" db:"is_active"`
	CreatedAt             time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at" db:"updated_at"`
}

// UserPhone represents a user's phone number (multi-phone support)
type UserPhone struct {
	ID                    string         `json:"id" db:"id"`
	UserID                string         `json:"user_id" db:"user_id"`
	Phone                 string         `json:"phone" db:"phone"`
	PhoneE164             sql.NullString `json:"phone_e164,omitempty" db:"phone_e164"`
	IsPrimary             bool           `json:"is_primary" db:"is_primary"`
	IsRecovery            bool           `json:"is_recovery" db:"is_recovery"`
	IsVerified            bool           `json:"is_verified" db:"is_verified"`
	VerifiedAt            sql.NullTime   `json:"verified_at,omitempty" db:"verified_at"`
	VerificationCode      sql.NullString `json:"-" db:"verification_code"`
	VerificationExpiresAt sql.NullTime   `json:"verification_expires_at,omitempty" db:"verification_expires_at"`
	IsActive              bool           `json:"is_active" db:"is_active"`
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at" db:"updated_at"`
}

// Role represents a user role
type Role struct {
	ID             string          `json:"id" db:"id"`
	Name           string          `json:"name" db:"name"` // admin, manager, user
	DisplayName    sql.NullString  `json:"display_name,omitempty" db:"display_name"`
	Description    sql.NullString  `json:"description,omitempty" db:"description"`
	IsSystemRole   bool            `json:"is_system_role" db:"is_system_role"`
	IsAdminRole    bool            `json:"is_admin_role" db:"is_admin_role"`
	HierarchyLevel int             `json:"hierarchy_level" db:"hierarchy_level"`
	Permissions    json.RawMessage `json:"permissions" db:"permissions"` // JSONB array
	IsActive       bool            `json:"is_active" db:"is_active"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy      sql.NullString  `json:"created_by,omitempty" db:"created_by"`
}

// UserRole represents a user's role assignment
type UserRole struct {
	ID                 string         `json:"id" db:"id"`
	UserID             string         `json:"user_id" db:"user_id"`
	RoleID             string         `json:"role_id" db:"role_id"`
	GrantedBy          sql.NullString `json:"granted_by,omitempty" db:"granted_by"`
	GrantedAt          time.Time      `json:"granted_at" db:"granted_at"`
	ExpiresAt          sql.NullTime   `json:"expires_at,omitempty" db:"expires_at"`
	AssignmentReason   sql.NullString `json:"assignment_reason,omitempty" db:"assignment_reason"`
	BusinessContext    sql.NullString `json:"business_context,omitempty" db:"business_context"`
	IsActive           bool           `json:"is_active" db:"is_active"`
	SuspendedAt        sql.NullTime   `json:"suspended_at,omitempty" db:"suspended_at"`
	SuspendedBy        sql.NullString `json:"suspended_by,omitempty" db:"suspended_by"`
	SuspensionReason   sql.NullString `json:"suspension_reason,omitempty" db:"suspension_reason"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

// UserSession represents a user session
type UserSession struct {
	ID                 string         `json:"id" db:"id"`
	SessionID          string         `json:"session_id" db:"session_id"`
	UserID             string         `json:"user_id" db:"user_id"`
	IPAddress          sql.NullString `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent          sql.NullString `json:"user_agent,omitempty" db:"user_agent"`
	UserAgentHash      sql.NullString `json:"user_agent_hash,omitempty" db:"user_agent_hash"`
	DeviceFingerprint  sql.NullString `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
	Country            sql.NullString `json:"country,omitempty" db:"country"`
	Region             sql.NullString `json:"region,omitempty" db:"region"`
	City               sql.NullString `json:"city,omitempty" db:"city"`
	OAuthClientID      sql.NullString `json:"oauth_client_id,omitempty" db:"oauth_client_id"`
	OAuthScopes        []string       `json:"oauth_scopes,omitempty" db:"oauth_scopes"` // TEXT[]

	// Auth tracking
	AuthMethodID       sql.NullString `json:"auth_method_id,omitempty" db:"auth_method_id"`
	LoginMethodType    string         `json:"login_method_type" db:"login_method_type"` // password, google, facebook, phone_otp, etc.

	// Device tracking (from 030_session_devices)
	DeviceName         sql.NullString `json:"device_name,omitempty" db:"device_name"`
	DeviceType         sql.NullString `json:"device_type,omitempty" db:"device_type"` // mobile, tablet, desktop, unknown
	OSName             sql.NullString `json:"os_name,omitempty" db:"os_name"`
	OSVersion          sql.NullString `json:"os_version,omitempty" db:"os_version"`
	BrowserName        sql.NullString `json:"browser_name,omitempty" db:"browser_name"`
	BrowserVersion     sql.NullString `json:"browser_version,omitempty" db:"browser_version"`
	IsTrusted          bool           `json:"is_trusted" db:"is_trusted"`
	TrustedAt          sql.NullTime   `json:"trusted_at,omitempty" db:"trusted_at"`

	IsActive           bool           `json:"is_active" db:"is_active"`
	LastActivityAt     time.Time      `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	ExpiresAt          sql.NullTime   `json:"expires_at,omitempty" db:"expires_at"`
	TerminatedAt       sql.NullTime   `json:"terminated_at,omitempty" db:"terminated_at"`
	TerminationReason  sql.NullString `json:"termination_reason,omitempty" db:"termination_reason"`
}

// UserPasswordHistory tracks password changes
type UserPasswordHistory struct {
	ID           string         `json:"id" db:"id"`
	UserID       string         `json:"user_id" db:"user_id"`
	PasswordHash string         `json:"-" db:"password_hash"`
	ChangedBy    sql.NullString `json:"changed_by,omitempty" db:"changed_by"`
	ChangeReason sql.NullString `json:"change_reason,omitempty" db:"change_reason"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}

// UserOAuthAccount represents a linked OAuth provider account
type UserOAuthAccount struct {
	ID                 string         `json:"id" db:"id"`
	UserID             string         `json:"user_id" db:"user_id"`
	Provider           string         `json:"provider" db:"provider"` // google, microsoft, github, apple
	ProviderUserID     string         `json:"provider_user_id" db:"provider_user_id"`
	Email              sql.NullString `json:"email,omitempty" db:"email"`
	AvatarURL          sql.NullString `json:"avatar_url,omitempty" db:"avatar_url"`
	AccessTokenHash    sql.NullString `json:"-" db:"access_token_hash"`
	RefreshTokenHash   sql.NullString `json:"-" db:"refresh_token_hash"`
	ExpiresAt          sql.NullTime   `json:"expires_at,omitempty" db:"expires_at"`
	IsPrimary          bool           `json:"is_primary" db:"is_primary"`
	LinkedAt           time.Time      `json:"linked_at" db:"linked_at"`
}

// UserProfile represents additional user profile information (if still needed)
type UserProfile struct {
	UserID      string         `json:"user_id" db:"user_id"`
	Bio         sql.NullString `json:"bio,omitempty" db:"bio"`
	DateOfBirth sql.NullTime   `json:"date_of_birth,omitempty" db:"date_of_birth"`
	Gender      sql.NullString `json:"gender,omitempty" db:"gender"`
	Address     sql.NullString `json:"address,omitempty" db:"address"`
	City        sql.NullString `json:"city,omitempty" db:"city"`
	State       sql.NullString `json:"state,omitempty" db:"state"`
	Country     sql.NullString `json:"country,omitempty" db:"country"`
	ZipCode     sql.NullString `json:"zip_code,omitempty" db:"zip_code"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsUserActive returns true if user is active
func (u *User) IsUserActive() bool {
	return u.IsActive && !u.DeletedAt.Valid
}

// IsLocked returns true if user account is locked
func (u *User) IsLocked() bool {
	if u.LockedUntil.Valid {
		return time.Now().Before(u.LockedUntil.Time)
	}
	return false
}

// HasRole checks if user has a specific role (requires loading user roles)
func (u *User) HasRole(roleName string) bool {
	// This will need to be implemented with a proper role check
	// For now, return false - implementation depends on context
	return false
}

// LoginChallenge represents an OTP challenge for phone/email login
type LoginChallenge struct {
	ID            string         `json:"id" db:"id"`
	UserID        sql.NullString `json:"user_id,omitempty" db:"user_id"`
	ChallengeType string         `json:"challenge_type" db:"challenge_type"` // phone_otp, email_otp
	Destination   string         `json:"destination" db:"destination"`       // phone or email
	CodeHash      string         `json:"-" db:"code_hash"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	ExpiresAt     time.Time      `json:"expires_at" db:"expires_at"`
	VerifiedAt    sql.NullTime   `json:"verified_at,omitempty" db:"verified_at"`
	Attempts      int            `json:"attempts" db:"attempts"`
	MaxAttempts   int            `json:"max_attempts" db:"max_attempts"`
	LockedUntil   sql.NullTime   `json:"locked_until,omitempty" db:"locked_until"`
	IPAddress     sql.NullString `json:"ip_address,omitempty" db:"ip_address"`
}
