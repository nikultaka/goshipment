
-- ========================================================
-- 030_session_devices.sql: USER SESSIONS - ADD device_name
-- ========================================================

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS device_name VARCHAR(100);
-- e.g., "iPhone 15 Pro", "Chrome on Windows", "Safari on MacBook"

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS device_type VARCHAR(30);
-- mobile, tablet, desktop, unknown

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS os_name VARCHAR(50);
-- iOS, Android, Windows, macOS, Linux

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS os_version VARCHAR(20);

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS browser_name VARCHAR(50);
-- Chrome, Safari, Firefox, Edge

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS browser_version VARCHAR(20);

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS is_trusted BOOLEAN DEFAULT false;
-- User marked as "trusted device"

ALTER TABLE user_sessions 
ADD COLUMN IF NOT EXISTS trusted_at TIMESTAMP;

CREATE INDEX idx_user_sessions_device_type ON user_sessions(device_type);
CREATE INDEX idx_user_sessions_is_trusted ON user_sessions(is_trusted);

