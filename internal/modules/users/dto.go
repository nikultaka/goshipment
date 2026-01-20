package users

import (
	"time"
)

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	FirstName    string `json:"first_name" binding:"required"`
	LastName     string `json:"last_name" binding:"required"`
	ReferralCode string `json:"referral_code,omitempty"` // Optional: referral code from another user
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UserResponse represents a user response (without sensitive data)
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Phone         string    `json:"phone,omitempty"`
	Avatar        string    `json:"avatar,omitempty"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LoginResponse represents a login response with tokens
type LoginResponse struct {
	AccessToken   string        `json:"access_token,omitempty"`
	RefreshToken  string        `json:"refresh_token,omitempty"`
	TokenType     string        `json:"token_type,omitempty"`
	ExpiresIn     int           `json:"expires_in,omitempty"`
	SessionID     string        `json:"session_id,omitempty"` // Session ID for multi-device tracking
	User          *UserResponse `json:"user,omitempty"`
	TwoFARequired bool          `json:"2fa_required,omitempty"` // If true, client must verify 2FA
	TwoFAMethod   string        `json:"2fa_method,omitempty"`   // sms or authenticator
}

// UsersListResponse represents a paginated list of users
type UsersListResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// TwoFA DTOs
type Setup2FARequest struct {
	Method string `json:"method" binding:"required,oneof=sms authenticator" example:"authenticator"`
}

type Setup2FAResponse struct {
	Method      string   `json:"method"`
	Secret      string   `json:"secret,omitempty"`
	QRCodeURL   string   `json:"qr_code_url,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
	Message     string   `json:"message,omitempty"`
}

type Verify2FASetupRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

type Verify2FALoginRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

type TwoFAStatusResponse struct {
	Enabled bool   `json:"enabled"`
	Method  string `json:"method,omitempty"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents a token refresh response
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"` // Optional: if rotating refresh tokens
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Multi-Email Management DTOs

// AddEmailRequest represents a request to add a new email
type AddEmailRequest struct {
	Email     string `json:"email" binding:"required,email"`
	IsPrimary bool   `json:"is_primary"`
}

// UserEmailResponse represents a user email response
type UserEmailResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	IsPrimary  bool      `json:"is_primary"`
	IsRecovery bool      `json:"is_recovery"`
	IsVerified bool      `json:"is_verified"`
	VerifiedAt time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UserEmailsListResponse represents a list of user emails
type UserEmailsListResponse struct {
	Emails []*UserEmailResponse `json:"emails"`
	Total  int                  `json:"total"`
}

// Multi-Phone Management DTOs

// AddPhoneRequest represents a request to add a new phone
type AddPhoneRequest struct {
	Phone     string `json:"phone" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}

// UserPhoneResponse represents a user phone response
type UserPhoneResponse struct {
	ID         string    `json:"id"`
	Phone      string    `json:"phone"`
	PhoneE164  string    `json:"phone_e164,omitempty"`
	IsPrimary  bool      `json:"is_primary"`
	IsRecovery bool      `json:"is_recovery"`
	IsVerified bool      `json:"is_verified"`
	VerifiedAt time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UserPhonesListResponse represents a list of user phones
type UserPhonesListResponse struct {
	Phones []*UserPhoneResponse `json:"phones"`
	Total  int                  `json:"total"`
}

// VerifyPhoneRequest represents a request to verify a phone number
type VerifyPhoneRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// Admin DTOs

// AdminCreateUserRequest represents an admin request to create a user
type AdminCreateUserRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone"`
	Role      string `json:"role"` // admin, manager, user (defaults to user)
	IsActive  *bool  `json:"is_active"`
}

// Password Reset DTOs

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents a reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// Export DTOs

// CreateExportRequest represents a request to create an export
type CreateExportRequest struct {
	ExportType     string `json:"export_type" binding:"required"`                      // wallet_transactions, shipments, invoices, etc.
	Format         string `json:"format" binding:"omitempty,oneof=csv json xlsx"`      // csv, json, xlsx (default: csv)
	DeliveryMethod string `json:"delivery_method" binding:"omitempty,oneof=download email"` // download, email (default: download)
}

// ExportResponse represents an export response
type ExportResponse struct {
	ID             string  `json:"id"`
	ExportType     string  `json:"export_type"`
	Format         string  `json:"format"`
	DeliveryMethod string  `json:"delivery_method"`
	Status         string  `json:"status"` // queued, processing, ready, failed, expired
	ErrorMessage   *string `json:"error_message,omitempty"`
	FileID         *string `json:"file_id,omitempty"`
	FileSize       *int64  `json:"file_size,omitempty"`
	RowCount       *int32  `json:"row_count,omitempty"`
	DownloadURL    *string `json:"download_url,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// ExportsListResponse represents a list of exports
type ExportsListResponse struct {
	Exports    []*ExportResponse `json:"exports"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// Business Profile DTOs

// CreateBusinessRequest represents a request to create a business profile
type CreateBusinessRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=255"`
	Description  string `json:"description" binding:"omitempty,max=1000"`
	AddressLine1 string `json:"address_line1" binding:"omitempty,max=255"`
	AddressLine2 string `json:"address_line2" binding:"omitempty,max=255"`
	City         string `json:"city" binding:"omitempty,max=100"`
	Province     string `json:"province" binding:"omitempty,max=100"`
	PostalCode   string `json:"postal_code" binding:"omitempty,max=20"`
	Country      string `json:"country" binding:"omitempty,len=2"` // ISO 3166-1 alpha-2
	Phone        string `json:"phone" binding:"omitempty,max=20"`
	Email        string `json:"email" binding:"omitempty,email,max=255"`
}

// UpdateBusinessRequest represents a request to update a business profile (partial updates)
type UpdateBusinessRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description  *string `json:"description" binding:"omitempty,max=1000"`
	AddressLine1 *string `json:"address_line1" binding:"omitempty,max=255"`
	AddressLine2 *string `json:"address_line2" binding:"omitempty,max=255"`
	City         *string `json:"city" binding:"omitempty,max=100"`
	Province     *string `json:"province" binding:"omitempty,max=100"`
	PostalCode   *string `json:"postal_code" binding:"omitempty,max=20"`
	Country      *string `json:"country" binding:"omitempty,len=2"`
	Phone        *string `json:"phone" binding:"omitempty,max=20"`
	Email        *string `json:"email" binding:"omitempty,email,max=255"`
}

