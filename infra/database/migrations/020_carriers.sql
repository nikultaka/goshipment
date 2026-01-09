-- =====================================================
-- 020_carriers.sql
-- Carriers (Direct API Integrations)
-- =====================================================
-- Supported: FedEx, UPS, Purolator, DHL Express, Canada Post
-- API credentials stored in .env / secret manager (NOT in database)
-- =====================================================
-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE carrier_type_enum AS ENUM (
    'parcel',           -- Standard parcel shipping (our focus)
    'freight_ltl',      -- Less than truckload (future)
    'freight_ftl'       -- Full truckload (future)
);

CREATE TYPE service_level_category_enum AS ENUM (
    'ground',           -- 3-7 business days
    'express',          -- 2-3 business days
    'priority',         -- Next day / overnight
    'same_day',         -- Same day delivery
    'economy',          -- Cheapest, slowest
    'international'     -- Cross-border
);

CREATE TYPE day_of_week_enum AS ENUM (
    'monday',
    'tuesday',
    'wednesday',
    'thursday',
    'friday',
    'saturday',
    'sunday'
);

-- =====================================================
-- CARRIERS
-- =====================================================
-- Platform-wide carrier definitions
-- One row per supported carrier

CREATE TABLE carriers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- IDENTITY
    -- -------------------------------------------------
    code VARCHAR(50) NOT NULL UNIQUE,           -- fedex, ups, dhl_express, canada_post, purolator
    name VARCHAR(150) NOT NULL,                 -- FedEx, UPS, DHL Express, Canada Post, Purolator
    display_name VARCHAR(150),                  -- Optional display override

    -- -------------------------------------------------
    -- CLASSIFICATION
    -- -------------------------------------------------
    carrier_type carrier_type_enum NOT NULL DEFAULT 'parcel',

    -- -------------------------------------------------
    -- CAPABILITIES
    -- -------------------------------------------------
    supports_domestic BOOLEAN NOT NULL DEFAULT TRUE,
    supports_international BOOLEAN NOT NULL DEFAULT FALSE,
    supports_pickup BOOLEAN NOT NULL DEFAULT TRUE,
    supports_drop_off BOOLEAN NOT NULL DEFAULT TRUE,
    supports_returns BOOLEAN NOT NULL DEFAULT FALSE,
    supports_labels BOOLEAN NOT NULL DEFAULT TRUE,
    supports_tracking BOOLEAN NOT NULL DEFAULT TRUE,
    supports_insurance BOOLEAN NOT NULL DEFAULT FALSE,
    supports_signature BOOLEAN NOT NULL DEFAULT TRUE,
    supports_cod BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- API INTEGRATION (Non-sensitive)
    -- -------------------------------------------------
    api_version VARCHAR(20),                    -- e.g., 'v1', '2024-01'
    api_base_url VARCHAR(255),                  -- e.g., 'https://apis.fedex.com'
    -- Credentials resolved from env: {CODE}_CLIENT_ID, {CODE}_CLIENT_SECRET

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,   -- Show in UI / allow new shipments
    is_active BOOLEAN NOT NULL DEFAULT TRUE,    -- System-wide availability

    -- -------------------------------------------------
    -- UI
    -- -------------------------------------------------
    logo_file_id UUID,
    sort_order INTEGER NOT NULL DEFAULT 0,
    color_hex VARCHAR(7),

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- CARRIER SETTINGS
-- =====================================================
-- Platform-wide operational settings per carrier
-- Pickup times are in carrier_operating_schedule (per day)

CREATE TABLE carrier_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL UNIQUE
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- PICKUP CONFIGURATION
    -- -------------------------------------------------
    min_pickup_lead_hours INTEGER DEFAULT 2,
    max_pickup_days_ahead INTEGER DEFAULT 14,

    -- -------------------------------------------------
    -- PACKAGE LIMITS
    -- -------------------------------------------------
    max_packages_per_shipment INTEGER DEFAULT 10,
    max_package_weight_kg NUMERIC(10,2),
    max_package_length_cm NUMERIC(10,2),
    max_declared_value NUMERIC(14,2),

    -- -------------------------------------------------
    -- PRICING (Platform markup)
    -- -------------------------------------------------
    rate_multiplier NUMERIC(6,4) NOT NULL DEFAULT 1.0000,
    flat_markup_amount NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    pickup_fee_amount NUMERIC(10,2) NOT NULL DEFAULT 0.00,

    -- -------------------------------------------------
    -- DISCOUNTS
    -- -------------------------------------------------
    carrier_discount_percent NUMERIC(5,2),
    pass_through_discount_percent NUMERIC(5,2),

    -- -------------------------------------------------
    -- FUEL SURCHARGE
    -- -------------------------------------------------
    fuel_surcharge_percent NUMERIC(5,2),
    fuel_surcharge_updated_at TIMESTAMP,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_rate_multiplier_positive
        CHECK (rate_multiplier > 0),

    CONSTRAINT chk_flat_markup_non_negative
        CHECK (flat_markup_amount >= 0),

    CONSTRAINT chk_pickup_fee_non_negative
        CHECK (pickup_fee_amount >= 0),

    CONSTRAINT chk_carrier_discount_range
        CHECK (
            carrier_discount_percent IS NULL
            OR (carrier_discount_percent >= 0 AND carrier_discount_percent <= 100)
        )
);

