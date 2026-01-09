-- =====================================================
-- 024_geo_reference.sql
-- Global Geographic Reference Tables
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE region_type_enum AS ENUM (
    'state',
    'province',
    'territory',
    'region'
);

-- =====================================================
-- COUNTRIES
-- ISO-3166-1 alpha-2
-- =====================================================

CREATE TABLE countries (
    code CHAR(2) PRIMARY KEY,
    -- ISO-3166-1 alpha-2 (CA, US, GB, etc.)

    name VARCHAR(100) NOT NULL,
    -- Display name

    currency_code CHAR(3) NOT NULL,
    -- ISO-4217 (CAD, USD, EUR, etc.)

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_countries_active
    ON countries(is_active);

-- =====================================================
-- REGIONS
-- States / Provinces / Territories / Regions
-- =====================================================

CREATE TABLE regions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    country_code CHAR(2) NOT NULL,
    -- FK → countries.code

    code VARCHAR(10) NOT NULL,
    -- Region code (ON, QC, CA, NY, TX, etc.)

    name VARCHAR(100) NOT NULL,
    -- Full region name

    type region_type_enum NOT NULL,

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_regions_country
        FOREIGN KEY (country_code)
        REFERENCES countries(code)
        ON DELETE CASCADE,

    CONSTRAINT uq_region_per_country
        UNIQUE (country_code, code)
);

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_regions_country
    ON regions(country_code);

CREATE INDEX idx_regions_country_active
    ON regions(country_code, is_active);

CREATE INDEX idx_regions_active
    ON regions(is_active);

-- =====================================================
-- NOTES
-- =====================================================
-- 1. Countries use ISO-3166-1 alpha-2 codes exclusively
-- 2. Regions are unified globally (single table)
-- 3. Transactional tables should store country_code + region_code
-- 4. Names are for display only
-- 5. Validation occurs at API level, not DB level
-- =====================================================
