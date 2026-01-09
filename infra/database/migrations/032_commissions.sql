-- =====================================================
-- 032_commissions.sql
-- Commissions: Plans + Transactions (Earnings Ledger) + Payout Requests + Payouts
-- Organization-scoped
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

-- Plan configuration
CREATE TYPE commission_base_enum AS ENUM (
  'profit',        -- (revenue - cost) * %
  'revenue',       -- revenue * %
  'per_shipment'   -- fixed amount per completed shipment
);

CREATE TYPE commission_value_type_enum AS ENUM (
  'percent',
  'fixed_amount'
);

CREATE TYPE commission_plan_status_enum AS ENUM (
  'active',
  'paused'
);

CREATE TYPE commission_apply_when_enum AS ENUM (
  'shipment_completed',
  'include_refunded_shipments'
);

-- Transactions (earnings ledger)
CREATE TYPE commission_tx_status_enum AS ENUM (
  'pending',      -- shipment exists, not completed yet (not payable)
  'eligible',     -- shipment completed and qualifies (payable)
  'processing',   -- locked for payout processing
  'paid',         -- included in a completed payout
  'voided'        -- invalidated (e.g., refund excluded / rule changed / admin void)
);

-- Payout requests (rep asks for payout)
CREATE TYPE commission_payout_request_status_enum AS ENUM (
  'pending',      -- requested, awaiting admin review
  'approved',     -- admin approved, ready to process
  'rejected',     -- admin rejected (final)
  'cancelled'     -- requester cancelled before processing
);

-- Payouts (money sent)
CREATE TYPE commission_payout_status_enum AS ENUM (
  'pending',      -- created (often after request approval)
  'processing',   -- sending money / awaiting confirmation
  'paid',         -- paid successfully
  'rejected',     -- rejected/blocked (e.g. compliance)
  'failed',       -- attempted but failed
  'cancelled'     -- cancelled before payment
);

CREATE TYPE commission_payout_method_enum AS ENUM (
  'stripe',
  'paypal',
  'manual'
);

-- =====================================================
-- COMMISSION PLANS (what the UI creates)
-- =====================================================

CREATE TABLE commission_plans (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  commission_number VARCHAR(20) NOT NULL UNIQUE,
  -- COM-XXXXXXXX

  organization_id UUID NOT NULL REFERENCES organizations(id),

  created_by_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  name VARCHAR(120) NOT NULL,
  description TEXT,

  -- UI: Percentage of Profit / Percentage of Revenue / Fixed per Shipment
  commission_base commission_base_enum NOT NULL,
  commission_value_type commission_value_type_enum NOT NULL,
  value NUMERIC(14,2) NOT NULL,

  -- UI: Apply when shipment completed
  apply_when commission_apply_when_enum NOT NULL DEFAULT 'shipment_completed',

  -- UI: Include refunded shipments (not recommended)
  include_refunded_shipments BOOLEAN NOT NULL DEFAULT FALSE,

  -- UI: Active / Paused
  status commission_plan_status_enum NOT NULL DEFAULT 'active',

  metadata JSONB,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_commission_plan_value
    CHECK (
      (commission_value_type = 'percent' AND value > 0 AND value <= 100)
      OR
      (commission_value_type = 'fixed_amount' AND value > 0)
    ),

  -- per_shipment should be fixed_amount only
  CONSTRAINT chk_commission_plan_per_shipment_fixed
    CHECK (
      commission_base <> 'per_shipment'
      OR commission_value_type = 'fixed_amount'
    )
);

CREATE INDEX idx_commission_plans_org_id ON commission_plans(organization_id);
CREATE INDEX idx_commission_plans_status ON commission_plans(status);

-- =====================================================
-- COMMISSION TRANSACTIONS (EARNINGS LEDGER)
-- One row = one earned commission event (usually per shipment/invoice)
-- Tracks: who triggered it, who earns it, how much, and status lifecycle
-- =====================================================

