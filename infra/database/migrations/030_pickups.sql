-- =====================================================
-- 030_pickups.sql
-- Pickups (Direct Carrier APIs, Org-Scoped)
-- =====================================================

-- =====================================================
-- ENUM: pickup_status_enum
-- =====================================================

CREATE TYPE pickup_status_enum AS ENUM (
    'draft',        -- created but not scheduled yet
    'scheduled',    -- scheduled successfully with carrier
    'rescheduled',  -- schedule changed after scheduling
    'cancelled',    -- cancelled by user/admin or provider
    'completed',    -- pickup completed successfully
    'failed'        -- scheduling or pickup failed
);

-- =====================================================
-- ENUM: pickup_building_type_enum
-- =====================================================

CREATE TYPE pickup_building_type_enum AS ENUM (
    'house',
    'apartment',
    'condo',
    'office',
    'warehouse',
    'retail',
    'industrial',
    'medical',
    'government',
    'other'
);

-- =====================================================
-- ENUM: pickup_location_enum
-- =====================================================

CREATE TYPE pickup_location_enum AS ENUM (
    'front_door',
    'basement',
    'between_doors',
    'counter',
    'desk',
    'front_desk',
    'back_door',
    'front_porch',
    'garage',
    'gate_house',
    'kiosk',
    'lab',
    'loading_dock',
    'lobby',
    'mailroom',
    'mailbox',
    'outside_door',
    'parts_department',
    'pharmacy',
    'pro_shop',
    'receiving',
    'security',
    'service_counter',
    'shipping',
    'side_door',
    'switchboard',
    'vault',
    'other'
);

-- =====================================================
-- TABLE: pickups
-- =====================================================

CREATE TABLE pickups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- HUMAN IDENTIFIER
    -- -------------------------------------------------
    pickup_number VARCHAR(20) NOT NULL UNIQUE,
    -- PKU-XXXXXXXX (app-generated)

    -- -------------------------------------------------
    -- OWNERSHIP / SCOPE
    -- -------------------------------------------------
    organization_id UUID NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id),

    -- -------------------------------------------------
    -- RELATIONSHIP
    -- -------------------------------------------------
    shipment_id UUID NOT NULL REFERENCES shipments(id),
    
    -- -------------------------------------------------
    -- CARRIER
    -- -------------------------------------------------
    carrier_id UUID NOT NULL REFERENCES carriers(id),

    -- -------------------------------------------------
    -- PICKUP ADDRESS (FROM ADDRESS BOOK)
    -- -------------------------------------------------
    pickup_address_book_id UUID NOT NULL REFERENCES address_book(id),

    -- -------------------------------------------------
    -- PICKUP ADDRESS SNAPSHOT (IMMUTABLE)
    -- -------------------------------------------------
    pickup_name VARCHAR(150),
    pickup_company VARCHAR(150),
    pickup_phone VARCHAR(50),
    pickup_email VARCHAR(150),

    pickup_country CHAR(2) NOT NULL REFERENCES countries(code),
    pickup_region_code VARCHAR(10),
    pickup_city VARCHAR(100) NOT NULL,
    pickup_postal_code VARCHAR(20) NOT NULL,
    pickup_address_line1 VARCHAR(255) NOT NULL,
    pickup_address_line2 VARCHAR(255),

    pickup_is_residential BOOLEAN DEFAULT FALSE,

    -- -------------------------------------------------
    -- LOCATION DETAILS
    -- -------------------------------------------------
    building_type pickup_building_type_enum NOT NULL DEFAULT 'house',
    pickup_location pickup_location_enum NOT NULL DEFAULT 'front_door',
    pickup_instructions TEXT,

    -- -------------------------------------------------
    -- SCHEDULING (NEXT-DAY OR LATER)
    -- -------------------------------------------------
    pickup_date DATE NOT NULL,
    pickup_window_start TIME NOT NULL,
    pickup_window_end   TIME NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Toronto',

    -- -------------------------------------------------
    -- PROVIDER RESPONSE
    -- -------------------------------------------------
    provider_pickup_id VARCHAR(150),
    provider_reference VARCHAR(150),
    provider_response JSONB,

    -- -------------------------------------------------
    -- POST-PICKUP ADJUSTMENTS (EMBEDDED)
    -- -------------------------------------------------
    adjustments_total NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    has_pending_adjustments BOOLEAN NOT NULL DEFAULT FALSE,
    adjustments_breakdown JSONB,
    -- [{ type, amount, reason, created_at }]

    invoice_id UUID,
    -- optional linkage to unpaid invoice

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    pickup_status pickup_status_enum NOT NULL DEFAULT 'draft',

    scheduled_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    failure_reason TEXT,

    -- -------------------------------------------------
    -- SAFETY / AUDIT
    -- -------------------------------------------------
    idempotency_key VARCHAR(100),
    revision INTEGER DEFAULT 1,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT fk_pickups_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id),

    CONSTRAINT chk_pickups_next_day_only
        CHECK (pickup_date >= (CURRENT_DATE + 1)),

    CONSTRAINT chk_pickups_window_valid
        CHECK (pickup_window_end > pickup_window_start),

    CONSTRAINT chk_pickups_instructions_for_other
        CHECK (
            pickup_location <> 'other'
            OR (pickup_instructions IS NOT NULL AND length(trim(pickup_instructions)) > 0)
        )
);

-- =====================================================
-- UNIQUENESS RULE
-- =====================================================

CREATE UNIQUE INDEX uq_pickups_active_carrier_address_day
    ON pickups (carrier_id, pickup_address_book_id, pickup_date)
    WHERE pickup_status IN ('draft', 'scheduled', 'rescheduled')
      AND is_deleted = FALSE;

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_pickups_org ON pickups(organization_id);
CREATE INDEX idx_pickups_shipment ON pickups(shipment_id);
CREATE INDEX idx_pickups_carrier ON pickups(carrier_id);
CREATE INDEX idx_pickups_pickup_date ON pickups(pickup_date);
CREATE INDEX idx_pickups_status ON pickups(pickup_status);
CREATE INDEX idx_pickups_address_book ON pickups(pickup_address_book_id);
CREATE INDEX idx_pickups_provider_pickup_id ON pickups(provider_pickup_id);
CREATE INDEX idx_pickups_created_at ON pickups(created_at);
