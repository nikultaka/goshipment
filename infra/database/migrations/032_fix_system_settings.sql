-- =====================================================
-- 032_fix_system_settings.sql - FIX SYSTEM SETTINGS TABLE
-- =====================================================
-- This migration recreates the system_settings table to use a key-value
-- structure instead of a single-row table with specific columns.

-- Drop existing tables (if they depend on system_settings, drop them first)
DROP TABLE IF EXISTS settings_audit_log CASCADE;
DROP TABLE IF EXISTS system_settings CASCADE;

-- Create new system_settings table with key-value structure
CREATE TABLE system_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Setting Details
    category VARCHAR(50) NOT NULL,  -- e.g., 'core', 'security', 'notifications', 'storage', 'rate_limiting'
    key VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,  -- JSON or string value
    value_type VARCHAR(20) NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'number', 'boolean', 'json')),

    -- Metadata
    description TEXT,
    is_encrypted BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    UNIQUE(key)
);

-- Indexes for faster lookups
CREATE INDEX idx_system_settings_category ON system_settings(category);
CREATE INDEX idx_system_settings_key ON system_settings(key);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_system_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_system_settings_updated_at();

-- Insert default system settings
INSERT INTO system_settings (category, key, value, value_type, description) VALUES
    -- Core Application
    ('core', 'app_name', 'Skeleton API', 'string', 'Application name'),
    ('core', 'app_url', 'https://api.example.com', 'string', 'Application URL'),
    ('core', 'environment', 'production', 'string', 'Environment (development, staging, production)'),
    ('core', 'maintenance_mode', 'false', 'boolean', 'Enable maintenance mode'),
    ('core', 'requires_restart', 'false', 'boolean', 'Indicates if application requires restart'),

    -- Security & Authentication
    ('security', 'login_max_attempts', '5', 'number', 'Maximum login attempts before lockout'),
    ('security', 'login_lockout_minutes', '15', 'number', 'Account lockout duration in minutes'),
    ('security', 'password_reset_ttl_minutes', '30', 'number', 'Password reset token TTL in minutes'),

    -- Notifications
    ('notifications', 'email_from_name', 'Skeleton', 'string', 'Default email sender name'),
    ('notifications', 'email_from_address', 'no-reply@example.com', 'string', 'Default email sender address'),
    ('notifications', 'sms_default_country', 'CA', 'string', 'Default country code for SMS'),

    -- Storage
    ('storage', 'default_user_quota_mb', '5120', 'number', 'Default user storage quota in MB'),

    -- Rate Limiting
    ('rate_limiting', 'enable_all_rate_limiting', 'true', 'boolean', 'Enable all rate limiting'),
    ('rate_limiting', 'enable_redis_rate_limiting', 'true', 'boolean', 'Enable Redis-based rate limiting'),
    ('rate_limiting', 'enable_api_client_rate_limiting', 'true', 'boolean', 'Enable API client rate limiting');

-- Recreate settings_audit_log table
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

-- Index for audit log
CREATE INDEX idx_settings_audit_log_scope ON settings_audit_log(setting_scope);
CREATE INDEX idx_settings_audit_log_user_id ON settings_audit_log(user_id);
CREATE INDEX idx_settings_audit_log_changed_at ON settings_audit_log(changed_at);

-- Comments
COMMENT ON TABLE system_settings IS 'System-wide settings in key-value format';
COMMENT ON COLUMN system_settings.category IS 'Setting category for grouping (e.g., core, security, notifications)';
COMMENT ON COLUMN system_settings.key IS 'Unique setting key';
COMMENT ON COLUMN system_settings.value IS 'Setting value stored as string, number, boolean, or JSON';
COMMENT ON COLUMN system_settings.value_type IS 'Type of the value: string, number, boolean, or json';
