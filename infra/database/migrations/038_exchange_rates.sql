-- =====================================================
-- 038_exchange_rates.sql
-- Daily Exchange Rates (Central Source of Truth)
-- =====================================================
--
-- PURPOSE:
-- Single source of truth for all exchange rate lookups.
-- All code (estimates, invoices, reports) should query this table.
-- Base currency is always CAD (hardcoded assumption).
--
-- RATE SOURCE OPTIONS:
-- 1. Bank of Canada (official CAD rates) - FREE, daily at ~12:30 ET
--    https://www.bankofcanada.ca/valet/observations/FXUSDCAD/json?recent=1
--
-- 2. Open Exchange Rates (more currencies, faster updates) - PAID
--    https://openexchangerates.org/api/latest.json
--
-- 3. Exchange Rate API - FREE tier available
--    https://api.exchangerate-api.com/v4/latest/CAD
--
-- RECOMMENDED: Bank of Canada for official CAD/USD rate
-- =====================================================

-- =====================================================
-- EXCHANGE RATES TABLE
-- =====================================================

CREATE TABLE exchange_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- CURRENCY PAIR
    -- -------------------------------------------------
    -- Base currency is always CAD (not stored, implied)

    target_currency CHAR(3) NOT NULL,
    -- Target currency (e.g., 'USD')

    -- -------------------------------------------------
    -- RATE
    -- -------------------------------------------------
    rate NUMERIC(18,10) NOT NULL,
    -- Exchange rate: 1 CAD = rate target_currency
    -- Example: CAD to USD might be 0.7412345678
    -- So $100 CAD = $74.12 USD

    inverse_rate NUMERIC(18,10) NOT NULL,
    -- Inverse for quick lookup: 1 target = inverse_rate CAD
    -- Example: 1 USD = 1.3491 CAD

    -- -------------------------------------------------
    -- DATE
    -- -------------------------------------------------
    rate_date DATE NOT NULL,
    -- The date this rate applies to

    -- -------------------------------------------------
    -- SOURCE
    -- -------------------------------------------------
    source VARCHAR(50) NOT NULL,
    -- bank_of_canada, open_exchange_rates, exchange_rate_api, manual

    source_rate_id VARCHAR(100),
    -- Provider's identifier for this rate (if available)

    -- -------------------------------------------------
    -- TIMING
    -- -------------------------------------------------
    published_at TIMESTAMP,
    -- When the source published this rate

    fetched_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- When we fetched this rate

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    -- Only one active rate per pair per date

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_exchange_rate_positive
        CHECK (rate > 0),

    CONSTRAINT chk_exchange_inverse_positive
        CHECK (inverse_rate > 0),

    -- Only one active rate per currency pair per date
    -- Base is always CAD, so just target + date
    CONSTRAINT uq_exchange_rate_pair_date
        UNIQUE (target_currency, rate_date)
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Primary lookup: Get today's rate for a currency
CREATE INDEX idx_exchange_rates_lookup
    ON exchange_rates(target_currency, rate_date DESC)
    WHERE is_active = TRUE;

-- Get latest rate for any currency
CREATE INDEX idx_exchange_rates_latest
    ON exchange_rates(target_currency, rate_date DESC);

-- Historical lookups by date
CREATE INDEX idx_exchange_rates_date
    ON exchange_rates(rate_date);

-- Cleanup old rates
CREATE INDEX idx_exchange_rates_created
    ON exchange_rates(created_at);

-- =====================================================
-- HELPER FUNCTION: Get Current Rate
-- =====================================================
-- Returns the most recent rate for CAD to target currency
-- Falls back to previous day if today's rate not yet available

CREATE OR REPLACE FUNCTION get_exchange_rate(
    p_target_currency CHAR(3),
    p_for_date DATE DEFAULT CURRENT_DATE
)
RETURNS NUMERIC(18,10) AS $$
DECLARE
    v_rate NUMERIC(18,10);
BEGIN
    -- Same currency (CAD to CAD), return 1
    IF p_target_currency = 'CAD' THEN
        RETURN 1.0;
    END IF;

    -- Try to get rate for the specified date
    SELECT rate INTO v_rate
    FROM exchange_rates
    WHERE target_currency = p_target_currency
      AND rate_date <= p_for_date
      AND is_active = TRUE
    ORDER BY rate_date DESC
    LIMIT 1;

    -- If no rate found, return NULL (caller should handle)
    RETURN v_rate;
END;
$$ LANGUAGE plpgsql STABLE;

-- =====================================================
-- HELPER FUNCTION: Convert Amount
-- =====================================================
-- Converts an amount from one currency to another
-- Assumes base is CAD for all stored rates

CREATE OR REPLACE FUNCTION convert_currency(
    p_amount NUMERIC(14,2),
    p_from_currency CHAR(3),
    p_to_currency CHAR(3),
    p_for_date DATE DEFAULT CURRENT_DATE
)
RETURNS NUMERIC(14,2) AS $$
DECLARE
    v_rate NUMERIC(18,10);
    v_inverse_rate NUMERIC(18,10);
