-- =====================================================
-- MODULE: RATE LIMITING CONFIGURATION
-- =====================================================

CREATE TABLE rate_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Who the limit applies to
    target_type VARCHAR(20) NOT NULL,   -- 'user' or 'client'
    target_id   VARCHAR(100) NOT NULL,  -- user_id OR client_id

    -- Token bucket configuration
    capacity     INT NOT NULL,          -- burst size
    refill_rate  INT NOT NULL,          -- tokens per minute

    -- Audit fields
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE (target_type, target_id)
);
