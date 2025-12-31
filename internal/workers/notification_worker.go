package workers

import (
	"encoding/json"
	"fmt"
	"log"

	"gogin/internal/clients"
	"gogin/internal/config"
	"gogin/internal/modules/notifications"
	"gogin/internal/modules/sendgrid"
	"gogin/internal/modules/twilio"

	"github.com/nats-io/nats.go"
)

// NotificationWorker processes notification delivery jobs
type NotificationWorker struct {
	db       *clients.Database
	nats     *clients.NATSClient
	sendgrid *sendgrid.SendGridClient
	twilio   *twilio.TwilioClient
	config   *config.Config
}

// NewNotificationWorker creates a new notification worker
func NewNotificationWorker(db *clients.Database, nats *clients.NATSClient, cfg *config.Config) *NotificationWorker {
	return &NotificationWorker{
		db:       db,
		nats:     nats,
		sendgrid: sendgrid.NewSendGridClient(cfg.SMTP),
		twilio:   twilio.NewTwilioClient(cfg.Twilio),
		config:   cfg,
	}
}

// Start starts the notification worker
func (w *NotificationWorker) Start() error {
	log.Println("📬 Starting notification worker...")

	// Use core NATS subscription (not JetStream) for simplicity
	conn := w.nats.GetConnection()
	subject := "NOTIFICATIONS.notification.send"

	_, err := conn.Subscribe(subject, w.handleNotificationSend)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	log.Println("✓ Notification worker started successfully")
	log.Printf("   Listening for notifications on: %s", subject)
	return nil
}

// handleNotificationSend handles notification send messages
func (w *NotificationWorker) handleNotificationSend(msg *nats.Msg) {
	var req notifications.SendNotificationRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal notification: %v", err)
		msg.Nak()
		return
	}

	log.Printf("Processing notification: %s to %s via %s", req.Type, req.UserID, req.Channel)

	var err error
	switch req.Channel {
	case "email":
		err = w.sendEmail(&req)
	case "sms":
		err = w.sendSMS(&req)
	case "push":
		err = w.sendPushNotification(&req)
	default:
		log.Printf("Unknown notification channel: %s", req.Channel)
		msg.Nak()
		return
	}

	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		// Update status to failed
		w.updateNotificationStatus(req.ID, "failed", err.Error())
		msg.Nak()
		return
	}

	// Update status to sent
	w.updateNotificationStatus(req.ID, "sent", "")
	msg.Ack()
	log.Printf("✓ Notification sent successfully")
}

// sendEmail sends an email notification
func (w *NotificationWorker) sendEmail(req *notifications.SendNotificationRequest) error {
	// Get user email from user_emails table (primary email first, fallback to any active verified email)
	var email string
	query := `
		SELECT email
		FROM user_emails
		WHERE user_id = $1
		  AND is_active = true
		  AND is_verified = true
		ORDER BY is_primary DESC, created_at ASC
		LIMIT 1
	`
	err := w.db.QueryRow(query, req.UserID).Scan(&email)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	msg := &sendgrid.EmailMessage{
		To:          []string{email},
		Subject:     req.Title,
		TextContent: req.Content,
		HTMLContent: fmt.Sprintf("<h2>%s</h2><p>%s</p>", req.Title, req.Content),
	}

	return w.sendgrid.SendEmail(msg)
}

// sendSMS sends an SMS notification
func (w *NotificationWorker) sendSMS(req *notifications.SendNotificationRequest) error {
	// Get user phone from database
	var phone string
	err := w.db.QueryRow("SELECT phone FROM users WHERE id = $1", req.UserID).Scan(&phone)
	if err != nil {
		return fmt.Errorf("failed to get user phone: %w", err)
	}

	if phone == "" {
		return fmt.Errorf("user has no phone number")
	}

	msg := &twilio.SMSMessage{
		To:   phone,
		Body: fmt.Sprintf("%s: %s", req.Title, req.Content),
	}

	return w.twilio.SendSMS(msg)
}

// sendPushNotification sends a push notification (placeholder)
func (w *NotificationWorker) sendPushNotification(req *notifications.SendNotificationRequest) error {
	// Implement push notification logic here (FCM, APNs, etc.)
	log.Printf("Push notification: %s", req.Title)
	return nil
}

// updateNotificationStatus updates notification status in database
func (w *NotificationWorker) updateNotificationStatus(notificationID, status, errorMsg string) {
	var query string

	if status == "sent" {
		// Update with sent_at timestamp when successful
		query = `
			UPDATE notifications
			SET status = $1, error_message = $2, sent_at = NOW(), updated_at = NOW()
			WHERE id = $3
		`
	} else if status == "failed" {
		// Update with failed_at timestamp when failed
		query = `
			UPDATE notifications
			SET status = $1, error_message = $2, failed_at = NOW(), updated_at = NOW()
			WHERE id = $3
		`
	} else {
		// Generic update for other statuses
		query = `
			UPDATE notifications
			SET status = $1, error_message = $2, updated_at = NOW()
			WHERE id = $3
		`
	}

	_, err := w.db.Exec(query, status, errorMsg, notificationID)
	if err != nil {
		log.Printf("Failed to update notification status: %v", err)
	}
}
