-- =====================================================
-- 019_address_book.sql
-- Address Book (Organization-scoped)
-- =====================================================
-- =====================================================
-- ADDRESS TYPE ENUM
-- =====================================================
CREATE TYPE address_type_enum AS ENUM (
    'sender',
    'recipient',
    'both'
);

-- =====================================================
-- ADDRESS BOOK
-- =====================================================
CREATE TABLE address_book (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Human-readable identifier
    address_number VARCHAR(20) UNIQUE NOT NULL, -- ADR-XXXXXXXX

    -- Ownership
    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    -- Audit
    created_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    -- Identity / Labeling
    nickname VARCHAR(100) NOT NULL,
    company VARCHAR(150),
    contact_name VARCHAR(150) NOT NULL,

    phone VARCHAR(30),
    email VARCHAR(255),

    -- Address lines
    address_line_1 VARCHAR(255) NOT NULL,
    address_line_2 VARCHAR(255),

    city VARCHAR(100) NOT NULL,
    province_code VARCHAR(50),
    postal_code VARCHAR(20) NOT NULL,
    country_code CHAR(2) NOT NULL,

    -- Shipping semantics
    address_type address_type_enum NOT NULL DEFAULT 'both',
    is_residential BOOLEAN DEFAULT FALSE,

    -- Defaults & status
    -- is_default BOOLEAN DEFAULT FALSE,
    is_default_sender BOOLEAN NOT NULL DEFAULT FALSE,
    is_default_recipient BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- =================================================
    -- CONSTRAINTS
    -- =================================================
    CONSTRAINT chk_address_number_format
        CHECK (address_number ~ '^ADR-[A-Z0-9]{8}$')
);

-- =====================================================
-- INDEXES
-- =====================================================
CREATE UNIQUE INDEX ux_org_default_sender
ON address_book (organization_id)
WHERE is_default_sender = true;

CREATE UNIQUE INDEX ux_org_default_recipient
ON address_book (organization_id)
WHERE is_default_recipient = true;

CREATE INDEX idx_address_book_org
    ON address_book(organization_id);

CREATE INDEX idx_address_book_type
    ON address_book(address_type);

CREATE INDEX idx_address_book_active
    ON address_book(is_active);

CREATE INDEX idx_address_book_created_by
    ON address_book(created_by_user_id);

-- =====================================================
-- NOTES
-- =====================================================
-- Ownership:
-- - Addresses belong to organizations ONLY
-- - Users and teams consume addresses via organization access
--
-- address_number:
-- - Generated at API level
-- - Format: ADR-XXXXXXXX (uppercase alphanumeric)
--
-- Defaults:
-- - Only one default address per organization should exist
--   (enforced at application level)
--
-- address_type_enum usage:
-- - Ship From selector → ('sender','both')
-- - Ship To selector   → ('recipient','both')
--
-- is_residential maps directly to carrier "residential" flag
-- =====================================================
