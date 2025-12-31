-- Storage quotas (system + admin user)

-- System-wide quota reference (for reporting/limits)
INSERT INTO storage_quotas (
    id,
    user_id,
    business_id,
    quota_type,
    storage_limit_bytes,
    file_count_limit,
    storage_used_bytes,
    file_count,
    last_calculated_at,
    custom_quota,
    quota_reason,
    created_at,
    updated_at
) VALUES (
    uuid_generate_v4(),
    NULL,
    NULL,
    'system',
    1099511627776,      -- 1 TB
    NULL,
    0,
    0,
    NOW(),
    TRUE,
    'System-wide default storage limits.',
    NOW(),
    NOW()
);

-- Admin user quota
INSERT INTO storage_quotas (
    id,
    user_id,
    business_id,
    quota_type,
    storage_limit_bytes,
    file_count_limit,
    storage_used_bytes,
    file_count,
    last_calculated_at,
    custom_quota,
    quota_reason,
    created_at,
    updated_at
) VALUES (
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'user',
    10737418240,        -- 10 GB
    NULL,
    0,
    0,
    NOW(),
    TRUE,
    'Initial admin quota for test uploads and invoices.',
    NOW(),
    NOW()
);
