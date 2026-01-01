package audit

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"gogin/internal/clients"
	"math"
	"strings"
	"time"
)

// AuditService handles audit log operations
type AuditService struct {
	db *clients.Database
}

// NewAuditService creates a new audit service
func NewAuditService(db *clients.Database) *AuditService {
	return &AuditService{db: db}
}

// GetAuditLogs retrieves audit logs with filtering and pagination
func (s *AuditService) GetAuditLogs(filters *AuditLogFilters) (*AuditLogsListResponse, error) {
	// Set default pagination
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 50
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	offset := (filters.Page - 1) * filters.Limit

	// Set default sorting
	if filters.SortBy == "" {
		filters.SortBy = "occurred_at"
	}
	if filters.SortOrder == "" {
		filters.SortOrder = "desc"
	}

	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argCounter := 1

	if filters.UserID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argCounter))
		args = append(args, filters.UserID)
		argCounter++
	}
	if filters.ActorType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_type = $%d", argCounter))
		args = append(args, filters.ActorType)
		argCounter++
	}
	if filters.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action ILIKE $%d", argCounter))
		args = append(args, "%"+filters.Action+"%")
		argCounter++
	}
	if filters.ResourceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource_type = $%d", argCounter))
		args = append(args, filters.ResourceType)
		argCounter++
	}
	if filters.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argCounter))
		args = append(args, filters.Status)
		argCounter++
	}
	if filters.StartDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("occurred_at >= $%d", argCounter))
		args = append(args, filters.StartDate)
		argCounter++
	}
	if filters.EndDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("occurred_at <= $%d", argCounter))
		args = append(args, filters.EndDate+" 23:59:59")
		argCounter++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int
	err := s.db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get logs with pagination
	query := fmt.Sprintf(`
		SELECT
			id, event_id, actor_type, user_id, api_client_id,
			actor_identifier, actor_ip, action, resource_type,
			resource_id, business_purpose, status, records_affected,
			response_time_ms, error_code, error_message,
			occurred_at, recorded_at
		FROM audit_logs
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, filters.SortBy, filters.SortOrder, argCounter, argCounter+1)

	args = append(args, filters.Limit, offset)

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*AuditLogResponse{}
	for rows.Next() {
		log := &AuditLogResponse{}
		var (
			userID          sql.NullString
			apiClientID     sql.NullString
			actorIdentifier sql.NullString
			actorIP         sql.NullString
			resourceType    sql.NullString
			resourceID      sql.NullString
			responseTimeMs  sql.NullInt32
			errorCode       sql.NullString
			errorMessage    sql.NullString
		)

		err := rows.Scan(
			&log.ID,
			&log.EventID,
			&log.ActorType,
			&userID,
			&apiClientID,
			&actorIdentifier,
			&actorIP,
			&log.Action,
			&resourceType,
			&resourceID,
			&log.BusinessPurpose,
			&log.Status,
			&log.RecordsAffected,
			&responseTimeMs,
			&errorCode,
			&errorMessage,
			&log.OccurredAt,
			&log.RecordedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Handle nullable fields
		if userID.Valid {
			log.UserID = &userID.String
		}
		if apiClientID.Valid {
			log.APIClientID = &apiClientID.String
		}
		if actorIdentifier.Valid {
			log.ActorIdentifier = &actorIdentifier.String
		}
		if actorIP.Valid {
			log.ActorIP = &actorIP.String
		}
		if resourceType.Valid {
			log.ResourceType = &resourceType.String
		}
		if resourceID.Valid {
			log.ResourceID = &resourceID.String
		}
		if responseTimeMs.Valid {
			val := int(responseTimeMs.Int32)
			log.ResponseTimeMs = &val
		}
		if errorCode.Valid {
			log.ErrorCode = &errorCode.String
		}
		if errorMessage.Valid {
			log.ErrorMessage = &errorMessage.String
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filters.Limit)))

	return &AuditLogsListResponse{
		Logs:       logs,
		Total:      total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

// ExportAuditLogsToCSV exports audit logs to CSV format
func (s *AuditService) ExportAuditLogsToCSV(filters *AuditLogFilters, writer *csv.Writer) error {
	// Remove pagination for export
	filters.Limit = 0
	filters.Page = 0

	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argCounter := 1

	if filters.UserID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argCounter))
		args = append(args, filters.UserID)
		argCounter++
	}
	if filters.ActorType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_type = $%d", argCounter))
		args = append(args, filters.ActorType)
		argCounter++
	}
	if filters.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action ILIKE $%d", argCounter))
		args = append(args, "%"+filters.Action+"%")
		argCounter++
	}
	if filters.ResourceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource_type = $%d", argCounter))
		args = append(args, filters.ResourceType)
		argCounter++
	}
	if filters.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argCounter))
		args = append(args, filters.Status)
		argCounter++
	}
	if filters.StartDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("occurred_at >= $%d", argCounter))
		args = append(args, filters.StartDate)
		argCounter++
	}
	if filters.EndDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("occurred_at <= $%d", argCounter))
		args = append(args, filters.EndDate+" 23:59:59")
		argCounter++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Set default sorting
	if filters.SortBy == "" {
		filters.SortBy = "occurred_at"
	}
	if filters.SortOrder == "" {
		filters.SortOrder = "desc"
	}

	query := fmt.Sprintf(`
		SELECT
			id, event_id, actor_type, user_id, api_client_id,
			actor_identifier, actor_ip, action, resource_type,
			resource_id, business_purpose, status, records_affected,
			response_time_ms, error_code, error_message,
			occurred_at, recorded_at
		FROM audit_logs
		%s
		ORDER BY %s %s
	`, whereClause, filters.SortBy, filters.SortOrder)

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query audit logs for export: %w", err)
	}
	defer rows.Close()

	// Write CSV header
	header := []string{
		"ID", "Event ID", "Actor Type", "User ID", "API Client ID",
		"Actor Identifier", "Actor IP", "Action", "Resource Type",
		"Resource ID", "Business Purpose", "Status", "Records Affected",
		"Response Time (ms)", "Error Code", "Error Message",
		"Occurred At", "Recorded At",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for rows.Next() {
		var (
			id              string
			eventID         string
			actorType       string
			userID          sql.NullString
			apiClientID     sql.NullString
			actorIdentifier sql.NullString
			actorIP         sql.NullString
			action          string
			resourceType    sql.NullString
			resourceID      sql.NullString
			businessPurpose string
			status          string
			recordsAffected int
			responseTimeMs  sql.NullInt32
			errorCode       sql.NullString
			errorMessage    sql.NullString
			occurredAt      time.Time
			recordedAt      time.Time
		)

		err := rows.Scan(
			&id, &eventID, &actorType, &userID, &apiClientID,
			&actorIdentifier, &actorIP, &action, &resourceType,
			&resourceID, &businessPurpose, &status, &recordsAffected,
			&responseTimeMs, &errorCode, &errorMessage,
			&occurredAt, &recordedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan row for export: %w", err)
		}

		row := []string{
			id,
			eventID,
			actorType,
			nullStringToString(userID),
			nullStringToString(apiClientID),
			nullStringToString(actorIdentifier),
			nullStringToString(actorIP),
			action,
			nullStringToString(resourceType),
			nullStringToString(resourceID),
			businessPurpose,
			status,
			fmt.Sprintf("%d", recordsAffected),
			nullInt32ToString(responseTimeMs),
			nullStringToString(errorCode),
			nullStringToString(errorMessage),
			occurredAt.Format(time.RFC3339),
			recordedAt.Format(time.RFC3339),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows for export: %w", err)
	}

	writer.Flush()
	return writer.Error()
}

// Helper functions
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt32ToString(ni sql.NullInt32) string {
	if ni.Valid {
		return fmt.Sprintf("%d", ni.Int32)
	}
	return ""
}

// GetAuditLogByID retrieves a specific audit log by ID (NEW)
func (s *AuditService) GetAuditLogByID(id string) (*AuditLogResponse, error) {
	query := `
		SELECT id, user_id, client_id, actor_type, action, resource_type, resource_id,
		       changes, metadata, ip_address, user_agent, session_id, trace_id,
		       business_purpose, status, records_affected, response_time_ms,
		       error_code, error_message, occurred_at, recorded_at
		FROM audit_logs
		WHERE id = $1
	`

	row := s.db.QueryRow(query, id)

	var (
		userID, clientID, resourceID, sessionID, traceID sql.NullString
		changes, metadata                                 sql.NullString
		ipAddress, userAgent                              sql.NullString
		businessPurpose                                   sql.NullString
		recordsAffected                                   int
		responseTimeMs                                    sql.NullInt32
		errorCode, errorMessage                           sql.NullString
		occurredAt, recordedAt                            time.Time
		logID, actorType, action, resourceType, status    string
	)

	err := row.Scan(
		&logID, &userID, &clientID, &actorType, &action, &resourceType, &resourceID,
		&changes, &metadata, &ipAddress, &userAgent, &sessionID, &traceID,
		&businessPurpose, &status, &recordsAffected, &responseTimeMs,
		&errorCode, &errorMessage, &occurredAt, &recordedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch audit log: %w", err)
	}

	var responseTimeMsInt *int
	if responseTimeMs.Valid {
		val := int(responseTimeMs.Int32)
		responseTimeMsInt = &val
	}

	return &AuditLogResponse{
		ID:               logID,
		EventID:          logID, // Using ID as EventID
		UserID:           nullStringPointer(userID),
		APIClientID:      nullStringPointer(clientID),
		ActorIdentifier:  nullStringPointer(userID),
		ActorIP:          nullStringPointer(ipAddress),
		ActorType:        actorType,
		Action:           action,
		ResourceType:     &resourceType,
		ResourceID:       nullStringPointer(resourceID),
		BusinessPurpose:  nullStringToString(businessPurpose),
		Status:           status,
		RecordsAffected:  recordsAffected,
		ResponseTimeMs:   responseTimeMsInt,
		ErrorCode:        nullStringPointer(errorCode),
		ErrorMessage:     nullStringPointer(errorMessage),
		OccurredAt:       occurredAt,
		RecordedAt:       recordedAt,
	}, nil
}

func nullStringPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullInt32Pointer(ni sql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}
