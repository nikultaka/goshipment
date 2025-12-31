-- ============================
-- Seed OAuth scopes
-- Pattern: <module>.<action>
-- ============================

INSERT INTO oauth_scopes (name, description, is_active, created_at, updated_at) VALUES
    -- OIDC-style basics (future-proof even if OIDC module comes later)
    ('openid',            'Base scope for OpenID Connect identity.', TRUE, NOW(), NOW()),
    ('profile',           'Access to basic profile information.',   TRUE, NOW(), NOW()),
    ('email',             'Access to primary email address.',       TRUE, NOW(), NOW()),

    -- Users
    ('users.read',        'Read own user profile.',                 TRUE, NOW(), NOW()),
    ('users.read.all',    'Read any user profile (admin).',         TRUE, NOW(), NOW()),
    ('users.write',       'Update own profile.',                    TRUE, NOW(), NOW()),
    ('users.manage',      'Create/update/deactivate users.',        TRUE, NOW(), NOW()),

    -- Roles & Teams
    ('roles.manage',      'Manage roles and permissions.',          TRUE, NOW(), NOW()),
    ('teams.read',        'Read teams and team members.',           TRUE, NOW(), NOW()),
    ('teams.manage',      'Manage teams and assignments.',          TRUE, NOW(), NOW()),

    -- API clients
    ('api_clients.read',  'Read API client configuration.',         TRUE, NOW(), NOW()),
    ('api_clients.manage','Create/update/delete API clients.',      TRUE, NOW(), NOW()),

    -- Logs
    ('logs.audit.read',   'Read audit logs.',                       TRUE, NOW(), NOW()),
    ('logs.system.read',  'Read system logs.',                      TRUE, NOW(), NOW()),
    ('logs.security.read','Read security logs.',                    TRUE, NOW(), NOW()),

    -- Notifications
    ('notifications.send',              'Send notifications via worker.', TRUE, NOW(), NOW()),
    ('notifications.templates.manage',  'Manage notification templates.', TRUE, NOW(), NOW()),

    -- Settings
    ('settings.system.read',  'View system settings.',             TRUE, NOW(), NOW()),
    ('settings.system.write', 'Modify system settings.',           TRUE, NOW(), NOW()),
    ('settings.user.read',    'Read user settings.',               TRUE, NOW(), NOW()),
    ('settings.user.write',   'Modify user settings.',             TRUE, NOW(), NOW()),

    -- Storage
    ('storage.files.read',    'Read/download accessible files.',   TRUE, NOW(), NOW()),
    ('storage.files.write',   'Upload files.',                     TRUE, NOW(), NOW()),
    ('storage.files.delete',  'Delete files (where permitted).',   TRUE, NOW(), NOW()),
    ('storage.shares.manage', 'Manage public/protected shares.',   TRUE, NOW(), NOW()),

    -- Support / Tickets
    ('support.tickets.read',  'Read own tickets.',                 TRUE, NOW(), NOW()),
    ('support.tickets.write', 'Create and reply to tickets.',      TRUE, NOW(), NOW()),
    ('support.tickets.manage','Manage all support tickets.',       TRUE, NOW(), NOW()),

    -- Analytics / Reports / Dashboard
    ('analytics.read',        'Read analytics data.',              TRUE, NOW(), NOW()),
    ('reports.read',          'Read reports and exports.',         TRUE, NOW(), NOW()),

    -- Shipping / Wallet (Phase 2+)
    ('shipments.create',      'Create shipments.',                 TRUE, NOW(), NOW()),
    ('shipments.read',        'Read shipments.',                   TRUE, NOW(), NOW()),
    ('shipments.cancel',      'Cancel shipments.',                 TRUE, NOW(), NOW()),
    ('rates.quote',           'Get carrier rate quotes.',          TRUE, NOW(), NOW()),
    ('wallet.read',           'Read wallet balance and history.',  TRUE, NOW(), NOW()),
    ('wallet.transfer',       'Transfer wallet funds.',            TRUE, NOW(), NOW()),
    ('invoices.read',         'Read invoices.',                    TRUE, NOW(), NOW()),

    -- Integrations
    ('sendgrid.manage',       'Manage SendGrid templates/settings.', TRUE, NOW(), NOW()),
    ('twilio.manage',         'Manage Twilio services/settings.',    TRUE, NOW(), NOW()),
    ('pdf.generate',          'Generate PDFs via worker.',           TRUE, NOW(), NOW()),
    ('pdf.templates.manage',  'Manage PDF templates.',               TRUE, NOW(), NOW());

-- ============================
-- Seed OAuth clients
-- ============================

-- First-party web SPA (public client with PKCE)
INSERT INTO oauth_clients (
    id,
    client_id,
    client_secret_hash,
    name,
    description,
    redirect_uris,
    scopes,
    client_type,
    allowed_grant_types,
    require_pkce,
    pkce_challenge_method,
    access_token_lifetime,
    refresh_token_lifetime,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000201',      -- fixed UUID
    'ph_web_app',
    NULL,                                        -- public client → no secret
    'ParcelHub Web App',
    'First-party SPA dashboard for end users and admins.',
    ARRAY['https://app.example.com/oauth/callback'],
    ARRAY[
        'openid','profile','email',
        'users.read','users.write',
        'shipments.create','shipments.read',
        'address_book.read','address_book.write',
        'notifications.send'
    ],
    'public',
    ARRAY['authorization_code','refresh_token'],
    TRUE,
    'S256',
    3600,
    2592000,
    TRUE,
    NOW(),
    NOW()
);

-- Internal service-to-service client (confidential)
INSERT INTO oauth_clients (
    id,
    client_id,
    client_secret_hash,
    name,
    description,
    redirect_uris,
    scopes,
    client_type,
    allowed_grant_types,
    require_pkce,
    pkce_challenge_method,
    access_token_lifetime,
    refresh_token_lifetime,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000202',
    'ph_internal_service',
    'changeme-in-production',  -- TODO: replace with real hash
    'Internal Service Client',
    'Internal workers and admin tools (Stripe webhooks, Shippo webhooks, etc.).',
    ARRAY['https://api.example.com/oauth/internal/callback'],
    ARRAY[
        'users.read.all',
        'api_clients.manage',
        'logs.audit.read','logs.system.read','logs.security.read',
        'settings.system.read','settings.system.write',
        'notifications.send',
        'storage.files.read','storage.files.write',
        'support.tickets.manage',
        'analytics.read','reports.read',
        'sendgrid.manage','twilio.manage','pdf.generate','pdf.templates.manage'
    ],
    'confidential',
    ARRAY['client_credentials','refresh_token'],
    TRUE,
    'S256',
    3600,
    2592000,
    TRUE,
    NOW(),
    NOW()
);

-- (Optional) You can later fill oauth_client_scopes once FK types are fixed.