CREATE TABLE commission_transactions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  commission_transaction_number VARCHAR(20) NOT NULL UNIQUE,
  -- CMT-XXXXXXXX

  organization_id UUID NOT NULL REFERENCES organizations(id),

  -- The user who earns commission (sales rep / referrer / affiliate)
  beneficiary_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- The user who generated the revenue (customer)
  source_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- Shipment / Invoice linkage
  shipment_id UUID NOT NULL
    REFERENCES shipments(id) ON DELETE CASCADE,

  invoice_id UUID,

  -- Plan snapshot (never rely on live plan later)
  commission_plan_id UUID NOT NULL
    REFERENCES commission_plans(id) ON DELETE RESTRICT,

  commission_number VARCHAR(20) NOT NULL,                 -- snapshot
  commission_base commission_base_enum NOT NULL,          -- snapshot
  commission_value_type commission_value_type_enum NOT NULL, -- snapshot
  commission_value NUMERIC(14,2) NOT NULL,                -- snapshot

  -- Financial snapshot
  currency CHAR(3) NOT NULL DEFAULT 'CAD',

  -- Revenue/profit snapshots used for calculation (optional but recommended)
  revenue_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  cost_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  profit_amount NUMERIC(14,2) NOT NULL DEFAULT 0,

  commission_amount NUMERIC(14,2) NOT NULL DEFAULT 0,

  status commission_tx_status_enum NOT NULL DEFAULT 'pending',

  -- This becomes the time anchor for reports
  earned_at TIMESTAMP NOT NULL DEFAULT NOW(),

  -- When it became eligible (e.g. shipment completed)
  eligible_at TIMESTAMP,

  -- When locked for payout
  processing_at TIMESTAMP,

  -- When paid
  paid_at TIMESTAMP,

  -- Link to payout if paid
  payout_id UUID,

  metadata JSONB,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_commission_amount_non_negative
    CHECK (commission_amount >= 0),

  CONSTRAINT chk_commission_snapshots_non_negative
    CHECK (
      revenue_amount >= 0 AND
      cost_amount >= 0 AND
      profit_amount >= 0
    )
);

-- payout_id FK added after payouts table exists (below)

CREATE INDEX idx_commission_tx_org_id ON commission_transactions(organization_id);
CREATE INDEX idx_commission_tx_beneficiary ON commission_transactions(beneficiary_user_id);
CREATE INDEX idx_commission_tx_status ON commission_transactions(status);
CREATE INDEX idx_commission_tx_shipment ON commission_transactions(shipment_id);
-- CREATE INDEX idx_commission_tx_invoice ON commission_transactions(invoice_id);
CREATE INDEX idx_commission_tx_earned_at ON commission_transactions(earned_at);

-- Prevent duplicate commission transaction per shipment per beneficiary per plan (common policy)
CREATE UNIQUE INDEX uq_commission_tx_unique_event
  ON commission_transactions(organization_id, shipment_id, beneficiary_user_id, commission_plan_id);

-- =====================================================
-- COMMISSION TRANSACTION EVENTS (AUDIT)
-- Immutable history of status changes / admin actions
-- =====================================================

CREATE TABLE commission_transaction_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  commission_transaction_id UUID NOT NULL
    REFERENCES commission_transactions(id) ON DELETE CASCADE,

  actor_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  from_status commission_tx_status_enum,
  to_status commission_tx_status_enum NOT NULL,

  message TEXT,
  metadata JSONB,

  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_commission_tx_events_tx ON commission_transaction_events(commission_transaction_id);
CREATE INDEX idx_commission_tx_events_created_at ON commission_transaction_events(created_at);

-- =====================================================
-- PAYOUT REQUESTS (USER/REP REQUESTS PAYOUT)
-- Includes notes and admin review notes
-- =====================================================

CREATE TABLE commission_payout_requests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  payout_request_number VARCHAR(20) NOT NULL UNIQUE,
  -- PRQ-XXXXXXXX

  organization_id UUID NOT NULL REFERENCES organizations(id),

  requested_by_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- The beneficiary (usually same as requested_by_user_id, but flexible)
  beneficiary_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  amount_requested NUMERIC(14,2) NOT NULL CHECK (amount_requested > 0),
  currency CHAR(3) NOT NULL DEFAULT 'CAD',

  method commission_payout_method_enum NOT NULL DEFAULT 'manual',

  status commission_payout_request_status_enum NOT NULL DEFAULT 'pending',

  -- User note (shown to admin)
  request_note TEXT,

  -- Admin review
  reviewed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at TIMESTAMP,
  review_note TEXT,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comm_payout_req_org ON commission_payout_requests(organization_id);
CREATE INDEX idx_comm_payout_req_status ON commission_payout_requests(status);
CREATE INDEX idx_comm_payout_req_beneficiary ON commission_payout_requests(beneficiary_user_id);
CREATE INDEX idx_comm_payout_req_created_at ON commission_payout_requests(created_at);

