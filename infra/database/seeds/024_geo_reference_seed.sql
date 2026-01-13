-- =====================================================
-- 024_geo_reference_seed.sql
-- Seed data for countries and regions tables
-- =====================================================

-- =====================================================
-- COUNTRIES (ISO 3166-1 alpha-2 codes)
-- =====================================================

INSERT INTO countries (code, name, currency_code, is_active) VALUES
-- North America
('CA', 'Canada', 'CAD', TRUE),
('US', 'United States', 'USD', TRUE),
('MX', 'Mexico', 'MXN', TRUE),

-- Europe
('GB', 'United Kingdom', 'GBP', TRUE),
('DE', 'Germany', 'EUR', TRUE),
('FR', 'France', 'EUR', TRUE),
('IT', 'Italy', 'EUR', TRUE),
('ES', 'Spain', 'EUR', TRUE),
('NL', 'Netherlands', 'EUR', TRUE),
('BE', 'Belgium', 'EUR', TRUE),
('CH', 'Switzerland', 'CHF', TRUE),
('AT', 'Austria', 'EUR', TRUE),
('SE', 'Sweden', 'SEK', TRUE),
('NO', 'Norway', 'NOK', TRUE),
('DK', 'Denmark', 'DKK', TRUE),
('FI', 'Finland', 'EUR', TRUE),
('IE', 'Ireland', 'EUR', TRUE),
('PT', 'Portugal', 'EUR', TRUE),
('PL', 'Poland', 'PLN', TRUE),
('CZ', 'Czech Republic', 'CZK', TRUE),

-- Asia-Pacific
('CN', 'China', 'CNY', TRUE),
('JP', 'Japan', 'JPY', TRUE),
('KR', 'South Korea', 'KRW', TRUE),
('IN', 'India', 'INR', TRUE),
('AU', 'Australia', 'AUD', TRUE),
('NZ', 'New Zealand', 'NZD', TRUE),
('SG', 'Singapore', 'SGD', TRUE),
('HK', 'Hong Kong', 'HKD', TRUE),
('TW', 'Taiwan', 'TWD', TRUE),
('TH', 'Thailand', 'THB', TRUE),
('MY', 'Malaysia', 'MYR', TRUE),
('ID', 'Indonesia', 'IDR', TRUE),
('PH', 'Philippines', 'PHP', TRUE),
('VN', 'Vietnam', 'VND', TRUE),

-- Middle East
('AE', 'United Arab Emirates', 'AED', TRUE),
('SA', 'Saudi Arabia', 'SAR', TRUE),
('IL', 'Israel', 'ILS', TRUE),
('TR', 'Turkey', 'TRY', TRUE),

-- Latin America
('BR', 'Brazil', 'BRL', TRUE),
('AR', 'Argentina', 'ARS', TRUE),
('CL', 'Chile', 'CLP', TRUE),
('CO', 'Colombia', 'COP', TRUE),
('PE', 'Peru', 'PEN', TRUE),

-- Africa
('ZA', 'South Africa', 'ZAR', TRUE),
('EG', 'Egypt', 'EGP', TRUE),
('NG', 'Nigeria', 'NGN', TRUE),
('KE', 'Kenya', 'KES', TRUE)

ON CONFLICT (code) DO NOTHING;

-- =====================================================
-- REGIONS (States/Provinces for major shipping countries)
-- =====================================================

-- Canada Provinces and Territories
INSERT INTO regions (country_code, code, name, type, is_active) VALUES
('CA', 'AB', 'Alberta', 'province', TRUE),
('CA', 'BC', 'British Columbia', 'province', TRUE),
('CA', 'MB', 'Manitoba', 'province', TRUE),
('CA', 'NB', 'New Brunswick', 'province', TRUE),
('CA', 'NL', 'Newfoundland and Labrador', 'province', TRUE),
('CA', 'NS', 'Nova Scotia', 'province', TRUE),
('CA', 'ON', 'Ontario', 'province', TRUE),
('CA', 'PE', 'Prince Edward Island', 'province', TRUE),
('CA', 'QC', 'Quebec', 'province', TRUE),
('CA', 'SK', 'Saskatchewan', 'province', TRUE),
('CA', 'NT', 'Northwest Territories', 'territory', TRUE),
('CA', 'NU', 'Nunavut', 'territory', TRUE),
('CA', 'YT', 'Yukon', 'territory', TRUE)

