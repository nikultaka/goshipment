-- =====================================================
-- MODULE 4: USERS (UPDATED FOR MULTI-EMAIL, MULTI-PHONE)
-- =====================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL,            -- usr_xxx

    -- Identity Pointers (NOT raw values)
    primary_email_id UUID,
    primary_phone_id UUID,

    -- Authentication
    password_hash VARCHAR(255),

    -- internal user_uid and client_id e.g., PH-xxxxx 
    client_number VARCHAR(50) UNIQUE,   -- e.g., CL-00001234
    client_number_generated_at TIMESTAMP,
    username VARCHAR(30) UNIQUE DEFAULT NULL,
    username_lower VARCHAR(30) UNIQUE GENERATED ALWAYS AS (lower(username)) STORED,
    
    -- Profile
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(200),
    -- avatar_url VARCHAR(500),
    avatar_file_id UUID,

    -- Preferences
    timezone VARCHAR(50) DEFAULT 'UTC',
    locale VARCHAR(10) DEFAULT 'en_CA',
    date_format VARCHAR(20) DEFAULT 'YYYY-MM-DD',
    time_format VARCHAR(10) DEFAULT '24h',
    currency CHAR(3) NOT NULL DEFAULT 'CAD',

    -- Account Status
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,    -- means ID verified, KYC or similar
    verification_level INTEGER DEFAULT 0,

    -- Login Security
    last_login_at TIMESTAMP,
    last_login_ip INET,
    login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    lockout_reason VARCHAR(100),

    -- Token Security
    token_version INTEGER DEFAULT 1,       -- force logout on rotation

    -- 2FA (TOTP)
    has_2fa_enabled BOOLEAN NOT NULL DEFAULT false,

    -- Privacy & Consent
    privacy_consent JSONB DEFAULT '{}',
    marketing_consent BOOLEAN DEFAULT false,
    marketing_consent_date TIMESTAMP,
    last_privacy_notice_accepted TIMESTAMP,

    -- Account Lifecycle
    account_deletion_requested_at TIMESTAMP,
    account_deletion_reason VARCHAR(255),
    account_anonymized_at TIMESTAMP,

    -- Referral System
    referral_code VARCHAR(50) UNIQUE,
    referred_by_user_id UUID REFERENCES users(id),
        
    -- UI Preferences
    ui_theme VARCHAR(20) NOT NULL DEFAULT 'light', -- dark / light
    in_app_notifications BOOLEAN NOT NULL DEFAULT TRUE,

    -- Registration Metadata
    registration_ip INET,
    registration_user_agent_hash VARCHAR(64),
    utm_source VARCHAR(100),
    utm_medium VARCHAR(100),
    utm_campaign VARCHAR(100),
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,

    CONSTRAINT users_verification_level CHECK (verification_level BETWEEN 0 AND 3)
);

-- ===================================================================
-- NEW TABLE: user_emails (MULTI-EMAIL SUPPORT)
-- ===================================================================
CREATE TABLE user_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,

    email VARCHAR(255) NOT NULL,
    email_lower VARCHAR(255) GENERATED ALWAYS AS (lower(email)) STORED,

    is_primary BOOLEAN DEFAULT false,
    is_recovery BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP,

    verification_token VARCHAR(255),
    verification_expires_at TIMESTAMP,

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(email_lower),
    UNIQUE(user_id, email_lower)
);

-- ===================================================================
-- NEW TABLE: user_phones (MULTI-PHONE SUPPORT)
-- ===================================================================
CREATE TABLE user_phones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,

    phone VARCHAR(20) NOT NULL,
    phone_e164 VARCHAR(20),

    is_primary BOOLEAN DEFAULT false,
    is_recovery BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP,

    verification_code VARCHAR(10),
    verification_expires_at TIMESTAMP,

    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(phone_e164),
    UNIQUE(user_id, phone_e164)
);


-- =====================================================
-- ROLES (NO ENUMS)
-- =====================================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL, -- admin, manager, user, etc.
    display_name VARCHAR(100),
    description TEXT,
    
    -- Role Properties
    is_system_role BOOLEAN DEFAULT false,
    is_admin_role BOOLEAN DEFAULT false,
    hierarchy_level INTEGER DEFAULT 0,
    
    -- Permissions JSON array
    permissions JSONB DEFAULT '[]',
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);


-- =====================================================
-- USER ROLE ASSIGNMENTS
-- =====================================================
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    assignment_reason VARCHAR(255),
    business_context VARCHAR(255),

    is_active BOOLEAN DEFAULT true,
    suspended_at TIMESTAMP,
    suspended_by UUID REFERENCES users(id),
    suspension_reason VARCHAR(255),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, role_id) DEFERRABLE INITIALLY DEFERRED
);


-- =====================================================
-- USER SESSIONS
-- =====================================================
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(128) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    ip_address INET,
    user_agent TEXT,
    user_agent_hash VARCHAR(64),
    device_fingerprint VARCHAR(128),
    
    country VARCHAR(3),
    region VARCHAR(100),
    city VARCHAR(100),
    
    oauth_client_id UUID REFERENCES oauth_clients(id),
    oauth_scopes TEXT[],

    auth_method_id UUID,
    login_method_type VARCHAR(30) NOT NULL,

    is_active BOOLEAN DEFAULT true,
    last_activity_at TIMESTAMP DEFAULT NOW(),

    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    terminated_at TIMESTAMP,
    termination_reason VARCHAR(100)
);


-- =====================================================
-- PASSWORD HISTORY
-- =====================================================
CREATE TABLE user_password_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    
    changed_by UUID REFERENCES users(id),
    change_reason VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);


-- =====================================================
-- OAUTH ACCOUNTS (MULTIPLE PROVIDERS)
-- =====================================================
CREATE TABLE user_oauth_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    provider VARCHAR(50) NOT NULL,          -- google, microsoft, github, apple, etc
    provider_user_id VARCHAR(255) NOT NULL,

    email VARCHAR(255),
    avatar_url VARCHAR(500),

    access_token_hash VARCHAR(255),
    refresh_token_hash VARCHAR(255),
    expires_at TIMESTAMP,

    is_primary BOOLEAN DEFAULT false,       -- NEW: choose preferred OAuth login
    linked_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(provider, provider_user_id)
);

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token_hash VARCHAR(64) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,

    ip_address INET,
    user_agent_hash VARCHAR(64)
);


-- Now add foreign keys (AFTER users is created)
ALTER TABLE user_emails
  ADD CONSTRAINT fk_user_emails_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_phones
  ADD CONSTRAINT fk_user_phones_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE users
  ADD CONSTRAINT fk_users_primary_email
  FOREIGN KEY (primary_email_id) REFERENCES user_emails(id);

ALTER TABLE users
  ADD CONSTRAINT fk_users_primary_phone
  FOREIGN KEY (primary_phone_id) REFERENCES user_phones(id);

CREATE UNIQUE INDEX uq_password_reset_tokens_unused
ON password_reset_tokens(user_id)
WHERE used_at IS NULL;
