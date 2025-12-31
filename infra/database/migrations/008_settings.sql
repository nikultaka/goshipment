-- =====================================================
-- 008_settings.sql SETTINGS (FIELD-BASED)
-- =====================================================

CREATE TABLE system_settings (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE),

    -- ============================
    -- Core Application
    -- ============================
    app_name VARCHAR(255) NOT NULL DEFAULT 'Skeleton API',
    app_url VARCHAR(500) NOT NULL DEFAULT 'https://api.example.com',

    -- ============================
    -- Environment
    -- ============================
    environment VARCHAR(20) NOT NULL DEFAULT 'production',
    maintenance_mode BOOLEAN NOT NULL DEFAULT FALSE,
    requires_restart BOOLEAN NOT NULL DEFAULT FALSE,

    -- ============================
    -- Security / Auth
    -- ============================
    login_max_attempts INT NOT NULL DEFAULT 5,
    login_lockout_minutes INT NOT NULL DEFAULT 15,
    password_reset_ttl_minutes INT NOT NULL DEFAULT 30,

    -- ============================
    -- Notifications
    -- ============================
    email_from_name VARCHAR(255) NOT NULL DEFAULT 'Skeleton',
    email_from_address VARCHAR(255) NOT NULL DEFAULT 'no-reply@example.com',
    sms_default_country CHAR(2) NOT NULL DEFAULT 'CA',

    -- ============================
    -- Storage
    -- ============================
    default_user_quota_mb INT NOT NULL DEFAULT 5120,

    -- ============================
    -- Rate Limiting
    -- ============================
    enable_all_rate_limiting BOOLEAN NOT NULL DEFAULT TRUE,
    enable_redis_rate_limiting BOOLEAN NOT NULL DEFAULT TRUE,
    enable_api_client_rate_limiting BOOLEAN NOT NULL DEFAULT TRUE,

    -- ============================
    -- Audit
    -- ============================
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Setting change history (for auditing)
CREATE TABLE settings_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Context
    setting_scope VARCHAR(20) NOT NULL CHECK (setting_scope IN ('system', 'user')),
    user_id UUID REFERENCES users(id),

    -- Change
    field_name VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,

    -- Metadata
    change_reason VARCHAR(255),
    changed_by UUID REFERENCES users(id),
    change_method VARCHAR(50), -- admin_panel, api, migration
    ip_address INET,
    user_agent_hash VARCHAR(64),

    -- Impact
    affected_users INTEGER DEFAULT 1,
    requires_notification BOOLEAN DEFAULT FALSE,

    changed_at TIMESTAMP NOT NULL DEFAULT NOW()
);