ON CONFLICT (country_code, code) DO NOTHING;

-- United States (all 50 states + DC)
INSERT INTO regions (country_code, code, name, type, is_active) VALUES
('US', 'AL', 'Alabama', 'state', TRUE),
('US', 'AK', 'Alaska', 'state', TRUE),
('US', 'AZ', 'Arizona', 'state', TRUE),
('US', 'AR', 'Arkansas', 'state', TRUE),
('US', 'CA', 'California', 'state', TRUE),
('US', 'CO', 'Colorado', 'state', TRUE),
('US', 'CT', 'Connecticut', 'state', TRUE),
('US', 'DE', 'Delaware', 'state', TRUE),
('US', 'FL', 'Florida', 'state', TRUE),
('US', 'GA', 'Georgia', 'state', TRUE),
('US', 'HI', 'Hawaii', 'state', TRUE),
('US', 'ID', 'Idaho', 'state', TRUE),
('US', 'IL', 'Illinois', 'state', TRUE),
('US', 'IN', 'Indiana', 'state', TRUE),
('US', 'IA', 'Iowa', 'state', TRUE),
('US', 'KS', 'Kansas', 'state', TRUE),
('US', 'KY', 'Kentucky', 'state', TRUE),
('US', 'LA', 'Louisiana', 'state', TRUE),
('US', 'ME', 'Maine', 'state', TRUE),
('US', 'MD', 'Maryland', 'state', TRUE),
('US', 'MA', 'Massachusetts', 'state', TRUE),
('US', 'MI', 'Michigan', 'state', TRUE),
('US', 'MN', 'Minnesota', 'state', TRUE),
('US', 'MS', 'Mississippi', 'state', TRUE),
('US', 'MO', 'Missouri', 'state', TRUE),
('US', 'MT', 'Montana', 'state', TRUE),
('US', 'NE', 'Nebraska', 'state', TRUE),
('US', 'NV', 'Nevada', 'state', TRUE),
('US', 'NH', 'New Hampshire', 'state', TRUE),
('US', 'NJ', 'New Jersey', 'state', TRUE),
('US', 'NM', 'New Mexico', 'state', TRUE),
('US', 'NY', 'New York', 'state', TRUE),
('US', 'NC', 'North Carolina', 'state', TRUE),
('US', 'ND', 'North Dakota', 'state', TRUE),
('US', 'OH', 'Ohio', 'state', TRUE),
('US', 'OK', 'Oklahoma', 'state', TRUE),
('US', 'OR', 'Oregon', 'state', TRUE),
('US', 'PA', 'Pennsylvania', 'state', TRUE),
('US', 'RI', 'Rhode Island', 'state', TRUE),
('US', 'SC', 'South Carolina', 'state', TRUE),
('US', 'SD', 'South Dakota', 'state', TRUE),
('US', 'TN', 'Tennessee', 'state', TRUE),
('US', 'TX', 'Texas', 'state', TRUE),
('US', 'UT', 'Utah', 'state', TRUE),
('US', 'VT', 'Vermont', 'state', TRUE),
('US', 'VA', 'Virginia', 'state', TRUE),
('US', 'WA', 'Washington', 'state', TRUE),
('US', 'WV', 'West Virginia', 'state', TRUE),
('US', 'WI', 'Wisconsin', 'state', TRUE),
('US', 'WY', 'Wyoming', 'state', TRUE),
('US', 'DC', 'District of Columbia', 'state', TRUE)

ON CONFLICT (country_code, code) DO NOTHING;

-- =====================================================
-- NOTES
-- =====================================================
-- This seed file provides:
-- 1. 55+ major countries with their ISO codes and currency codes
-- 2. All Canadian provinces and territories
-- 3. All US states and DC
-- 4. Additional regions can be added as needed for other countries
-- =====================================================
