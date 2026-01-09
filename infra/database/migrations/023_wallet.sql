-- =====================================================
-- 023_wallet.sql (UPDATED)
-- Organization Wallets, Holds & Immutable Ledger
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE wallet_status_enum AS ENUM (
    'active',
    'locked',
    'disabled'
);

CREATE TYPE wallet_tx_direction_enum AS ENUM (
    'credit',
    'debit'
);

CREATE TYPE wallet_tx_type_enum AS ENUM (
    'topup',                -- Customer adds funds
    'topup_refund',         -- Topup refunded to card
    'shipment_payment',     -- Payment for shipment
    'shipment_refund',      -- Refund for cancelled/voided shipment
    'hold_capture',         -- Convert hold to actual debit
    'hold_release',         -- Release held funds (failed checkout)
    'adjustment_charge',    -- Post-shipment adjustment (weight, duties)
    'adjustment_refund',    -- Refund of adjustment
    'commission_credit',    -- Commission earned (if paid to wallet)
    'promotional_credit',   -- Promo/marketing credit
    'manual_credit',        -- Admin manual credit
    'manual_debit',         -- Admin manual debit
    'chargeback',           -- Stripe chargeback
    'chargeback_reversal'   -- Chargeback reversed
);

CREATE TYPE wallet_hold_status_enum AS ENUM (
    'pending',      -- Hold created, waiting for capture
    'captured',     -- Hold converted to actual debit
    'released',     -- Hold released (checkout failed)
    'expired'       -- Hold expired (timeout)
);

-- =====================================================
-- WALLETS
-- =====================================================

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    wallet_number VARCHAR(20) UNIQUE NOT NULL,  -- WAL-XXXXXXXX

    organization_id UUID NOT NULL 
        REFERENCES organizations(id) 
        ON DELETE RESTRICT,

    -- -------------------------------------------------
    -- BALANCES
    -- -------------------------------------------------
    balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    -- Available balance (excludes holds)
    
    held_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    -- Currently held for pending checkouts
    
    -- Total funds = balance + held_balance
    -- Available for new transactions = balance only

    -- -------------------------------------------------
    -- STATUS & LOCKING
    -- -------------------------------------------------
    status wallet_status_enum NOT NULL DEFAULT 'active',
    version BIGINT NOT NULL DEFAULT 0,  -- Optimistic locking

    -- -------------------------------------------------
    -- LIMITS (Optional, per-org overrides)
    -- -------------------------------------------------
    max_balance NUMERIC(14,2),          -- Max wallet can hold
    min_topup_amount NUMERIC(14,2) DEFAULT 10.00,
    max_topup_amount NUMERIC(14,2) DEFAULT 10000.00,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    UNIQUE (organization_id),

    CONSTRAINT chk_wallet_balance_non_negative
        CHECK (balance >= 0),

    CONSTRAINT chk_wallet_held_balance_non_negative
        CHECK (held_balance >= 0),

    CONSTRAINT chk_wallet_number_format
        CHECK (wallet_number ~ '^WAL-[A-Z0-9]{8}$')
);

