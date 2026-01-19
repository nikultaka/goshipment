package clients

import (
	"bytes"
	"fmt"
	"html/template"

	"gogin/internal/config"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// EmailClient handles email sending via SendGrid
type EmailClient struct {
	config *config.SendGridConfig
	client *sendgrid.Client
}

// NewEmailClient creates a new email client
func NewEmailClient(cfg *config.SendGridConfig) *EmailClient {
	var client *sendgrid.Client
	if cfg.APIKey != "" {
		client = sendgrid.NewSendClient(cfg.APIKey)
	}

	return &EmailClient{
		config: cfg,
		client: client,
	}
}

// EmailMessage represents an email to be sent
type EmailMessage struct {
	To       []string
	Subject  string
	Body     string
	HTML     string
	ReplyTo  string
}

// SendEmail sends an email via SendGrid
func (e *EmailClient) SendEmail(msg EmailMessage) error {
	if !e.config.Enabled {
		// Email is disabled, just log and return
		fmt.Printf("[EMAIL DISABLED] Would send email to %v: %s\n", msg.To, msg.Subject)
		return nil
	}

	if e.client == nil {
		return fmt.Errorf("SendGrid client not initialized - check SENDGRID_API_KEY")
	}

	// Create SendGrid message
	from := mail.NewEmail(e.config.FromName, e.config.FromEmail)
	to := mail.NewEmail("", msg.To[0])

	// Use HTML if provided, otherwise plain text
	var message *mail.SGMailV3
	if msg.HTML != "" {
		message = mail.NewSingleEmail(from, msg.Subject, to, msg.Body, msg.HTML)
	} else {
		message = mail.NewSingleEmail(from, msg.Subject, to, msg.Body, "")
	}

	// Add reply-to if specified
	replyTo := msg.ReplyTo
	if replyTo == "" {
		replyTo = e.config.ReplyToEmail
	}
	if replyTo != "" {
		message.SetReplyTo(mail.NewEmail("", replyTo))
	}

	// Send email
	response, err := e.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid returned error status %d: %s", response.StatusCode, response.Body)
	}

	fmt.Printf("[EMAIL SENT] Successfully sent email to %v: %s (Status: %d)\n", msg.To, msg.Subject, response.StatusCode)
	return nil
}

