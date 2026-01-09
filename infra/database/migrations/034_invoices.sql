-- =====================================================
-- 034_invoices.sql
-- Invoices, Credit Notes, Items & Payments
-- Organization-Scoped with Full Audit Trail
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

-- for Refunds and Returns
CREATE TYPE invoice_payment_direction_enum AS ENUM ('in', 'out');

-- For charges or packages in invoice_items
CREATE TYPE invoice_item_section_enum AS ENUM ('charges', 'packages');

-- Document type
CREATE TYPE invoice_type_enum AS ENUM (
    'invoice',        -- INV-xxxxx (standard shipment invoice)
    'credit_note',    -- CR-xxxxx  (refund/adjustment credit)
    'adjustment'      -- ADJ-xxxxx (weight/import/export charges)
);

-- Document lifecycle
CREATE TYPE invoice_status_enum AS ENUM (
    'draft',
    'issued',
    'void'
);

-- Payment state (financial)
CREATE TYPE invoice_payment_status_enum AS ENUM (
    'unpaid',
    'partial',
    'paid',
    'overpaid'
);

-- How the invoice is funded/paid
CREATE TYPE invoice_payment_method_enum AS ENUM (
    'unassigned',       -- draft/issued but not yet paid
    'wallet',           -- prepaid wallet debit
    'carrier_account',  -- billed to carrier (admin/manager/sales)
    'stripe',           -- card payment
    'manual'            -- cash/e-transfer/wire/manual entry
);

-- Status of a payment record
CREATE TYPE invoice_payment_event_status_enum AS ENUM (
    'pending',
    'completed',
    'failed',
    'refunded',
    'void'
);

-- Credit note reasons
CREATE TYPE credit_note_reason_enum AS ENUM (
    'full_refund',           -- Complete shipment refund
    'partial_refund',        -- Partial refund (e.g., one package)
    'rate_adjustment',       -- Carrier rate was lower than quoted
    'service_issue',         -- Carrier service failure
    'duplicate_charge',      -- Accidental double charge
    'wallet_topup_return',   -- Customer wants topup refunded
    'promotional_credit',    -- Marketing/goodwill credit
    'billing_error',         -- Internal billing mistake
    'other'
);

-- Invoice item types
CREATE TYPE invoice_item_type_enum AS ENUM (
    'base_rate',        -- Carrier base shipping rate
    'fuel_surcharge',   -- Fuel surcharge
    'insurance',        -- Declared value insurance
    'residential',      -- Residential delivery surcharge
    'signature',        -- Signature required surcharge
    'oversize',         -- Oversize package surcharge
    'weight_adjustment',-- Weight correction charge
    'import_duty',      -- Import/customs duty
    'export_fee',       -- Export documentation fee
    'pickup_fee',       -- Scheduled pickup fee
    'saturday_delivery',-- Weekend delivery surcharge
    'handling',         -- Special handling
    'markup',           -- Platform markup/margin
    'discount',         -- Applied discount (negative)
    'tax_hst',          -- HST tax
    'tax_gst',          -- GST tax
    'tax_pst',          -- PST tax
    'other'             -- Other charges
);

