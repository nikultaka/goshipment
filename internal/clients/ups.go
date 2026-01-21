package clients

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gogin/internal/config"
)

// UPSClient handles UPS API interactions
type UPSClient struct {
	config     *config.UPSConfig
	httpClient *http.Client
	token      *upsToken
	tokenMu    sync.RWMutex
}

type upsToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

// UPSTokenResponse represents the OAuth token response from UPS
type UPSTokenResponse struct {
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	ClientID    string `json:"client_id"`
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	Status      string `json:"status"`
}

// UPSError represents an error response from UPS API
type UPSError struct {
	Response struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"response"`
}

// NewUPSClient creates a new UPS API client
func NewUPSClient(cfg *config.UPSConfig) *UPSClient {
	return &UPSClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if UPS credentials are configured
func (c *UPSClient) IsConfigured() bool {
	return c.config.ClientID != "" && c.config.ClientSecret != "" && c.config.AccountNumber != ""
}

// GetToken retrieves a valid OAuth token, refreshing if necessary
func (c *UPSClient) GetToken() (string, error) {
	c.tokenMu.RLock()
	if c.token != nil && time.Now().Before(c.token.ExpiresAt) {
		token := c.token.AccessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	return c.refreshToken()
}

// refreshToken fetches a new OAuth token from UPS
func (c *UPSClient) refreshToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.token != nil && time.Now().Before(c.token.ExpiresAt) {
		return c.token.AccessToken, nil
	}

	tokenURL := c.config.TokenURL()

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	// UPS uses Basic Auth for OAuth token request
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.ClientID + ":" + c.config.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-merchant-id", c.config.AccountNumber)

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
		var upsErr UPSError
		if json.Unmarshal(body, &upsErr) == nil && len(upsErr.Response.Errors) > 0 {
			return "", fmt.Errorf("UPS OAuth error: %s - %s", upsErr.Response.Errors[0].Code, upsErr.Response.Errors[0].Message)
		}
		return "", fmt.Errorf("UPS OAuth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp UPSTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// Parse expires_in (UPS returns it as string in seconds)
	expiresIn := 14399 // Default ~4 hours
	if tokenResp.ExpiresIn != "" {
		fmt.Sscanf(tokenResp.ExpiresIn, "%d", &expiresIn)
	}

	// Store token with 5-minute buffer before expiration
	c.token = &upsToken{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn-300) * time.Second),
	}

	return c.token.AccessToken, nil
}

// DoRequest performs an authenticated request to UPS API
func (c *UPSClient) DoRequest(method, endpoint string, body interface{}) ([]byte, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get UPS token: %w", err)
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
	req.Header.Set("transId", fmt.Sprintf("%d", time.Now().UnixNano()))
	req.Header.Set("transactionSrc", "goshipment")

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
		var upsErr UPSError
		if json.Unmarshal(respBody, &upsErr) == nil && len(upsErr.Response.Errors) > 0 {
			return nil, fmt.Errorf("UPS API error: %s - %s", upsErr.Response.Errors[0].Code, upsErr.Response.Errors[0].Message)
		}
		return nil, fmt.Errorf("UPS API failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetRates fetches shipping rates from UPS
func (c *UPSClient) GetRates(request *UPSRateRequest) (*UPSRateResponse, error) {
	respBody, err := c.DoRequest("POST", "/api/rating/v1/Shop", request)
	if err != nil {
		return nil, err
	}

	var rateResp UPSRateResponse
	if err := json.Unmarshal(respBody, &rateResp); err != nil {
		return nil, fmt.Errorf("failed to parse rate response: %w", err)
	}

	return &rateResp, nil
}

// CreateShipment creates a shipment and returns label
func (c *UPSClient) CreateShipment(request *UPSShipmentRequest) (*UPSShipmentResponse, error) {
	respBody, err := c.DoRequest("POST", "/api/shipments/v1/ship", request)
	if err != nil {
		return nil, err
	}

	var shipResp UPSShipmentResponse
	if err := json.Unmarshal(respBody, &shipResp); err != nil {
		return nil, fmt.Errorf("failed to parse shipment response: %w", err)
	}

	return &shipResp, nil
}

// TrackShipment tracks a shipment by tracking number
func (c *UPSClient) TrackShipment(trackingNumber string) (*UPSTrackResponse, error) {
	endpoint := fmt.Sprintf("/api/track/v1/details/%s", trackingNumber)

	respBody, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var trackResp UPSTrackResponse
	if err := json.Unmarshal(respBody, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to parse track response: %w", err)
	}

	return &trackResp, nil
}

// AccountNumber returns the configured account number
func (c *UPSClient) AccountNumber() string {
	return c.config.AccountNumber
}

// Environment returns the current environment (TEST or PRODUCTION)
func (c *UPSClient) Environment() string {
	return c.config.Env
}

// --- Request/Response Types ---

// UPSRateRequest represents a rate request
type UPSRateRequest struct {
	RateRequest struct {
		Request struct {
			SubVersion string `json:"SubVersion,omitempty"`
		} `json:"Request"`
		Shipment UPSRateShipment `json:"Shipment"`
	} `json:"RateRequest"`
}

type UPSRateShipment struct {
	Shipper         UPSParty   `json:"Shipper"`
	ShipTo          UPSParty   `json:"ShipTo"`
	ShipFrom        UPSParty   `json:"ShipFrom,omitempty"`
	PaymentDetails  *UPSPaymentDetails `json:"PaymentDetails,omitempty"`
	Service         *UPSService `json:"Service,omitempty"`
	Package         []UPSPackage `json:"Package"`
	ShipmentRatingOptions *UPSShipmentRatingOptions `json:"ShipmentRatingOptions,omitempty"`
}

type UPSParty struct {
	Name          string     `json:"Name,omitempty"`
	ShipperNumber string     `json:"ShipperNumber,omitempty"`
	Address       UPSAddress `json:"Address"`
}

type UPSAddress struct {
	AddressLine       []string `json:"AddressLine,omitempty"`
	City              string   `json:"City"`
	StateProvinceCode string   `json:"StateProvinceCode,omitempty"`
	PostalCode        string   `json:"PostalCode"`
	CountryCode       string   `json:"CountryCode"`
	ResidentialAddressIndicator string `json:"ResidentialAddressIndicator,omitempty"`
}

type UPSPaymentDetails struct {
	ShipmentCharge []UPSShipmentCharge `json:"ShipmentCharge"`
}

type UPSShipmentCharge struct {
	Type        string `json:"Type"`
	BillShipper struct {
		AccountNumber string `json:"AccountNumber"`
	} `json:"BillShipper"`
}

type UPSService struct {
	Code        string `json:"Code"`
	Description string `json:"Description,omitempty"`
}

type UPSPackage struct {
	PackagingType UPSPackagingType `json:"PackagingType"`
	Dimensions    *UPSDimensions   `json:"Dimensions,omitempty"`
	PackageWeight UPSPackageWeight `json:"PackageWeight"`
}

type UPSPackagingType struct {
	Code        string `json:"Code"`
	Description string `json:"Description,omitempty"`
}

type UPSDimensions struct {
	UnitOfMeasurement struct {
		Code string `json:"Code"` // IN or CM
	} `json:"UnitOfMeasurement"`
	Length string `json:"Length"`
	Width  string `json:"Width"`
	Height string `json:"Height"`
}

type UPSPackageWeight struct {
	UnitOfMeasurement struct {
		Code string `json:"Code"` // LBS or KGS
	} `json:"UnitOfMeasurement"`
	Weight string `json:"Weight"`
}

type UPSShipmentRatingOptions struct {
	NegotiatedRatesIndicator string `json:"NegotiatedRatesIndicator,omitempty"`
}

// UPSRateResponse represents the rate response
type UPSRateResponse struct {
	RateResponse struct {
		Response struct {
			ResponseStatus struct {
				Code        string `json:"Code"`
				Description string `json:"Description"`
			} `json:"ResponseStatus"`
		} `json:"Response"`
		RatedShipment []struct {
			Service struct {
				Code        string `json:"Code"`
				Description string `json:"Description"`
			} `json:"Service"`
			TotalCharges struct {
				CurrencyCode  string `json:"CurrencyCode"`
				MonetaryValue string `json:"MonetaryValue"`
			} `json:"TotalCharges"`
			NegotiatedRateCharges *struct {
				TotalCharge struct {
					CurrencyCode  string `json:"CurrencyCode"`
					MonetaryValue string `json:"MonetaryValue"`
				} `json:"TotalCharge"`
			} `json:"NegotiatedRateCharges,omitempty"`
			GuaranteedDelivery *struct {
				BusinessDaysInTransit string `json:"BusinessDaysInTransit"`
				DeliveryByTime        string `json:"DeliveryByTime,omitempty"`
			} `json:"GuaranteedDelivery,omitempty"`
			TimeInTransit *struct {
				ServiceSummary struct {
					EstimatedArrival struct {
						Arrival struct {
							Date string `json:"Date"`
							Time string `json:"Time"`
						} `json:"Arrival"`
					} `json:"EstimatedArrival"`
				} `json:"ServiceSummary"`
			} `json:"TimeInTransit,omitempty"`
		} `json:"RatedShipment"`
	} `json:"RateResponse"`
}

// UPSShipmentRequest represents a shipment creation request
type UPSShipmentRequest struct {
	ShipmentRequest struct {
		Request struct {
			SubVersion string `json:"SubVersion,omitempty"`
		} `json:"Request"`
		Shipment struct {
			Description     string     `json:"Description,omitempty"`
			Shipper         UPSShipmentParty `json:"Shipper"`
			ShipTo          UPSShipmentParty `json:"ShipTo"`
			ShipFrom        UPSShipmentParty `json:"ShipFrom,omitempty"`
			PaymentInformation struct {
				ShipmentCharge []UPSShipmentCharge `json:"ShipmentCharge"`
			} `json:"PaymentInformation"`
			Service UPSService `json:"Service"`
			Package []UPSShipmentPackage `json:"Package"`
		} `json:"Shipment"`
		LabelSpecification struct {
			LabelImageFormat struct {
				Code string `json:"Code"` // GIF, PNG, ZPL, etc.
			} `json:"LabelImageFormat"`
			LabelStockSize *struct {
				Height string `json:"Height"`
				Width  string `json:"Width"`
			} `json:"LabelStockSize,omitempty"`
		} `json:"LabelSpecification"`
	} `json:"ShipmentRequest"`
}

type UPSShipmentParty struct {
	Name          string     `json:"Name"`
	ShipperNumber string     `json:"ShipperNumber,omitempty"`
	AttentionName string     `json:"AttentionName,omitempty"`
	Phone         *UPSPhone  `json:"Phone,omitempty"`
	Address       UPSAddress `json:"Address"`
}

type UPSPhone struct {
	Number string `json:"Number"`
}

type UPSShipmentPackage struct {
	Description   string `json:"Description,omitempty"`
	Packaging     UPSPackagingType `json:"Packaging"`
	Dimensions    *UPSDimensions   `json:"Dimensions,omitempty"`
	PackageWeight UPSPackageWeight `json:"PackageWeight"`
}

// UPSShipmentResponse represents the shipment creation response
type UPSShipmentResponse struct {
	ShipmentResponse struct {
		Response struct {
			ResponseStatus struct {
				Code        string `json:"Code"`
				Description string `json:"Description"`
			} `json:"ResponseStatus"`
		} `json:"Response"`
		ShipmentResults struct {
			ShipmentCharges struct {
				TotalCharges struct {
					CurrencyCode  string `json:"CurrencyCode"`
					MonetaryValue string `json:"MonetaryValue"`
				} `json:"TotalCharges"`
			} `json:"ShipmentCharges"`
			ShipmentIdentificationNumber string `json:"ShipmentIdentificationNumber"`
			PackageResults []struct {
				TrackingNumber string `json:"TrackingNumber"`
				ShippingLabel  struct {
					ImageFormat struct {
						Code string `json:"Code"`
					} `json:"ImageFormat"`
					GraphicImage string `json:"GraphicImage"` // Base64 encoded
				} `json:"ShippingLabel"`
			} `json:"PackageResults"`
		} `json:"ShipmentResults"`
	} `json:"ShipmentResponse"`
}

// UPSTrackResponse represents tracking response
type UPSTrackResponse struct {
	TrackResponse struct {
		Shipment []struct {
			InquiryNumber string `json:"inquiryNumber"`
			Package       []struct {
				TrackingNumber string `json:"trackingNumber"`
				Activity       []struct {
					Location struct {
						Address struct {
							City              string `json:"city"`
							StateProvince     string `json:"stateProvince"`
							CountryCode       string `json:"countryCode"`
						} `json:"address"`
					} `json:"location"`
					Status struct {
						Type        string `json:"type"`
						Description string `json:"description"`
						Code        string `json:"code"`
					} `json:"status"`
					Date string `json:"date"`
					Time string `json:"time"`
				} `json:"activity"`
				CurrentStatus struct {
					Description string `json:"description"`
					Code        string `json:"code"`
				} `json:"currentStatus"`
				DeliveryDate []struct {
					Type string `json:"type"`
					Date string `json:"date"`
				} `json:"deliveryDate"`
			} `json:"package"`
		} `json:"shipment"`
	} `json:"trackResponse"`
}
