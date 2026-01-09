package users

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"

	"gogin/internal/utils"

	"github.com/pquerna/otp/totp"
)

type TwoFAMethod string

const (
	MethodAuthenticator TwoFAMethod = "authenticator"
	MethodSMS           TwoFAMethod = "sms"
)

type TwoFAMetadata struct {
	Method      string   `json:"2fa_method,omitempty"`
	TOTPSecret  string   `json:"totp_secret,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

type TwoFAService struct {
	db          *sql.DB
	userService *UserService
}

func NewTwoFAService(db *sql.DB, userService *UserService) *TwoFAService {
	return &TwoFAService{
		db:          db,
		userService: userService,
	}
}

// GenerateTOTP creates a new TOTP secret and returns QR code URL
func (s *TwoFAService) GenerateTOTP(userID, email string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "SkeletonAPI",
		AccountName: email,
		SecretSize:  32,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// VerifyTOTP checks if a TOTP code is valid
func (s *TwoFAService) VerifyTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateBackupCodes creates 10 backup codes
func (s *TwoFAService) GenerateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code, err := generateSecureCode(8)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

// Enable2FA enables 2FA for a user with the specified method
func (s *TwoFAService) Enable2FA(userID string, method TwoFAMethod, secret string, backupCodes []string) error {
	// Get current metadata
	metadata, err := s.getUserMetadata(userID)
	if err != nil {
		return err
	}

	// Update 2FA info in metadata
	metadata.Method = string(method)
	if method == MethodAuthenticator {
		metadata.TOTPSecret = secret
		metadata.BackupCodes = backupCodes
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update user record
	query := `
		UPDATE users
		SET has_2fa_enabled = true,
			metadata = $1,
			updated_at = $2
		WHERE id = $3
	`
	_, err = s.db.Exec(query, metadataJSON, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	return nil
}

// Disable2FA disables 2FA for a user
func (s *TwoFAService) Disable2FA(userID string) error {
	// Get current metadata
	metadata, err := s.getUserMetadata(userID)
	if err != nil {
		return err
	}

	// Clear 2FA data
	metadata.Method = ""
	metadata.TOTPSecret = ""
	metadata.BackupCodes = nil

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE users
		SET has_2fa_enabled = false,
			metadata = $1,
			updated_at = $2
		WHERE id = $3
	`
	_, err = s.db.Exec(query, metadataJSON, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	return nil
}

// Get2FAInfo retrieves 2FA configuration for a user
func (s *TwoFAService) Get2FAInfo(userID string) (*TwoFAMetadata, bool, error) {
	query := `SELECT has_2fa_enabled, metadata FROM users WHERE id = $1`

	var enabled bool
	var metadataJSON []byte

	err := s.db.QueryRow(query, userID).Scan(&enabled, &metadataJSON)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get 2FA info: %w", err)
	}

	metadata := &TwoFAMetadata{}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, metadata); err != nil {
			return nil, false, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return metadata, enabled, nil
}

// Verify2FACode verifies a 2FA code (TOTP, SMS OTP, or backup code)
func (s *TwoFAService) Verify2FACode(userID, code string) (bool, error) {
	metadata, enabled, err := s.Get2FAInfo(userID)
	if err != nil {
		return false, err
	}

	if !enabled {
		return false, fmt.Errorf("2FA not enabled")
	}

	// Handle SMS OTP verification
	if metadata.Method == string(MethodSMS) {
		// Get user's phone number
		_, _, userPhone, _, err := s.userService.GetUserByID(userID)
		if err != nil {
			return false, fmt.Errorf("failed to get user phone: %w", err)
		}

		if userPhone == nil || userPhone.Phone == "" {
			return false, fmt.Errorf("no phone number found for SMS verification")
		}

		// Verify SMS OTP
		verified, err := s.VerifySMSOTP(userPhone.Phone, code)
		if err != nil {
			return false, err
		}

		return verified, nil
	}

	// Handle Authenticator TOTP verification
	if metadata.Method == string(MethodAuthenticator) && metadata.TOTPSecret != "" {
		if s.VerifyTOTP(metadata.TOTPSecret, code) {
			return true, nil
		}

		// Check backup codes
		for i, backupCode := range metadata.BackupCodes {
			if utils.VerifyPassword(code, backupCode) {
				// Remove used backup code
				metadata.BackupCodes = append(metadata.BackupCodes[:i], metadata.BackupCodes[i+1:]...)
				s.updateUserMetadata(userID, metadata)
				return true, nil
			}
		}
	}

	return false, nil
}

