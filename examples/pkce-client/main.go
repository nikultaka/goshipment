package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Configuration
type Config struct {
	AuthServerURL string
	ClientID      string
	RedirectURI   string
	CallbackPort  string
}

// Token response from the server
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// PKCE parameters
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	State         string
	Nonce         string
}

func main() {
	// Parse command-line flags
	authServerURL := flag.String("server", "http://localhost:8080", "Authorization server URL")
	clientID := flag.String("client", "web-app-client", "OAuth2 client ID")
	redirectURI := flag.String("redirect", "http://localhost:8085/callback", "Redirect URI")
	callbackPort := flag.String("port", "8085", "Local callback server port")
	accessToken := flag.String("token", "", "Access token for authorization (get from /users/login)")
	flag.Parse()

	if *accessToken == "" {
		log.Fatal("Error: -token flag is required. Get an access token from POST /api/v1/users/login")
	}

	cfg := &Config{
		AuthServerURL: *authServerURL,
		ClientID:      *clientID,
		RedirectURI:   *redirectURI,
		CallbackPort:  *callbackPort,
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         PKCE OAuth2/OIDC Client Example                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Generate PKCE parameters
	fmt.Println("Step 1: Generating PKCE parameters...")
	pkce := generatePKCEParams()
	fmt.Printf("  ✓ Code Verifier:  %s\n", pkce.CodeVerifier[:20]+"...")
	fmt.Printf("  ✓ Code Challenge: %s\n", pkce.CodeChallenge[:20]+"...")
	fmt.Printf("  ✓ State:          %s\n", pkce.State)
	fmt.Printf("  ✓ Nonce:          %s\n", pkce.Nonce)
	fmt.Println()

	// Step 2: Start local callback server
	authCodeChan := make(chan string)
	go startCallbackServer(cfg.CallbackPort, pkce.State, authCodeChan)
	time.Sleep(500 * time.Millisecond) // Give server time to start

	// Step 3: Build authorization URL
	authURL := buildAuthorizationURL(cfg, pkce)
	fmt.Println("Step 2: Authorization URL generated")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(authURL)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("⚠️  Copy the URL above and open it in your browser")
	fmt.Println("⚠️  Or use curl to make the request")
	fmt.Println()

	// Make the authorization request programmatically
	fmt.Println("Step 3: Making authorization request...")
	authCode, err := makeAuthorizationRequest(cfg, pkce, *accessToken)
	if err != nil {
		log.Fatalf("Authorization request failed: %v", err)
	}
	fmt.Printf("  ✓ Authorization code received: %s\n", authCode[:20]+"...")
	fmt.Println()

	// Step 4: Exchange authorization code for tokens
	fmt.Println("Step 4: Exchanging authorization code for tokens...")
	tokens, err := exchangeCodeForTokens(cfg, authCode, pkce.CodeVerifier)
	if err != nil {
		log.Fatalf("Token exchange failed: %v", err)
	}
	fmt.Println("  ✓ Tokens received successfully!")
	fmt.Println()

	// Step 5: Display tokens
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                      TOKEN RESPONSE                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Printf("Access Token:  %s...\n", tokens.AccessToken[:50])
	fmt.Printf("Token Type:    %s\n", tokens.TokenType)
	fmt.Printf("Expires In:    %d seconds\n", tokens.ExpiresIn)
	if tokens.RefreshToken != "" {
		fmt.Printf("Refresh Token: %s...\n", tokens.RefreshToken[:50])
	}
	if tokens.IDToken != "" {
		fmt.Printf("ID Token:      %s...\n", tokens.IDToken[:50])
	}
	fmt.Println()

	// Step 6: Validate ID token
	if tokens.IDToken != "" {
		fmt.Println("Step 5: Validating ID Token...")
		claims, err := validateIDToken(cfg, tokens.IDToken, pkce.Nonce)
		if err != nil {
			log.Printf("⚠️  ID Token validation failed: %v", err)
		} else {
			fmt.Println("  ✓ ID Token is valid!")
			fmt.Println()
			fmt.Println("ID Token Claims:")
			displayClaims(claims)
		}
	}

	fmt.Println()
	fmt.Println("✅ PKCE flow completed successfully!")
}

// generatePKCEParams generates PKCE code_verifier and code_challenge
func generatePKCEParams() *PKCEParams {
	// Generate code_verifier (43-128 characters, URL-safe)
	codeVerifier := generateRandomString(43)

	// Generate code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate state and nonce
	state := generateRandomString(16)
	nonce := generateRandomString(16)

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		State:         state,
		Nonce:         nonce,
	}
}

