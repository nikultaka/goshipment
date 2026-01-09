-- =====================================================
-- 026_shipment_estimates.sql
-- Shipment Estimates (Rate Quotes)
-- =====================================================
-- -----------------------------------------------------
-- ENUMS
-- -----------------------------------------------------

CREATE TYPE estimate_source_enum AS ENUM (
    'frontend',
    'dashboard',
    'api',
    'admin'
);

-- -----------------------------------------------------
-- TABLE: shipment_estimates
-- -----------------------------------------------------

CREATE TABLE shipment_estimates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- OWNERSHIP & ORIGIN
    -- -------------------------------------------------
    organization_id UUID,
    user_id UUID,

    source estimate_source_enum NOT NULL DEFAULT 'frontend',

    -- -------------------------------------------------
    -- OPTIONAL ADDRESS BOOK LINKS (UX ONLY)
    -- -------------------------------------------------
    from_address_book_id UUID,
    to_address_book_id UUID,

    -- -------------------------------------------------
    -- FROM ADDRESS (SNAPSHOT)
    -- -------------------------------------------------
    from_country CHAR(2) NOT NULL,
    from_region_code VARCHAR(10),
    from_city VARCHAR(100),
    from_postal_code VARCHAR(20) NOT NULL,

    -- -------------------------------------------------
    -- TO ADDRESS (SNAPSHOT)
    -- -------------------------------------------------
    to_country CHAR(2) NOT NULL,
    to_region_code VARCHAR(10),
    to_city VARCHAR(100),
    to_postal_code VARCHAR(20) NOT NULL,

    -- -------------------------------------------------
    -- PACKAGE AGGREGATES
    -- -------------------------------------------------
    total_packages INTEGER NOT NULL DEFAULT 0,
    total_weight NUMERIC(14,3) NOT NULL DEFAULT 0,
    weight_unit weight_unit_enum NOT NULL DEFAULT 'lb',

    -- -------------------------------------------------
    -- PRICING TOTALS
    -- -------------------------------------------------
    subtotal_amount NUMERIC(14,2),
    taxes_amount NUMERIC(14,2),
    total_amount NUMERIC(14,2),
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- VALIDITY & LOCKING
    -- -------------------------------------------------
    expires_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    locked_at TIMESTAMP WITHOUT TIME ZONE,

    -- -------------------------------------------------
    -- SELECTED RATE
    -- -------------------------------------------------
    selected_carrier_id UUID,
    selected_servicelevel_name VARCHAR(150),
    selected_servicelevel_code VARCHAR(100), -- e.g. FEDEX_GROUND, UPS_STANDARD, DHL_EXPRESS
    selected_amount NUMERIC(14,2),
    rate_selected_at TIMESTAMP WITHOUT TIME ZONE,

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT fk_estimate_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id),

    CONSTRAINT fk_estimate_user
        FOREIGN KEY (user_id)
        REFERENCES users(id),

    CONSTRAINT fk_estimate_carrier
        FOREIGN KEY (selected_carrier_id)
        REFERENCES carriers(id),

    CONSTRAINT fk_estimate_from_country
        FOREIGN KEY (from_country)
        REFERENCES countries(code),

    CONSTRAINT fk_estimate_to_country
        FOREIGN KEY (to_country)
        REFERENCES countries(code),

    CONSTRAINT chk_estimate_scope
        CHECK (
            (organization_id IS NULL AND user_id IS NULL)
            OR (organization_id IS NOT NULL)
        ),

    CONSTRAINT chk_estimate_currency_cad
        CHECK (currency = 'CAD'),

    CONSTRAINT chk_estimate_rate_when_locked
        CHECK (
            is_locked = FALSE
            OR rate_selected_at IS NOT NULL
        ),

    CONSTRAINT chk_estimate_not_expired_when_selected
        CHECK (
            rate_selected_at IS NULL
            OR expires_at > rate_selected_at
        )
);

-- -----------------------------------------------------
-- INDEXES
-- -----------------------------------------------------

CREATE INDEX idx_estimate_org ON shipment_estimates(organization_id);
CREATE INDEX idx_estimate_user ON shipment_estimates(user_id);
CREATE INDEX idx_estimate_source ON shipment_estimates(source);
CREATE INDEX idx_estimate_created_at ON shipment_estimates(created_at);
CREATE INDEX idx_estimate_expires_at ON shipment_estimates(expires_at);
CREATE INDEX idx_estimate_selected_carrier ON shipment_estimates(selected_carrier_id);

