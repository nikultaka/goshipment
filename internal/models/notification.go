package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID                        string          `json:"id" db:"id"`
	TemplateID                string          `json:"template_id" db:"template_id"` // welcome_email, password_reset
	Name                      string          `json:"name" db:"name"`
	Description               sql.NullString  `json:"description,omitempty" db:"description"`
	Channel                   string          `json:"channel" db:"channel"` // email, sms, push, webhook
	Subject                   sql.NullString  `json:"subject,omitempty" db:"subject"` // For email
	BodyTemplate              string          `json:"body_template" db:"body_template"` // Template with {{variables}}
	HTMLTemplate              sql.NullString  `json:"html_template,omitempty" db:"html_template"`
	RequiredVariables         json.RawMessage `json:"required_variables" db:"required_variables"` // JSONB array
	OptionalVariables         json.RawMessage `json:"optional_variables" db:"optional_variables"` // JSONB array
	Locale                    string          `json:"locale" db:"locale"`
	IsActive                  bool            `json:"is_active" db:"is_active"`
	IsTransactional           bool            `json:"is_transactional" db:"is_transactional"`
	AutoSend                  bool            `json:"auto_send" db:"auto_send"`
	MaxSendsPerUserPerHour    sql.NullInt64   `json:"max_sends_per_user_per_hour,omitempty" db:"max_sends_per_user_per_hour"`
	MaxSendsPerUserPerDay     sql.NullInt64   `json:"max_sends_per_user_per_day,omitempty" db:"max_sends_per_user_per_day"`
	Priority                  string          `json:"priority" db:"priority"` // low, normal, high, urgent
	RetryAttempts             int             `json:"retry_attempts" db:"retry_attempts"`
	RetryDelaySeconds         int             `json:"retry_delay_seconds" db:"retry_delay_seconds"`
	TrackOpens                bool            `json:"track_opens" db:"track_opens"`
	TrackClicks               bool            `json:"track_clicks" db:"track_clicks"`
	SendGridTemplateID        sql.NullString  `json:"sendgrid_template_id,omitempty" db:"sendgrid_template_id"`
	TwilioMessagingServiceSID sql.NullString  `json:"twilio_messaging_service_sid,omitempty" db:"twilio_messaging_service_sid"`
	Version                   int             `json:"version" db:"version"`
	PreviousVersionID         sql.NullString  `json:"previous_version_id,omitempty" db:"previous_version_id"`
	CreatedAt                 time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy                 sql.NullString  `json:"created_by,omitempty" db:"created_by"`
	LastModifiedBy            sql.NullString  `json:"last_modified_by,omitempty" db:"last_modified_by"`
}

