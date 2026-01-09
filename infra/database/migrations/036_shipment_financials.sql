-- =====================================================
-- 036_shipment_financials.sql
-- Shipment Financial Records (Profit & Cost Tracking)
-- =====================================================
--
-- PURPOSE:
-- This is the SINGLE SOURCE OF TRUTH for understanding:
-- - What the customer paid
-- - What we paid the carrier
-- - Our gross profit
-- - Commissions owed
-- - Net profit
--
-- WHY SEPARATE FROM INVOICES?
-- - Invoices are customer-facing documents
-- - Carrier costs should NOT be visible to customers
-- - This table is internal/admin only
--
-- WHY NOT IN SHIPMENTS?
-- - Shipments is already large
-- - Financial data may be updated post-shipment (adjustments)
-- - Keeps concerns separated
-- =====================================================


-- =====================================================
-- SHIPMENT FINANCIALS
-- =====================================================
-- One row per shipment
-- Created when shipment is finalized (label purchased)
-- Updated when adjustments occur

CREATE TABLE shipment_financials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- LINKAGE
    -- -------------------------------------------------
    shipment_id UUID NOT NULL UNIQUE
        REFERENCES shipments(id)
        ON DELETE CASCADE,

    organization_id UUID NOT NULL
        REFERENCES organizations(id),

    invoice_id UUID
        REFERENCES invoices(id)
        ON DELETE SET NULL,

    -- -------------------------------------------------
    -- CUSTOMER REVENUE (What we charged)
    -- -------------------------------------------------
    customer_subtotal NUMERIC(14,2) NOT NULL,
    -- Base rate + surcharges (before tax)

    customer_taxes NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- HST/GST/PST

    customer_total NUMERIC(14,2) NOT NULL,
    -- Total charged to customer

    discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Discount applied (reduces revenue)

    customer_net_revenue NUMERIC(14,2) NOT NULL,
    -- customer_total - discount_amount
    -- This is our actual revenue

    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- CARRIER COST (What we paid)
    -- -------------------------------------------------
    carrier_base_cost NUMERIC(14,2) NOT NULL,
    -- Base rate from carrier

    carrier_fuel_surcharge NUMERIC(14,2) NOT NULL DEFAULT 0,
    carrier_surcharges NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Other carrier surcharges (residential, oversize, etc.)

    carrier_taxes NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Taxes carrier charged us (if any)

    carrier_total_cost NUMERIC(14,2) NOT NULL,
    -- Total we paid to carrier

    carrier_currency CHAR(3) NOT NULL DEFAULT 'CAD',
    -- Usually same as customer currency

    -- -------------------------------------------------
    -- RATE SNAPSHOT (At time of quote)
    -- -------------------------------------------------
    quoted_rate NUMERIC(14,2) NOT NULL,
    -- What we quoted customer

    carrier_rate_at_quote NUMERIC(14,2) NOT NULL,
    -- What carrier quoted us

    rate_locked_at TIMESTAMP NOT NULL,
    -- When rate was locked in

    -- -------------------------------------------------
    -- GROSS PROFIT
    -- -------------------------------------------------
    gross_profit NUMERIC(14,2) NOT NULL,
    -- customer_net_revenue - carrier_total_cost

    gross_margin_percent NUMERIC(5,2) NOT NULL,
    -- (gross_profit / customer_net_revenue) * 100

    -- -------------------------------------------------
    -- PLATFORM FEES (Our markup breakdown)
    -- -------------------------------------------------
    platform_markup_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Fixed markup we added

    platform_markup_percent NUMERIC(5,2),
    -- Percentage markup we used

    -- -------------------------------------------------
    -- COMMISSIONS (What we owe)
    -- -------------------------------------------------
    referral_commission NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Commission to referrer (if referred user)

    affiliate_commission NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Commission to affiliate/partner

    sales_commission NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Commission to sales rep (if applicable)

    total_commission NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Sum of all commissions

    -- -------------------------------------------------
    -- NET PROFIT
    -- -------------------------------------------------
    net_profit NUMERIC(14,2) NOT NULL,
    -- gross_profit - total_commission

    net_margin_percent NUMERIC(5,2) NOT NULL,
    -- (net_profit / customer_net_revenue) * 100

    -- -------------------------------------------------
    -- ADJUSTMENTS (Post-shipment changes)
    -- -------------------------------------------------
    adjustment_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Additional charges/credits after initial shipment
    -- Positive = we charged more
    -- Negative = we refunded

    adjustment_reason TEXT,

    has_adjustments BOOLEAN NOT NULL DEFAULT FALSE,

    -- -------------------------------------------------
    -- BILLING MODE
    -- -------------------------------------------------
    is_carrier_billed BOOLEAN NOT NULL DEFAULT FALSE,
    -- TRUE = Admin/manager shipment, no customer charge
    -- Revenue = 0, but we still track carrier cost

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_finalized BOOLEAN NOT NULL DEFAULT FALSE,
    -- TRUE = all costs confirmed, no more changes expected

    finalized_at TIMESTAMP,

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    rate_source VARCHAR(50),
    -- 'carrier_api', 'cached_rate', 'manual_quote'

    carrier_invoice_number VARCHAR(100),
    -- Carrier's invoice to us (for reconciliation)

    carrier_invoice_date DATE,

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
    CONSTRAINT chk_sf_customer_total
        CHECK (customer_total >= 0),

    CONSTRAINT chk_sf_carrier_cost
        CHECK (carrier_total_cost >= 0),

    CONSTRAINT chk_sf_gross_profit_math
        CHECK (
            is_carrier_billed = TRUE
            OR ABS(gross_profit - (customer_net_revenue - carrier_total_cost)) < 0.01
        ),

    CONSTRAINT chk_sf_net_profit_math
        CHECK (
            ABS(net_profit - (gross_profit - total_commission)) < 0.01
        ),

    CONSTRAINT chk_sf_total_commission
        CHECK (
            ABS(total_commission - (referral_commission + affiliate_commission + sales_commission)) < 0.01
        )
);

