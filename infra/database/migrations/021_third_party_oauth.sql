-- 021_third_party_oauth.sql
-- Third-party OAuth / OIDC providers (Google, Facebook, X, Apple, GitHub)

CREATE TABLE oauth_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    provider VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,

    issuer VARCHAR(255),
    authorization_endpoint VARCHAR(500),
    token_endpoint VARCHAR(500),
    userinfo_endpoint VARCHAR(500),
    jwks_uri VARCHAR(500),

    client_id VARCHAR(255) NOT NULL,
    client_secret_encrypted TEXT, -- encrypted at rest (KMS/envelope); NOT a hash

    scopes TEXT[] NOT NULL,

    supports_pkce BOOLEAN DEFAULT true,
    supports_oidc BOOLEAN DEFAULT true,

    email_claim VARCHAR(100) DEFAULT 'email',
    email_verified_claim VARCHAR(100) DEFAULT 'email_verified',
    avatar_claim VARCHAR(100) DEFAULT 'picture',
    subject_claim VARCHAR(50) DEFAULT 'sub',

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Track last login per provider
ALTER TABLE user_oauth_accounts
ADD COLUMN last_login_at TIMESTAMP;
