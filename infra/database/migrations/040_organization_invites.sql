-- =====================================================
-- Organization Invites Migration
-- =====================================================
-- Purpose: Store invitation records for users to join organizations
-- Features: Secure token-based invitations with expiry and status tracking

-- Create invite status enum
CREATE TYPE invite_status_enum AS ENUM ('pending', 'accepted', 'rejected', 'expired', 'revoked');

-- Create organization_invites table
CREATE TABLE IF NOT EXISTS organization_invites (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Foreign Keys
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invited_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Invitee Information
    email VARCHAR(255) NOT NULL,
    role organization_role_enum NOT NULL,

    -- Security & Status
    invite_token VARCHAR(64) NOT NULL UNIQUE,
    status invite_status_enum NOT NULL DEFAULT 'pending',

    -- Timestamps
    expires_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    accepted_at TIMESTAMP WITHOUT TIME ZONE,
    rejected_at TIMESTAMP WITHOUT TIME ZONE,
    revoked_at TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_invite_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT chk_invite_expires_future CHECK (expires_at > created_at),
    CONSTRAINT chk_invite_token_length CHECK (LENGTH(invite_token) = 64)
);

-- Indexes for performance
CREATE INDEX idx_org_invites_organization ON organization_invites(organization_id);
CREATE INDEX idx_org_invites_email ON organization_invites(email);
CREATE INDEX idx_org_invites_token ON organization_invites(invite_token);
CREATE INDEX idx_org_invites_status ON organization_invites(status);
CREATE INDEX idx_org_invites_invited_by ON organization_invites(invited_by_user_id);
CREATE INDEX idx_org_invites_expires_at ON organization_invites(expires_at);

-- Composite index for common queries
CREATE INDEX idx_org_invites_org_status ON organization_invites(organization_id, status);

-- Comments
COMMENT ON TABLE organization_invites IS 'Stores organization invitation records with secure token-based access';
COMMENT ON COLUMN organization_invites.invite_token IS 'Cryptographically secure random token (64 characters)';
COMMENT ON COLUMN organization_invites.status IS 'Current status: pending, accepted, rejected, expired, or revoked';
COMMENT ON COLUMN organization_invites.expires_at IS 'Invitation expiry timestamp (typically 7 days from creation)';
