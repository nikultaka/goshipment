package apiclient

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"gogin/internal/clients"
	"gogin/internal/models"
	"gogin/internal/modules/redishelper"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// APIClientService handles API client business logic
type APIClientService struct {
	db          *clients.Database
	redisHelper *redishelper.RedisHelper
}

// NewAPIClientService creates a new API client service
func NewAPIClientService(db *clients.Database, redisHelper *redishelper.RedisHelper) *APIClientService {
	return &APIClientService{
		db:          db,
		redisHelper: redisHelper,
	}
}

// CreateClient creates a new OAuth client
func (s *APIClientService) CreateClient(userID string, req *CreateClientRequest) (*ClientResponse, error) {
	clientID := s.generateClientID()

	// Determine client type and secret handling
	var clientType string
	var clientSecret string
	var clientSecretHash sql.NullString

	if req.IsPublic {
		clientType = "public"
		clientSecret = ""
		clientSecretHash = sql.NullString{Valid: false} // NULL for public clients
	} else {
		clientType = "confidential"
		clientSecret = s.generateClientSecret()
		// Hash the secret before storing
		hash := s.hashSecret(clientSecret)
		clientSecretHash = sql.NullString{String: hash, Valid: true}
	}

	id := uuid.New().String()
	query := `
		INSERT INTO oauth_clients
		(id, client_id, client_secret_hash, name, description, redirect_uris, scopes,
		 client_type, allowed_grant_types, require_pkce, pkce_challenge_method,
		 access_token_lifetime, refresh_token_lifetime, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(query,
		id,
		clientID,
		clientSecretHash,
		req.Name,
		req.Description,
		pq.Array(req.RedirectURIs),  // PostgreSQL array type
		pq.Array(req.Scopes),         // PostgreSQL array type
		clientType,
		pq.Array(req.GrantTypes),     // PostgreSQL array type
		true,                          // require_pkce (OAuth 2.1 best practice)
		"S256",                        // pkce_challenge_method
		3600,                          // access_token_lifetime (1 hour)
		2592000,                       // refresh_token_lifetime (30 days)
		true,                          // is_active
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &ClientResponse{
		ID:                   id,
		ClientID:             clientID,
		ClientSecret:         clientSecret, // Only returned on creation!
		Name:                 req.Name,
		Description:          req.Description,
		RedirectURIs:         req.RedirectURIs,
		Scopes:               req.Scopes,
		ClientType:           clientType,
		AllowedGrantTypes:    req.GrantTypes,
		RequirePKCE:          true,
		PKCEChallengeMethod:  "S256",
		AccessTokenLifetime:  3600,
		RefreshTokenLifetime: 2592000,
		IsActive:             true,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}

// GetClient retrieves a client by ID
func (s *APIClientService) GetClient(id string) (*ClientResponse, error) {
	var client models.OAuthClient
	query := `
		SELECT id, client_id, client_secret_hash, name, description, redirect_uris,
		       scopes, client_type, allowed_grant_types, require_pkce, pkce_challenge_method,
		       access_token_lifetime, refresh_token_lifetime, is_active, created_at, updated_at
		FROM oauth_clients
		WHERE id = $1
	`

	err := s.db.QueryRow(query, id).Scan(
		&client.ID,
		&client.ClientID,
		&client.ClientSecretHash,
		&client.Name,
		&client.Description,
		pq.Array(&client.RedirectURIs),
		pq.Array(&client.Scopes),
		&client.ClientType,
		pq.Array(&client.AllowedGrantTypes),
		&client.RequirePKCE,
		&client.PKCEChallengeMethod,
		&client.AccessTokenLifetime,
		&client.RefreshTokenLifetime,
		&client.IsActive,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return s.toClientResponse(&client), nil
}

// ListClients lists all clients with pagination
func (s *APIClientService) ListClients(page, limit int) ([]*ClientResponse, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM oauth_clients").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get clients
	query := `
		SELECT id, client_id, client_secret_hash, name, description, redirect_uris,
		       scopes, client_type, allowed_grant_types, require_pkce, pkce_challenge_method,
		       access_token_lifetime, refresh_token_lifetime, is_active, created_at, updated_at
		FROM oauth_clients
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []*ClientResponse
	for rows.Next() {
		var client models.OAuthClient
		err := rows.Scan(
			&client.ID,
			&client.ClientID,
			&client.ClientSecretHash,
			&client.Name,
			&client.Description,
			pq.Array(&client.RedirectURIs),
			pq.Array(&client.Scopes),
			&client.ClientType,
			pq.Array(&client.AllowedGrantTypes),
			&client.RequirePKCE,
			&client.PKCEChallengeMethod,
			&client.AccessTokenLifetime,
			&client.RefreshTokenLifetime,
			&client.IsActive,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, s.toClientResponse(&client))
	}

	return clients, total, nil
}

