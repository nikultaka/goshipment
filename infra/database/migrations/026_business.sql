-- ================================================
-- 026_business.sql
-- Businesses
-- ================================================

CREATE TYPE business_type_enum AS ENUM (
    'individual',
    'business',
    'nonprofit',
    'government'
);

CREATE TABLE businesses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL UNIQUE
        REFERENCES users(id) ON DELETE CASCADE,

    business_name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),

    business_type business_type_enum NOT NULL,

    tax_id VARCHAR(100),

    avatar_file_id UUID REFERENCES files(id) ON DELETE SET NULL,

    website VARCHAR(255),
    description TEXT,

    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    province VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(2),

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE business_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    business_id UUID NOT NULL
        REFERENCES businesses(id) ON DELETE CASCADE,

    email VARCHAR(255) NOT NULL,
    email_lower VARCHAR(255) GENERATED ALWAYS AS (lower(email)) STORED,

    email_type VARCHAR(30) DEFAULT 'general',
    -- general, billing, support, accounting, notifications

    is_primary BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP,

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE (business_id, email_lower)
);


CREATE TABLE business_phones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    business_id UUID NOT NULL
        REFERENCES businesses(id) ON DELETE CASCADE,

    phone VARCHAR(20) NOT NULL,
    phone_e164 VARCHAR(20),

    phone_type VARCHAR(30) DEFAULT 'general',
    -- general, billing, support, sales, whatsapp

    is_primary BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP,

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE (business_id, phone_e164)
);

CREATE UNIQUE INDEX ux_business_primary_email
ON business_emails (business_id)
WHERE is_primary = true;

CREATE UNIQUE INDEX ux_business_primary_phone
ON business_phones (business_id)
WHERE is_primary = true;

CREATE INDEX idx_businesses_avatar_file_id ON businesses(avatar_file_id);

ALTER TABLE files
ADD CONSTRAINT fk_files_business_id
FOREIGN KEY (business_id) REFERENCES businesses(id) ON DELETE SET NULL;

CREATE INDEX idx_files_business_id ON files(business_id);

ALTER TABLE storage_quotas ADD CONSTRAINT fk_storage_quotas_business FOREIGN KEY (business_id) REFERENCES businesses(id) ON DELETE CASCADE;