-- =====================================================
-- SHIPMENT FINANCIAL ADJUSTMENTS
-- =====================================================
-- Track individual post-shipment adjustments

CREATE TABLE shipment_financial_adjustments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    adjustment_number VARCHAR(20) UNIQUE NOT NULL,  -- ADJ-XXXXXXXX

    shipment_financial_id UUID NOT NULL
        REFERENCES shipment_financials(id)
        ON DELETE CASCADE,

    shipment_id UUID NOT NULL
        REFERENCES shipments(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- ADJUSTMENT TYPE
    -- -------------------------------------------------
    adjustment_type VARCHAR(50) NOT NULL,
    -- weight_correction, dimension_correction, address_correction,
    -- import_duty, export_fee, fuel_adjustment, service_upgrade,
    -- residential_correction, carrier_audit, refund, etc.

    -- -------------------------------------------------
    -- AMOUNTS
    -- -------------------------------------------------
    customer_amount NUMERIC(14,2) NOT NULL,
    -- What we charged/credited customer

    carrier_amount NUMERIC(14,2) NOT NULL,
    -- What carrier charged/credited us

    profit_impact NUMERIC(14,2) NOT NULL,
    -- customer_amount - carrier_amount

    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- -------------------------------------------------
    -- INVOICE LINKAGE
    -- -------------------------------------------------
    adjustment_invoice_id UUID,
    -- Links to adjustment invoice (if created)

    -- -------------------------------------------------
    -- SOURCE
    -- -------------------------------------------------
    source VARCHAR(50) NOT NULL,
    -- carrier_audit, manual, webhook, etc.

    carrier_reference VARCHAR(100),
    -- Carrier's reference for this adjustment

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    created_by_user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    reason TEXT NOT NULL,
    notes TEXT,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_sfa_number_format
        CHECK (adjustment_number ~ '^ADJ-[A-Z0-9]{8}$'),

    CONSTRAINT chk_sfa_profit_math
        CHECK (ABS(profit_impact - (customer_amount - carrier_amount)) < 0.01)
);

