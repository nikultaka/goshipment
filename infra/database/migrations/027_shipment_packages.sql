-- =====================================================
-- 027_shipment_packages.sql
-- Shipment Packages (Multi-Package Support)
--
-- SINGLE authoritative package table used by:
--   - shipment_estimates (quote stage)
--   - shipments (final stage)
--
-- IMPORTANT:
-- - Tracking + labels are PER PACKAGE (per parcel)
-- - Therefore label_url, tracking_number, etc. live HERE
-- =====================================================

-- =====================================================
-- TABLE: shipment_packages
-- =====================================================

CREATE TABLE shipment_packages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- Human Identifier
    -- -------------------------------------------------
    package_number VARCHAR(30) UNIQUE,
    -- -------------------------------------------------
    -- PARENT LINKAGE
    -- -------------------------------------------------
    -- A package belongs to EITHER:
    --   - shipment_estimate_id (quote stage)
    --   - shipment_id          (final shipment stage)
    --
    -- NOTE:
    -- shipment_id FK is added in 038_shipments.sql
    shipment_estimate_id UUID,
    shipment_id UUID,

    -- -------------------------------------------------
    -- PACKAGE ORDERING
    -- -------------------------------------------------
    -- Stable ordering for UI and carrier payload mapping
    package_index INTEGER NOT NULL,
    -- 1..N

    -- -------------------------------------------------
    -- PACKAGE LABEL (UX ONLY)
    -- -------------------------------------------------
    package_name VARCHAR(80),
    -- e.g. "Box 1", "Documents", "Shoebox", "Laptop"

    -- -------------------------------------------------
    -- DIMENSIONS
    -- -------------------------------------------------
    length NUMERIC(10,2) NOT NULL,
    width  NUMERIC(10,2) NOT NULL,
    height NUMERIC(10,2) NOT NULL,

    dimension_unit VARCHAR(10) NOT NULL,
    -- cm | in

    -- -------------------------------------------------
    -- WEIGHT
    -- -------------------------------------------------
    weight NUMERIC(10,3) NOT NULL,
    weight_unit VARCHAR(10) NOT NULL,
    -- kg | lb

    -- -------------------------------------------------
    -- DECLARED VALUE (OPTIONAL)
    -- -------------------------------------------------
    declared_value NUMERIC(14,2),
    declared_currency CHAR(3),

    -- -------------------------------------------------
    -- CARRIER OBJECT LINKS (OPTIONAL)
    -- -------------------------------------------------
    -- These are populated once you start interacting with Shippo.
    -- For quote stage: you may create parcels to rate, or create them later.
    carrier_parcel_id VARCHAR(150),

    -- -------------------------------------------------
    -- LABEL & TRACKING (PER PACKAGE / PER PARCEL)
    -- -------------------------------------------------
    -- In multi-piece shipments, label purchase returns one transaction per parcel.
    carrier_label_transaction_id  VARCHAR(150),

    label_url TEXT,
    label_file_type VARCHAR(20),
    -- pdf | zpl | png (depending on carrier)

    tracking_number VARCHAR(100),
    tracking_url TEXT,
    label_purchased_at TIMESTAMP WITHOUT TIME ZONE,

    -- -------------------------------------------------
    -- OPTIONAL PACKAGE DELIVERY STATE
    -- -------------------------------------------------
    package_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending | label_purchased | in_transit | delivered | failed | voided | returned

    failure_reason TEXT,
    carrier_error JSONB,

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- SAFETY CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_shipment_packages_parent
        CHECK (
            shipment_estimate_id IS NOT NULL
            OR shipment_id IS NOT NULL
        ),

    CONSTRAINT chk_shipment_packages_package_index
        CHECK (package_index >= 1),
    
    CONSTRAINT chk_package_number_format
        CHECK (package_number IS NULL OR package_number ~ '^PKG-[A-Z0-9]+-[0-9]{2}$'),

    -- -------------------------------------------------
    -- FOREIGN KEYS
    -- -------------------------------------------------
    CONSTRAINT fk_shipment_packages_estimate
        FOREIGN KEY (shipment_estimate_id)
        REFERENCES shipment_estimates(id)
        ON DELETE CASCADE
    
);

-- =====================================================
-- INDEXES
-- =====================================================
-- Add index
CREATE INDEX idx_shipment_packages_number
    ON shipment_packages(package_number);

CREATE INDEX idx_shipment_packages_estimate_id
    ON shipment_packages(shipment_estimate_id);

CREATE INDEX idx_shipment_packages_shipment_id
    ON shipment_packages(shipment_id);

CREATE INDEX idx_shipment_packages_package_index
    ON shipment_packages(package_index);

CREATE INDEX idx_shipment_packages_tracking_number
    ON shipment_packages(tracking_number);

CREATE INDEX idx_shipment_packages_label_tx_id
    ON shipment_packages(carrier_label_transaction_id);
-- =====================================================
-- NOTES
-- =====================================================
COMMENT ON COLUMN shipment_packages.package_number IS 'Format: PKG-{SHIPMENT_SHORT}-{INDEX}, e.g., PKG-K9R2B7X4-01';
-- WHY LABEL/TRACKING IS HERE:
-- -----------------------------------------------------
-- Multi-package shipments can produce:
--   - 1 shipment (financial + address + service)
--   - N parcels (each parcel can have its own label + tracking)
--
-- Therefore, shipment_packages is the correct place for:
--   label_url, tracking_number, tracking_url, transaction_id
-- =====================================================
