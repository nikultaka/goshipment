-- =====================================================
-- MODULE 6: LOGS - COMPREHENSIVE AUDIT SYSTEM
-- =====================================================

-- Primary audit log (application events)
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(50) UNIQUE NOT NULL,       -- evt_xxx

    -- Actor (WHO)
    actor_type VARCHAR(20) NOT NULL,           -- user, admin, api_client, system
    user_id UUID REFERENCES users(id),
    api_client_id UUID REFERENCES api_clients(id),
    actor_identifier VARCHAR(255),             -- email, client_id, cron_name
    actor_ip INET,
    actor_user_agent_hash VARCHAR(64),

    -- Action (WHAT)
    action VARCHAR(100) NOT NULL,              -- login, create_user, update_order
    resource_type VARCHAR(50),                 -- users, orders, files
    resource_id VARCHAR(100),
    resource_owner_id UUID REFERENCES users(id),

    -- Purpose (WHY – required by GDPR/PIPEDA)
    business_purpose VARCHAR(100) NOT NULL,    -- service_delivery, billing, support, security
    legal_basis VARCHAR(50),                   -- consent, contract, legitimate_interest
    business_justification TEXT,

    -- Request Context
    request_id VARCHAR(50),
    session_id VARCHAR(100),
    correlation_id VARCHAR(50),

    -- Results
    status VARCHAR(20) NOT NULL,               -- success, failure, denied
    records_affected INTEGER DEFAULT 0,
    response_time_ms INTEGER,
    error_code VARCHAR(50),
    error_message TEXT,

    -- Data Changes (Only for write ops)
    before_values JSONB,
    after_values JSONB,
    field_changes TEXT[],

    -- Timing
    occurred_at TIMESTAMP NOT NULL,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- System events log (technical events)
CREATE TABLE system_event_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(50) UNIQUE NOT NULL,    -- sev_xxx

    -- Classification
    event_type VARCHAR(50) NOT NULL,         -- worker_start, cache_miss, error
    event_category VARCHAR(50),              -- system, db, cache, worker, integration
    severity VARCHAR(20) DEFAULT 'info',     -- debug, info, warn, error, fatal

    -- Source
    component VARCHAR(100),                  -- api, worker, scheduler
    instance_id VARCHAR(100),                -- pod_id, server_id

    -- Details
    message TEXT NOT NULL,
    event_data JSONB DEFAULT '{}',
    stack_trace TEXT,

    -- Correlation
    request_id VARCHAR(50),
    correlation_id VARCHAR(50),
    user_id UUID REFERENCES users(id),

    -- Timing
    occurred_at TIMESTAMP NOT NULL,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- Security events log
CREATE TABLE security_event_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(50) UNIQUE NOT NULL,     -- sec_xxx

    -- Classification
    event_type VARCHAR(50) NOT NULL,          -- login_attempt, brute_force, rate_limit
    threat_level VARCHAR(20) DEFAULT 'low',   -- low, medium, high, critical
    event_category VARCHAR(50),               -- auth, api_security, data_access

    -- Source
    source_ip INET,
    source_country VARCHAR(3),
    user_agent TEXT,
    user_id UUID REFERENCES users(id),
    session_id VARCHAR(100),

    -- Attack Details
    attack_type VARCHAR(50),                  -- brute_force, sql_injection, xss, csrf
    attack_vector VARCHAR(100),               -- endpoint name or param

    -- System Action
    action_taken VARCHAR(50),                 -- block, throttle, log_only

    -- Target
    endpoint_targeted VARCHAR(200),
    resource_targeted VARCHAR(100),

    -- Timing
    occurred_at TIMESTAMP NOT NULL,
    recorded_at TIMESTAMP DEFAULT NOW()
);