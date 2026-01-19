-- Quick fix: Populate countries table with essential shipping countries
-- This fixes the foreign key constraint error in shipment-estimates

-- Insert countries
INSERT INTO countries (code, name, currency_code, is_active) VALUES
('CA', 'Canada', 'CAD', TRUE),
('US', 'United States', 'USD', TRUE),
('MX', 'Mexico', 'MXN', TRUE),
('GB', 'United Kingdom', 'GBP', TRUE),
('DE', 'Germany', 'EUR', TRUE),
('FR', 'France', 'EUR', TRUE),
('IT', 'Italy', 'EUR', TRUE),
('ES', 'Spain', 'EUR', TRUE),
('AU', 'Australia', 'AUD', TRUE),
('NZ', 'New Zealand', 'NZD', TRUE),
('CN', 'China', 'CNY', TRUE),
('JP', 'Japan', 'JPY', TRUE),
('KR', 'South Korea', 'KRW', TRUE),
('IN', 'India', 'INR', TRUE),
('SG', 'Singapore', 'SGD', TRUE),
('HK', 'Hong Kong', 'HKD', TRUE),
('BR', 'Brazil', 'BRL', TRUE),
('NL', 'Netherlands', 'EUR', TRUE),
('BE', 'Belgium', 'EUR', TRUE),
('CH', 'Switzerland', 'CHF', TRUE)
ON CONFLICT (code) DO NOTHING;

-- Insert Canadian provinces
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

-- Insert US states (major ones)
INSERT INTO regions (country_code, code, name, type, is_active) VALUES
('US', 'CA', 'California', 'state', TRUE),
('US', 'NY', 'New York', 'state', TRUE),
('US', 'TX', 'Texas', 'state', TRUE),
('US', 'FL', 'Florida', 'state', TRUE),
('US', 'IL', 'Illinois', 'state', TRUE),
('US', 'PA', 'Pennsylvania', 'state', TRUE),
('US', 'OH', 'Ohio', 'state', TRUE),
('US', 'GA', 'Georgia', 'state', TRUE),
('US', 'NC', 'North Carolina', 'state', TRUE),
('US', 'MI', 'Michigan', 'state', TRUE),
('US', 'NJ', 'New Jersey', 'state', TRUE),
('US', 'VA', 'Virginia', 'state', TRUE),
('US', 'WA', 'Washington', 'state', TRUE),
('US', 'AZ', 'Arizona', 'state', TRUE),
('US', 'MA', 'Massachusetts', 'state', TRUE),
('US', 'TN', 'Tennessee', 'state', TRUE),
('US', 'IN', 'Indiana', 'state', TRUE),
('US', 'MO', 'Missouri', 'state', TRUE),
('US', 'MD', 'Maryland', 'state', TRUE),
('US', 'WI', 'Wisconsin', 'state', TRUE)
ON CONFLICT (country_code, code) DO NOTHING;

-- Verify the data
SELECT 'Countries inserted:' as message, COUNT(*) as count FROM countries;
SELECT 'Regions inserted:' as message, COUNT(*) as count FROM regions;
