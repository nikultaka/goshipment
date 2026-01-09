-- =====================================================
-- 018_business.sql
-- MODULE 26: Businesses (Organization-Scoped)
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE business_type_enum AS ENUM (
    'individual',
    'business',
    'nonprofit',
    'government'
);

-- =====================================================
-- BUSINESSES
-- One business per organization (hard constraint)
-- =====================================================

CREATE TABLE businesses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope
    organization_id UUID NOT NULL UNIQUE REFERENCES organizations(id),

    -- Identity
    business_name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),

    business_type business_type_enum,

    -- -------------------------------------------------
    -- COMPLIANCE (Optional)
    -- -------------------------------------------------
    tax_id VARCHAR(100),                        -- GST/HST number, EIN, etc.
    tax_id_verified BOOLEAN NOT NULL DEFAULT FALSE,
    tax_id_verified_at TIMESTAMP,

    -- Branding
    avatar_file_id UUID
        REFERENCES files(id) ON DELETE SET NULL,

    -- Public Info
    website VARCHAR(255),
    description TEXT,

    -- Address
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    province VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(2),

    -- -------------------------------------------------
    -- B2B COMPLIANCE (Optional)
    -- For enterprise relationships
    -- -------------------------------------------------
    dpa_signed_at TIMESTAMP,                    -- Data Processing Agreement
    dpa_signatory_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,
    dpa_document_file_id UUID
        REFERENCES files(id)
        ON DELETE SET NULL,
    compliance_contact_email VARCHAR(255),
    billing_contact_email VARCHAR(255),

     -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE, -- Business verification status
    verified_at TIMESTAMP,
    verified_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =====================================================
-- BUSINESS EMAILS
-- =====================================================

CREATE TABLE business_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    business_id UUID NOT NULL
        REFERENCES businesses(id) ON DELETE CASCADE,

    email VARCHAR(255) NOT NULL,
    email_lower VARCHAR(255)
        GENERATED ALWAYS AS (lower(email)) STORED,

    email_type VARCHAR(30) NOT NULL DEFAULT 'general',
    -- general | billing | support | accounting | notifications

    is_primary BOOLEAN NOT NULL DEFAULT false,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMP,

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (business_id, email_lower)
);

-- Only one primary email per business
CREATE UNIQUE INDEX ux_business_primary_email
ON business_emails (business_id)
WHERE is_primary = true;

-- =====================================================
-- BUSINESS PHONES
-- =====================================================

CREATE TABLE business_phones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    business_id UUID NOT NULL
        REFERENCES businesses(id) ON DELETE CASCADE,

    phone VARCHAR(20) NOT NULL,
    phone_e164 VARCHAR(20),

    phone_type VARCHAR(30) NOT NULL DEFAULT 'general',
    -- general | billing | support | sales | whatsapp

    is_primary BOOLEAN NOT NULL DEFAULT false,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMP,

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (business_id, phone_e164)
);

-- Only one primary phone per business
CREATE UNIQUE INDEX ux_business_primary_phone
ON business_phones (business_id)
WHERE is_primary = true;

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_businesses_org
    ON businesses(organization_id);

CREATE INDEX idx_businesses_avatar
    ON businesses(avatar_file_id);

CREATE INDEX idx_business_emails_business
    ON business_emails(business_id);

CREATE INDEX idx_business_phones_business
    ON business_phones(business_id);

-- =====================================================
-- NOTES
-- =====================================================
-- 1. One business per organization (enforced by UNIQUE org_id)
-- 2. Business identity is owned by organization, not user
-- 3. Emails/phones support multiple entries with one primary
-- 4. Files are referenced directly (no ALTERs used)
-- 5. Storage quotas can safely FK to businesses(id) in their own file
-- =====================================================
