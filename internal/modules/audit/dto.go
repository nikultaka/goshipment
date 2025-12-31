package audit

import "time"

// AuditLogResponse represents an audit log entry response
type AuditLogResponse struct {
	ID                   string     `json:"id"`
	EventID              string     `json:"event_id"`
	ActorType            string     `json:"actor_type"`
	UserID               *string    `json:"user_id,omitempty"`
	APIClientID          *string    `json:"api_client_id,omitempty"`
	ActorIdentifier      *string    `json:"actor_identifier,omitempty"`
	ActorIP              *string    `json:"actor_ip,omitempty"`
	Action               string     `json:"action"`
	ResourceType         *string    `json:"resource_type,omitempty"`
	ResourceID           *string    `json:"resource_id,omitempty"`
	BusinessPurpose      string     `json:"business_purpose"`
	Status               string     `json:"status"`
	RecordsAffected      int        `json:"records_affected"`
	ResponseTimeMs       *int       `json:"response_time_ms,omitempty"`
	ErrorCode            *string    `json:"error_code,omitempty"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	OccurredAt           time.Time  `json:"occurred_at"`
	RecordedAt           time.Time  `json:"recorded_at"`
}

// AuditLogsListResponse represents a paginated list of audit logs
type AuditLogsListResponse struct {
	Logs       []*AuditLogResponse `json:"logs"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// AuditLogFilters represents filters for audit log queries
type AuditLogFilters struct {
	UserID       string `form:"user_id"`
	ActorType    string `form:"actor_type"`
	Action       string `form:"action"`
	ResourceType string `form:"resource_type"`
	Status       string `form:"status"`
	StartDate    string `form:"start_date"` // Format: 2006-01-02
	EndDate      string `form:"end_date"`   // Format: 2006-01-02
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
	SortBy       string `form:"sort_by"`   // occurred_at, recorded_at
	SortOrder    string `form:"sort_order"` // asc, desc
}