-- =====================================================
-- INVOICES (HEADER)
-- =====================================================

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- OWNERSHIP / SCOPE
    -- -------------------------------------------------
    organization_id UUID NOT NULL REFERENCES organizations(id),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- -------------------------------------------------
    -- INVOICE TYPE & CATEGORY
    -- -------------------------------------------------
    invoice_type invoice_type_enum NOT NULL DEFAULT 'invoice',

    -- For credit notes: link to original invoice
    parent_invoice_id UUID
        REFERENCES invoices(id)
        ON DELETE SET NULL,

    -- For adjustments: link to related shipment invoice
    related_invoice_id UUID
        REFERENCES invoices(id)
        ON DELETE SET NULL,

    -- -------------------------------------------------
    -- CREDIT NOTE REASON (only for credit_note type)
    -- -------------------------------------------------
    credit_note_reason credit_note_reason_enum,

    -- -------------------------------------------------
    -- SHIPMENT LINKAGE
    -- -------------------------------------------------
    shipment_id UUID,
    -- Required for 'invoice' type
    -- Optional for 'adjustment' type (weight changes after delivery)
    -- NULL for 'credit_note' type

    -- -------------------------------------------------
    -- INVOICE IDENTITY
    -- -------------------------------------------------
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    -- INV-1JTO541L
    -- CR-1JTO54M1
    -- ADJ-Z51XO541

    status invoice_status_enum NOT NULL DEFAULT 'issued',
    payment_status invoice_payment_status_enum NOT NULL DEFAULT 'unpaid',

    -- -------------------------------------------------
    -- PAYMENT METHOD (PRIMARY / SUMMARY)
    -- -------------------------------------------------
    payment_method invoice_payment_method_enum NOT NULL DEFAULT 'unassigned',

    -- External reference for payment or reconciliation
    payment_reference VARCHAR(150),

    -- Optional human note
    payment_note TEXT,

    -- -------------------------------------------------
    -- BILLING MODE
    -- -------------------------------------------------
    -- TRUE = billed to carrier account (admin/manager/sales shipments)
    -- FALSE = billed to customer wallet (normal flow)
    is_carrier_billed BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- BILL TO (SNAPSHOT – LEGAL RECORD)
    -- -------------------------------------------------
    bill_to_name VARCHAR(150),
    bill_to_company VARCHAR(150),
    bill_to_email VARCHAR(150),
    bill_to_phone VARCHAR(50),

    bill_to_country CHAR(2),
    bill_to_region_code VARCHAR(10),
    bill_to_city VARCHAR(100),
    bill_to_postal_code VARCHAR(20),
    bill_to_address_line1 VARCHAR(255),
    bill_to_address_line2 VARCHAR(255),
    bill_to_tax_id VARCHAR(50),

    -- -------------------------------------------------
    -- SHIPMENT SUMMARY (PAGE 1 OF PDF)
    -- -------------------------------------------------
    shipment_number VARCHAR(50),
    shipment_from_summary VARCHAR(255),
    shipment_to_summary VARCHAR(255),
    carrier_code VARCHAR(50),
    carrier_name VARCHAR(100),
    service_code VARCHAR(100),
    service_name VARCHAR(150),
    total_packages INTEGER NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- MONEY (ACCOUNTING SNAPSHOT)
    -- -------------------------------------------------
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    subtotal_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    taxes_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,

    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    amount_paid NUMERIC(14,2) NOT NULL DEFAULT 0,
    balance_due NUMERIC(14,2) NOT NULL DEFAULT 0,

    -- USD equivalent (for display only)
    total_amount_usd NUMERIC(14,2),
    exchange_rate_used NUMERIC(10,6),

    -- Add for keeping a track of max refund allowed - match total amount paid
    refundable_amount NUMERIC(14,2) NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- DISCOUNTS INFORMATION
    -- -------------------------------------------------
    discount_id UUID,
    discount_code VARCHAR(50),
    discount_description TEXT,

    -- -------------------------------------------------
    -- FILES
    -- -------------------------------------------------
    pdf_file_id UUID REFERENCES files(id),

    -- -------------------------------------------------
    -- DATES
    -- -------------------------------------------------
    issued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    due_at TIMESTAMP,
    paid_at TIMESTAMP,
    voided_at TIMESTAMP,

    -- -------------------------------------------------
    -- METADATA & AUDIT
    -- -------------------------------------------------
    notes TEXT,
    internal_notes TEXT,  -- Admin-only notes
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_invoice_amounts_non_negative
        CHECK (
            (invoice_type IN ('invoice', 'adjustment') AND subtotal_amount >= 0)
            OR invoice_type = 'credit_note'
        ),

    CONSTRAINT chk_invoice_balance_math
        CHECK (ROUND(amount_paid + balance_due, 2) = ROUND(total_amount, 2)),

    CONSTRAINT chk_credit_note_reason_required
        CHECK (
            (invoice_type = 'credit_note' AND credit_note_reason IS NOT NULL)
            OR invoice_type <> 'credit_note'
        ),

    CONSTRAINT chk_credit_note_amounts
        CHECK (
            (invoice_type = 'credit_note' AND total_amount <= 0 AND balance_due = 0)
            OR invoice_type <> 'credit_note'
        ),

    CONSTRAINT chk_invoice_shipment_rules
        CHECK (
            -- Standard invoice requires shipment
            (invoice_type = 'invoice' AND shipment_id IS NOT NULL)
            -- Adjustment invoice: shipment optional (may be post-delivery)
            OR invoice_type = 'adjustment'
            -- Credit note: no shipment, must have parent
            OR (invoice_type = 'credit_note' AND shipment_id IS NULL AND parent_invoice_id IS NOT NULL)
        ),

    CONSTRAINT chk_carrier_billed_rules
        CHECK (
            (is_carrier_billed = FALSE)
        OR (
            is_carrier_billed = TRUE
            AND payment_method = 'carrier_account'
            AND payment_status = 'paid'
            AND balance_due = 0
        )
    ),

    CONSTRAINT chk_credit_note_paid_math
        CHECK (
            invoice_type <> 'credit_note'
        OR (
            payment_status = 'paid'
            AND ROUND(amount_paid, 2) = ROUND(total_amount, 2)
            AND balance_due = 0
        )
    )
);