-- =====================================================
-- CARRIER OPERATING SCHEDULE
-- =====================================================
-- Per-day pickup windows and cutoff times
-- 7 rows per carrier (one per day of week)

CREATE TABLE carrier_operating_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    day_of_week day_of_week_enum NOT NULL,

    -- -------------------------------------------------
    -- OPERATING STATUS
    -- -------------------------------------------------
    is_operating BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- PICKUP WINDOW
    -- -------------------------------------------------
    pickup_start_time TIME,
    pickup_end_time TIME,

    -- -------------------------------------------------
    -- CUTOFF TIMES
    -- -------------------------------------------------
    same_day_cutoff_time TIME,
    next_day_cutoff_time TIME,

    -- -------------------------------------------------
    -- NOTES
    -- -------------------------------------------------
    notes TEXT,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT uq_carrier_day UNIQUE (carrier_id, day_of_week),

    CONSTRAINT chk_pickup_window
        CHECK (
            NOT is_operating 
            OR pickup_start_time IS NULL 
            OR pickup_end_time IS NULL 
            OR pickup_start_time < pickup_end_time
        ),

    CONSTRAINT chk_cutoff_within_window
        CHECK (
            same_day_cutoff_time IS NULL
            OR pickup_end_time IS NULL
            OR same_day_cutoff_time <= pickup_end_time
        )
);

-- =====================================================
-- CARRIER HOLIDAYS
-- =====================================================
-- Days when carriers don't operate or have modified hours

CREATE TABLE carrier_holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- HOLIDAY INFO
    -- -------------------------------------------------
    holiday_date DATE NOT NULL,
    holiday_name VARCHAR(100),

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_closed BOOLEAN NOT NULL DEFAULT TRUE,

    -- Modified hours if not fully closed
    modified_pickup_start TIME,
    modified_pickup_end TIME,
    modified_cutoff TIME,

    -- -------------------------------------------------
    -- SCOPE
    -- -------------------------------------------------
    country_code CHAR(2),
    region_code VARCHAR(10),

    notes TEXT,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT uq_carrier_holiday UNIQUE (carrier_id, holiday_date, country_code),

    CONSTRAINT chk_modified_hours
        CHECK (
            is_closed = TRUE
            OR (modified_pickup_start IS NOT NULL AND modified_pickup_end IS NOT NULL)
        )
);

-- =====================================================
-- CARRIER SERVICE LEVELS
-- =====================================================
-- Service options per carrier (Ground, Express, Priority, etc.)

CREATE TABLE carrier_service_levels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- IDENTITY
    -- -------------------------------------------------
    code VARCHAR(100) NOT NULL,                 -- FEDEX_GROUND, UPS_NEXT_DAY_AIR
    carrier_code VARCHAR(100) NOT NULL,         -- Carrier's actual API code
    name VARCHAR(150) NOT NULL,
    display_name VARCHAR(150),

    -- -------------------------------------------------
    -- CLASSIFICATION
    -- -------------------------------------------------
    category service_level_category_enum NOT NULL DEFAULT 'ground',

    -- -------------------------------------------------
    -- DELIVERY ESTIMATES
    -- -------------------------------------------------
    min_transit_days INTEGER,
    max_transit_days INTEGER,
    delivery_time_description VARCHAR(100),

    -- -------------------------------------------------
    -- SCOPE
    -- -------------------------------------------------
    is_domestic BOOLEAN NOT NULL DEFAULT TRUE,
    is_international BOOLEAN NOT NULL DEFAULT FALSE,
    origin_countries CHAR(2)[],
    destination_countries CHAR(2)[],

    -- -------------------------------------------------
    -- CAPABILITIES
    -- -------------------------------------------------
    supports_pickup BOOLEAN NOT NULL DEFAULT TRUE,
    supports_saturday_delivery BOOLEAN NOT NULL DEFAULT FALSE,
    supports_signature BOOLEAN NOT NULL DEFAULT TRUE,
    supports_insurance BOOLEAN NOT NULL DEFAULT FALSE,
    requires_dimensions BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- RESTRICTIONS
    -- -------------------------------------------------
    max_weight_kg NUMERIC(10,2),
    max_length_cm NUMERIC(10,2),
    max_declared_value NUMERIC(14,2),

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT uq_carrier_service_code UNIQUE (carrier_id, code),

    CONSTRAINT chk_transit_days_valid
        CHECK (
            min_transit_days IS NULL
            OR max_transit_days IS NULL
            OR min_transit_days <= max_transit_days
        )
);