-- =====================================================
-- WALLET TRANSACTIONS (IMMUTABLE LEDGER)
-- =====================================================
-- NEVER UPDATE, NEVER DELETE
-- Corrections via compensating entries only

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    transaction_number VARCHAR(20) UNIQUE NOT NULL,  -- TRX-XXXXXXXX

    wallet_id UUID NOT NULL
        REFERENCES wallets(id)
        ON DELETE RESTRICT,  -- Never delete wallet with transactions

    -- -------------------------------------------------
    -- TRANSACTION TYPE
    -- -------------------------------------------------
    direction wallet_tx_direction_enum NOT NULL,
    tx_type wallet_tx_type_enum NOT NULL,
    
    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- BALANCE SNAPSHOT (for reconciliation)
    -- -------------------------------------------------
    balance_before NUMERIC(14,2) NOT NULL,
    balance_after NUMERIC(14,2) NOT NULL,

    -- -------------------------------------------------
    -- BUSINESS CONTEXT
    -- -------------------------------------------------
    description VARCHAR(255),
    
    -- What triggered this transaction
    reference_type VARCHAR(50),  -- shipment, invoice, hold, topup, etc.
    reference_id UUID,

    -- -------------------------------------------------
    -- STRIPE LINKAGE
    -- -------------------------------------------------
    stripe_payment_intent_id VARCHAR(150),
    stripe_charge_id VARCHAR(150),
    stripe_refund_id VARCHAR(150),
    stripe_balance_transaction_id VARCHAR(150),

    -- -------------------------------------------------
    -- ACTOR & AUDIT
    -- -------------------------------------------------
    performed_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    is_system_generated BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- IDEMPOTENCY
    -- -------------------------------------------------
    idempotency_key VARCHAR(100) NOT NULL,

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_wallet_tx_amount_positive
        CHECK (amount > 0),

    CONSTRAINT chk_wallet_tx_number_format
        CHECK (transaction_number ~ '^TRX-[A-Z0-9]{8}$'),

    CONSTRAINT chk_wallet_tx_balance_math
        CHECK (
            (direction = 'credit' AND balance_after = balance_before + amount)
            OR (direction = 'debit' AND balance_after = balance_before - amount)
        ),

    CONSTRAINT uq_wallet_idempotency
        UNIQUE (wallet_id, idempotency_key)
);

-- =====================================================
-- WALLET HOLDS
-- =====================================================
-- Temporary holds during checkout process
-- Prevents overselling balance

CREATE TABLE wallet_holds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    hold_number VARCHAR(20) UNIQUE NOT NULL,  -- HLD-XXXXXXXX

    wallet_id UUID NOT NULL
        REFERENCES wallets(id)
        ON DELETE RESTRICT,

    -- -------------------------------------------------
    -- HOLD DETAILS
    -- -------------------------------------------------
    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',
    
    status wallet_hold_status_enum NOT NULL DEFAULT 'pending',

    -- -------------------------------------------------
    -- REFERENCE
    -- -------------------------------------------------
    reference_type VARCHAR(50) NOT NULL,  -- 'shipment_estimate', 'checkout'
    reference_id UUID NOT NULL,

    description VARCHAR(255),

    -- -------------------------------------------------
    -- LIFECYCLE
    -- -------------------------------------------------
    expires_at TIMESTAMP NOT NULL,
    -- Holds auto-expire if not captured/released
    -- Default: 15 minutes from creation

    captured_at TIMESTAMP,
    released_at TIMESTAMP,

    -- -------------------------------------------------
    -- LINKED TRANSACTION
    -- -------------------------------------------------
    -- When captured, links to the actual wallet_transaction
    capture_transaction_id UUID
        REFERENCES wallet_transactions(id),

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    created_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_hold_amount_positive
        CHECK (amount > 0),

    CONSTRAINT chk_hold_number_format
        CHECK (hold_number ~ '^HLD-[A-Z0-9]{8}$'),

    CONSTRAINT chk_hold_status_timestamps
        CHECK (
            (status = 'pending' AND captured_at IS NULL AND released_at IS NULL)
            OR (status = 'captured' AND captured_at IS NOT NULL AND released_at IS NULL)
            OR (status = 'released' AND released_at IS NOT NULL AND captured_at IS NULL)
            OR (status = 'expired' AND captured_at IS NULL)
        )
);

-- =====================================================
-- WALLET TOPUPS (Stripe Payment Tracking)
-- =====================================================
-- Every topup has a Stripe payment, tracked for refunds