-- =====================================================
-- INVOICE ITEMS (LINE ITEMS)
-- Page 1: Charges breakdown
-- Page 2+: Package details
-- =====================================================

CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    invoice_id UUID NOT NULL
        REFERENCES invoices(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- ITEM ORDERING
    -- -------------------------------------------------
    item_index INTEGER NOT NULL DEFAULT 1,
    -- For display order on invoice


    -- -------------------------------------------------
    -- ITEM CLASSIFICATION
    -- -------------------------------------------------
    item_type invoice_item_type_enum NOT NULL,

    -- Section grouping for PDF rendering
    -- 'charges' = Page 1 (rates, surcharges, taxes)
    -- 'packages' = Page 2+ (package details)
    section invoice_item_section_enum NOT NULL DEFAULT 'charges',

    -- -------------------------------------------------
    -- PACKAGE LINKAGE (for package items only)
    -- -------------------------------------------------
    package_id UUID
        REFERENCES shipment_packages(id)
        ON DELETE SET NULL,

    -- Package details snapshot (for PDF, even if package deleted)
    package_index INTEGER,
    package_number VARCHAR(30),        -- PKG-K9R2B7X4-01
    tracking_number VARCHAR(100),

    -- -------------------------------------------------
    -- PACKAGE SPECS (snapshot for Page 2)
    -- -------------------------------------------------
    package_name VARCHAR(150),         -- "Electronics - Laptop"
    package_weight NUMERIC(10,3),
    package_weight_unit VARCHAR(10), -- "lb/kg"
    package_dimensions VARCHAR(50),    -- "25x15x8"
    package_dimensions_unit VARCHAR(10), -- "cm/inches"
    package_declared_value NUMERIC(14,2),

    -- -------------------------------------------------
    -- ITEM DETAILS
    -- -------------------------------------------------
    description TEXT,
    quantity INTEGER DEFAULT 1,
    unit_price NUMERIC(14,2),
    amount NUMERIC(14,2),

    -- Tax details (if this is a tax line)
    tax_rate NUMERIC(5,2),
    tax_name VARCHAR(50),

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_invoice_item_index_positive
        CHECK (item_index >= 1),

    CONSTRAINT chk_invoice_item_package_section
        CHECK (
            section <> 'packages'
        OR (section = 'packages' AND (package_id IS NOT NULL OR package_number IS NOT NULL))
        )
);

-- =====================================================
-- INVOICE PAYMENTS (PAYMENT RECORDS)
-- Multiple payments can fund one invoice
-- =====================================================

