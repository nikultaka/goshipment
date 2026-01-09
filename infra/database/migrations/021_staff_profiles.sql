-- =====================================================
-- 021_staff_profiles.sql
-- Staff Profiles (Frontend "About Us" Display)
-- =====================================================
--
-- PURPOSE:
-- Display team/staff on website frontend (About Us page)
-- NOT for operational team management
--
-- DESIGN:
-- - Links to user_id if person is a platform user
-- - Falls back to manual entry if not a user
-- - Override fields allow custom display even if user exists
-- =====================================================

CREATE TABLE staff_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- OWNERSHIP
    -- -------------------------------------------------
    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- LINKED USER (Optional)
    -- -------------------------------------------------
    -- If set, display info can pull from users table
    -- Override fields below take precedence if set
    user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    -- -------------------------------------------------
    -- DISPLAY INFO
    -- -------------------------------------------------
    -- If user_id is set AND these are NULL, pull from users table
    -- If these are set, they OVERRIDE user data for display
    
    display_name VARCHAR(150),
    -- Override: shown instead of users.display_name
    
    title VARCHAR(100),
    -- e.g., "CEO", "Operations Manager", "Lead Developer"
    
    bio TEXT,
    -- Short biography for the website
    
    -- -------------------------------------------------
    -- AVATAR
    -- -------------------------------------------------
    -- If NULL and user_id is set, use users.avatar_file_id
    avatar_file_id UUID
        REFERENCES files(id)
        ON DELETE SET NULL,

    -- -------------------------------------------------
    -- CONTACT (Optional, for public display)
    -- -------------------------------------------------
    public_email VARCHAR(255),
    -- Separate from user's actual email (may want different)
    
    linkedin_url VARCHAR(255),
    twitter_url VARCHAR(255),

    -- -------------------------------------------------
    -- DISPLAY SETTINGS
    -- -------------------------------------------------
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL DEFAULT 0,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    -- One profile per user per organization
    CONSTRAINT uq_staff_profile_user
        UNIQUE (organization_id, user_id),

    CONSTRAINT chk_display_order_non_negative
        CHECK (display_order >= 0),

    -- Must have either user_id OR display_name
    CONSTRAINT chk_has_identity
        CHECK (user_id IS NOT NULL OR display_name IS NOT NULL)
);

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_staff_profiles_org 
    ON staff_profiles(organization_id);

CREATE INDEX idx_staff_profiles_user 
    ON staff_profiles(user_id);

CREATE INDEX idx_staff_profiles_visible 
    ON staff_profiles(organization_id, display_order) 
    WHERE is_visible = TRUE AND is_active = TRUE;

-- =====================================================
-- VIEW: Resolved Staff Display
-- =====================================================
-- Automatically resolves user data with overrides

CREATE OR REPLACE VIEW v_staff_display AS
SELECT 
    sp.id,
    sp.organization_id,
    sp.user_id,
    sp.display_order,
    
    -- Resolved display name (override > user > fallback)
    COALESCE(
        sp.display_name,
        u.display_name,
        CONCAT(u.first_name, ' ', u.last_name)
    ) AS name,
    
    sp.title,
    sp.bio,
    
    -- Resolved avatar (override > user)
    COALESCE(sp.avatar_file_id, u.avatar_file_id) AS avatar_file_id,
    
    -- Public contact
    sp.public_email,
    sp.linkedin_url,
    sp.twitter_url
    
FROM staff_profiles sp
LEFT JOIN users u ON u.id = sp.user_id
WHERE sp.is_visible = TRUE 
  AND sp.is_active = TRUE
ORDER BY sp.display_order ASC;
