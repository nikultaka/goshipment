-- =====================================================
-- 028_shipments.sql
-- Shipments
-- Authoritative financial & logistics record
--
-- IMPORTANT:
-- - A shipment is created FROM a shipment_estimate
-- - shipment_packages contains the 1..N packages
-- - Labels/tracking are PER PACKAGE and live in shipment_packages
-- - shipment_status is the shipment lifecycle (not UI archiving)
-- =====================================================

-- =====================================================
-- ENUM: shipment_status_enum
-- Created safely (idempotent)
-- =====================================================


CREATE TYPE shipment_status_enum AS ENUM (

    -- -----------------------------------------
    -- PRE-LABEL STATES
    -- -----------------------------------------
    'draft',                -- created, not finalized
    'awaiting_payment',     -- checkout started
    'payment_failed',       -- payment attempt failed

    -- -----------------------------------------
    -- LABEL & PICKUP
    -- -----------------------------------------
    'label_purchased',      -- labels purchased (packages now have tracking)
    'ready_for_pickup',     -- ready for carrier pickup
    'pickup_scheduled',     -- pickup scheduled

    -- -----------------------------------------
    -- IN-TRANSIT
    -- -----------------------------------------
    'in_transit',           -- carrier has it
    'out_for_delivery',     -- final mile

    -- -----------------------------------------
    -- FINAL STATES
    -- -----------------------------------------
    'delivered',            -- delivered
    'delivery_failed',      -- delivery attempt failed
    'returned',             -- returned to sender

    -- -----------------------------------------
    -- TERMINAL / EXCEPTION STATES
    -- -----------------------------------------
    'cancelled',            -- cancelled
    'refunded',             -- refunded
    'voided'                -- label voided
);


-- =====================================================
-- TABLE: shipments
-- =====================================================

