package geo

import (
	"fmt"
	"time"

	"gogin/internal/clients"
)

// GeoService handles geography-related business logic
type GeoService struct {
	db *clients.Database
}

// NewGeoService creates a new geo service
func NewGeoService(db *clients.Database) *GeoService {
	return &GeoService{
		db: db,
	}
}

// Country represents a country from the database
type Country struct {
	Code         string
	Name         string
	CurrencyCode string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Region represents a region/state/province from the database
type Region struct {
	ID          string
	CountryCode string
	Code        string
	Name        string
	Type        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListCountries retrieves all countries, optionally filtered by is_active
func (s *GeoService) ListCountries(isActive *bool) ([]*CountryResponse, error) {
	query := `
		SELECT
			code, name, currency_code, is_active, created_at, updated_at
		FROM countries
	`

	args := []interface{}{}

	if isActive != nil {
		query += " WHERE is_active = $1"
		args = append(args, *isActive)
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list countries: %w", err)
	}
	defer rows.Close()

	countries := []*CountryResponse{}
	for rows.Next() {
		var c Country
		err := rows.Scan(
			&c.Code, &c.Name, &c.CurrencyCode, &c.IsActive,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}

		countries = append(countries, &CountryResponse{
			Code:         c.Code,
			Name:         c.Name,
			CurrencyCode: c.CurrencyCode,
			IsActive:     c.IsActive,
			CreatedAt:    c.CreatedAt,
			UpdatedAt:    c.UpdatedAt,
		})
	}

	return countries, nil
}

// ListRegionsByCountry retrieves all regions for a specific country, optionally filtered by is_active
func (s *GeoService) ListRegionsByCountry(countryCode string, isActive *bool) ([]*RegionResponse, error) {
	query := `
		SELECT
			id, country_code, code, name, type, is_active, created_at, updated_at
		FROM regions
		WHERE country_code = $1
	`

	args := []interface{}{countryCode}

	if isActive != nil {
		query += " AND is_active = $2"
		args = append(args, *isActive)
	}

	query += " ORDER BY name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	defer rows.Close()

	regions := []*RegionResponse{}
	for rows.Next() {
		var r Region
		err := rows.Scan(
			&r.ID, &r.CountryCode, &r.Code, &r.Name, &r.Type,
			&r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region: %w", err)
		}

		regions = append(regions, &RegionResponse{
			ID:          r.ID,
			CountryCode: r.CountryCode,
			Code:        r.Code,
			Name:        r.Name,
			Type:        r.Type,
			IsActive:    r.IsActive,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}

	return regions, nil
}

// CountryExists checks if a country code exists in the database
func (s *GeoService) CountryExists(countryCode string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM countries WHERE code = $1)
	`, countryCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check country: %w", err)
	}
	return exists, nil
}
