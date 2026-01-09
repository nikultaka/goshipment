-- =====================================================
-- 022_organization_preferences.sql
-- Organization Preferences & Carrier Defaults
-- =====================================================
-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE signature_option_enum AS ENUM (
    'none',
    'signature_required',
    'adult_signature_required'
);

CREATE TYPE label_format_enum AS ENUM (
    'thermal_4x6',
    'letter_8_5x11'
);

CREATE TYPE weight_unit_enum AS ENUM (
    'kg',
    'lb'
);

CREATE TYPE dimension_unit_enum AS ENUM (
    'cm',
    'in'
);

-- =====================================================
-- ORGANIZATION PREFERENCES
-- =====================================================
-- User-controlled settings for shipping behavior
-- All users in org inherit these settings

CREATE TABLE organization_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    organization_id UUID NOT NULL UNIQUE
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- SHIPPING DEFAULTS (UX pre-fill)
    -- -------------------------------------------------
    default_sender_address_id UUID,
    default_recipient_address_id UUID,
    -- FKs added after address_book exists

    -- -------------------------------------------------
    -- SHIPPING OPTIONS
    -- -------------------------------------------------
    package_reference VARCHAR(150),
    signature_option signature_option_enum NOT NULL DEFAULT 'none',
    delivery_instructions TEXT,

    -- -------------------------------------------------
    -- UNITS & LOCALE
    -- -------------------------------------------------
    weight_unit weight_unit_enum NOT NULL DEFAULT 'kg',
    dimension_unit dimension_unit_enum NOT NULL DEFAULT 'cm',
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Toronto',

    -- -------------------------------------------------
    -- LABEL & DISPLAY
    -- -------------------------------------------------
    label_format label_format_enum NOT NULL DEFAULT 'thermal_4x6',
    hide_shipper_phone_on_label BOOLEAN NOT NULL DEFAULT FALSE,
    show_usd_estimates BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- ORGANIZATION CARRIER DEFAULTS
-- =====================================================
-- Per-carrier settings for each organization
-- Pickup times, default carrier selection, enable/disable

CREATE TABLE organization_carrier_defaults (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    carrier_id UUID NOT NULL
        REFERENCES carriers(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- DEFAULT CARRIER FLAG
    -- -------------------------------------------------
    is_default BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- ENABLE/DISABLE CARRIER FOR THIS ORG
    -- -------------------------------------------------
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- PICKUP DEFAULTS (Per-carrier)
    -- -------------------------------------------------
    pickup_start_time TIME,
    pickup_end_time TIME,
    pickup_instructions TEXT,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT uq_org_carrier
        UNIQUE (organization_id, carrier_id),

    CONSTRAINT chk_pickup_time_window
        CHECK (
            pickup_start_time IS NULL
            OR pickup_end_time IS NULL
            OR pickup_start_time < pickup_end_time
        )
);

-- Only ONE default carrier per org
CREATE UNIQUE INDEX ux_org_default_carrier
ON organization_carrier_defaults (organization_id)
WHERE is_default = TRUE;

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_org_preferences_org
    ON organization_preferences(organization_id);

CREATE INDEX idx_org_carrier_defaults_org
    ON organization_carrier_defaults(organization_id);

CREATE INDEX idx_org_carrier_defaults_carrier
    ON organization_carrier_defaults(carrier_id);

CREATE INDEX idx_org_carrier_defaults_enabled
    ON organization_carrier_defaults(organization_id)
    WHERE is_enabled = TRUE;

-- =====================================================
-- ADD FKs (after address_book exists)
-- =====================================================
-- ALTER TABLE organization_preferences
--     ADD CONSTRAINT fk_org_prefs_sender_address
--     FOREIGN KEY (default_sender_address_id)
--     REFERENCES address_book(id)
--     ON DELETE SET NULL;
--
-- ALTER TABLE organization_preferences
--     ADD CONSTRAINT fk_org_prefs_recipient_address
--     FOREIGN KEY (default_recipient_address_id)
--     REFERENCES address_book(id)
--     ON DELETE SET NULL;

-- =====================================================
-- NOTES
-- =====================================================

/*
SETTINGS HIERARCHY:
────────────────────────────────────────────────────────
1. organization_preferences (user-controlled)
   - Shipping defaults
   - Units, timezone
   - Label format

2. organization_carrier_defaults (user-controlled, per-carrier)
   - Default carrier selection
   - Enable/disable carriers
   - Pickup window defaults

3. organization_settings (admin-controlled) - separate table
   - allow_same_day_pickup
   - allow_next_day_pickup
   - Feature flags

USER PREFERENCES (in users table):
────────────────────────────────────────────────────────
- timezone (display)
- locale
- date_format
- time_format
- ui_theme
- in_app_notifications

These are PERSONAL UI preferences, not shipping settings.
*/