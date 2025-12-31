-- ============================
-- System settings
-- ============================

INSERT INTO system_settings (
    id,
    app_name,
    app_url,
    environment,
    login_max_attempts,
    login_lockout_minutes,
    email_from_name,
    email_from_address,
    sms_default_country,
    default_user_quota_mb,
    updated_by
) VALUES (
    TRUE,
    'Skeleton API',
    'https://api.example.com',
    'production',
    5,
    15,
    'Skeleton',
    'no-reply@example.com',
    'CA',
    5120,
    '00000000-0000-0000-0000-000000000001'
);
