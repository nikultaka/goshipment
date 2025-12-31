-- ============================
-- Seed roles
-- ============================

INSERT INTO roles (
    id,
    name,
    display_name,
    description,
    is_system_role,
    is_admin_role,
    hierarchy_level,
    permissions,
    is_active,
    created_at,
    updated_at,
    created_by
) VALUES
    (
        '00000000-0000-0000-0000-000000000101',
        'admin',
        'Administrator',
        'Full platform administrator with access to all modules and settings.',
        TRUE,
        TRUE,
        100,
        '["*"]'::jsonb,
        TRUE,
        NOW(),
        NOW(),
        NULL
    ),
    (
        '00000000-0000-0000-0000-000000000102',
        'manager',
        'Manager',
        'Operational manager with access to reports, shipments, and support.',
        TRUE,
        FALSE,
        80,
        '["analytics.read","reports.read","shipments.read","support.tickets.manage"]'::jsonb,
        TRUE,
        NOW(),
        NOW(),
        NULL
    ),
    (
        '00000000-0000-0000-0000-000000000103',
        'user',
        'User',
        'Standard customer account for creating and managing shipments.',
        TRUE,
        FALSE,
        10,
        '["users.read","shipments.create","shipments.read"]'::jsonb,
        TRUE,
        NOW(),
        NOW(),
        NULL
    ),
    (
        '00000000-0000-0000-0000-000000000104',
        'support',
        'Support Agent',
        'Handles support tickets and user issues.',
        TRUE,
        FALSE,
        60,
        '["support.tickets.manage","users.read","shipments.read"]'::jsonb,
        TRUE,
        NOW(),
        NOW(),
        NULL
    ),
    (
        '00000000-0000-0000-0000-000000000105',
        'system',
        'System',
        'System automation / background processes.',
        TRUE,
        TRUE,
        1000,
        '["*"]'::jsonb,
        TRUE,
        NOW(),
        NOW(),
        NULL
    );

-- ============================
-- Seed core users
-- ============================

-- Admin user
INSERT INTO users (
    id,
    external_id,
    username,
    first_name,
    last_name,
    display_name,
    timezone,
    locale,
    is_active,
    is_verified,
    verification_level,
    referral_code,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'usr_admin',
    'system_admin',
    'System',
    'Admin',
    'System Admin',
    'America/Toronto',
    'en_CA',
    TRUE,
    TRUE,
    3,
    'REF-ADMIN-0001',
    NOW(),
    NOW()
);

-- Admin email
INSERT INTO user_emails (
    id,
    user_id,
    email,
    is_primary,
    is_recovery,
    is_verified,
    verified_at
) VALUES (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000001',
    'admin@example.com',
    TRUE,
    TRUE,
    TRUE,
    NOW()
);

-- Admin phone
INSERT INTO user_phones (
    id,
    user_id,
    phone,
    phone_e164,
    is_primary,
    is_recovery,
    is_verified,
    verified_at
) VALUES (
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000001',
    '+14165551234',
    '+14165551234',
    TRUE,
    TRUE,
    TRUE,
    NOW()
);

-- Wire admin primary email/phone pointers
UPDATE users
SET primary_email_id = '00000000-0000-0000-0000-000000000401',
    primary_phone_id = '00000000-0000-0000-0000-000000000501'
WHERE id = '00000000-0000-0000-0000-000000000001';


-- Demo user
INSERT INTO users (
    id,
    external_id,
    username,
    first_name,
    last_name,
    display_name,
    timezone,
    locale,
    is_active,
    is_verified,
    verification_level,
    referral_code,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000002',
    'usr_demo',
    'demo_user',
    'Demo',
    'User',
    'Demo User',
    'America/Toronto',
    'en_CA',
    TRUE,
    FALSE,
    0,
    'REF-DEMO-0001',
    NOW(),
    NOW()
);

-- Demo email
INSERT INTO user_emails (
    id,
    user_id,
    email,
    is_primary,
    is_recovery,
    is_verified
) VALUES (
    '00000000-0000-0000-0000-000000000402',
    '00000000-0000-0000-0000-000000000002',
    'user@example.com',
    TRUE,
    TRUE,
    FALSE
);

-- Demo phone
INSERT INTO user_phones (
    id,
    user_id,
    phone,
    phone_e164,
    is_primary,
    is_recovery,
    is_verified
) VALUES (
    '00000000-0000-0000-0000-000000000502',
    '00000000-0000-0000-0000-000000000002',
    '+14165550000',
    '+14165550000',
    TRUE,
    TRUE,
    FALSE
);

-- Wire demo primary email/phone pointers
UPDATE users
SET primary_email_id = '00000000-0000-0000-0000-000000000402',
    primary_phone_id = '00000000-0000-0000-0000-000000000502'
WHERE id = '00000000-0000-0000-0000-000000000002';


-- ============================
-- Seed user role assignments
-- ============================

INSERT INTO user_roles (
    id,
    user_id,
    role_id,
    granted_at,
    is_active,
    created_at,
    updated_at
) VALUES
    (
        '00000000-0000-0000-0000-000000000301',
        '00000000-0000-0000-0000-000000000001', -- admin user
        '00000000-0000-0000-0000-000000000101', -- admin role
        NOW(),
        TRUE,
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0000-000000000302',
        '00000000-0000-0000-0000-000000000002', -- demo user
        '00000000-0000-0000-0000-000000000103', -- user role
        NOW(),
        TRUE,
        NOW(),
        NOW()
    );



-- =====================================================
-- ADD TO: 004_users_seed.sql
-- UPDATE existing user INSERTs to include new columns:
--   currency, has_2fa_enabled, ui_theme, in_app_notifications
--
-- Example for admin user (add these columns to your INSERT):
--   currency = 'CAD'
--   has_2fa_enabled = FALSE
--   ui_theme = 'light'
--   in_app_notifications = TRUE
-- =====================================================
