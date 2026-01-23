package clients

import (
	"fmt"
	"log"

	"gogin/internal/config"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSClient handles SMS sending via Twilio
type SMSClient struct {
	client     *twilio.RestClient
	fromNumber string
	enabled    bool
}

// NewSMSClient creates a new Twilio SMS client
func NewSMSClient(cfg *config.TwilioConfig) *SMSClient {
	if cfg.AccountSID == "" || cfg.AuthToken == "" {
		log.Println("⚠️  Twilio credentials not configured - SMS sending disabled")
		return &SMSClient{enabled: false}
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})

	return &SMSClient{
		client:     client,
		fromNumber: cfg.FromNumber,
		enabled:    cfg.Enabled,
	}
}

// SendOTP sends an OTP code via SMS
func (s *SMSClient) SendOTP(to, otp string) error {
	if !s.enabled {
		log.Printf("📱 [SMS DISABLED] Would send OTP %s to %s\n", otp, to)
		return fmt.Errorf("SMS sending is disabled")
	}

	if s.client == nil {
		return fmt.Errorf("twilio client not initialized")
	}

	message := fmt.Sprintf("Your verification code is: %s. Valid for 5 minutes. Do not share this code.", otp)

	return s.SendSMS(to, message)
}

// SendSMS sends a generic SMS message
func (s *SMSClient) SendSMS(to, body string) error {
	if !s.enabled {
		log.Printf("📱 [SMS DISABLED] Would send to %s: %s\n", to, body)
		return fmt.Errorf("SMS sending is disabled")
	}

	if s.client == nil {
		return fmt.Errorf("twilio client not initialized")
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(s.fromNumber)
	params.SetBody(body)

	resp, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("❌ Failed to send SMS to %s: %v\n", to, err)
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	log.Printf("✅ SMS sent successfully to %s (SID: %s)\n", to, *resp.Sid)
	return nil
}

// IsEnabled returns whether SMS sending is enabled
func (s *SMSClient) IsEnabled() bool {
	return s.enabled
}