// generateRandomString generates a URL-safe random string
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}

// buildAuthorizationURL builds the OAuth2 authorization URL
func buildAuthorizationURL(cfg *Config, pkce *PKCEParams) string {
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", pkce.State)
	params.Set("nonce", pkce.Nonce)
	params.Set("code_challenge", pkce.CodeChallenge)
	params.Set("code_challenge_method", "S256")

	return fmt.Sprintf("%s/api/v1/oauth/authorize?%s", cfg.AuthServerURL, params.Encode())
}

// makeAuthorizationRequest makes the authorization request programmatically
func makeAuthorizationRequest(cfg *Config, pkce *PKCEParams, accessToken string) (string, error) {
	reqBody := map[string]string{
		"client_id":               cfg.ClientID,
		"redirect_uri":            cfg.RedirectURI,
		"response_type":           "code",
		"scope":                   "openid profile email",
		"state":                   pkce.State,
		"nonce":                   pkce.Nonce,
		"code_challenge":          pkce.CodeChallenge,
		"code_challenge_method":   "S256",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", cfg.AuthServerURL+"/api/v1/oauth/authorize", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authorization failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.State != pkce.State {
		return "", fmt.Errorf("state mismatch: expected %s, got %s", pkce.State, result.State)
	}

	return result.Code, nil
}

// exchangeCodeForTokens exchanges the authorization code for tokens
func exchangeCodeForTokens(cfg *Config, code, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", cfg.RedirectURI)
	data.Set("client_id", cfg.ClientID)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST", cfg.AuthServerURL+"/api/v1/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %d - %s", resp.StatusCode, string(body))
	}

	var tokens TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

// validateIDToken validates the ID token signature and claims
func validateIDToken(cfg *Config, idToken, expectedNonce string) (jwt.MapClaims, error) {
	// Fetch JWKS from server
	jwksURL := cfg.AuthServerURL + "/api/v1/oauth/jwks.json"
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("no keys found in JWKS")
	}

	// Parse token without validation first to get the claims
	token, _, err := jwt.NewParser().ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate nonce
	if nonce, ok := claims["nonce"].(string); ok {
		if nonce != expectedNonce {
			return nil, fmt.Errorf("nonce mismatch: expected %s, got %s", expectedNonce, nonce)
		}
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); ok {
		if iss != cfg.AuthServerURL {
			log.Printf("⚠️  Issuer mismatch: expected %s, got %s", cfg.AuthServerURL, iss)
		}
	}

	// Validate audience
	if aud, ok := claims["aud"].([]interface{}); ok {
		if len(aud) > 0 && aud[0].(string) != cfg.ClientID {
			return nil, fmt.Errorf("audience mismatch: expected %s, got %v", cfg.ClientID, aud)
		}
	}

	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token has expired")
		}
	}

	return claims, nil
}

// displayClaims displays ID token claims in a formatted way
func displayClaims(claims jwt.MapClaims) {
	claimsToDisplay := []string{
		"sub", "iss", "aud", "exp", "iat", "nonce",
		"name", "given_name", "family_name", "email", "email_verified",
		"phone_number", "phone_number_verified", "picture", "locale", "zoneinfo",
	}

	for _, key := range claimsToDisplay {
		if val, ok := claims[key]; ok {
			switch v := val.(type) {
			case float64:
				if key == "exp" || key == "iat" {
					fmt.Printf("  %-20s: %s\n", key, time.Unix(int64(v), 0).Format(time.RFC3339))
				} else {
					fmt.Printf("  %-20s: %.0f\n", key, v)
				}
			case []interface{}:
				fmt.Printf("  %-20s: %v\n", key, v)
			default:
				fmt.Printf("  %-20s: %v\n", key, v)
			}
		}
	}
}

// startCallbackServer starts a local HTTP server to receive the authorization code
func startCallbackServer(port, expectedState string, authCodeChan chan string) {
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if state != expectedState {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			authCodeChan <- ""
			return
		}

		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			authCodeChan <- ""
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Authorization Successful</title></head>
			<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
				<h1 style="color: green;">✅ Authorization Successful!</h1>
				<p>You can close this window now.</p>
			</body>
			</html>
		`)

		authCodeChan <- code
	})

	log.Printf("Callback server listening on http://localhost:%s/callback", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Printf("Callback server error: %v", err)
	}
}
