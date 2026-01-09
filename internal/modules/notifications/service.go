package notifications

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"

	"github.com/google/uuid"
)

// NotificationsService handles notifications business logic
type NotificationsService struct {
	db       *clients.Database
	nats     *clients.NATSClient
}

// NewNotificationsService creates a new notifications service
func NewNotificationsService(db *clients.Database, nats *clients.NATSClient) *NotificationsService {
	return &NotificationsService{
		db:       db,
		nats:     nats,
	}
}

// SendNotification creates and queues a notification
func (s *NotificationsService) SendNotification(req *SendNotificationRequest) (*NotificationResponse, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO user_notifications (id, user_id, type, title, message, is_read)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	var createdAt time.Time
	err := s.db.QueryRow(query,
		id,
		req.UserID,
		req.Type,
		req.Title,
		req.Content,
		false,
	).Scan(&createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Queue for async delivery with notification ID
	req.ID = id // Add notification ID to the request
	notifData, _ := json.Marshal(req)
	go s.nats.Publish("notification.send", notifData)

	return &NotificationResponse{
		ID:        id,
		UserID:    req.UserID,
		Type:      req.Type,
		Channel:   req.Channel,
		Title:     req.Title,
		Content:   req.Content,
		IsRead:    false,
		Status:    "pending",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

// ListNotifications lists user notifications
func (s *NotificationsService) ListNotifications(userID string, page, limit int) ([]*NotificationResponse, int, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var total, unread int
	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_read = FALSE THEN 1 ELSE 0 END), 0)
		FROM user_notifications
		WHERE user_id = $1
	`, userID).Scan(&total, &unread)
	if err != nil {
		return nil, 0, 0, err
	}

	// Get notifications
	query := `
		SELECT id, user_id, type, title, message, is_read, created_at
		FROM user_notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var notifications []*NotificationResponse
	for rows.Next() {
		var notif models.UserNotification
		err := rows.Scan(
			&notif.ID,
			&notif.UserID,
			&notif.Type,
			&notif.Title,
			&notif.Message,
			&notif.IsRead,
			&notif.CreatedAt,
		)
		if err != nil {
			return nil, 0, 0, err
		}
		notifications = append(notifications, s.toUserNotificationResponse(&notif))
	}

	return notifications, total, unread, nil
}

// GetNotification retrieves a notification by ID
func (s *NotificationsService) GetNotification(id, userID string) (*NotificationResponse, error) {
	var notif models.UserNotification
	query := `
		SELECT id, user_id, type, title, message, is_read, created_at
		FROM user_notifications
		WHERE id = $1 AND user_id = $2
	`

	err := s.db.QueryRow(query, id, userID).Scan(
		&notif.ID,
		&notif.UserID,
		&notif.Type,
		&notif.Title,
		&notif.Message,
		&notif.IsRead,
		&notif.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return s.toUserNotificationResponse(&notif), nil
}

// MarkAsRead marks a notification as read
func (s *NotificationsService) MarkAsRead(id, userID string) error {
	query := `UPDATE user_notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`
	result, err := s.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationsService) MarkAllAsRead(userID string) (int64, error) {
	query := `UPDATE user_notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`
	result, err := s.db.Exec(query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark notifications as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected, nil
}

// DeleteNotification deletes a notification
func (s *NotificationsService) DeleteNotification(id, userID string) error {
	query := `DELETE FROM user_notifications WHERE id = $1 AND user_id = $2`
	result, err := s.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}


// Notification Template Management

// ListNotificationTemplates lists all notification templates with pagination
func (s *NotificationsService) ListNotificationTemplates(page, limit int) (*NotificationTemplatesListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM notification_templates").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// Get templates
	query := `
		SELECT
			id, template_id, name, description, channel, subject,
			body_template, html_template, required_variables, optional_variables,
			locale, is_active, is_transactional, priority, created_at, updated_at
		FROM notification_templates
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	templates := []*NotificationTemplateResponse{}
	for rows.Next() {
		template := &NotificationTemplateResponse{}
		var description, subject, htmlTemplate []byte
		var requiredVars, optionalVars []byte

		err := rows.Scan(
			&template.ID,
			&template.TemplateID,
			&template.Name,
			&description,
			&template.Channel,
			&subject,
			&template.BodyTemplate,
			&htmlTemplate,
			&requiredVars,
			&optionalVars,
			&template.Locale,
			&template.IsActive,
			&template.IsTransactional,
			&template.Priority,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		// Handle nullable fields
		if description != nil && len(description) > 0 && string(description) != "null" {
			desc := string(description)
			template.Description = &desc
		}
		if subject != nil && len(subject) > 0 && string(subject) != "null" {
			subj := string(subject)
			template.Subject = &subj
		}
		if htmlTemplate != nil && len(htmlTemplate) > 0 && string(htmlTemplate) != "null" {
			html := string(htmlTemplate)
			template.HTMLTemplate = &html
		}

		// Parse JSON arrays
		if requiredVars != nil && len(requiredVars) > 0 {
			json.Unmarshal(requiredVars, &template.RequiredVariables)
		}
		if optionalVars != nil && len(optionalVars) > 0 {
			json.Unmarshal(optionalVars, &template.OptionalVariables)
		}

		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating templates: %w", err)
	}

	totalPages := (total + limit - 1) / limit

	return &NotificationTemplatesListResponse{
		Templates:  templates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// CreateNotificationTemplate creates a new notification template
func (s *NotificationsService) CreateNotificationTemplate(req *CreateNotificationTemplateRequest, userID string) (*NotificationTemplateResponse, error) {
	// Set defaults
	if req.Locale == "" {
		req.Locale = "en_CA"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isTransactional := true
	if req.IsTransactional != nil {
		isTransactional = *req.IsTransactional
	}

	// Marshal JSON arrays
	requiredVarsJSON, _ := json.Marshal(req.RequiredVariables)
	optionalVarsJSON, _ := json.Marshal(req.OptionalVariables)

	id := uuid.New()
	query := `
		INSERT INTO notification_templates (
			id, template_id, name, description, channel, subject,
			body_template, html_template, required_variables, optional_variables,
			locale, is_active, is_transactional, priority, created_by, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(query,
		id,
		req.TemplateID,
		req.Name,
		nullStringOrNil(req.Description),
		req.Channel,
		nullStringOrNil(req.Subject),
		req.BodyTemplate,
		nullStringOrNil(req.HTMLTemplate),
		requiredVarsJSON,
		optionalVarsJSON,
		req.Locale,
		isActive,
		isTransactional,
		req.Priority,
		uuid.MustParse(userID),
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	template := &NotificationTemplateResponse{
		ID:                id.String(),
		TemplateID:        req.TemplateID,
		Name:              req.Name,
		Channel:           req.Channel,
		BodyTemplate:      req.BodyTemplate,
		RequiredVariables: req.RequiredVariables,
		OptionalVariables: req.OptionalVariables,
		Locale:            req.Locale,
		IsActive:          isActive,
		IsTransactional:   isTransactional,
		Priority:          req.Priority,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	if req.Description != "" {
		template.Description = &req.Description
	}
	if req.Subject != "" {
		template.Subject = &req.Subject
	}
	if req.HTMLTemplate != "" {
		template.HTMLTemplate = &req.HTMLTemplate
	}

	return template, nil
}

// UpdateNotificationTemplate updates an existing notification template
func (s *NotificationsService) UpdateNotificationTemplate(id string, req *UpdateNotificationTemplateRequest, userID string) (*NotificationTemplateResponse, error) {
	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argCounter := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argCounter))
		args = append(args, req.Name)
		argCounter++
	}
	if req.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argCounter))
		args = append(args, req.Description)
		argCounter++
	}
	if req.Subject != "" {
		updates = append(updates, fmt.Sprintf("subject = $%d", argCounter))
		args = append(args, req.Subject)
		argCounter++
	}
	if req.BodyTemplate != "" {
		updates = append(updates, fmt.Sprintf("body_template = $%d", argCounter))
		args = append(args, req.BodyTemplate)
		argCounter++
	}
	if req.HTMLTemplate != "" {
		updates = append(updates, fmt.Sprintf("html_template = $%d", argCounter))
		args = append(args, req.HTMLTemplate)
		argCounter++
	}
	if req.RequiredVariables != nil {
		requiredVarsJSON, _ := json.Marshal(req.RequiredVariables)
		updates = append(updates, fmt.Sprintf("required_variables = $%d", argCounter))
		args = append(args, requiredVarsJSON)
		argCounter++
	}
	if req.OptionalVariables != nil {
		optionalVarsJSON, _ := json.Marshal(req.OptionalVariables)
		updates = append(updates, fmt.Sprintf("optional_variables = $%d", argCounter))
		args = append(args, optionalVarsJSON)
		argCounter++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argCounter))
		args = append(args, *req.IsActive)
		argCounter++
	}
	if req.Priority != "" {
		updates = append(updates, fmt.Sprintf("priority = $%d", argCounter))
		args = append(args, req.Priority)
		argCounter++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add last_modified_by and updated_at
	updates = append(updates, fmt.Sprintf("last_modified_by = $%d", argCounter))
	args = append(args, uuid.MustParse(userID))
	argCounter++

	updates = append(updates, "updated_at = NOW()")

	// Add id to args
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE notification_templates
		SET %s
		WHERE id = $%d
		RETURNING
			id, template_id, name, description, channel, subject,
			body_template, html_template, required_variables, optional_variables,
			locale, is_active, is_transactional, priority, created_at, updated_at
	`, strings.Join(updates, ", "), argCounter)

	template := &NotificationTemplateResponse{}
	var description, subject, htmlTemplate []byte
	var requiredVars, optionalVars []byte

	err := s.db.QueryRow(query, args...).Scan(
		&template.ID,
		&template.TemplateID,
		&template.Name,
		&description,
		&template.Channel,
		&subject,
		&template.BodyTemplate,
		&htmlTemplate,
		&requiredVars,
		&optionalVars,
		&template.Locale,
		&template.IsActive,
		&template.IsTransactional,
		&template.Priority,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Handle nullable fields
	if description != nil && len(description) > 0 && string(description) != "null" {
		desc := string(description)
		template.Description = &desc
	}
	if subject != nil && len(subject) > 0 && string(subject) != "null" {
		subj := string(subject)
		template.Subject = &subj
	}
	if htmlTemplate != nil && len(htmlTemplate) > 0 && string(htmlTemplate) != "null" {
		html := string(htmlTemplate)
		template.HTMLTemplate = &html
	}

	// Parse JSON arrays
	if requiredVars != nil && len(requiredVars) > 0 {
		json.Unmarshal(requiredVars, &template.RequiredVariables)
	}
	if optionalVars != nil && len(optionalVars) > 0 {
		json.Unmarshal(optionalVars, &template.OptionalVariables)
	}

	return template, nil
}

// Helper functions

func (s *NotificationsService) toUserNotificationResponse(notif *models.UserNotification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:        notif.ID,
		UserID:    notif.UserID,
		Type:      notif.Type,
		Channel:   "",     // Not stored in user_notifications
		Title:     notif.Title,
		Content:   notif.Message,
		IsRead:    notif.IsRead,
		Status:    "",     // Not stored in user_notifications
		CreatedAt: notif.CreatedAt,
		UpdatedAt: notif.CreatedAt, // user_notifications doesn't have updated_at
	}

	return resp
}

// Deprecated: Use toUserNotificationResponse instead
func (s *NotificationsService) toNotificationResponse(notif *models.Notification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:        notif.ID,
		UserID:    notif.UserID,
		Type:      notif.Type,
		Channel:   notif.Channel,
		Title:     notif.Title,
		Content:   notif.Content,
		IsRead:    notif.IsRead,
		Status:    notif.Status,
		CreatedAt: notif.CreatedAt,
		UpdatedAt: notif.UpdatedAt,
	}

	if notif.ReadAt.Valid {
		readAt := notif.ReadAt.Time
		resp.ReadAt = &readAt
	}

	return resp
}

func nullStringOrNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetUserNotifications retrieves notifications for a user with pagination (NEW)
func (s *NotificationsService) GetUserNotifications(userID string, page, limit int, unreadOnly bool) ([]*NotificationResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build where clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	if unreadOnly {
		whereClause += " AND is_read = false"
	}

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM user_notifications ` + whereClause
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Get notifications
	query := `
		SELECT id, user_id, type, title, message, is_read, created_at
		FROM user_notifications
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*NotificationResponse
	for rows.Next() {
		var notif models.UserNotification
		err := rows.Scan(&notif.ID, &notif.UserID, &notif.Type, &notif.Title,
			&notif.Message, &notif.IsRead, &notif.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, s.toUserNotificationResponse(&notif))
	}

	if notifications == nil {
		notifications = []*NotificationResponse{}
	}

	return notifications, total, nil
}

// MarkNotificationAsRead marks a notification as read (NEW)
func (s *NotificationsService) MarkNotificationAsRead(notificationID, userID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// MarkAllNotificationsAsRead marks all notifications as read for a user (NEW)
func (s *NotificationsService) MarkAllNotificationsAsRead(userID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true
		WHERE user_id = $1 AND is_read = false
	`

	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}