BEGIN
    -- Same currency, no conversion needed
    IF p_from_currency = p_to_currency THEN
        RETURN p_amount;
    END IF;

    -- CAD to target: use rate directly
    IF p_from_currency = 'CAD' THEN
        v_rate := get_exchange_rate(p_to_currency, p_for_date);
        IF v_rate IS NULL THEN
            RAISE EXCEPTION 'No exchange rate found for CAD to %', p_to_currency;
        END IF;
        RETURN ROUND(p_amount * v_rate, 2);
    END IF;

    -- Target to CAD: use inverse_rate
    IF p_to_currency = 'CAD' THEN
        SELECT inverse_rate INTO v_inverse_rate
        FROM exchange_rates
        WHERE target_currency = p_from_currency
          AND rate_date <= p_for_date
          AND is_active = TRUE
        ORDER BY rate_date DESC
        LIMIT 1;
        
        IF v_inverse_rate IS NULL THEN
            RAISE EXCEPTION 'No exchange rate found for % to CAD', p_from_currency;
        END IF;
        RETURN ROUND(p_amount * v_inverse_rate, 2);
    END IF;

    -- Non-CAD to Non-CAD: convert through CAD
    -- First convert to CAD, then to target
    DECLARE
        v_cad_amount NUMERIC(14,2);
    BEGIN
        v_cad_amount := convert_currency(p_amount, p_from_currency, 'CAD', p_for_date);
        RETURN convert_currency(v_cad_amount, 'CAD', p_to_currency, p_for_date);
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- =====================================================
-- VIEW: Latest Rates
-- =====================================================

CREATE OR REPLACE VIEW v_exchange_rates_latest AS
SELECT DISTINCT ON (target_currency)
    id,
    'CAD'::CHAR(3) AS base_currency,
    target_currency,
    rate,
    inverse_rate,
    rate_date,
    source,
    fetched_at
FROM exchange_rates
WHERE is_active = TRUE
ORDER BY target_currency, rate_date DESC;

-- =====================================================
-- NOTES
-- =====================================================

-- /*
-- WORKER SCHEDULE:
-- ────────────────────────────────────────────────────────

-- Bank of Canada publishes rates around 12:30 PM ET daily.
-- Run worker at:
--   - 1:00 PM ET (primary fetch)
--   - 4:00 PM ET (backup fetch if primary failed)

-- For real-time needs, consider caching the rate in Redis
-- with a 1-hour TTL.


-- GO WORKER PSEUDO-CODE:
-- ────────────────────────────────────────────────────────

-- func FetchDailyExchangeRate() error {
--     // 1. Fetch from Bank of Canada API
--     resp, err := http.Get(
--         "https://www.bankofcanada.ca/valet/observations/FXUSDCAD/json?recent=1"
--     )
    
--     // 2. Parse response
--     // Response format:
--     // {
--     //   "observations": [{
--     //     "d": "2026-01-05",
--     //     "FXUSDCAD": {"v": "1.3605"}
--     //   }]
--     // }
    
--     // 3. Calculate inverse (CAD to USD)
--     usdToCad := parseFloat(data.Observations[0].FXUSDCAD.V)
--     cadToUsd := 1.0 / usdToCad
    
--     // 4. Upsert into database
--     _, err = db.Exec(`
--         INSERT INTO exchange_rates (
--             target_currency, rate, inverse_rate,
--             rate_date, source, published_at, fetched_at
--         ) VALUES (
--             'USD', $1, $2, $3, 'bank_of_canada', $4, NOW()
--         )
--         ON CONFLICT (target_currency, rate_date)
--         DO UPDATE SET
--             rate = EXCLUDED.rate,
--             inverse_rate = EXCLUDED.inverse_rate,
--             fetched_at = NOW()
--     `, cadToUsd, usdToCad, rateDate, publishedAt)
    
--     // 5. Cache in Redis for fast lookups
--     redis.Set("exchange:CAD:USD", cadToUsd, 1*time.Hour)
--     redis.Set("exchange:USD:CAD", usdToCad, 1*time.Hour)
    
--     return nil
-- }


-- USAGE IN CODE:
-- ────────────────────────────────────────────────────────

-- // Option 1: SQL function (single param now - target currency)
-- SELECT get_exchange_rate('USD');
-- -- Returns: 0.7350

-- // Option 2: Convert function
-- SELECT convert_currency(100.00, 'CAD', 'USD');
-- -- Returns: 73.50

-- // Option 3: Get rate from view
-- SELECT rate FROM v_exchange_rates_latest 
-- WHERE target_currency = 'USD';

-- // Option 4: Redis cache (fastest)
-- rate := redis.Get("exchange:CAD:USD")


-- RATE SOURCES COMPARISON:
-- ────────────────────────────────────────────────────────

-- Source                  | Cost   | Update Freq | Notes
-- ------------------------|--------|-------------|------------------
-- Bank of Canada          | FREE   | Daily 12:30 | Official CAD rate
-- Open Exchange Rates     | $12/mo | Hourly      | 200+ currencies
-- Exchange Rate API       | FREE   | Daily       | Rate limits apply
-- Fixer.io                | $10/mo | Hourly      | Good reliability
-- Stripe (via balance)    | N/A    | N/A         | Not a forex API


-- RECOMMENDED APPROACH:
-- ────────────────────────────────────────────────────────

-- 1. Use Bank of Canada as primary source (free, official)
-- 2. Cache rate in Redis with 1-hour TTL
-- 3. Worker runs at 1 PM ET daily
-- 4. Fallback to previous day's rate if fetch fails
-- 5. All code uses get_exchange_rate() or Redis cache


-- RETENTION:
-- ────────────────────────────────────────────────────────

-- Keep rates for 7 years (financial compliance).
-- Daily cleanup can archive old rates to cold storage.

-- DELETE FROM exchange_rates 
-- WHERE rate_date < CURRENT_DATE - INTERVAL '7 years';
-- */