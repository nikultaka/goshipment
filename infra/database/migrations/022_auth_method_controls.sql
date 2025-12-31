-- =====================================================
-- 022_auth_method_controls.sql
-- Authentication Method Controls
-- =====================================================
-- Purpose:
-- - Control how users authenticate
-- - Support password-only, OAuth-only, or mixed auth
-- - Enable admin and security enforcement
--
-- SAFE:
-- - Non-breaking
-- - Backward compatible
-- =====================================================

-- -----------------------------------------------------
-- Preferred authentication method (UX hint only)
-- Examples:
--   password
--   google
--   apple
--   github
-- -----------------------------------------------------
ALTER TABLE users
ADD COLUMN preferred_auth_method VARCHAR(20);

-- -----------------------------------------------------
-- Disable password login (OAuth-only enforcement)
-- Useful for:
-- - Enterprise policies
-- - Security incidents
-- - Passwordless-first accounts
-- -----------------------------------------------------
ALTER TABLE users
ADD COLUMN password_login_disabled BOOLEAN DEFAULT false;