// GetNotificationTemplate retrieves a specific notification template by ID (NEW)
func (s *NotificationsService) GetNotificationTemplate(id string) (*NotificationTemplateResponse, error) {
	query := `
		SELECT id, name, description, channel, subject, body_template,
		       required_variables, optional_variables, is_active, created_at, updated_at
		FROM notification_templates
		WHERE id = $1
	`

	row := s.db.QueryRow(query, id)

	var (
		template                       models.NotificationTemplate
		description, subject           sql.NullString
		requiredVars, optionalVars     []byte
	)

	err := row.Scan(
		&template.ID, &template.Name, &description, &template.Channel,
		&subject, &template.BodyTemplate,
		&requiredVars, &optionalVars, &template.IsActive,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	// Convert to response
	resp := &NotificationTemplateResponse{
		ID:                template.ID,
		TemplateID:        template.ID,
		Name:              template.Name,
		Channel:           template.Channel,
		BodyTemplate:      template.BodyTemplate,
		IsActive:          template.IsActive,
		IsTransactional:   false, // default
		Priority:          "normal", // default
		Locale:            "en", // default
		RequiredVariables: []string{},
		OptionalVariables: []string{},
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
	}

	if description.Valid {
		resp.Description = &description.String
	}
	if subject.Valid {
		resp.Subject = &subject.String
	}

	// Parse JSON arrays
	if requiredVars != nil && len(requiredVars) > 0 {
		json.Unmarshal(requiredVars, &resp.RequiredVariables)
	}
	if optionalVars != nil && len(optionalVars) > 0 {
		json.Unmarshal(optionalVars, &resp.OptionalVariables)
	}

	return resp, nil
}
