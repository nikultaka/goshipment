package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gogin/internal/config"
)

// FedExClient handles FedEx API interactions
type FedExClient struct {
	config     *config.FedExConfig
	httpClient *http.Client
	token      *fedexToken
	tokenMu    sync.RWMutex
}

type fedexToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

// FedExTokenResponse represents the OAuth token response from FedEx
type FedExTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// FedExError represents an error response from FedEx API
type FedExError struct {
	TransactionID string `json:"transactionId"`
	Errors        []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// NewFedExClient creates a new FedEx API client
func NewFedExClient(cfg *config.FedExConfig) *FedExClient {
	return &FedExClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if FedEx credentials are configured
func (c *FedExClient) IsConfigured() bool {
	return c.config.ClientID != "" && c.config.ClientSecret != "" && c.config.AccountNumber != ""
}

// GetToken retrieves a valid OAuth token, refreshing if necessary
func (c *FedExClient) GetToken() (string, error) {
	c.tokenMu.RLock()
	if c.token != nil && time.Now().Before(c.token.ExpiresAt) {
		token := c.token.AccessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	return c.refreshToken()
}

// refreshToken fetches a new OAuth token from FedEx
func (c *FedExClient) refreshToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.token != nil && time.Now().Before(c.token.ExpiresAt) {
		return c.token.AccessToken, nil
	}

	tokenURL := c.config.TokenURL()

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var fedexErr FedExError
		if json.Unmarshal(body, &fedexErr) == nil && len(fedexErr.Errors) > 0 {
			return "", fmt.Errorf("FedEx OAuth error: %s - %s", fedexErr.Errors[0].Code, fedexErr.Errors[0].Message)
		}
		return "", fmt.Errorf("FedEx OAuth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp FedExTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// Store token with 5-minute buffer before expiration
	c.token = &fedexToken{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second),
	}

	return c.token.AccessToken, nil
}

// DoRequest performs an authenticated request to FedEx API
func (c *FedExClient) DoRequest(method, endpoint string, body interface{}) ([]byte, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get FedEx token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.config.BaseURL() + endpoint
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-locale", "en_CA")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var fedexErr FedExError
		if json.Unmarshal(respBody, &fedexErr) == nil && len(fedexErr.Errors) > 0 {
			return nil, fmt.Errorf("FedEx API error: %s - %s", fedexErr.Errors[0].Code, fedexErr.Errors[0].Message)
		}
		return nil, fmt.Errorf("FedEx API failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetRates fetches shipping rates from FedEx
func (c *FedExClient) GetRates(request *FedExRateRequest) (*FedExRateResponse, error) {
	respBody, err := c.DoRequest("POST", "/rate/v1/rates/quotes", request)
	if err != nil {
		return nil, err
	}

	var rateResp FedExRateResponse
	if err := json.Unmarshal(respBody, &rateResp); err != nil {
		return nil, fmt.Errorf("failed to parse rate response: %w", err)
	}

	return &rateResp, nil
}

// CreateShipment creates a shipment and returns label
func (c *FedExClient) CreateShipment(request *FedExShipmentRequest) (*FedExShipmentResponse, error) {
	respBody, err := c.DoRequest("POST", "/ship/v1/shipments", request)
	if err != nil {
		return nil, err
	}

	var shipResp FedExShipmentResponse
	if err := json.Unmarshal(respBody, &shipResp); err != nil {
		return nil, fmt.Errorf("failed to parse shipment response: %w", err)
	}

	return &shipResp, nil
}

// TrackShipment tracks a shipment by tracking number
func (c *FedExClient) TrackShipment(trackingNumber string) (*FedExTrackResponse, error) {
	request := map[string]interface{}{
		"includeDetailedScans": true,
		"trackingInfo": []map[string]interface{}{
			{
				"trackingNumberInfo": map[string]string{
					"trackingNumber": trackingNumber,
				},
			},
		},
	}

	respBody, err := c.DoRequest("POST", "/track/v1/trackingnumbers", request)
	if err != nil {
		return nil, err
	}

	var trackResp FedExTrackResponse
	if err := json.Unmarshal(respBody, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to parse track response: %w", err)
	}

	return &trackResp, nil
}

// AccountNumber returns the configured account number
func (c *FedExClient) AccountNumber() string {
	return c.config.AccountNumber
}

// Environment returns the current environment (TEST or PRODUCTION)
func (c *FedExClient) Environment() string {
	return c.config.Env
}

// --- Request/Response Types ---

// FedExRateRequest represents a rate quote request
type FedExRateRequest struct {
	AccountNumber struct {
		Value string `json:"value"`
	} `json:"accountNumber"`
	RequestedShipment FedExRequestedShipment `json:"requestedShipment"`
}

type FedExRequestedShipment struct {
	Shipper          FedExParty    `json:"shipper"`
	Recipient        FedExParty    `json:"recipient"`
	PickupType       string        `json:"pickupType"`
	RateRequestType  []string      `json:"rateRequestType"`
	RequestedPackageLineItems []FedExPackage `json:"requestedPackageLineItems"`
}

type FedExParty struct {
	Address FedExAddress `json:"address"`
}

type FedExAddress struct {
	PostalCode             string   `json:"postalCode"`
	CountryCode            string   `json:"countryCode"`
	City                   string   `json:"city,omitempty"`
	StateOrProvinceCode    string   `json:"stateOrProvinceCode,omitempty"`
	StreetLines            []string `json:"streetLines,omitempty"`
	Residential            bool     `json:"residential,omitempty"`
}

type FedExPackage struct {
	Weight FedExWeight `json:"weight"`
	Dimensions *FedExDimensions `json:"dimensions,omitempty"`
}

type FedExWeight struct {
	Units string  `json:"units"` // LB or KG
	Value float64 `json:"value"`
}

type FedExDimensions struct {
	Length int    `json:"length"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Units  string `json:"units"` // IN or CM
}

// FedExRateResponse represents the rate response
type FedExRateResponse struct {
	TransactionID string `json:"transactionId"`
	Output        struct {
		RateReplyDetails []struct {
			ServiceType     string `json:"serviceType"`
			ServiceName     string `json:"serviceName"`
			RatedShipmentDetails []struct {
				TotalNetCharge       float64 `json:"totalNetCharge"`
				TotalNetChargeWithDutiesAndTaxes float64 `json:"totalNetChargeWithDutiesAndTaxes"`
				Currency             string  `json:"currency"`
			} `json:"ratedShipmentDetails"`
			DeliveryTimestamp string `json:"deliveryTimestamp,omitempty"`
			DeliveryDayOfWeek string `json:"deliveryDayOfWeek,omitempty"`
			TransitTime       string `json:"transitTime,omitempty"`
		} `json:"rateReplyDetails"`
	} `json:"output"`
}

// FedExShipmentRequest represents a shipment creation request
type FedExShipmentRequest struct {
	LabelResponseOptions string `json:"labelResponseOptions"` // URL_ONLY or LABEL
	RequestedShipment    struct {
		Shipper         FedExShipmentParty `json:"shipper"`
		Recipients      []FedExShipmentParty `json:"recipients"`
		PickupType      string `json:"pickupType"`
		ServiceType     string `json:"serviceType"`
		PackagingType   string `json:"packagingType"`
		ShippingChargesPayment struct {
			PaymentType string `json:"paymentType"`
			Payor       struct {
				ResponsibleParty struct {
					AccountNumber struct {
						Value string `json:"value"`
					} `json:"accountNumber"`
				} `json:"responsibleParty"`
			} `json:"payor"`
		} `json:"shippingChargesPayment"`
		LabelSpecification struct {
			LabelStockType string `json:"labelStockType"`
			ImageType      string `json:"imageType"`
		} `json:"labelSpecification"`
		RequestedPackageLineItems []FedExShipmentPackage `json:"requestedPackageLineItems"`
	} `json:"requestedShipment"`
	AccountNumber struct {
		Value string `json:"value"`
	} `json:"accountNumber"`
}

type FedExShipmentParty struct {
	Contact FedExContact `json:"contact"`
	Address FedExAddress `json:"address"`
}

type FedExContact struct {
	PersonName  string `json:"personName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	PhoneNumber string `json:"phoneNumber"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

type FedExShipmentPackage struct {
	Weight     FedExWeight     `json:"weight"`
	Dimensions *FedExDimensions `json:"dimensions,omitempty"`
}

// FedExShipmentResponse represents the shipment creation response
type FedExShipmentResponse struct {
	TransactionID string `json:"transactionId"`
	Output        struct {
		TransactionShipments []struct {
			MasterTrackingNumber string `json:"masterTrackingNumber"`
			ServiceType          string `json:"serviceType"`
			ShipDatestamp        string `json:"shipDatestamp"`
			PieceResponses       []struct {
				TrackingNumber string `json:"trackingNumber"`
				PackageDocuments []struct {
					DocType   string `json:"docType"`
					URL       string `json:"url,omitempty"`
					EncodedLabel string `json:"encodedLabel,omitempty"`
				} `json:"packageDocuments"`
			} `json:"pieceResponses"`
			CompletedShipmentDetail struct {
				ShipmentRating struct {
					ActualRateType string `json:"actualRateType"`
					ShipmentRateDetails []struct {
						TotalNetCharge float64 `json:"totalNetCharge"`
						Currency       string  `json:"currency"`
					} `json:"shipmentRateDetails"`
				} `json:"shipmentRating"`
			} `json:"completedShipmentDetail"`
		} `json:"transactionShipments"`
	} `json:"output"`
}

// FedExTrackResponse represents tracking response
type FedExTrackResponse struct {
	TransactionID string `json:"transactionId"`
	Output        struct {
		CompleteTrackResults []struct {
			TrackingNumber string `json:"trackingNumber"`
			TrackResults   []struct {
				TrackingNumberInfo struct {
					TrackingNumber string `json:"trackingNumber"`
				} `json:"trackingNumberInfo"`
				LatestStatusDetail struct {
					Code        string `json:"code"`
					Description string `json:"description"`
					StatusByLocale string `json:"statusByLocale"`
				} `json:"latestStatusDetail"`
				DateAndTimes []struct {
					Type     string `json:"type"`
					DateTime string `json:"dateTime"`
				} `json:"dateAndTimes"`
				DeliveryDetails struct {
					ActualDeliveryAddress FedExAddress `json:"actualDeliveryAddress"`
					DeliveryAttempts      string       `json:"deliveryAttempts"`
					ReceivedByName        string       `json:"receivedByName"`
				} `json:"deliveryDetails"`
				ScanEvents []struct {
					Date           string `json:"date"`
					EventType      string `json:"eventType"`
					EventDescription string `json:"eventDescription"`
					ScanLocation   struct {
						City                string `json:"city"`
						StateOrProvinceCode string `json:"stateOrProvinceCode"`
						CountryCode         string `json:"countryCode"`
					} `json:"scanLocation"`
				} `json:"scanEvents"`
			} `json:"trackResults"`
		} `json:"completeTrackResults"`
	} `json:"output"`
}