// SendSMSOTP sends an OTP via SMS for 2FA
func (s *TwoFAService) SendSMSOTP(userID, phone, ipAddress string) error {
	// Generate 6-digit OTP
	otp := generateOTP()
	otpHash := hashOTP(otp)
	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// Create login challenge
	err := s.userService.CreateLoginChallenge(phone, otpHash, expiresAt, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to create OTP challenge: %w", err)
	}

	// SMS sending has been removed (Twilio wrapper deleted)
	// In development mode, show OTP in console
	if s.userService.config.IsDevelopment() {
		fmt.Printf("📱 [DEV MODE] 2FA OTP for %s: %s\n", phone, otp)
		fmt.Printf("   (This OTP expires in 5 minutes)\n")
		fmt.Printf("   Note: SMS sending disabled - external SMS service required for production\n")
	} else {
		// Production mode - log warning
		fmt.Printf("⚠️  SMS sending not configured for %s - 2FA OTP: %s\n", phone, otp)
	}

	// Also log in dev mode for debugging
	if s.userService.config.IsDevelopment() {
		fmt.Printf("📱 [DEV MODE] 2FA OTP sent to %s: %s\n", phone, otp)
	}

	return nil
}

// Verify SMS OTP
func (s *TwoFAService) VerifySMSOTP(phone, code string) (bool, error) {
	challenge, err := s.userService.GetActiveChallenge(phone)
	if err != nil {
		return false, fmt.Errorf("no active OTP challenge found")
	}

	// Check if expired
	if time.Now().UTC().After(challenge.ExpiresAt) {
		return false, fmt.Errorf("OTP expired")
	}

	// Check if locked due to too many attempts
	if challenge.LockedUntil.Valid && time.Now().UTC().Before(challenge.LockedUntil.Time) {
		return false, fmt.Errorf("too many failed attempts, please request a new OTP")
	}

	// Check attempts limit
	if challenge.Attempts >= challenge.MaxAttempts {
		s.userService.LockChallenge(challenge.ID)
		return false, fmt.Errorf("too many failed attempts, please request a new OTP")
	}

	// Verify OTP hash
	otpHash := hashOTP(code)
	if challenge.CodeHash != otpHash {
		s.userService.IncrementChallengeAttempts(challenge.ID)
		return false, nil
	}

	// Mark challenge as verified (prevents reuse)
	s.userService.MarkChallengeVerified(challenge.ID)
	return true, nil
}

// Helper functions
func (s *TwoFAService) getUserMetadata(userID string) (*TwoFAMetadata, error) {
	query := `SELECT metadata FROM users WHERE id = $1`
	var metadataJSON []byte

	err := s.db.QueryRow(query, userID).Scan(&metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	metadata := &TwoFAMetadata{}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, metadata); err != nil {
			// If unmarshal fails, return empty metadata (not an error)
			return &TwoFAMetadata{}, nil
		}
	}

	return metadata, nil
}

func (s *TwoFAService) updateUserMetadata(userID string, metadata *TwoFAMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	query := `UPDATE users SET metadata = $1, updated_at = $2 WHERE id = $3`
	_, err = s.db.Exec(query, metadataJSON, time.Now().UTC(), userID)
	return err
}

func generateSecureCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode as base32 without padding
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	return code[:length], nil
}

// GetBackupCodes retrieves the user's backup codes
func (s *TwoFAService) GetBackupCodes(userID string) ([]string, error) {
	metadata, enabled, err := s.Get2FAInfo(userID)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, fmt.Errorf("2FA is not enabled")
	}

	if metadata.Method != string(MethodAuthenticator) {
		return nil, fmt.Errorf("backup codes are only available for authenticator-based 2FA")
	}

	return metadata.BackupCodes, nil
}

// RegenerateBackupCodes generates new backup codes and replaces the old ones
func (s *TwoFAService) RegenerateBackupCodes(userID string) ([]string, error) {
	metadata, enabled, err := s.Get2FAInfo(userID)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, fmt.Errorf("2FA is not enabled")
	}

	if metadata.Method != string(MethodAuthenticator) {
		return nil, fmt.Errorf("backup codes are only available for authenticator-based 2FA")
	}

	// Generate new backup codes
	newCodes, err := s.GenerateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Update metadata with new codes
	metadata.BackupCodes = newCodes
	if err := s.updateUserMetadata(userID, metadata); err != nil {
		return nil, fmt.Errorf("failed to save backup codes: %w", err)
	}

	return newCodes, nil
}
