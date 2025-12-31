-- =====================================================
-- SEED: ANALYTICS MODULE
-- =====================================================

-- ============================
-- DAILY SUMMARY SEED
-- ============================
INSERT INTO analytics_daily_summary (
    id,
    date,
    total_users,
    new_users,
    active_users,
    verified_users,
    tickets_open,
    tickets_resolved,
    tickets_new,
    total_files,
    total_storage_bytes,
    new_files,
    emails_sent,
    sms_sent,
    api_requests,
    api_errors,
    average_response_time_ms,
    error_rate,
    created_at
) VALUES
(
    uuid_generate_v4(),
    CURRENT_DATE,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0.00,
    NOW()
)
ON CONFLICT (date) DO NOTHING;



-- ============================
-- HOURLY SUMMARY SEED
-- ============================
INSERT INTO analytics_hourly_summary (
    id,
    hour_bucket,
    active_users,
    api_requests,
    api_errors,
    avg_response_time_ms,
    emails_sent,
    sms_sent,
    tickets_new,
    created_at
) VALUES
(
    uuid_generate_v4(),
    date_trunc('hour', NOW()),
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    NOW()
)
ON CONFLICT (hour_bucket) DO NOTHING;



-- ============================
-- MONTHLY SUMMARY SEED
-- ============================
INSERT INTO analytics_monthly_summary (
    id,
    month,
    total_users,
    new_users,
    active_users,
    tickets_open,
    tickets_resolved,
    files_uploaded,
    storage_used_bytes,
    api_requests,
    api_errors,
    emails_sent,
    sms_sent,
    created_at
) VALUES
(
    uuid_generate_v4(),
    to_char(NOW(), 'YYYY-MM'),
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,
    0,          -- FIXED: add value for sms_sent
    NOW()
)
ON CONFLICT (month) DO NOTHING;

