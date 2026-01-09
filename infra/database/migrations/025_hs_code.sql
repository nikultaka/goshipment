-- =====================================================
-- 025_hs_code.sql
-- Harmonized System (HS) Codes
-- Global Customs Classification (Versioned)
-- =====================================================

-- IMPORTANT DESIGN NOTES
-- -----------------------------------------------------
-- 1. HS codes are GLOBAL and VERSIONED
-- 2. Base system is 6 digits (international standard)
-- 3. Country-specific extensions (8–10 digits)
--    are intentionally NOT stored here
-- 4. HS codes are APPEND-ONLY
--    (never delete, only deactivate)
-- 5. Enforcement happens at API / shipment-item level
-- =====================================================
-- =====================================================
-- HS CODE VERSIONS
-- Tracks official HS system releases (e.g. HS2022)
-- =====================================================

CREATE TABLE hs_code_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    code VARCHAR(20) NOT NULL UNIQUE,
    -- Example: HS2017, HS2022

    year INTEGER NOT NULL,
    -- Example: 2022

    is_active BOOLEAN DEFAULT FALSE,
    -- Only one version should be active at a time

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_hs_code_versions_active
    ON hs_code_versions(is_active);

-- =====================================================
-- HS CODES
-- Tree-structured classification
-- =====================================================

CREATE TABLE hs_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    version_id UUID NOT NULL,
    -- FK → hs_code_versions.id

    code VARCHAR(10) NOT NULL,
    -- Stored as STRING
    -- Examples:
    -- 01        (chapter)
    -- 0101      (heading)
    -- 010121    (subheading)

    parent_code VARCHAR(10),
    -- Self-referencing by code (not FK)
    -- Allows simple tree reconstruction

    description TEXT NOT NULL,
    -- Official HS description (English)

    level VARCHAR(20) NOT NULL,
    -- chapter | heading | subheading

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_hs_codes_version
        FOREIGN KEY (version_id)
        REFERENCES hs_code_versions(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_hs_code_per_version
        UNIQUE (version_id, code)
);

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_hs_codes_version
    ON hs_codes(version_id);

CREATE INDEX idx_hs_codes_code
    ON hs_codes(code);

CREATE INDEX idx_hs_codes_active
    ON hs_codes(is_active);

CREATE INDEX idx_hs_codes_level
    ON hs_codes(level);

-- =====================================================
-- NOTES
-- =====================================================
-- 1. Codes are stored as strings to preserve leading zeros
-- 2. parent_code is not a FK by design
--    (avoids import-order complexity)
-- 3. Tree can be reconstructed using:
--      LEFT(code, 2) / LEFT(code, 4)
-- 4. Do NOT store country-specific HS extensions here
-- 5. Shipment items will reference hs_codes.id
-- =====================================================