-- =====================================================
-- PAYOUTS (ACTUAL MONEY SENT)
-- PAY-XXXXXXXX as requested
-- Includes: date, amount, method, status, admin reference/transaction id visible to user
-- =====================================================

CREATE TABLE commission_payouts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  payout_number VARCHAR(20) NOT NULL UNIQUE,
  -- PAY-XXXXXXXX

  organization_id UUID NOT NULL REFERENCES organizations(id),

  -- Who got paid
  beneficiary_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- Optional linkage to request
  payout_request_id UUID
    REFERENCES commission_payout_requests(id) ON DELETE SET NULL,

  method commission_payout_method_enum NOT NULL DEFAULT 'manual',

  amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
  currency CHAR(3) NOT NULL DEFAULT 'CAD',

  status commission_payout_status_enum NOT NULL DEFAULT 'pending',

  -- Notes
  payout_note TEXT,        -- internal/admin note
  user_visible_note TEXT,  -- shown to user (e.g., eTransfer ref, PayPal txn id)

  -- External provider references
  stripe_transfer_id VARCHAR(150),
  stripe_payout_id VARCHAR(150),
  paypal_payout_batch_id VARCHAR(150),
  paypal_transaction_id VARCHAR(150),

  -- Manual confirmation reference (bank transfer / eTransfer / cheque)
  manual_reference VARCHAR(150),

  -- Admin actor
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  processed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMP,
  paid_at TIMESTAMP,

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comm_payouts_org ON commission_payouts(organization_id);
CREATE INDEX idx_comm_payouts_beneficiary ON commission_payouts(beneficiary_user_id);
CREATE INDEX idx_comm_payouts_status ON commission_payouts(status);
CREATE INDEX idx_comm_payouts_created_at ON commission_payouts(created_at);

-- =====================================================
-- PAYOUT ITEMS (WHICH COMMISSION TX WERE PAID IN THIS PAYOUT)
-- This is the accounting binding between payout and ledger rows.
-- =====================================================

CREATE TABLE commission_payout_items (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  payout_id UUID NOT NULL
    REFERENCES commission_payouts(id) ON DELETE CASCADE,

  commission_transaction_id UUID NOT NULL
    REFERENCES commission_transactions(id) ON DELETE RESTRICT,

  amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
  currency CHAR(3) NOT NULL DEFAULT 'CAD',

  created_at TIMESTAMP NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_comm_payout_items_unique_tx
    UNIQUE (commission_transaction_id)
);

CREATE INDEX idx_comm_payout_items_payout ON commission_payout_items(payout_id);

-- Now add payout_id FK to commission_transactions (after payout table exists)
ALTER TABLE commission_transactions
  ADD CONSTRAINT fk_commission_tx_payout
  FOREIGN KEY (payout_id)
  REFERENCES commission_payouts(id)
  ON DELETE SET NULL;

-- =====================================================
-- PAYOUT EVENTS (AUDIT)
-- =====================================================

CREATE TABLE commission_payout_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  payout_id UUID NOT NULL
    REFERENCES commission_payouts(id) ON DELETE CASCADE,

  actor_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  from_status commission_payout_status_enum,
  to_status commission_payout_status_enum NOT NULL,

  message TEXT,
  metadata JSONB,

  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comm_payout_events_payout ON commission_payout_events(payout_id);
CREATE INDEX idx_comm_payout_events_created_at ON commission_payout_events(created_at);

-- =====================================================
-- NOTES / LIFECYCLE
-- =====================================================
-- 1) Commission transactions are created when shipment is created OR when invoice is created.
--    - status='pending' until shipment completed.
--
-- 2) On shipment completion:
--    - if include_refunded_shipments=false, ensure shipment not refunded before marking eligible
--    - set status='eligible', eligible_at=NOW()
--
-- 3) Payout processing:
--    - select eligible transactions FOR UPDATE
--    - move them to status='processing' and set processing_at
--    - create commission_payouts row (PAY-XXXXXXXX)
--    - create commission_payout_items rows
--
-- 4) When payout completes:
--    - update payout status='paid', paid_at=NOW()
--    - update linked commission_transactions status='paid', paid_at=NOW()
--
-- 5) If payout fails:
--    - payout status='failed'
--    - return tx from 'processing' back to 'eligible' (or 'voided' if necessary)
-- =====================================================
