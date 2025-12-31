package sendgrid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gogin/internal/config"
)

// SendGridClient wraps SendGrid API
type SendGridClient struct {
	apiKey      string
	fromEmail   string
	fromName    string
	replyToEmail string
}

// NewSendGridClient creates a new SendGrid client
func NewSendGridClient(cfg config.SMTPConfig) *SendGridClient {
	return &SendGridClient{
		apiKey:       cfg.APIKey,
		fromEmail:    cfg.FromEmail,
		fromName:     cfg.FromName,
		replyToEmail: cfg.ReplyToEmail,
	}
}

// EmailMessage represents an email message
type EmailMessage struct {
	To          []string
	Subject     string
	TextContent string
	HTMLContent string
	ReplyTo     string
}

// SendEmail sends an email via SendGrid
func (c *SendGridClient) SendEmail(msg *EmailMessage) error {
	if c.apiKey == "" {
		return fmt.Errorf("SendGrid API key not configured")
	}

	personalizations := make([]map[string]interface{}, 0)
	toList := make([]map[string]string, 0)
	for _, email := range msg.To {
		toList = append(toList, map[string]string{"email": email})
	}
	personalizations = append(personalizations, map[string]interface{}{
		"to":      toList,
		"subject": msg.Subject,
	})

	content := make([]map[string]string, 0)
	if msg.TextContent != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": msg.TextContent,
		})
	}
	if msg.HTMLContent != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": msg.HTMLContent,
		})
	}

	payload := map[string]interface{}{
		"personalizations": personalizations,
		"from": map[string]string{
			"email": c.fromEmail,
			"name":  c.fromName,
		},
		"content": content,
	}

	if msg.ReplyTo != "" {
		payload["reply_to"] = map[string]string{"email": msg.ReplyTo}
	} else if c.replyToEmail != "" {
		payload["reply_to"] = map[string]string{"email": c.replyToEmail}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SendGrid API error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
