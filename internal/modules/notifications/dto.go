package notifications

import "time"

// NotificationResponse represents a notification response
type NotificationResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationsListResponse represents a paginated list of notifications
type NotificationsListResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	Total         int                     `json:"total"`
	Unread        int                     `json:"unread"`
	Page          int                     `json:"page"`
	Limit         int                     `json:"limit"`
	TotalPages    int                     `json:"total_pages"`
}

// TestEmailRequest represents a test email request
type TestEmailRequest struct {
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
}

// TestSMSRequest represents a test SMS request
type TestSMSRequest struct {
	To   string `json:"to" binding:"required"`
	Body string `json:"body" binding:"required"`
}

// SendNotificationRequest represents a notification send request
type SendNotificationRequest struct {
	ID      string `json:"id,omitempty"` // Notification ID (populated by service layer)
	UserID  string `json:"user_id" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Channel string `json:"channel" binding:"required,oneof=email sms push"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// Notification Template DTOs

// NotificationTemplateResponse represents a notification template
type NotificationTemplateResponse struct {
	ID                string   `json:"id"`
	TemplateID        string   `json:"template_id"`
	Name              string   `json:"name"`
	Description       *string  `json:"description,omitempty"`
	Channel           string   `json:"channel"`
	Subject           *string  `json:"subject,omitempty"`
	BodyTemplate      string   `json:"body_template"`
	HTMLTemplate      *string  `json:"html_template,omitempty"`
	RequiredVariables []string `json:"required_variables"`
	OptionalVariables []string `json:"optional_variables"`
	Locale            string   `json:"locale"`
	IsActive          bool     `json:"is_active"`
	IsTransactional   bool     `json:"is_transactional"`
	Priority          string   `json:"priority"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NotificationTemplatesListResponse represents a list of notification templates
type NotificationTemplatesListResponse struct {
	Templates  []*NotificationTemplateResponse `json:"templates"`
	Total      int                             `json:"total"`
	Page       int                             `json:"page"`
	Limit      int                             `json:"limit"`
	TotalPages int                             `json:"total_pages"`
}

// CreateNotificationTemplateRequest represents a request to create a notification template
type CreateNotificationTemplateRequest struct {
	TemplateID        string   `json:"template_id" binding:"required"`
	Name              string   `json:"name" binding:"required"`
	Description       string   `json:"description"`
	Channel           string   `json:"channel" binding:"required,oneof=email sms push webhook"`
	Subject           string   `json:"subject"`
	BodyTemplate      string   `json:"body_template" binding:"required"`
	HTMLTemplate      string   `json:"html_template"`
	RequiredVariables []string `json:"required_variables"`
	OptionalVariables []string `json:"optional_variables"`
	Locale            string   `json:"locale"`
	IsActive          *bool    `json:"is_active"`
	IsTransactional   *bool    `json:"is_transactional"`
	Priority          string   `json:"priority"`
}

// UpdateNotificationTemplateRequest represents a request to update a notification template
type UpdateNotificationTemplateRequest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Subject           string   `json:"subject"`
	BodyTemplate      string   `json:"body_template"`
	HTMLTemplate      string   `json:"html_template"`
	RequiredVariables []string `json:"required_variables"`
	OptionalVariables []string `json:"optional_variables"`
	IsActive          *bool    `json:"is_active"`
	Priority          string   `json:"priority"`
}