// NotificationQueue represents a queued notification (for NATS processing)
type NotificationQueue struct {
	ID                   string          `json:"id" db:"id"`
	QueueID              string          `json:"queue_id" db:"queue_id"` // nq_123abc
	TemplateID           string          `json:"template_id" db:"template_id"`
	UserID               sql.NullString  `json:"user_id,omitempty" db:"user_id"`
	RecipientEmail       sql.NullString  `json:"recipient_email,omitempty" db:"recipient_email"`
	RecipientPhone       sql.NullString  `json:"recipient_phone,omitempty" db:"recipient_phone"`
	RecipientName        sql.NullString  `json:"recipient_name,omitempty" db:"recipient_name"`
	Subject              sql.NullString  `json:"subject,omitempty" db:"subject"`
	Body                 string          `json:"body" db:"body"`
	HTMLBody             sql.NullString  `json:"html_body,omitempty" db:"html_body"`
	TemplateVariables    json.RawMessage `json:"template_variables" db:"template_variables"` // JSONB
	Channel              string          `json:"channel" db:"channel"`
	Priority             string          `json:"priority" db:"priority"`
	ScheduledAt          time.Time       `json:"scheduled_at" db:"scheduled_at"`
	SendAfter            time.Time       `json:"send_after" db:"send_after"`
	ExpiresAt            sql.NullTime    `json:"expires_at,omitempty" db:"expires_at"`
	Status               string          `json:"status" db:"status"` // queued, processing, sent, failed, expired, cancelled
	Attempts             int             `json:"attempts" db:"attempts"`
	MaxAttempts          int             `json:"max_attempts" db:"max_attempts"`
	LastAttemptAt        sql.NullTime    `json:"last_attempt_at,omitempty" db:"last_attempt_at"`
	NextRetryAt          sql.NullTime    `json:"next_retry_at,omitempty" db:"next_retry_at"`
	ProviderUsed         sql.NullString  `json:"provider_used,omitempty" db:"provider_used"`
	ProviderMessageID    sql.NullString  `json:"provider_message_id,omitempty" db:"provider_message_id"`
	ProviderResponse     json.RawMessage `json:"provider_response" db:"provider_response"` // JSONB
	SentAt               sql.NullTime    `json:"sent_at,omitempty" db:"sent_at"`
	DeliveredAt          sql.NullTime    `json:"delivered_at,omitempty" db:"delivered_at"`
	FailedAt             sql.NullTime    `json:"failed_at,omitempty" db:"failed_at"`
	ErrorCode            sql.NullString  `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage         sql.NullString  `json:"error_message,omitempty" db:"error_message"`
	TriggeredByEvent     sql.NullString  `json:"triggered_by_event,omitempty" db:"triggered_by_event"`
	RelatedResourceType  sql.NullString  `json:"related_resource_type,omitempty" db:"related_resource_type"`
	RelatedResourceID    sql.NullString  `json:"related_resource_id,omitempty" db:"related_resource_id"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// NotificationDeliveryEvent represents tracking events for notifications
type NotificationDeliveryEvent struct {
	ID                string         `json:"id" db:"id"`
	NotificationID    string         `json:"notification_id" db:"notification_id"`
	EventType         string         `json:"event_type" db:"event_type"` // sent, delivered, opened, clicked, bounced, unsubscribed
	EventTimestamp    time.Time      `json:"event_timestamp" db:"event_timestamp"`
	ProviderEventID   sql.NullString `json:"provider_event_id,omitempty" db:"provider_event_id"`
	ProviderEventData json.RawMessage `json:"provider_event_data" db:"provider_event_data"` // JSONB
	EmailSubject      sql.NullString `json:"email_subject,omitempty" db:"email_subject"`
	EmailFrom         sql.NullString `json:"email_from,omitempty" db:"email_from"`
	EmailTo           sql.NullString `json:"email_to,omitempty" db:"email_to"`
	ClickedURL        sql.NullString `json:"clicked_url,omitempty" db:"clicked_url"`
	IPAddress         sql.NullString `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         sql.NullString `json:"user_agent,omitempty" db:"user_agent"`
	Location          sql.NullString `json:"location,omitempty" db:"location"`
	SMSTo             sql.NullString `json:"sms_to,omitempty" db:"sms_to"`
	SMSFrom           sql.NullString `json:"sms_from,omitempty" db:"sms_from"`
	SMSSegments       sql.NullInt64  `json:"sms_segments,omitempty" db:"sms_segments"`
	BounceType        sql.NullString `json:"bounce_type,omitempty" db:"bounce_type"` // hard, soft
	BounceReason      sql.NullString `json:"bounce_reason,omitempty" db:"bounce_reason"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
}

// UserNotificationPreference represents user notification preferences
type UserNotificationPreference struct {
	ID                       string         `json:"id" db:"id"`
	UserID                   string         `json:"user_id" db:"user_id"`
	EmailEnabled             bool           `json:"email_enabled" db:"email_enabled"`
	SMSEnabled               bool           `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled              bool           `json:"push_enabled" db:"push_enabled"`
	MarketingEmails          bool           `json:"marketing_emails" db:"marketing_emails"`
	PromotionalSMS           bool           `json:"promotional_sms" db:"promotional_sms"`
	SecurityAlerts           bool           `json:"security_alerts" db:"security_alerts"`
	OrderUpdates             bool           `json:"order_updates" db:"order_updates"`
	BillingNotifications     bool           `json:"billing_notifications" db:"billing_notifications"`
	SystemAnnouncements      bool           `json:"system_announcements" db:"system_announcements"`
	DigestFrequency          string         `json:"digest_frequency" db:"digest_frequency"` // immediate, daily, weekly, never
	QuietHoursEnabled        bool           `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
	QuietHoursStart          sql.NullString `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"` // TIME
	QuietHoursEnd            sql.NullString `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"` // TIME
	QuietHoursTimezone       sql.NullString `json:"quiet_hours_timezone,omitempty" db:"quiet_hours_timezone"`
	EmailUnsubscribeToken    sql.NullString `json:"email_unsubscribe_token,omitempty" db:"email_unsubscribe_token"`
	SMSUnsubscribeToken      sql.NullString `json:"sms_unsubscribe_token,omitempty" db:"sms_unsubscribe_token"`
	PreferencesUpdatedAt     time.Time      `json:"preferences_updated_at" db:"preferences_updated_at"`
	UnsubscribedFromAllAt    sql.NullTime   `json:"unsubscribed_from_all_at,omitempty" db:"unsubscribed_from_all_at"`
	CreatedAt                time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at" db:"updated_at"`
}

// UserNotification represents an in-app notification for users
type UserNotification struct {
	ID                  string         `json:"id" db:"id"`
	UserID              string         `json:"user_id" db:"user_id"`
	Type                string         `json:"type" db:"type"` // shipment_created, funds_added, pickup_scheduled
	Title               string         `json:"title" db:"title"`
	Message             string         `json:"message" db:"message"`
	RelatedResourceType sql.NullString `json:"related_resource_type,omitempty" db:"related_resource_type"`
	RelatedResourceID   sql.NullString `json:"related_resource_id,omitempty" db:"related_resource_id"`
	IsRead              bool           `json:"is_read" db:"is_read"`
	Metadata            json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
}

// Notification represents a notification record (old model - kept for backward compatibility if needed)
type Notification struct {
	ID          string         `json:"id" db:"id"`
	UserID      string         `json:"user_id" db:"user_id"`
	Type        string         `json:"type" db:"type"`
	Channel     string         `json:"channel" db:"channel"` // email, sms, push
	Title       string         `json:"title" db:"title"`
	Content     string         `json:"content" db:"content"`
	IsRead      bool           `json:"is_read" db:"is_read"`
	ReadAt      sql.NullTime   `json:"read_at,omitempty" db:"read_at"`
	Status      string         `json:"status" db:"status"` // pending, sent, failed
	Recipient   sql.NullString `json:"recipient,omitempty" db:"recipient"`
	Subject     sql.NullString `json:"subject,omitempty" db:"subject"`
	Provider    sql.NullString `json:"provider,omitempty" db:"provider"`
	ProviderID  sql.NullString `json:"provider_id,omitempty" db:"provider_id"`
	ErrorMsg    sql.NullString `json:"error_msg,omitempty" db:"error_msg"`
	Attempts    int            `json:"attempts" db:"attempts"`
	SentAt      sql.NullTime   `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}
