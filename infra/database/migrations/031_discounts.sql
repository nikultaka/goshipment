-- =====================================================
-- 031_discounts.sql
-- Discounts / Promo Codes + Redemptions (Organization-Scoped)
-- Default Currency: CAD
-- =====================================================

-- ============================
-- ENUMS
-- ============================

CREATE TYPE discount_type_enum AS ENUM (
    'percent',
    'fixed_amount',
    'free_shipping'
);

CREATE TYPE discount_scope_enum AS ENUM (
    'invoice',
    'shipment'
);

CREATE TYPE discount_redemption_status_enum AS ENUM (
    'applied',
    'voided'
);

-- ============================
-- DISCOUNTS (PROMO CODES)
-- ============================

CREATE TABLE discounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discount_number VARCHAR(20) UNIQUE,

    -- -------------------------------------------------
    -- OWNERSHIP / SCOPE
    -- -------------------------------------------------
    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    created_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    -- -------------------------------------------------
    -- CODE IDENTITY
    -- -------------------------------------------------
    code VARCHAR(50) NOT NULL,
    name VARCHAR(120),
    description TEXT,

    -- -------------------------------------------------
    -- DISCOUNT DEFINITION
    -- -------------------------------------------------
    discount_type discount_type_enum NOT NULL,
    value NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',
    scope discount_scope_enum NOT NULL DEFAULT 'invoice',

    -- -------------------------------------------------
    -- ELIGIBILITY RULES
    -- -------------------------------------------------
    min_subtotal_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    max_discount_amount NUMERIC(14,2),
    allowed_country CHAR(2),

    -- -------------------------------------------------
    -- LIMITS
    -- -------------------------------------------------
    max_uses_total INTEGER,
    max_uses_per_user INTEGER,
    max_total_discount_amount NUMERIC(14,2),

    -- -------------------------------------------------
    -- VALIDITY
    -- -------------------------------------------------
    starts_at TIMESTAMP,
    ends_at TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    first_time_only BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_discount_free_shipping_scope
        CHECK (
            discount_type <> 'free_shipping'
            OR scope = 'shipment'
        ),

    CONSTRAINT discounts_code_unique_per_org
        UNIQUE (organization_id, code),

    CONSTRAINT discounts_value_positive_chk
        CHECK (value > 0),

    CONSTRAINT discounts_min_subtotal_non_negative_chk
        CHECK (min_subtotal_amount >= 0),

    CONSTRAINT discounts_max_discount_non_negative_chk
        CHECK (max_discount_amount IS NULL OR max_discount_amount >= 0),

    CONSTRAINT discounts_percent_range_chk
        CHECK (
            (discount_type = 'percent' AND value > 0 AND value <= 100)
            OR
            (discount_type <> 'percent' AND value > 0)
        )
);

CREATE INDEX idx_discounts_org_id ON discounts(organization_id);
CREATE INDEX idx_discounts_code ON discounts(code);
CREATE INDEX idx_discounts_active ON discounts(is_active);
CREATE INDEX idx_discounts_window ON discounts(starts_at, ends_at);
CREATE INDEX idx_discounts_scope ON discounts(scope);


-- =====================================================
-- DISCOUNT ALLOWED USERS
-- =====================================================

CREATE TABLE discount_allowed_users (
    discount_id UUID NOT NULL
        REFERENCES discounts(id)
        ON DELETE CASCADE,

    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (discount_id, user_id)
);

CREATE INDEX idx_discount_allowed_users_user
    ON discount_allowed_users(user_id);


-- ============================
-- DISCOUNT REDEMPTIONS
-- ============================

CREATE TABLE discount_redemptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    usage_number VARCHAR(20) UNIQUE,

    -- -------------------------------------------------
    -- OWNERSHIP
    -- -------------------------------------------------
    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    discount_id UUID NOT NULL
        REFERENCES discounts(id)
        ON DELETE CASCADE,

    redeemed_by_user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- LINKAGE
    -- -------------------------------------------------
    invoice_id UUID,
    -- FK to invoices deferred to 035_indexes.sql (invoices created in 034)

    shipment_id UUID
        REFERENCES shipments(id)
        ON DELETE SET NULL,

    status discount_redemption_status_enum NOT NULL DEFAULT 'applied',

    -- -------------------------------------------------
    -- SNAPSHOT (WHAT WAS APPLIED)
    -- -------------------------------------------------
    currency CHAR(3) NOT NULL DEFAULT 'CAD',
    discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,

    redeemed_at TIMESTAMP NOT NULL DEFAULT NOW(),

    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT discount_redemptions_amount_non_negative_chk
        CHECK (discount_amount >= 0),

    CONSTRAINT discount_redemptions_invoice_or_shipment_chk
        CHECK (
            invoice_id IS NOT NULL OR shipment_id IS NOT NULL
        )
);

CREATE INDEX idx_discount_redemptions_org_id
    ON discount_redemptions(organization_id);

CREATE INDEX idx_discount_redemptions_discount_id
    ON discount_redemptions(discount_id);

CREATE INDEX idx_discount_redemptions_user_id
    ON discount_redemptions(redeemed_by_user_id);

CREATE INDEX idx_discount_redemptions_shipment_id
    ON discount_redemptions(shipment_id);

CREATE INDEX idx_discount_redemptions_redeemed_at
    ON discount_redemptions(redeemed_at);
