package users

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
	"github.com/mileusna/useragent"
)

// SessionService handles user session management
type SessionService struct {
	db *clients.Database
}

// NewSessionService creates a new session service
func NewSessionService(db *clients.Database) *SessionService {
	return &SessionService{db: db}
}

// CreateSession creates a new session record for a user login
func (s *SessionService) CreateSession(userID, ipAddress, userAgent, loginMethod string) (*models.UserSession, error) {
	// Parse user agent to extract device info
	ua := useragent.Parse(userAgent)

	deviceType := "unknown"
	if ua.Mobile {
		deviceType = "mobile"
	} else if ua.Tablet {
		deviceType = "tablet"
	} else if ua.Desktop {
		deviceType = "desktop"
	}

	// Create device name
	deviceName := fmt.Sprintf("%s on %s", ua.Name, ua.OS)
	if deviceName == " on " {
		deviceName = "Unknown Device"
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	// Hash user agent for comparison
	uaHash := fmt.Sprintf("%x", sha256.Sum256([]byte(userAgent)))

	// Set expiration (30 days from now)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	query := `
		INSERT INTO user_sessions (
			session_id, user_id, ip_address, user_agent, user_agent_hash,
			login_method_type, device_name, device_type,
			os_name, os_version, browser_name, browser_version,
			is_active, is_trusted, last_activity_at, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var id string
	err := s.db.QueryRow(
		query,
		sessionID, userID, ipAddress, userAgent, uaHash,
		loginMethod, deviceName, deviceType,
		ua.OS, ua.OSVersion, ua.Name, ua.Version,
		true, false, time.Now().UTC(), expiresAt, time.Now().UTC(),
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &models.UserSession{
		ID:              id,
		SessionID:       sessionID,
		UserID:          userID,
		IPAddress:       sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:       sql.NullString{String: userAgent, Valid: userAgent != ""},
		UserAgentHash:   sql.NullString{String: uaHash, Valid: true},
		LoginMethodType: loginMethod,
		DeviceName:      sql.NullString{String: deviceName, Valid: true},
		DeviceType:      sql.NullString{String: deviceType, Valid: true},
		OSName:          sql.NullString{String: ua.OS, Valid: ua.OS != ""},
		OSVersion:       sql.NullString{String: ua.OSVersion, Valid: ua.OSVersion != ""},
		BrowserName:     sql.NullString{String: ua.Name, Valid: ua.Name != ""},
		BrowserVersion:  sql.NullString{String: ua.Version, Valid: ua.Version != ""},
		IsActive:        true,
		IsTrusted:       false,
		LastActivityAt:  time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
		ExpiresAt:       sql.NullTime{Time: expiresAt, Valid: true},
	}, nil
}

// GetActiveSessions retrieves all active sessions for a user
func (s *SessionService) GetActiveSessions(userID string) ([]models.UserSession, error) {
	query := `
		SELECT
			id, session_id, user_id, ip_address, user_agent, user_agent_hash,
			device_fingerprint, country, region, city,
			oauth_client_id, auth_method_id, login_method_type,
			device_name, device_type, os_name, os_version,
			browser_name, browser_version, is_trusted, trusted_at,
			is_active, last_activity_at, created_at, expires_at,
			terminated_at, termination_reason
		FROM user_sessions
		WHERE user_id = $1
			AND is_active = true
			AND terminated_at IS NULL
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY last_activity_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sessions: %w", err)
	}
	defer rows.Close()

	sessions := []models.UserSession{}
	for rows.Next() {
		var session models.UserSession
		var oauthScopes []string // Handle TEXT[] type

		err := rows.Scan(
			&session.ID, &session.SessionID, &session.UserID,
			&session.IPAddress, &session.UserAgent, &session.UserAgentHash,
			&session.DeviceFingerprint, &session.Country, &session.Region, &session.City,
			&session.OAuthClientID, &session.AuthMethodID, &session.LoginMethodType,
			&session.DeviceName, &session.DeviceType, &session.OSName, &session.OSVersion,
			&session.BrowserName, &session.BrowserVersion, &session.IsTrusted, &session.TrustedAt,
			&session.IsActive, &session.LastActivityAt, &session.CreatedAt, &session.ExpiresAt,
			&session.TerminatedAt, &session.TerminationReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session.OAuthScopes = oauthScopes
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// TerminateSession terminates a specific session
func (s *SessionService) TerminateSession(userID, sessionID string) error {
	query := `
		UPDATE user_sessions
		SET is_active = false,
			terminated_at = $1,
			termination_reason = $2
		WHERE user_id = $3
			AND (id::text = $4 OR session_id = $4)
			AND is_active = true
	`

	result, err := s.db.Exec(query, time.Now().UTC(), "user_logout", userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found or already terminated")
	}

	return nil
}

// TerminateAllSessions terminates all active sessions for a user
func (s *SessionService) TerminateAllSessions(userID string) (int, error) {
	query := `
		UPDATE user_sessions
		SET is_active = false,
			terminated_at = $1,
			termination_reason = $2
		WHERE user_id = $3
			AND is_active = true
			AND terminated_at IS NULL
	`

	result, err := s.db.Exec(query, time.Now().UTC(), "user_logout_all", userID)
	if err != nil {
		return 0, fmt.Errorf("failed to terminate all sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// TerminateAllSessionsExcept terminates all sessions except the current one
func (s *SessionService) TerminateAllSessionsExcept(userID, currentSessionID string) (int, error) {
	query := `
		UPDATE user_sessions
		SET is_active = false,
			terminated_at = $1,
			termination_reason = $2
		WHERE user_id = $3
			AND session_id != $4
			AND is_active = true
			AND terminated_at IS NULL
	`

	result, err := s.db.Exec(query, time.Now().UTC(), "user_logout_other_devices", userID, currentSessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to terminate other sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// ValidateSession checks if a session is valid and active
func (s *SessionService) ValidateSession(sessionID string) (*models.UserSession, error) {
	query := `
		SELECT
			id, session_id, user_id, ip_address, user_agent, user_agent_hash,
			device_fingerprint, country, region, city,
			oauth_client_id, auth_method_id, login_method_type,
			device_name, device_type, os_name, os_version,
			browser_name, browser_version, is_trusted, trusted_at,
			is_active, last_activity_at, created_at, expires_at,
			terminated_at, termination_reason
		FROM user_sessions
		WHERE session_id = $1
			AND is_active = true
			AND terminated_at IS NULL
			AND (expires_at IS NULL OR expires_at > NOW())
	`

	var session models.UserSession
	var oauthScopes []string

	err := s.db.QueryRow(query, sessionID).Scan(
		&session.ID, &session.SessionID, &session.UserID,
		&session.IPAddress, &session.UserAgent, &session.UserAgentHash,
		&session.DeviceFingerprint, &session.Country, &session.Region, &session.City,
		&session.OAuthClientID, &session.AuthMethodID, &session.LoginMethodType,
		&session.DeviceName, &session.DeviceType, &session.OSName, &session.OSVersion,
		&session.BrowserName, &session.BrowserVersion, &session.IsTrusted, &session.TrustedAt,
		&session.IsActive, &session.LastActivityAt, &session.CreatedAt, &session.ExpiresAt,
		&session.TerminatedAt, &session.TerminationReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	session.OAuthScopes = oauthScopes

	// Update last activity timestamp
	s.updateLastActivity(sessionID)

	return &session, nil
}

// updateLastActivity updates the last_activity_at timestamp for a session
func (s *SessionService) updateLastActivity(sessionID string) {
	query := `UPDATE user_sessions SET last_activity_at = $1 WHERE session_id = $2`
	s.db.Exec(query, time.Now().UTC(), sessionID)
}

// TrustSession marks a session as trusted
func (s *SessionService) TrustSession(userID, sessionID string) error {
	query := `
		UPDATE user_sessions
		SET is_trusted = true,
			trusted_at = $1
		WHERE user_id = $2
			AND (id::text = $3 OR session_id = $3)
			AND is_active = true
	`

	result, err := s.db.Exec(query, time.Now().UTC(), userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to trust session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions (should be run periodically)
func (s *SessionService) CleanupExpiredSessions() (int, error) {
	query := `
		UPDATE user_sessions
		SET is_active = false,
			terminated_at = NOW(),
			termination_reason = 'expired'
		WHERE is_active = true
			AND expires_at < NOW()
	`

	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// GetUserSessions retrieves all sessions for a user (alias for GetActiveSessions for admin endpoints)
func (s *SessionService) GetUserSessions(userID string) ([]models.UserSession, error) {
	return s.GetActiveSessions(userID)
}

// LogoutAllDevices terminates all sessions for a user (alias for TerminateAllSessions)
func (s *SessionService) LogoutAllDevices(userID string) error {
	_, err := s.TerminateAllSessions(userID)
	return err
}
