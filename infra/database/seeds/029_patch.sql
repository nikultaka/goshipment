
-- =====================================================
-- ADD TO: 029_patch.sql (at the end)
-- NEW TABLE: auth_methods
-- =====================================================

INSERT INTO auth_methods (
    id, user_id, method_type,
    provider_name, provider_user_id, provider_email,
    is_enabled, is_verified, is_primary,
    last_used_at, use_count, created_at, updated_at
) VALUES
    ('00000000-0000-0000-0000-000000000601',
     '00000000-0000-0000-0000-000000000001',  -- admin
     'password', NULL, NULL, NULL,
     TRUE, TRUE, TRUE, NOW(), 1, NOW(), NOW()),

    ('00000000-0000-0000-0000-000000000602',
     '00000000-0000-0000-0000-000000000002',  -- demo
     'password', NULL, NULL, NULL,
     TRUE, TRUE, TRUE, NULL, 0, NOW(), NOW());
