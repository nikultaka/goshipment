-- =====================================================
-- 012_organizations_seed.sql
-- Seeds for Organizations, Members, and Settings
-- =====================================================

-- -----------------------------------------------------
-- 1. Create a default organization for the first user
-- -----------------------------------------------------

INSERT INTO organizations (
    id,
    external_id,
    name,
    owner_user_id,
    status,
    plan
)
SELECT
    uuid_generate_v4(),
    'org_default',
    'Default Organization',
    u.id,
    'active',
    'tier_1'
FROM users u
ORDER BY u.created_at
LIMIT 1;

-- -----------------------------------------------------
-- 2. Add owner as organization member (role = owner)
-- -----------------------------------------------------

INSERT INTO organization_members (
    organization_id,
    user_id,
    role,
    is_active
)
SELECT
    o.id,
    o.owner_user_id,
    'owner',
    TRUE
FROM organizations o
WHERE o.external_id = 'org_default';

-- -----------------------------------------------------
-- 3. Create organization settings (default behavior)
-- -----------------------------------------------------

INSERT INTO organization_settings (
    organization_id,
    allow_same_day_pickup,
    allow_next_day_pickup
)
SELECT
    o.id,
    FALSE,  -- same-day pickup disabled by default
    TRUE    -- next-day pickup enabled
FROM organizations o
WHERE o.external_id = 'org_default';