// UpdateClient updates a client
func (s *APIClientService) UpdateClient(id string, req *UpdateClientRequest) (*ClientResponse, error) {
	query := `
		UPDATE oauth_clients
		SET name = $1, description = $2, redirect_uris = $3, scopes = $4,
		    allowed_grant_types = $5, updated_at = NOW()
		WHERE id = $6
	`

	result, err := s.db.Exec(query,
		req.Name,
		req.Description,
		pq.Array(req.RedirectURIs),
		pq.Array(req.Scopes),
		pq.Array(req.GrantTypes),
		id,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("client not found")
	}

	return s.GetClient(id)
}

// DeleteClient soft deletes a client
func (s *APIClientService) DeleteClient(id string) error {
	query := `UPDATE oauth_clients SET is_active = false, updated_at = NOW() WHERE id = $1`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

// RegenerateSecret generates a new client secret
func (s *APIClientService) RegenerateSecret(id string) (*ClientResponse, error) {
	// Check if client exists and is confidential
	var clientType string
	err := s.db.QueryRow("SELECT client_type FROM oauth_clients WHERE id = $1", id).Scan(&clientType)
	if err != nil {
		return nil, fmt.Errorf("client not found")
	}

	if clientType == "public" {
		return nil, fmt.Errorf("cannot regenerate secret for public clients")
	}

	// Generate new secret
	newSecret := s.generateClientSecret()
	hash := s.hashSecret(newSecret)

	query := `UPDATE oauth_clients SET client_secret_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err = s.db.Exec(query, hash, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	client, err := s.GetClient(id)
	if err != nil {
		return nil, err
	}

	// Return new secret (only time it's shown)
	client.ClientSecret = newSecret
	return client, nil
}

// UpdateStatus updates client active status
func (s *APIClientService) UpdateStatus(id string, isActive bool) error {
	query := `UPDATE oauth_clients SET is_active = $1, updated_at = NOW() WHERE id = $2`
	result, err := s.db.Exec(query, isActive, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

// Helper methods

func (s *APIClientService) toClientResponse(client *models.OAuthClient) *ClientResponse {
	resp := &ClientResponse{
		ID:                   client.ID,
		ClientID:             client.ClientID,
		Name:                 client.Name,
		RedirectURIs:         client.RedirectURIs,
		Scopes:               client.Scopes,
		ClientType:           client.ClientType,
		AllowedGrantTypes:    client.AllowedGrantTypes,
		RequirePKCE:          client.RequirePKCE,
		PKCEChallengeMethod:  client.PKCEChallengeMethod,
		AccessTokenLifetime:  client.AccessTokenLifetime,
		RefreshTokenLifetime: client.RefreshTokenLifetime,
		IsActive:             client.IsActive,
		CreatedAt:            client.CreatedAt,
		UpdatedAt:            client.UpdatedAt,
	}

	if client.Description.Valid {
		resp.Description = client.Description.String
	}

	// Never return the secret hash
	resp.ClientSecret = ""

	return resp
}

func (s *APIClientService) generateClientID() string {
	return "client_" + s.generateRandomString(16)
}

func (s *APIClientService) generateClientSecret() string {
	return "secret_" + s.generateRandomString(32)
}

func (s *APIClientService) generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}

func (s *APIClientService) hashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(hash[:])
}