// BusinessResponse represents a business profile response
type BusinessResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Address      string    `json:"address,omitempty"`      // Formatted full address
	AddressLine1 string    `json:"address_line1,omitempty"`
	AddressLine2 string    `json:"address_line2,omitempty"`
	City         string    `json:"city,omitempty"`
	Province     string    `json:"province,omitempty"`
	PostalCode   string    `json:"postal_code,omitempty"`
	Country      string    `json:"country,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Email        string    `json:"email,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// =====================================================
// Organization User Management DTOs
// =====================================================

// UserProfileResponse represents the current user's profile with organization context
type UserProfileResponse struct {
	UserID         string    `json:"user_id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone,omitempty"`
	Role           string    `json:"role,omitempty"`           // Organization role: owner, administrator, manager, operator, viewer (optional if not in org)
	IsSalesRep     bool      `json:"is_sales_rep"`   // Sales representative flag
	ReferralCode   string    `json:"referral_code,omitempty"`
	OrganizationID string    `json:"organization_id,omitempty"` // Optional if user not in organization
	Avatar         string    `json:"avatar,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UpdateOwnProfileRequest represents a request to update own profile
type UpdateOwnProfileRequest struct {
	FirstName    *string `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName     *string `json:"last_name" binding:"omitempty,min=1,max=100"`
	Phone        *string `json:"phone" binding:"omitempty"`
	AvatarFileID *string `json:"avatar_file_id" binding:"omitempty"`
	Password     *string `json:"password" binding:"omitempty,min=8"`
	OldPassword  *string `json:"old_password" binding:"omitempty"` // Required if password is being changed
}

// OrganizationUserResponse represents a user in an organization (admin view)
type OrganizationUserResponse struct {
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`         // Full name (first + last)
	Email      string    `json:"email"`
	Role       string    `json:"role"`         // Organization role
	IsSalesRep bool      `json:"is_sales_rep"`
	IsActive   bool      `json:"is_active"`
	JoinedAt   time.Time `json:"joined_at"`
}

// OrganizationUsersListResponse represents a list of organization users
type OrganizationUsersListResponse struct {
	Users []*OrganizationUserResponse `json:"users"`
	Total int                         `json:"total"`
}

// ChangeUserRoleRequest represents a request to change a user's role
type ChangeUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=owner administrator manager operator viewer"`
}

// ToggleSalesRepRequest represents a request to toggle sales rep status
type ToggleSalesRepRequest struct {
	IsSalesRep bool `json:"is_sales_rep"`
}

// AnonymizeUserRequest represents a request to anonymize a user
type AnonymizeUserRequest struct {
	Reason string `json:"reason" binding:"required,min=10,max=500"` // Reason for anonymization (audit)
}