-- =====================================================
-- CARRIER PACKAGE TYPES
-- =====================================================
-- Package type codes per carrier

CREATE TABLE carrier_package_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- IDENTITY
    -- -------------------------------------------------
    code VARCHAR(100) NOT NULL,
    carrier_code VARCHAR(100) NOT NULL,
    name VARCHAR(150) NOT NULL,
    display_name VARCHAR(150),

    -- -------------------------------------------------
    -- TYPE
    -- -------------------------------------------------
    is_custom BOOLEAN NOT NULL DEFAULT TRUE,
    is_carrier_provided BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- DIMENSIONS (For carrier-provided packaging)
    -- -------------------------------------------------
    length_cm NUMERIC(10,2),
    width_cm NUMERIC(10,2),
    height_cm NUMERIC(10,2),
    max_weight_kg NUMERIC(10,2),

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_carrier_package_code UNIQUE (carrier_id, code)
);

-- =====================================================
-- CARRIER SURCHARGES
-- =====================================================
-- Common surcharges for estimates/display

CREATE TABLE carrier_surcharges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- IDENTITY
    -- -------------------------------------------------
    code VARCHAR(100) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,

    -- -------------------------------------------------
    -- PRICING
    -- -------------------------------------------------
    surcharge_type VARCHAR(20) NOT NULL DEFAULT 'fixed',
    -- fixed | percent | per_package | per_kg

    amount NUMERIC(10,2),
    percent NUMERIC(5,2),

    -- -------------------------------------------------
    -- APPLICATION
    -- -------------------------------------------------
    applies_to_domestic BOOLEAN NOT NULL DEFAULT TRUE,
    applies_to_international BOOLEAN NOT NULL DEFAULT TRUE,
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    effective_date DATE,
    expiry_date DATE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_carrier_surcharge_code UNIQUE (carrier_id, code),

    CONSTRAINT chk_surcharge_amount
        CHECK (
            (surcharge_type = 'percent' AND percent IS NOT NULL)
            OR (surcharge_type <> 'percent' AND amount IS NOT NULL)
        )
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Carriers
CREATE INDEX idx_carriers_code ON carriers(code);
CREATE INDEX idx_carriers_enabled ON carriers(is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX idx_carriers_active ON carriers(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_carriers_sort ON carriers(sort_order);

-- Carrier Settings
CREATE INDEX idx_carrier_settings_carrier ON carrier_settings(carrier_id);
CREATE INDEX idx_carrier_settings_active ON carrier_settings(is_active) WHERE is_active = TRUE;

-- Operating Schedule
CREATE INDEX idx_carrier_schedule_carrier ON carrier_operating_schedule(carrier_id);
CREATE INDEX idx_carrier_schedule_day ON carrier_operating_schedule(day_of_week);
CREATE INDEX idx_carrier_schedule_operating ON carrier_operating_schedule(carrier_id, day_of_week) 
    WHERE is_operating = TRUE;

-- Holidays
CREATE INDEX idx_carrier_holidays_carrier ON carrier_holidays(carrier_id);
CREATE INDEX idx_carrier_holidays_date ON carrier_holidays(holiday_date);
-- CREATE INDEX idx_carrier_holidays_upcoming ON carrier_holidays(carrier_id, holiday_date) 
    -- WHERE holiday_date >= CURRENT_DATE;

-- Service Levels
CREATE INDEX idx_service_levels_carrier ON carrier_service_levels(carrier_id);
CREATE INDEX idx_service_levels_code ON carrier_service_levels(code);
CREATE INDEX idx_service_levels_category ON carrier_service_levels(category);
CREATE INDEX idx_service_levels_domestic ON carrier_service_levels(carrier_id) WHERE is_domestic = TRUE;
CREATE INDEX idx_service_levels_intl ON carrier_service_levels(carrier_id) WHERE is_international = TRUE;
CREATE INDEX idx_service_levels_enabled ON carrier_service_levels(carrier_id) WHERE is_enabled = TRUE;

-- Package Types
CREATE INDEX idx_package_types_carrier ON carrier_package_types(carrier_id);
CREATE INDEX idx_package_types_enabled ON carrier_package_types(carrier_id) WHERE is_enabled = TRUE;

-- Surcharges
CREATE INDEX idx_surcharges_carrier ON carrier_surcharges(carrier_id);
CREATE INDEX idx_surcharges_active ON carrier_surcharges(carrier_id) WHERE is_active = TRUE;

-- =====================================================
-- HELPER VIEW: Carrier Availability
-- =====================================================

CREATE OR REPLACE VIEW v_carrier_availability AS
SELECT 
    c.id AS carrier_id,
    c.code AS carrier_code,
    c.name AS carrier_name,
    c.is_enabled,
    c.is_active,
    cos.day_of_week,
    cos.is_operating,
    cos.pickup_start_time,
    cos.pickup_end_time,
    cos.same_day_cutoff_time,
    cos.next_day_cutoff_time,
    cos.notes AS schedule_notes
FROM carriers c
LEFT JOIN carrier_operating_schedule cos ON cos.carrier_id = c.id
WHERE c.is_enabled = TRUE 
  AND c.is_active = TRUE;

-- =====================================================
-- HELPER FUNCTION: Get Carrier Availability for Date
-- =====================================================

CREATE OR REPLACE FUNCTION get_carrier_availability(
    p_carrier_id UUID,
    p_date DATE
)
RETURNS TABLE (
    carrier_id UUID,
    carrier_code VARCHAR,
    is_available BOOLEAN,
    is_holiday BOOLEAN,
    holiday_name VARCHAR,
    pickup_start_time TIME,
    pickup_end_time TIME,
    same_day_cutoff_time TIME,
    notes TEXT
) AS $$
DECLARE
    v_day_of_week day_of_week_enum;
BEGIN
    v_day_of_week := TRIM(LOWER(TO_CHAR(p_date, 'day')))::day_of_week_enum;
    
    RETURN QUERY
    SELECT 
        c.id,
        c.code,
        (cos.is_operating AND NOT COALESCE(ch.is_closed, FALSE)),
        COALESCE(ch.is_closed, FALSE),
        ch.holiday_name::VARCHAR,
        COALESCE(ch.modified_pickup_start, cos.pickup_start_time),
        COALESCE(ch.modified_pickup_end, cos.pickup_end_time),
        COALESCE(ch.modified_cutoff, cos.same_day_cutoff_time),
        COALESCE(ch.notes, cos.notes)
    FROM carriers c
    LEFT JOIN carrier_operating_schedule cos 
        ON cos.carrier_id = c.id 
        AND cos.day_of_week = v_day_of_week
    LEFT JOIN carrier_holidays ch 
        ON ch.carrier_id = c.id 
        AND ch.holiday_date = p_date
    WHERE c.id = p_carrier_id;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- NOTES
-- =====================================================

/*
TABLE HIERARCHY:
────────────────────────────────────────────────────────
carriers
├── carrier_settings (1:1)
│   ├── Pickup lead time & max days
│   ├── Package limits
│   └── Pricing markup
│
├── carrier_operating_schedule (1:7)
│   ├── Per day of week
│   ├── Pickup windows
│   └── Cutoff times
│
├── carrier_holidays (1:many)
│   └── Closures & modified hours
│
├── carrier_service_levels (1:many)
│   └── Ground, Express, Priority, etc.
│
├── carrier_package_types (1:many)
│   └── Your Packaging, FedEx Pak, etc.
│
└── carrier_surcharges (1:many)
    └── Fuel, residential, etc.

API CREDENTIAL RESOLUTION:
────────────────────────────────────────────────────────
carrier.code = 'fedex'
→ FEDEX_CLIENT_ID
→ FEDEX_CLIENT_SECRET
→ FEDEX_ACCOUNT_NUMBER

carrier.code = 'ups'
→ UPS_CLIENT_ID
→ UPS_CLIENT_SECRET
→ UPS_ACCOUNT_NUMBER

carrier.code = 'purolator'
→ PUROLATOR_KEY
→ PUROLATOR_PASSWORD
→ PUROLATOR_ACCOUNT_NUMBER

carrier.code = 'dhl_express'
→ DHL_API_KEY
→ DHL_API_SECRET
→ DHL_ACCOUNT_NUMBER

carrier.code = 'canada_post'
→ CANADA_POST_API_KEY
→ CANADA_POST_API_SECRET
→ CANADA_POST_CUSTOMER_NUMBER

USAGE:
────────────────────────────────────────────────────────
-- Check availability for specific date:
SELECT * FROM get_carrier_availability(carrier_uuid, '2025-12-25');

-- Get all carriers with Monday schedule:
SELECT * FROM v_carrier_availability WHERE day_of_week = 'monday';
*/