CREATE TABLE wallet_topups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    topup_number VARCHAR(20) UNIQUE NOT NULL,  -- TOP-XXXXXXXX

    wallet_id UUID NOT NULL
        REFERENCES wallets(id)
        ON DELETE RESTRICT,

    organization_id UUID NOT NULL
        REFERENCES organizations(id),

    -- -------------------------------------------------
    -- AMOUNT
    -- -------------------------------------------------
    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- Stripe fees (for reference)
    stripe_fee NUMERIC(14,2),
    net_amount NUMERIC(14,2),

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending, succeeded, failed, refunded, partially_refunded

    -- -------------------------------------------------
    -- STRIPE DETAILS (Critical for refunds!)
    -- -------------------------------------------------
    stripe_payment_intent_id VARCHAR(150) NOT NULL,
    stripe_charge_id VARCHAR(150),
    stripe_payment_method_id VARCHAR(150),
    stripe_receipt_url TEXT,

    -- -------------------------------------------------
    -- REFUND TRACKING
    -- -------------------------------------------------
    refundable_amount NUMERIC(14,2) NOT NULL,
    -- Decreases as refunds are issued
    -- Cannot refund more than this

    total_refunded NUMERIC(14,2) NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- LINKED TRANSACTION
    -- -------------------------------------------------
    wallet_transaction_id UUID
        REFERENCES wallet_transactions(id),

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    created_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_topup_amount_positive
        CHECK (amount > 0),

    CONSTRAINT chk_topup_number_format
        CHECK (topup_number ~ '^TOP-[A-Z0-9]{8}$'),

    CONSTRAINT chk_topup_refund_math
        CHECK (total_refunded <= amount),

    CONSTRAINT chk_topup_refundable
        CHECK (refundable_amount >= 0 AND refundable_amount <= amount)
);

-- =====================================================
-- WALLET TOPUP REFUNDS
-- =====================================================
-- Track individual refunds against topups

CREATE TABLE wallet_topup_refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    refund_number VARCHAR(20) UNIQUE NOT NULL,  -- TRF-XXXXXXXX

    topup_id UUID NOT NULL
        REFERENCES wallet_topups(id)
        ON DELETE RESTRICT,

    wallet_id UUID NOT NULL
        REFERENCES wallets(id)
        ON DELETE RESTRICT,

    -- -------------------------------------------------
    -- AMOUNT
    -- -------------------------------------------------
    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    -- pending, processing, succeeded, failed

    -- -------------------------------------------------
    -- STRIPE REFUND
    -- -------------------------------------------------
    stripe_refund_id VARCHAR(150),
    stripe_failure_reason VARCHAR(255),

    -- -------------------------------------------------
    -- REASON
    -- -------------------------------------------------
    reason VARCHAR(50) NOT NULL,
    -- customer_request, admin_adjustment, duplicate, fraud, etc.
    
    notes TEXT,

    -- -------------------------------------------------
    -- LINKED TRANSACTION
    -- -------------------------------------------------
    wallet_transaction_id UUID
        REFERENCES wallet_transactions(id),

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    requested_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    approved_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    processed_at TIMESTAMP,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_topup_refund_amount_positive
        CHECK (amount > 0),

    CONSTRAINT chk_topup_refund_number_format
        CHECK (refund_number ~ '^TRF-[A-Z0-9]{8}$')
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Wallets
CREATE INDEX idx_wallets_org ON wallets(organization_id);
CREATE INDEX idx_wallets_status ON wallets(status);

-- Transactions
CREATE INDEX idx_wallet_tx_wallet ON wallet_transactions(wallet_id);
CREATE INDEX idx_wallet_tx_type ON wallet_transactions(tx_type);
CREATE INDEX idx_wallet_tx_reference ON wallet_transactions(reference_type, reference_id);
CREATE INDEX idx_wallet_tx_created ON wallet_transactions(created_at);
CREATE INDEX idx_wallet_tx_stripe_pi ON wallet_transactions(stripe_payment_intent_id);

-- Holds
CREATE INDEX idx_wallet_holds_wallet ON wallet_holds(wallet_id);
CREATE INDEX idx_wallet_holds_status ON wallet_holds(status);
CREATE INDEX idx_wallet_holds_reference ON wallet_holds(reference_type, reference_id);
CREATE INDEX idx_wallet_holds_expires ON wallet_holds(expires_at) WHERE status = 'pending';

