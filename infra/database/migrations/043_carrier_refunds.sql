-- =====================================================
-- 043_carrier_refunds.sql
-- Carrier Refund Tracking
-- =====================================================
--
-- PURPOSE:
-- Track refunds requested from carriers that are in progress
-- This helps show the pending refund amount in wallet API
--
-- FLOW:
-- 1. Shipment is cancelled/voided
-- 2. Carrier refund is requested (status: requested)
-- 3. Carrier processes refund (status: processing)
-- 4. Refund is completed - money credited to wallet (status: completed)
--    OR refund is rejected/failed (status: rejected/failed)
-- =====================================================

-- Carrier Refund Status Enum
CREATE TYPE carrier_refund_status AS ENUM (
    'requested',     -- Refund request submitted to carrier
    'processing',    -- Carrier is processing the refund
    'completed',     -- Refund received and credited to wallet
    'rejected',      -- Carrier rejected the refund
    'failed',        -- Refund failed for technical reasons
    'cancelled'      -- Request cancelled before processing
);

-- Carrier Refunds Table
CREATE TABLE carrier_refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    refund_number VARCHAR(20) UNIQUE NOT NULL,  -- CRF-XXXXXXXX

    -- -------------------------------------------------
    -- LINKAGE
    -- -------------------------------------------------
    organization_id UUID NOT NULL
        REFERENCES organizations(id),

    wallet_id UUID NOT NULL
        REFERENCES wallets(id),

    shipment_id UUID NOT NULL
        REFERENCES shipments(id),

    invoice_id UUID
        REFERENCES invoices(id),

    -- -------------------------------------------------
    -- CARRIER INFO
    -- -------------------------------------------------
    carrier_id UUID NOT NULL
        REFERENCES carriers(id),

    carrier_tracking_number VARCHAR(100),
    -- Original tracking number

    carrier_refund_reference VARCHAR(100),
    -- Carrier's reference ID for the refund

    -- -------------------------------------------------
    -- AMOUNTS
    -- -------------------------------------------------
    requested_amount NUMERIC(14,2) NOT NULL,
    -- Amount requested from carrier

    approved_amount NUMERIC(14,2),
    -- Amount carrier approved (may differ from requested)

    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    status carrier_refund_status NOT NULL DEFAULT 'requested',

    -- -------------------------------------------------
    -- REASON
    -- -------------------------------------------------
    refund_reason VARCHAR(50) NOT NULL,
    -- shipment_cancelled, shipment_voided, label_unused,
    -- duplicate_charge, carrier_error, service_failure, etc.

    reason_notes TEXT,

    -- -------------------------------------------------
    -- TIMELINE
    -- -------------------------------------------------
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),

    carrier_acknowledged_at TIMESTAMP,
    -- When carrier acknowledged the request

    completed_at TIMESTAMP,
    -- When refund was completed/rejected/failed

    expected_completion_date DATE,
    -- Expected date for refund completion

    -- -------------------------------------------------
    -- WALLET TRANSACTION (when completed)
    -- -------------------------------------------------
    wallet_transaction_id UUID
        REFERENCES wallet_transactions(id),

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    requested_by_user_id UUID
        REFERENCES users(id),

    processed_by_user_id UUID
        REFERENCES users(id),

    -- -------------------------------------------------
    -- CARRIER RESPONSE
    -- -------------------------------------------------
    carrier_response_code VARCHAR(50),
    carrier_response_message TEXT,

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    notes TEXT,
    metadata JSONB,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_cr_number_format
        CHECK (refund_number ~ '^CRF-[A-Z0-9]{8}$'),

    CONSTRAINT chk_cr_amount_positive
        CHECK (requested_amount > 0),

    CONSTRAINT chk_cr_approved_positive
        CHECK (approved_amount IS NULL OR approved_amount >= 0)
);

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_cr_organization ON carrier_refunds(organization_id);
CREATE INDEX idx_cr_wallet ON carrier_refunds(wallet_id);
CREATE INDEX idx_cr_shipment ON carrier_refunds(shipment_id);
CREATE INDEX idx_cr_carrier ON carrier_refunds(carrier_id);
CREATE INDEX idx_cr_status ON carrier_refunds(status);
CREATE INDEX idx_cr_requested_at ON carrier_refunds(requested_at);

-- Index for "in progress" refunds (used by wallet API)
CREATE INDEX idx_cr_in_progress ON carrier_refunds(organization_id, status)
    WHERE status IN ('requested', 'processing');

-- =====================================================
-- COMMENTS
-- =====================================================

COMMENT ON TABLE carrier_refunds IS 'Tracks refunds requested from carriers for cancelled/voided shipments';
COMMENT ON COLUMN carrier_refunds.status IS 'requested = awaiting carrier, processing = carrier working on it, completed = refund received';
COMMENT ON COLUMN carrier_refunds.requested_amount IS 'Amount we requested from carrier (usually the label cost)';
COMMENT ON COLUMN carrier_refunds.approved_amount IS 'Amount carrier approved - may be less due to fees';