-- =====================================================
-- PLATFORM REVENUE SUMMARY (Daily)
-- =====================================================
-- Aggregated daily revenue for reporting

CREATE TABLE platform_revenue_daily (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- DATE
    -- -------------------------------------------------
    report_date DATE NOT NULL UNIQUE,

    -- -------------------------------------------------
    -- VOLUME
    -- -------------------------------------------------
    total_shipments INTEGER NOT NULL DEFAULT 0,
    total_packages INTEGER NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- REVENUE
    -- -------------------------------------------------
    gross_revenue NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Total customer_net_revenue

    discounts_given NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Total discounts

    net_revenue NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- gross_revenue - discounts_given

    -- -------------------------------------------------
    -- COSTS
    -- -------------------------------------------------
    carrier_costs NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Total carrier_total_cost

    -- -------------------------------------------------
    -- PROFIT
    -- -------------------------------------------------
    gross_profit NUMERIC(14,2) NOT NULL DEFAULT 0,
    gross_margin_percent NUMERIC(5,2),

    total_commissions NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- referral + affiliate + sales

    net_profit NUMERIC(14,2) NOT NULL DEFAULT 0,
    net_margin_percent NUMERIC(5,2),

    -- -------------------------------------------------
    -- ADJUSTMENTS
    -- -------------------------------------------------
    adjustment_revenue NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Additional revenue from adjustments

    adjustment_costs NUMERIC(14,2) NOT NULL DEFAULT 0,
    -- Additional costs from adjustments

    -- -------------------------------------------------
    -- BY CARRIER
    -- -------------------------------------------------
    revenue_by_carrier JSONB,
    -- { "fedex": 1000, "ups": 500, ... }

    profit_by_carrier JSONB,
    -- { "fedex": 500, "ups": 200, ... }

    -- -------------------------------------------------
    -- BY PAYMENT METHOD
    -- -------------------------------------------------
    wallet_payments NUMERIC(14,2) NOT NULL DEFAULT 0,
    stripe_payments NUMERIC(14,2) NOT NULL DEFAULT 0,
    carrier_billed NUMERIC(14,2) NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_finalized BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Shipment Financials
CREATE INDEX idx_sf_shipment ON shipment_financials(shipment_id);
CREATE INDEX idx_sf_org ON shipment_financials(organization_id);
CREATE INDEX idx_sf_invoice ON shipment_financials(invoice_id);
CREATE INDEX idx_sf_created ON shipment_financials(created_at);
CREATE INDEX idx_sf_carrier_billed ON shipment_financials(is_carrier_billed);
CREATE INDEX idx_sf_finalized ON shipment_financials(is_finalized);

-- Financial Adjustments
CREATE INDEX idx_sfa_financial ON shipment_financial_adjustments(shipment_financial_id);
CREATE INDEX idx_sfa_shipment ON shipment_financial_adjustments(shipment_id);
CREATE INDEX idx_sfa_type ON shipment_financial_adjustments(adjustment_type);
CREATE INDEX idx_sfa_created ON shipment_financial_adjustments(created_at);

-- Daily Revenue
CREATE INDEX idx_prd_date ON platform_revenue_daily(report_date);
CREATE INDEX idx_prd_finalized ON platform_revenue_daily(is_finalized);

-- =====================================================
-- HELPER VIEWS
-- =====================================================

-- Profit by carrier (current month)
CREATE OR REPLACE VIEW v_profit_by_carrier_mtd AS
SELECT 
    s.carrier_id,
    c.code AS carrier_code,
    c.name AS carrier_name,
    COUNT(*) AS shipment_count,
    SUM(sf.customer_net_revenue) AS total_revenue,
    SUM(sf.carrier_total_cost) AS total_cost,
    SUM(sf.gross_profit) AS total_gross_profit,
    AVG(sf.gross_margin_percent) AS avg_margin_percent,
    SUM(sf.total_commission) AS total_commissions,
    SUM(sf.net_profit) AS total_net_profit
FROM shipment_financials sf
JOIN shipments s ON s.id = sf.shipment_id
JOIN carriers c ON c.id = s.carrier_id
WHERE sf.created_at >= DATE_TRUNC('month', CURRENT_DATE)
GROUP BY s.carrier_id, c.code, c.name;

-- Commission summary by user
CREATE OR REPLACE VIEW v_commission_summary AS
SELECT 
    rr.referrer_user_id AS user_id,
    u.display_name AS user_name,
    'referral' AS commission_type,
    COUNT(DISTINCT sf.shipment_id) AS shipment_count,
    SUM(sf.referral_commission) AS total_commission
FROM shipment_financials sf
JOIN referral_redemptions rr ON rr.referred_user_id = (
    SELECT created_by_user_id FROM shipments WHERE id = sf.shipment_id
)
JOIN users u ON u.id = rr.referrer_user_id
WHERE sf.referral_commission > 0
GROUP BY rr.referrer_user_id, u.display_name

UNION ALL

SELECT 
    ct.beneficiary_user_id AS user_id,
    u.display_name AS user_name,
    'affiliate' AS commission_type,
    COUNT(DISTINCT sf.shipment_id) AS shipment_count,
    SUM(sf.affiliate_commission) AS total_commission
FROM shipment_financials sf
JOIN commission_transactions ct ON ct.shipment_id = sf.shipment_id
JOIN users u ON u.id = ct.beneficiary_user_id
WHERE sf.affiliate_commission > 0
GROUP BY ct.beneficiary_user_id, u.display_name;

-- =====================================================
-- NOTES
-- =====================================================

/*
FLOW: How Financial Records Are Created
────────────────────────────────────────────────────────

1. CHECKOUT
   - User pays $100 via wallet or Stripe
   - Carrier API returns cost of $50
   - shipment_financials created:
     customer_total: 100
     carrier_total_cost: 50
     gross_profit: 50
     commission: 5 (to referrer)
     net_profit: 45

2. POST-SHIPMENT ADJUSTMENT (e.g., weight correction)
   - Carrier charges us $5 more
   - We charge customer $8 more (our markup)
   - shipment_financial_adjustments created
   - shipment_financials.adjustment_amount updated
   - New adjustment invoice created

3. DAILY AGGREGATION (Cron job)
   - Aggregate all shipment_financials for the day
   - Create/update platform_revenue_daily row
   - Used for dashboards and reports


ADMIN/MANAGER BYPASS:
────────────────────────────────────────────────────────

When is_carrier_billed = TRUE:
- customer_total = 0
- customer_net_revenue = 0
- carrier_total_cost = actual cost
- gross_profit = negative (we paid, customer didn't)
- This is tracked as "cost" not "revenue"

This lets us see total carrier spend even for internal shipments.


RECONCILIATION:
────────────────────────────────────────────────────────

Monthly reconciliation process:
1. Get carrier invoice
2. Match carrier_invoice_number to our records
3. Compare carrier_total_cost vs their invoice
4. Create adjustments for any discrepancies
5. Mark shipment_financials.is_finalized = TRUE


PROFIT FORMULA:
────────────────────────────────────────────────────────

Gross Profit = Customer Net Revenue - Carrier Total Cost
Net Profit = Gross Profit - Total Commissions
Margin % = (Profit / Revenue) * 100

Example:
  Customer pays: $100
  Discount: $10
  Net Revenue: $90
  Carrier cost: $50
  Gross Profit: $40 (44.4% margin)
  Commissions: $4
  Net Profit: $36 (40% margin)
*/