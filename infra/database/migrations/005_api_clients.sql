-- =====================================================
-- MODULE 5: API CLIENTS
-- =====================================================

-- API client applications (different from OAuth clients for direct API access)
CREATE TABLE api_clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Identification
    client_id VARCHAR(100) UNIQUE NOT NULL,        -- public key
    client_secret_hash VARCHAR(255) NOT NULL,      -- hashed secret

    name VARCHAR(255) NOT NULL,
    description TEXT,
    contact_email VARCHAR(255),

    -- Access Control
    allowed_endpoints TEXT[],                       -- whitelist
    blocked_endpoints TEXT[],                       -- blacklist
    allowed_ips TEXT[],                              -- optional IP whitelist

    -- Authentication Method
    auth_method VARCHAR(20) DEFAULT 'api_key',      -- api_key or hmac
    hmac_secret_hash VARCHAR(255),                  -- only for HMAC

    -- Rate Limit (per client)
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    rate_limit_per_day INTEGER DEFAULT 20000,

    -- Quotas
    monthly_quota INTEGER DEFAULT 100000,
    quota_used_month INTEGER DEFAULT 0,
    quota_resets_on DATE,                           -- YYYY-MM-01

    -- Status
    is_active BOOLEAN DEFAULT true,
    is_sandbox BOOLEAN DEFAULT false,

    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);


-- API client usage logs
CREATE TABLE api_client_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,

    -- Request
    endpoint VARCHAR(200),
    http_method VARCHAR(10),
    status_code INTEGER,
    response_time_ms INTEGER,

    request_size_bytes INTEGER,
    response_size_bytes INTEGER,

    auth_method_used VARCHAR(20),

    -- Location & IP
    ip_address INET,
    country VARCHAR(3),

    -- Timing
    requested_at TIMESTAMP DEFAULT NOW(),
    date_bucket DATE GENERATED ALWAYS AS (DATE(requested_at)) STORED,
    hour_bucket TIMESTAMP GENERATED ALWAYS AS (date_trunc('hour', requested_at)) STORED,

    -- Errors
    error_code VARCHAR(50),
    error_message TEXT
);