-- =====================================================
-- MODULE 3: OAUTH 2.1 CORE SCHEMA (NO OIDC, NO CONSENT, NO DEVICE FLOW)
-- =====================================================

-- ============================
-- 1. SCOPES
-- ============================
CREATE TABLE oauth_scopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,        -- e.g. shipments.create
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================
-- 2. CLIENTS
-- ============================
CREATE TABLE oauth_clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id VARCHAR(100) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255),          -- NULL for public clients

    name VARCHAR(255) NOT NULL,
    description TEXT,

    redirect_uris TEXT[] NOT NULL,            -- must be HTTPS
    scopes TEXT[] NOT NULL,                   -- allowed scopes

    client_type VARCHAR(20) NOT NULL DEFAULT 'confidential', -- confidential, public
    allowed_grant_types TEXT[] DEFAULT ARRAY['authorization_code', 'refresh_token'],

    require_pkce BOOLEAN DEFAULT TRUE,        -- PKCE always required for public clients
    pkce_challenge_method VARCHAR(10) DEFAULT 'S256',

    access_token_lifetime INTEGER DEFAULT 3600,      -- 1 hour
    refresh_token_lifetime INTEGER DEFAULT 2592000,  -- 30 days

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================
-- 3. CLIENT ↔ SCOPES
-- ============================
CREATE TABLE oauth_client_scopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    scope_id UUID NOT NULL REFERENCES oauth_scopes(id) ON DELETE CASCADE,

    UNIQUE (client_id, scope_id)
);

-- ============================
-- 4. AUTHORIZATION CODES (PKCE)
-- ============================
CREATE TABLE oauth_authorization_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(128) UNIQUE NOT NULL,

    client_id UUID NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,                              -- references users(id)

    redirect_uri TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    state VARCHAR(500),

    -- PKCE
    code_challenge VARCHAR(128) NOT NULL,
    code_challenge_method VARCHAR(10) NOT NULL DEFAULT 'S256',

    -- lifecycle
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================
-- 5. ACCESS TOKENS (JWT METADATA)
-- ============================
CREATE TABLE oauth_access_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    jti VARCHAR(128) UNIQUE NOT NULL,                 -- the JWT ID

    client_id UUID NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id UUID,                                   -- null for client_credentials

    scopes TEXT[] NOT NULL,

    issued_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,                             -- null if active

    last_used_at TIMESTAMP,
    use_count INTEGER DEFAULT 0,

    issuing_ip_address INET,
    issuing_user_agent_hash VARCHAR(64)
);

-- ============================
-- 6. REFRESH TOKENS (ROTATION)
-- ============================
CREATE TABLE oauth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_hash VARCHAR(255) UNIQUE NOT NULL,                                     -- hashed RT (SHA-256)

    client_id UUID NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id UUID,

    scopes TEXT[] NOT NULL,

    token_family_id VARCHAR(128) NOT NULL,                                       -- link all tokens in the same chain
    parent_token_id UUID REFERENCES oauth_refresh_tokens(id),                  -- points to previous RT
    replaced_by UUID REFERENCES oauth_refresh_tokens(id),                      -- next RT in rotation chain

    issued_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP
);

-- ============================
-- 7. JWT SIGNING KEYS (ROTATION)
-- ============================
CREATE TABLE jwt_signing_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kid VARCHAR(100) UNIQUE NOT NULL,              -- Key ID

    algorithm VARCHAR(10) NOT NULL,                -- RS256, ES256
    key_type VARCHAR(10) NOT NULL,                 -- RSA, EC

    rsa_public_key TEXT,
    rsa_private_key TEXT,                          -- encrypted PEM if RSA

    ec_public_key TEXT,
    ec_private_key TEXT,                           -- encrypted PEM if EC

    is_active BOOLEAN DEFAULT TRUE,
    is_primary BOOLEAN DEFAULT FALSE,              -- which key signs new tokens

    created_at TIMESTAMP DEFAULT NOW()
);