package users

import (
	"net/http"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// SessionResponse represents a user session in API responses
type SessionResponse struct {
	ID                string  `json:"id"`
	SessionID         string  `json:"session_id"`
	DeviceName        string  `json:"device_name"`
	DeviceType        string  `json:"device_type"`
	OSName            string  `json:"os_name"`
	OSVersion         string  `json:"os_version,omitempty"`
	BrowserName       string  `json:"browser_name"`
	BrowserVersion    string  `json:"browser_version,omitempty"`
	IPAddress         string  `json:"ip_address"`
	City              string  `json:"city,omitempty"`
	Country           string  `json:"country,omitempty"`
	LoginMethodType   string  `json:"login_method_type"`
	IsTrusted         bool    `json:"is_trusted"`
	IsCurrentSession  bool    `json:"is_current_session"`
	LastActivityAt    string  `json:"last_activity_at"`
	CreatedAt         string  `json:"created_at"`
}

// getSessions retrieves all active sessions for the current user
// @Summary Get active sessions
// @Description Get all active sessions for the authenticated user across different devices
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{sessions=[]SessionResponse,total=int}}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/me/sessions [get]
func (m *UsersModule) getSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get current session ID from JWT claims (if stored)
	currentSessionID, _ := c.Get("session_id")

	sessions, err := m.sessionService.GetActiveSessions(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to retrieve sessions")
		return
	}

	// Convert to response format
	sessionResponses := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, SessionResponse{
			ID:               session.ID,
			SessionID:        session.SessionID,
			DeviceName:       session.DeviceName.String,
			DeviceType:       session.DeviceType.String,
			OSName:           session.OSName.String,
			OSVersion:        session.OSVersion.String,
			BrowserName:      session.BrowserName.String,
			BrowserVersion:   session.BrowserVersion.String,
			IPAddress:        session.IPAddress.String,
			City:             session.City.String,
			Country:          session.Country.String,
			LoginMethodType:  session.LoginMethodType,
			IsTrusted:        session.IsTrusted,
			IsCurrentSession: currentSessionID != nil && session.SessionID == currentSessionID.(string),
			LastActivityAt:   session.LastActivityAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt:        session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response.Success(c, http.StatusOK, "Sessions retrieved successfully", gin.H{
		"sessions": sessionResponses,
		"total":    len(sessionResponses),
	})
}

// terminateSession terminates a specific session (logout from a specific device)
// @Summary Terminate a session
// @Description Logout from a specific device/session
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID or SessionID to terminate"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/me/sessions/{session_id} [delete]
func (m *UsersModule) terminateSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.BadRequest(c, "Session ID is required")
		return
	}

	err := m.sessionService.TerminateSession(userID.(string), sessionID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Session terminated successfully", nil)
}

// logoutAllDevices terminates all sessions for the current user
// @Summary Logout from all devices
// @Description Terminate all active sessions (logout from all devices)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{terminated_count=int}}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/me/sessions/logout-all [post]
func (m *UsersModule) logoutAllDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	count, err := m.sessionService.TerminateAllSessions(userID.(string))
	if err != nil {
		response.InternalError(c, "Failed to logout from all devices")
		return
	}

	response.Success(c, http.StatusOK, "Logged out from all devices successfully", gin.H{
		"terminated_count": count,
	})
}

// logoutOtherDevices terminates all sessions except the current one
// @Summary Logout from other devices
// @Description Terminate all active sessions except the current one
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{terminated_count=int}}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/me/sessions/logout-others [post]
func (m *UsersModule) logoutOtherDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Get current session ID from JWT claims
	currentSessionID, exists := c.Get("session_id")
	if !exists {
		// If no session ID in token, logout all
		count, err := m.sessionService.TerminateAllSessions(userID.(string))
		if err != nil {
			response.InternalError(c, "Failed to logout from other devices")
			return
		}
		response.Success(c, http.StatusOK, "Logged out from all devices successfully", gin.H{
			"terminated_count": count,
		})
		return
	}

	count, err := m.sessionService.TerminateAllSessionsExcept(userID.(string), currentSessionID.(string))
	if err != nil {
		response.InternalError(c, "Failed to logout from other devices")
		return
	}

	response.Success(c, http.StatusOK, "Logged out from other devices successfully", gin.H{
		"terminated_count": count,
	})
}

// trustSession marks a session as trusted
// @Summary Trust a device
// @Description Mark a session/device as trusted to skip additional security checks
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID to trust"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/me/sessions/{session_id}/trust [patch]
func (m *UsersModule) trustSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.BadRequest(c, "Session ID is required")
		return
	}

	err := m.sessionService.TrustSession(userID.(string), sessionID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Session marked as trusted", nil)
}