-- Topups
CREATE INDEX idx_wallet_topups_wallet ON wallet_topups(wallet_id);
CREATE INDEX idx_wallet_topups_org ON wallet_topups(organization_id);
CREATE INDEX idx_wallet_topups_status ON wallet_topups(status);
CREATE INDEX idx_wallet_topups_stripe_pi ON wallet_topups(stripe_payment_intent_id);

-- Topup Refunds
CREATE INDEX idx_topup_refunds_topup ON wallet_topup_refunds(topup_id);
CREATE INDEX idx_topup_refunds_wallet ON wallet_topup_refunds(wallet_id);
CREATE INDEX idx_topup_refunds_status ON wallet_topup_refunds(status);

-- =====================================================
-- HELPER VIEW: Wallet Summary
-- =====================================================

CREATE OR REPLACE VIEW v_wallet_details AS
SELECT 
    w.id AS wallet_id,
    w.wallet_number,
    w.organization_id,
    w.balance AS available_balance,
    w.held_balance,
    (w.balance + w.held_balance) AS total_balance,
    w.status,
    
    -- Pending holds
    (
        SELECT COALESCE(SUM(amount), 0)
        FROM wallet_holds h
        WHERE h.wallet_id = w.id
        AND h.status = 'pending'
    ) AS pending_holds_amount,
    
    -- Total topped up
    (
        SELECT COALESCE(SUM(amount), 0)
        FROM wallet_topups t
        WHERE t.wallet_id = w.id
        AND t.status = 'succeeded'
    ) AS total_topped_up,
    
    -- Total refunded
    (
        SELECT COALESCE(SUM(amount), 0)
        FROM wallet_topup_refunds r
        WHERE r.wallet_id = w.id
        AND r.status = 'succeeded'
    ) AS total_refunded

FROM wallets w;

-- =====================================================
-- NOTES
-- =====================================================

/*
CHECKOUT FLOW WITH HOLDS:
────────────────────────────────────────────────────────

1. USER CLICKS "CHECKOUT"
   → Create wallet_hold for estimated amount
   → held_balance += amount
   → available balance unchanged (but effectively reduced)

2. CHECKOUT PROCESSING
   → Call carrier API for label
   → If amount changes, update hold

3A. SUCCESS
   → wallet_hold.status = 'captured'
   → Create wallet_transaction (debit, tx_type='hold_capture')
   → balance -= amount
   → held_balance -= amount
   → Create invoice (paid)

3B. FAILURE
   → wallet_hold.status = 'released'
   → held_balance -= amount
   → No wallet_transaction created
   → No charge to customer

4. HOLD EXPIRATION (Cron job)
   → Find holds WHERE status='pending' AND expires_at < NOW()
   → Set status = 'expired'
   → held_balance -= amount


REFUND RULES:
────────────────────────────────────────────────────────

1. TOPUP REFUNDS
   - Can only refund topups that exist in wallet_topups
   - Cannot refund more than topup.refundable_amount
   - Refund goes back to original payment method via Stripe
   - Must have stripe_payment_intent_id to refund

2. SHIPMENT REFUNDS
   - Created via credit_note on invoice
   - Credits back to wallet (not to card)
   - Creates wallet_transaction (credit, tx_type='shipment_refund')

3. ADMIN CAN'T REFUND > AVAILABLE
   - Check: refund_amount <= wallet.balance
   - If wallet.balance = $20 and refund = $50, reject
   - Admin can issue credit_note which creates negative invoice


ADMIN BYPASS (NO WALLET):
────────────────────────────────────────────────────────

Admin/Manager/Sales shipments:
- invoice.is_carrier_billed = TRUE
- invoice.payment_method = 'carrier_account'
- NO wallet transaction
- shipment_financials still tracks cost/profit
*/