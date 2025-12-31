package twilio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gogin/internal/config"
)

// TwilioClient wraps Twilio API
type TwilioClient struct {
	accountSID string
	authToken  string
	fromNumber string
}

// NewTwilioClient creates a new Twilio client
func NewTwilioClient(cfg config.TwilioConfig) *TwilioClient {
	return &TwilioClient{
		accountSID: cfg.AccountSID,
		authToken:  cfg.AuthToken,
		fromNumber: cfg.FromNumber,
	}
}

// SMSMessage represents an SMS message
type SMSMessage struct {
	To   string
	Body string
}

// SendSMS sends an SMS via Twilio
func (c *TwilioClient) SendSMS(msg *SMSMessage) error {
	if c.accountSID == "" || c.authToken == "" {
		return fmt.Errorf("Twilio credentials not configured")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.accountSID)

	data := url.Values{}
	data.Set("To", msg.To)
	data.Set("From", c.fromNumber)
	data.Set("Body", msg.Body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Twilio API error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendVerificationCode sends a verification code via SMS
func (c *TwilioClient) SendVerificationCode(phoneNumber, code string) error {
	msg := &SMSMessage{
		To:   phoneNumber,
		Body: fmt.Sprintf("Your verification code is: %s", code),
	}
	return c.SendSMS(msg)
}

// TwilioResponse represents Twilio API response
type TwilioResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	To          string `json:"to"`
	From        string `json:"from"`
	Body        string `json:"body"`
	ErrorCode   int    `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ParseResponse parses Twilio API response
func (c *TwilioClient) ParseResponse(body []byte) (*TwilioResponse, error) {
	var resp TwilioResponse
	err := json.Unmarshal(body, &resp)
	return &resp, err
}
