-- =====================================================
-- MODULE 12: Analytics
-- =====================================================

CREATE TABLE analytics_daily_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE UNIQUE NOT NULL,

    -- User Metrics
    total_users INTEGER DEFAULT 0,
    new_users INTEGER DEFAULT 0,
    active_users INTEGER DEFAULT 0,        -- last 24h
    verified_users INTEGER DEFAULT 0,

    -- Support Tickets
    tickets_open INTEGER DEFAULT 0,
    tickets_resolved INTEGER DEFAULT 0,
    tickets_new INTEGER DEFAULT 0,

    -- Files / Storage
    total_files INTEGER DEFAULT 0,
    total_storage_bytes BIGINT DEFAULT 0,
    new_files INTEGER DEFAULT 0,

    -- Notifications
    emails_sent INTEGER DEFAULT 0,
    sms_sent INTEGER DEFAULT 0,

    -- API Usage
    api_requests INTEGER DEFAULT 0,
    api_errors INTEGER DEFAULT 0,

    -- System Health
    average_response_time_ms INTEGER DEFAULT 0,
    error_rate DECIMAL(5,2), -- % errors

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE analytics_hourly_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hour_bucket TIMESTAMP UNIQUE NOT NULL, -- 2025-11-13 14:00:00

    -- User Metrics
    active_users INTEGER DEFAULT 0,   

    -- API
    api_requests INTEGER DEFAULT 0,
    api_errors INTEGER DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,

    -- Notifications
    emails_sent INTEGER DEFAULT 0,
    sms_sent INTEGER DEFAULT 0,

    -- Tickets
    tickets_new INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE analytics_monthly_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    month VARCHAR(7) UNIQUE NOT NULL, -- "2025-11"

    total_users INTEGER DEFAULT 0,
    new_users INTEGER DEFAULT 0,
    active_users INTEGER DEFAULT 0,

    tickets_open INTEGER DEFAULT 0,
    tickets_resolved INTEGER DEFAULT 0,

    files_uploaded INTEGER DEFAULT 0,
    storage_used_bytes BIGINT DEFAULT 0,

    api_requests INTEGER DEFAULT 0,
    api_errors INTEGER DEFAULT 0,

    emails_sent INTEGER DEFAULT 0,
    sms_sent INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW()
);