CREATE TABLE invoice_payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    invoice_id UUID NOT NULL
        REFERENCES invoices(id)
        ON DELETE CASCADE,

    -- Human-readable identifier
    payment_number VARCHAR(50) NOT NULL UNIQUE,
    -- PAY-XXXXXXXX

    method invoice_payment_method_enum NOT NULL,
    status invoice_payment_event_status_enum NOT NULL DEFAULT 'pending',

    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- for returns / refunds, ins
    direction invoice_payment_direction_enum NOT NULL DEFAULT 'in',

    -- Wallet linkage (if method = 'wallet')
    wallet_transaction_id UUID
        REFERENCES wallet_transactions(id)
        ON DELETE SET NULL,

    -- Stripe linkage (if method = 'stripe')
    stripe_payment_intent_id VARCHAR(150),
    stripe_charge_id VARCHAR(150),

    -- External reference
    provider_reference VARCHAR(150),

    -- Notes
    note TEXT,
    metadata JSONB,

    -- Actor
    processed_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    processed_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_invoice_payment_amount_positive
        CHECK (amount > 0),

    CONSTRAINT chk_wallet_tx_required_for_wallet
        CHECK (
            method <> 'wallet'
            OR (method = 'wallet' AND wallet_transaction_id IS NOT NULL)
        ),

    CONSTRAINT chk_payment_number_format
        CHECK (payment_number ~ '^PAY-[A-Z0-9]{8}$')
);

ALTER TABLE invoice_payments
ADD CONSTRAINT chk_payment_method_field_exclusivity
CHECK (
    (method = 'wallet' AND wallet_transaction_id IS NOT NULL AND stripe_payment_intent_id IS NULL AND stripe_charge_id IS NULL)
 OR (method = 'stripe' AND wallet_transaction_id IS NULL AND (stripe_payment_intent_id IS NOT NULL OR stripe_charge_id IS NOT NULL))
 OR (method IN ('manual','carrier_account') AND wallet_transaction_id IS NULL AND stripe_payment_intent_id IS NULL AND stripe_charge_id IS NULL)
);

ALTER TABLE invoice_payments
ADD CONSTRAINT chk_refund_direction_requires_completed_or_refunded
CHECK (
    direction <> 'out'
 OR status IN ('completed','refunded')
);

-- =====================================================
-- INVOICE EVENTS (AUDIT LOG)
-- =====================================================

CREATE TABLE invoice_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    invoice_id UUID NOT NULL
        REFERENCES invoices(id)
        ON DELETE CASCADE,

    actor_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    event_type VARCHAR(50) NOT NULL,
    -- created, issued, paid, partial_payment, voided, 
    -- payment_failed, refunded, adjusted

    from_status VARCHAR(30),
    to_status VARCHAR(30),

    from_payment_status VARCHAR(30),
    to_payment_status VARCHAR(30),

    amount NUMERIC(14,2),
    description TEXT,
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =====================================================
-- FOREIGN KEYS (DEFERRED)
-- =====================================================
ALTER TABLE invoices
    ADD CONSTRAINT chk_adjustment_linkage
    CHECK (
        invoice_type <> 'adjustment'
    OR (shipment_id IS NOT NULL OR related_invoice_id IS NOT NULL)
    );

ALTER TABLE invoices
    ADD CONSTRAINT invoices_shipment_id_fk
    FOREIGN KEY (shipment_id)
    REFERENCES shipments(id)
    ON DELETE SET NULL;

ALTER TABLE invoices
    ADD CONSTRAINT invoices_discount_id_fk
    FOREIGN KEY (discount_id)
    REFERENCES discounts(id)
    ON DELETE SET NULL;

-- =====================================================
-- INDEXES
-- =====================================================

