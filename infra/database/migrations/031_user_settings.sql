-- =====================================================
-- 031_user_settings.sql - USER SETTINGS TABLE
-- =====================================================

CREATE TABLE user_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Setting Details
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
    UNIQUE(user_id, key)
);

-- Index for faster lookups
CREATE INDEX idx_user_settings_user_id ON user_settings(user_id);
CREATE INDEX idx_user_settings_key ON user_settings(key);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_user_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_user_settings_updated_at();

-- Sample user settings (optional)
COMMENT ON TABLE user_settings IS 'User-specific settings and preferences';
COMMENT ON COLUMN user_settings.key IS 'Setting key (e.g., theme, language, notifications_enabled)';
COMMENT ON COLUMN user_settings.value IS 'Setting value stored as JSON or string';
COMMENT ON COLUMN user_settings.value_type IS 'Type of the value: string, number, boolean, or json';
