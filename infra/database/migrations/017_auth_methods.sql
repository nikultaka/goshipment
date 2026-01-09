-- =====================================================
-- 024_auth_methods.sql
-- Primary authentication methods per user
-- =====================================================

CREATE TABLE auth_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Owner
    user_id UUID NOT NULL,

    -- Authentication type
    method_type VARCHAR(30) NOT NULL,
    -- password | google | apple | github | microsoft | passkey

    -- Provider identity (OAuth / Passkeys)
    provider_name VARCHAR(30),
    provider_user_id VARCHAR(255),
    provider_email VARCHAR(255),

    -- Status
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_primary BOOLEAN NOT NULL DEFAULT false,

    -- Usage tracking
    last_used_at TIMESTAMP,
    use_count INTEGER NOT NULL DEFAULT 0,

    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    disabled_at TIMESTAMP,
    disabled_reason VARCHAR(255),

    CONSTRAINT fk_auth_methods_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_auth_method_type
        CHECK (
            method_type IN (
                'password',
                'phone_otp',     -- phone login, no password (OTP challenge)
                'email_otp',     -- optional if you ever do email magic-link/OTP login
                'google',
                'facebook',
                'apple',
                'github',
                'microsoft',
                'passkey'
            )
        ),

    CONSTRAINT uq_auth_method_per_user
        UNIQUE (user_id, method_type, provider_user_id)
);

-- One primary login method per user
CREATE UNIQUE INDEX uq_auth_methods_primary
    ON auth_methods(user_id)
    WHERE is_primary = true AND is_enabled = true;

CREATE INDEX idx_auth_methods_user_id
    ON auth_methods(user_id);

CREATE INDEX idx_auth_methods_method_type
    ON auth_methods(method_type);

CREATE INDEX idx_auth_methods_is_enabled
    ON auth_methods(is_enabled);

--
-- phone → OTP → challenge verified → session created
--
CREATE TABLE login_challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- User may NOT exist yet (first-time login)
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Type of challenge
    challenge_type VARCHAR(20) NOT NULL,
    -- phone_otp | email_otp

    -- Phone number (E.164) or email
    destination VARCHAR(255) NOT NULL,

    -- Hashed OTP / token (never store plaintext)
    code_hash VARCHAR(64) NOT NULL,

    -- Lifecycle
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,

    -- Abuse protection
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    locked_until TIMESTAMP,

    -- Audit / fingerprinting
    ip_address INET,
    user_agent_hash VARCHAR(64)
);

CREATE TABLE user_totp (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    secret_encrypted TEXT,
    enabled_at TIMESTAMP,
    last_used_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    code_hash VARCHAR(64) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    used_at TIMESTAMP,
    revoked_at TIMESTAMP
);

CREATE INDEX idx_login_challenges_user_id
ON login_challenges(user_id);

CREATE INDEX idx_login_challenges_destination
ON login_challenges(destination);

CREATE INDEX idx_login_challenges_expires_at
ON login_challenges(expires_at);

CREATE UNIQUE INDEX uq_auth_methods_password_per_user
ON auth_methods(user_id)
WHERE method_type = 'password';

CREATE UNIQUE INDEX uq_auth_methods_phone_otp_per_user
ON auth_methods(user_id)
WHERE method_type = 'phone_otp';

CREATE UNIQUE INDEX uq_auth_methods_email_otp_per_user
ON auth_methods(user_id)
WHERE method_type = 'email_otp';

CREATE UNIQUE INDEX uq_auth_methods_provider_identity
ON auth_methods(provider_name, provider_user_id)
WHERE provider_user_id IS NOT NULL;

ALTER TABLE user_sessions
    ADD CONSTRAINT fk_user_sessions_auth_method
    FOREIGN KEY (auth_method_id)
    REFERENCES auth_methods(id)
    ON DELETE SET NULL;

CREATE INDEX idx_user_totp_last_used_at
ON user_totp(last_used_at);

CREATE INDEX idx_user_backup_codes_user_id
ON user_backup_codes(user_id);

CREATE INDEX idx_user_backup_codes_unused
ON user_backup_codes(user_id)
WHERE used_at IS NULL AND revoked_at IS NULL;