// SendInvitationEmail sends an organization invitation email
func (e *EmailClient) SendInvitationEmail(to, inviterName, organizationName, inviteLink string) error {
	subject := fmt.Sprintf("You're invited to join %s on GoShipment", organizationName)

	// HTML email template
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 0;
            background-color: #f4f4f4;
        }
        .email-container {
            background-color: #ffffff;
            margin: 20px auto;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header .emoji {
            font-size: 48px;
            margin-bottom: 10px;
        }
        .content {
            padding: 40px 30px;
        }
        .content p {
            margin: 0 0 15px 0;
            font-size: 16px;
        }
        .highlight {
            background-color: #f8f9fa;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin: 20px 0;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            padding: 14px 40px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white !important;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 16px;
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
            transition: transform 0.2s;
        }
        .button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(102, 126, 234, 0.5);
        }
        .link-box {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 15px;
            margin: 20px 0;
            word-break: break-all;
        }
        .link-box p {
            margin: 5px 0;
            font-size: 14px;
            color: #666;
        }
        .link {
            color: #667eea;
            font-size: 13px;
        }
        .features {
            margin: 25px 0;
        }
        .features ul {
            margin: 10px 0;
            padding-left: 20px;
        }
        .features li {
            margin: 8px 0;
            font-size: 15px;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 30px;
            border-top: 1px solid #e9ecef;
            text-align: center;
        }
        .footer p {
            margin: 5px 0;
            color: #6c757d;
            font-size: 13px;
        }
        .note {
            background-color: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 12px;
            margin: 20px 0;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <div class="emoji">🎉</div>
            <h1>You're Invited!</h1>
        </div>

        <div class="content">
            <p>Hello,</p>

            <div class="highlight">
                <p><strong>{{.InviterName}}</strong> has invited you to join <strong>{{.OrganizationName}}</strong> on GoShipment.</p>
            </div>

            <div class="features">
                <p>GoShipment is a comprehensive shipping and logistics platform that helps teams:</p>
                <ul>
                    <li>📦 Get instant shipping quotes from multiple carriers</li>
                    <li>🏷️ Purchase shipping labels with ease</li>
                    <li>🚚 Schedule pickups and track deliveries</li>
                    <li>💰 Manage wallet and invoices</li>
                    <li>👥 Collaborate with team members</li>
                </ul>
            </div>

            <p>Click the button below to accept the invitation and join the team:</p>

            <div class="button-container">
                <a href="{{.InviteLink}}" class="button">Accept Invitation</a>
            </div>

            <div class="link-box">
                <p><strong>Or copy and paste this link:</strong></p>
                <p class="link">{{.InviteLink}}</p>
            </div>

            <div class="note">
                <strong>⏰ Note:</strong> This invitation will expire in 7 days.
            </div>

            <p style="margin-top: 30px;">If you did not expect this invitation, you can safely ignore this email.</p>
        </div>

        <div class="footer">
            <p><strong>GoShipment</strong></p>
            <p>© 2026 GoShipment. All rights reserved.</p>
            <p>This email was sent to {{.To}}</p>
        </div>
    </div>
</body>
</html>
`

	// Plain text version
	plainText := fmt.Sprintf(`Hello,

%s has invited you to join %s on GoShipment.

GoShipment is a comprehensive shipping and logistics platform that helps teams manage shipments, track deliveries, and handle invoicing all in one place.

Click the link below to accept the invitation:
%s

Note: This invitation will expire in 7 days.

If you did not expect this invitation, you can safely ignore this email.

© 2026 GoShipment. All rights reserved.
This email was sent to %s
`, inviterName, organizationName, inviteLink, to)

	// Parse template
	tmpl, err := template.New("invitation").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Execute template
	var htmlBody bytes.Buffer
	data := map[string]string{
		"InviterName":      inviterName,
		"OrganizationName": organizationName,
		"InviteLink":       inviteLink,
		"To":               to,
	}

	if err := tmpl.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Send email
	return e.SendEmail(EmailMessage{
		To:      []string{to},
		Subject: subject,
		Body:    plainText,
		HTML:    htmlBody.String(),
	})
}

// SendWelcomeEmail sends a welcome email to new users
func (e *EmailClient) SendWelcomeEmail(to, firstName string) error {
	subject := "Welcome to GoShipment!"

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 0;
            background-color: #f4f4f4;
        }
        .email-container {
            background-color: #ffffff;
            margin: 20px auto;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .header {
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .content {
            padding: 40px 30px;
        }
        .content p {
            margin: 0 0 15px 0;
            font-size: 16px;
        }
        .content ul {
            margin: 15px 0;
            padding-left: 20px;
        }
        .content li {
            margin: 10px 0;
            font-size: 15px;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 30px;
            border-top: 1px solid #e9ecef;
            text-align: center;
        }
        .footer p {
            margin: 5px 0;
            color: #6c757d;
            font-size: 13px;
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <h1>Welcome to GoShipment! 🚀</h1>
        </div>

        <div class="content">
            <p>Hi <strong>{{.FirstName}}</strong>,</p>

            <p>Thank you for joining GoShipment! We're excited to have you on board.</p>

            <p>With GoShipment, you can:</p>
            <ul>
                <li>📦 Get instant shipping quotes from multiple carriers</li>
                <li>🏷️ Purchase shipping labels with ease</li>
                <li>🚚 Schedule pickups and track deliveries</li>
                <li>💰 Manage your wallet and invoices</li>
                <li>👥 Collaborate with your team</li>
                <li>📊 View detailed reports and analytics</li>
            </ul>

            <p>Get started by creating your first shipment estimate!</p>

            <p style="margin-top: 30px;">If you have any questions, our support team is here to help.</p>

            <p>Best regards,<br><strong>The GoShipment Team</strong></p>
        </div>

        <div class="footer">
            <p><strong>GoShipment</strong></p>
            <p>© 2026 GoShipment. All rights reserved.</p>
            <p>This email was sent to {{.To}}</p>
        </div>
    </div>
</body>
</html>
`

	plainText := fmt.Sprintf(`Hi %s,

Thank you for joining GoShipment! We're excited to have you on board.

With GoShipment, you can:
- Get instant shipping quotes from multiple carriers
- Purchase shipping labels with ease
- Schedule pickups and track deliveries
- Manage your wallet and invoices
- Collaborate with your team
- View detailed reports and analytics

Get started by creating your first shipment estimate!

If you have any questions, our support team is here to help.

Best regards,
The GoShipment Team

© 2026 GoShipment. All rights reserved.
This email was sent to %s
`, firstName, to)

	tmpl, err := template.New("welcome").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var htmlBody bytes.Buffer
	data := map[string]string{
		"FirstName": firstName,
		"To":        to,
	}

	if err := tmpl.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	return e.SendEmail(EmailMessage{
		To:      []string{to},
		Subject: subject,
		Body:    plainText,
		HTML:    htmlBody.String(),
	})
}
