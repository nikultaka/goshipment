-- =====================================================
-- 033_referrals.sql
-- Referral Redemptions (Usage of User Referral Codes)
--
-- PURPOSE:
-- - Tracks successful referral usage (redemption)
-- - One row per referred user (prevents abuse)
-- - The referral code itself lives in users.referral_code
-- - This table is the ledger/event record used by:
--     - referral rewards
--     - commissions
--     - wallet credits
--     - analytics
--
-- IDENTIFIERS:
-- - reference_number = REF-XXXXXXXX (app-generated)
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE referral_redemption_status_enum AS ENUM (
    'active',       -- valid, reward eligible
    'revoked',      -- revoked by admin (fraud/abuse)
    'expired'       -- no longer eligible (policy-based)
);

-- =====================================================
-- TABLE: referral_redemptions
-- =====================================================

CREATE TABLE referral_redemptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- HUMAN IDENTIFIER
    -- -------------------------------------------------
    reference_number VARCHAR(20) NOT NULL UNIQUE,
    -- REF-XXXXXXXX (app-generated)

    -- -------------------------------------------------
    -- OWNERSHIP (OPTIONAL)
    -- -------------------------------------------------
    organization_id UUID,
    -- Optional: useful for reporting if referrals are used inside org context
    -- If you want it mandatory later, make it NOT NULL.

    -- -------------------------------------------------
    -- ACTORS
    -- -------------------------------------------------
    referrer_user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    referred_user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    status referral_redemption_status_enum
        NOT NULL DEFAULT 'active',

    -- -------------------------------------------------
    -- TIMING
    -- -------------------------------------------------
    redeemed_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- COMMISSION SNAPSHOT (IMMUTABLE)
    -- -------------------------------------------------
    commission_plan_id UUID
        REFERENCES commission_plans(id) ON DELETE RESTRICT,

    commission_number VARCHAR(20) NOT NULL,
    commission_base commission_base_enum NOT NULL,
    commission_value_type commission_value_type_enum NOT NULL,
    commission_value NUMERIC(14,2) NOT NULL,
    snapshot_taken_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_referral_not_self
        CHECK (referrer_user_id <> referred_user_id),

    -- One referral redemption per referred user (global)
    CONSTRAINT uq_referral_one_per_referred_user
        UNIQUE (referred_user_id),

    CONSTRAINT fk_referrals_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id)
);
-- EXAMPLE CODE
-- UPDATE referral_redemptions
-- SET
--   commission_plan_id = :new_plan_id,
--   commission_number = :new_plan_number,
--   commission_base = :new_plan_base,
--   commission_value_type = :new_plan_value_type,
--   commission_value = :new_plan_value,
--   snapshot_taken_at = NOW(),
--   metadata = jsonb_set(
--     COALESCE(metadata, '{}'),
--     '{admin_override}',
--     jsonb_build_object(
--       'old_plan', 'COM-001',
--       'new_plan', 'COM-002',
--       'reason', 'Admin override to 50%',
--       'changed_by', :admin_user_id,
--       'changed_at', NOW()
--     )
--   )
-- WHERE id = :referral_id
--   AND status = 'active';
-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_referrals_org
    ON referral_redemptions(organization_id);

CREATE INDEX idx_referrals_referrer
    ON referral_redemptions(referrer_user_id);

CREATE INDEX idx_referrals_referred
    ON referral_redemptions(referred_user_id);

CREATE INDEX idx_referrals_status
    ON referral_redemptions(status);

CREATE INDEX idx_referrals_redeemed_at
    ON referral_redemptions(redeemed_at);

-- =====================================================
-- NOTES / LIFECYCLE
-- =====================================================
-- 1) On user registration with referral code:
--    - users.referred_by_user_id is set
--    - referral_redemptions row is created with reference_number (REF-XXXXXXXX)
--
-- 2) Rewards/commissions should only consider:
--    status = 'active'
--
-- 3) Revoking a referral:
--    - status = 'revoked'
--    - do NOT delete rows (ledger integrity)
-- =====================================================