CREATE TABLE shipments (
    -- -------------------------------------------------
    -- INTERNAL ID
    -- -------------------------------------------------
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- PUBLIC ID (HUMAN)
    -- -------------------------------------------------
    shipment_number VARCHAR(20) NOT NULL UNIQUE,
    -- e.g. SIF-8F3K9A2Q

    -- -------------------------------------------------
    -- OWNERSHIP
    -- -------------------------------------------------
    organization_id UUID NOT NULL,
    created_by_user_id UUID NOT NULL,

    -- -------------------------------------------------
    -- ESTIMATE LINEAGE
    -- -------------------------------------------------
    shipment_estimate_id UUID NOT NULL,

    -- -------------------------------------------------
    -- CARRIER & SERVICE (LOCKED)
    -- -------------------------------------------------
    carrier_id UUID NOT NULL,
    servicelevel_code VARCHAR(100) NOT NULL,
    servicelevel_name VARCHAR(150) NOT NULL,

    carrier_shipment_id VARCHAR(150),
    carrier_payload JSONB,

    carrier_rate_id VARCHAR(150),
    -- provider rate/quote id used at purchase time

    carrier_purchase_id VARCHAR(150),
    -- provider purchase/transaction id for the overall shipment, if available

    -- -------------------------------------------------
    -- MASTER TRACKING
    -- -------------------------------------------------
    master_tracking_number VARCHAR(100),
    master_tracking_url TEXT,

    -- -------------------------------------------------
    -- ADDRESS BOOK LINKS
    -- -------------------------------------------------
    from_address_book_id UUID,
    to_address_book_id UUID,

    -- -------------------------------------------------
    -- FROM ADDRESS (SNAPSHOT)
    -- -------------------------------------------------
    from_name VARCHAR(150),
    from_company VARCHAR(150),
    from_phone VARCHAR(50),
    from_email VARCHAR(150),

    from_country CHAR(2) NOT NULL,
    from_region_code VARCHAR(10),
    from_city VARCHAR(100) NOT NULL,
    from_postal_code VARCHAR(20) NOT NULL,
    from_address_line1 VARCHAR(255) NOT NULL,
    from_address_line2 VARCHAR(255),

    -- -------------------------------------------------
    -- TO ADDRESS (SNAPSHOT)
    -- -------------------------------------------------
    to_name VARCHAR(150),
    to_company VARCHAR(150),
    to_phone VARCHAR(50),
    to_email VARCHAR(150),

    to_country CHAR(2) NOT NULL,
    to_region_code VARCHAR(10),
    to_city VARCHAR(100) NOT NULL,
    to_postal_code VARCHAR(20) NOT NULL,
    to_address_line1 VARCHAR(255) NOT NULL,
    to_address_line2 VARCHAR(255),

    -- -------------------------------------------------
    -- PACKAGE AGGREGATES
    -- -------------------------------------------------
    total_packages INTEGER NOT NULL,
    total_weight NUMERIC(14,3) NOT NULL,
    weight_unit VARCHAR(10) NOT NULL,
    -- kg | lb

    -- -------------------------------------------------
    -- DELIVERY OPTIONS
    -- -------------------------------------------------
    require_signature BOOLEAN NOT NULL DEFAULT FALSE,
    email_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    delivery_instructions TEXT,

    -- -------------------------------------------------
    -- BASE PRICING (LOCKED)
    -- -------------------------------------------------
    base_amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL,

    insurance_amount NUMERIC(14,2),
    pickup_fee_amount NUMERIC(14,2),
    taxes_amount NUMERIC(14,2),

    retail_rate NUMERIC(14,2),
    retail_currency CHAR(3),

    -- -------------------------------------------------
    -- POST-SHIPMENT ADJUSTMENTS (EMBEDDED)
    -- -------------------------------------------------
    adjustments_total NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    adjustments_currency CHAR(3),
    has_pending_adjustments BOOLEAN NOT NULL DEFAULT FALSE,

    adjustments_breakdown JSONB,

    -- -------------------------------------------------
    -- PAYMENT
    -- -------------------------------------------------
    wallet_transaction_id UUID,
    payment_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending | paid | partially_paid | failed | refunded

    stripe_payment_intent_id VARCHAR(150),

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    shipment_status shipment_status_enum NOT NULL DEFAULT 'draft',
    failure_reason TEXT,

    -- -------------------------------------------------
    -- PICKUP
    -- -------------------------------------------------
    pickup_required BOOLEAN DEFAULT FALSE,
    pickup_id UUID,

    -- -------------------------------------------------
    -- UI ARCHIVAL
    -- -------------------------------------------------
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    archived_at TIMESTAMP WITHOUT TIME ZONE,
    archived_by_user_id UUID,

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    idempotency_key VARCHAR(100),
    revision INTEGER DEFAULT 1,
    is_deleted BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- FOREIGN KEYS
    -- -------------------------------------------------
    CONSTRAINT fk_shipments_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id),

    CONSTRAINT fk_shipments_created_by
        FOREIGN KEY (created_by_user_id)
        REFERENCES users(id),

    CONSTRAINT fk_shipments_estimate
        FOREIGN KEY (shipment_estimate_id)
        REFERENCES shipment_estimates(id),

    CONSTRAINT fk_shipments_carrier
        FOREIGN KEY (carrier_id)
        REFERENCES carriers(id),

    CONSTRAINT fk_shipments_wallet_tx
        FOREIGN KEY (wallet_transaction_id)
        REFERENCES wallet_transactions(id),

    CONSTRAINT fk_shipments_from_address
        FOREIGN KEY (from_address_book_id)
        REFERENCES address_book(id),

    CONSTRAINT fk_shipments_to_address
        FOREIGN KEY (to_address_book_id)
        REFERENCES address_book(id),

    CONSTRAINT fk_shipments_archived_by
        FOREIGN KEY (archived_by_user_id)
        REFERENCES users(id)
);

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_shipments_org ON shipments(organization_id);
CREATE INDEX idx_shipments_status ON shipments(shipment_status);
CREATE INDEX idx_shipments_payment_status ON shipments(payment_status);
CREATE INDEX idx_shipments_created_at ON shipments(created_at);
CREATE INDEX idx_shipments_archived ON shipments(is_archived);

-- =====================================================
-- NOTES
-- =====================================================
-- KEY PRINCIPLES:
-- -----------------------------------------------------
-- 1. shipments = authoritative record for money + lifecycle + addresses + selected carrier/service
-- 2. shipment_packages = authoritative record for packages + labels + tracking (per parcel)
-- 3. shipment_estimates = pre-purchase quote container; safe to purge for guests
-- 4. 'archived' is NOT a lifecycle status. It is user/UI organization only.
-- =====================================================
