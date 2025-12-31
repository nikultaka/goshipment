package users

import (
	"time"
)

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
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
	Method string `json:"method" binding:"required,oneof=sms authenticator"`
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
	Email string `json:"email" binding:"required,email"`
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
	Phone string `json:"phone" binding:"required"`
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
