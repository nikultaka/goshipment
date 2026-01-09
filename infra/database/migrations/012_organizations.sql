-- =====================================================
-- 012_organizations.sql
-- MODULE 12: Organizations
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

-- Organization lifecycle status
CREATE TYPE organization_status AS ENUM (
    'active',
    'suspended',
    'closed'
);

-- Organization subscription / policy tiers
-- NOTE:
-- These are POLICY TIERS, not marketing plans.
-- Limits, pricing, and features are resolved in the policy layer.
CREATE TYPE organization_plan AS ENUM (
    'tier_1',   -- Entry / Starter
    'tier_2',   -- Growth / Pro
    'tier_3',   -- Business / Scale
    'tier_4'    -- Enterprise / Custom
);

-- Organization member authority roles
-- NOTE:
-- Roles define AUTHORITY, not job titles.
-- Scope (branch, team) is handled separately.
CREATE TYPE organization_role AS ENUM (
    'owner',           -- Business owner / founder
    'administrator',   -- Head office admin / corporate admin
    'manager',         -- Branch manager / operations manager
    'operator',        -- Employee / staff / clerk
    'viewer'           -- Read-only (accountant, auditor, external)
);

-- =====================================================
-- organizations
-- =====================================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    external_id VARCHAR(50) UNIQUE NOT NULL, -- org_xxxxx (human-friendly)
    name VARCHAR(200) NOT NULL,

    owner_user_id UUID NOT NULL,

    status organization_status NOT NULL DEFAULT 'active',
    plan organization_plan NOT NULL DEFAULT 'tier_1',

    -- -------------------------------------------------
    -- SOFT DELETE
    -- -------------------------------------------------
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP,
    deleted_by_user_id UUID,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_org_owner
        FOREIGN KEY (owner_user_id)
        REFERENCES users(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_org_deleted_by
        FOREIGN KEY (deleted_by_user_id)
        REFERENCES users(id)
        ON DELETE SET NULL
);

-- =====================================================
-- organization_members
-- =====================================================
-- Invariant:
-- The organization owner_user_id MUST also exist here
-- with role = 'owner'. Enforced at application/transaction level.

CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,

    role organization_role NOT NULL DEFAULT 'operator',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_org_member_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_org_member_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    UNIQUE (organization_id, user_id)
);

-- =====================================================
-- organization_settings
-- =====================================================

CREATE TABLE organization_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    allow_same_day_pickup BOOLEAN NOT NULL DEFAULT FALSE,
    allow_next_day_pickup BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- FUTURE: Feature Flags (Admin controls)
    -- -------------------------------------------------
    -- can_use_api BOOLEAN NOT NULL DEFAULT FALSE,
    -- can_use_webhooks BOOLEAN NOT NULL DEFAULT FALSE,
    -- can_use_batch_shipments BOOLEAN NOT NULL DEFAULT FALSE,
    -- max_team_members_override INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (organization_id)

);

-- Use audit_logs with resource_type='organization'

-- =====================================================
-- INDEXES
-- =====================================================
CREATE INDEX idx_org_active ON organizations(id) WHERE is_deleted = FALSE;
CREATE INDEX idx_org_owner ON organizations(owner_user_id);
CREATE INDEX idx_org_member_user ON organization_members(user_id);
CREATE INDEX idx_org_member_org ON organization_members(organization_id);