-- Invoices
CREATE INDEX idx_invoices_org_id ON invoices(organization_id);
CREATE INDEX idx_invoices_created_by ON invoices(created_by_user_id);
CREATE INDEX idx_invoices_shipment_id ON invoices(shipment_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_payment_status ON invoices(payment_status);
CREATE INDEX idx_invoices_payment_method ON invoices(payment_method);
CREATE INDEX idx_invoices_type ON invoices(invoice_type);
CREATE INDEX idx_invoices_parent ON invoices(parent_invoice_id);
CREATE INDEX idx_invoices_related ON invoices(related_invoice_id);
CREATE INDEX idx_invoices_issued_at ON invoices(issued_at);
CREATE INDEX idx_invoices_carrier_billed ON invoices(is_carrier_billed);

-- Unpaid invoices (for wallet widget)
CREATE INDEX idx_invoices_unpaid 
    ON invoices(organization_id, payment_status) 
    WHERE payment_status IN ('unpaid', 'partial');

-- Invoice items
CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_package ON invoice_items(package_id);
CREATE INDEX idx_invoice_items_type ON invoice_items(item_type);
CREATE INDEX idx_invoice_items_section ON invoice_items(section);

-- Invoice payments
CREATE INDEX idx_invoice_payments_invoice ON invoice_payments(invoice_id);
CREATE INDEX idx_invoice_payments_status ON invoice_payments(status);
CREATE INDEX idx_invoice_payments_wallet_tx ON invoice_payments(wallet_transaction_id);
CREATE INDEX idx_invoice_payments_stripe ON invoice_payments(stripe_payment_intent_id);

-- Invoice events
CREATE INDEX idx_invoice_events_invoice ON invoice_events(invoice_id);
CREATE INDEX idx_invoice_events_created ON invoice_events(created_at);

-- =====================================================
-- HELPER VIEW: Wallet Widget Data
-- =====================================================

-- =====================================================
-- NOTES
-- =====================================================
-- 
-- INVOICE TYPES:
-- 1. invoice     - Standard shipment invoice (INV-xxxxx)
-- 2. credit_note - Refund/credit (CR-xxxxx), always negative
-- 3. adjustment  - Post-shipment charges (ADJ-xxxxx)
--
-- PAYMENT FLOW:
-- 1. Invoice created with payment_status = 'unpaid'
-- 2. Payment recorded in invoice_payments
-- 3. Invoice amount_paid updated, balance_due recalculated
-- 4. When balance_due = 0, payment_status = 'paid'
--
-- ADMIN/MANAGER/SALES SHIPMENTS:
-- - is_carrier_billed = TRUE
-- - payment_method = 'carrier_account'
-- - Still generates invoice for tracking
--
-- PDF STRUCTURE:
-- - Page 1: Header, charges breakdown (section='charges')
-- - Page 2+: Package details (section='packages')
--
-- =====================================================

-- =====================================================
-- WALLET WIDGET VIEWS
-- =====================================================

-- =====================================================
-- FIX 7: Update v_wallet_summary to exclude credit_notes
-- =====================================================
CREATE OR REPLACE VIEW v_wallet_summary AS
SELECT 
    w.organization_id,
    w.wallet_number,
    w.balance AS available_balance,
    w.status AS wallet_status,
    COALESCE(unpaid.total_unpaid, 0) AS unpaid_amount,
    COALESCE(unpaid.unpaid_count, 0) AS unpaid_invoice_count
FROM wallets w
LEFT JOIN (
    SELECT 
        organization_id,
        SUM(balance_due) AS total_unpaid,
        COUNT(*) AS unpaid_count
    FROM invoices
    WHERE invoice_type IN ('invoice', 'adjustment')  -- Exclude credit_note
      AND payment_status IN ('unpaid', 'partial')
      AND status = 'issued'
      AND is_carrier_billed = FALSE
    GROUP BY organization_id
) unpaid ON unpaid.organization_id = w.organization_id;


-- =====================================================
-- FIX 8: Add v_invoice_refund_summary view
-- (Useful for preventing over-refunds)
-- =====================================================

CREATE OR REPLACE VIEW v_invoice_refund_summary AS
SELECT 
    i.id AS invoice_id,
    i.invoice_number,
    i.organization_id,
    i.total_amount AS original_amount,
    i.refundable_amount,
    COALESCE(ABS(SUM(cn.total_amount)), 0) AS total_refunded,
    i.refundable_amount - COALESCE(ABS(SUM(cn.total_amount)), 0) AS refundable_remaining
FROM invoices i
LEFT JOIN invoices cn ON cn.parent_invoice_id = i.id 
    AND cn.invoice_type = 'credit_note'
    AND cn.status <> 'void'
WHERE i.invoice_type = 'invoice'
GROUP BY i.id, i.invoice_number, i.organization_id, i.total_amount, i.refundable_amount;