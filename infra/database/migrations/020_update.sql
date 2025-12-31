-- ================================================
-- 020_update.sql
-- OIDC Support Update
-- Adds nonce + optional OIDC verification fields
-- Safe to run in production (non-breaking)
-- ================================================


-- ------------------------------------------------
-- 1. Add OIDC nonce column for Authorization Code Flow
-- This enables replay protection for SPAs & mobile apps.
-- ------------------------------------------------
ALTER TABLE oauth_authorization_codes
ADD COLUMN nonce VARCHAR(255);


-- ------------------------------------------------
-- 2. Add OIDC verification fields (trigger-based approach)
-- These help generate id_token claims (email_verified, phone_verified)
-- without manually joining user_emails / user_phones every time.
-- Using triggers since PostgreSQL doesn't allow subqueries in GENERATED columns.
-- ------------------------------------------------

-- Add the columns as regular boolean columns
ALTER TABLE users
ADD COLUMN IF NOT EXISTS oidc_email_verified BOOLEAN DEFAULT false;

ALTER TABLE users
ADD COLUMN IF NOT EXISTS oidc_phone_verified BOOLEAN DEFAULT false;

-- Create function to update OIDC verification status
CREATE OR REPLACE FUNCTION update_user_oidc_verification()
RETURNS TRIGGER AS $$
BEGIN
    -- Update email verification status
    UPDATE users
    SET oidc_email_verified = COALESCE(
        (SELECT is_verified FROM user_emails WHERE id = users.primary_email_id),
        false
    )
    WHERE id = NEW.user_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create function to update OIDC verification when user primary email/phone changes
CREATE OR REPLACE FUNCTION update_user_oidc_on_user_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Update email verification status
    NEW.oidc_email_verified := COALESCE(
        (SELECT is_verified FROM user_emails WHERE id = NEW.primary_email_id),
        false
    );

    -- Update phone verification status
    NEW.oidc_phone_verified := COALESCE(
        (SELECT is_verified FROM user_phones WHERE id = NEW.primary_phone_id),
        false
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on user_emails: when email verification changes, update users table
DROP TRIGGER IF EXISTS trg_user_email_oidc_sync ON user_emails;
CREATE TRIGGER trg_user_email_oidc_sync
AFTER INSERT OR UPDATE OF is_verified ON user_emails
FOR EACH ROW
EXECUTE FUNCTION update_user_oidc_verification();

-- Trigger on user_phones: when phone verification changes, update users table
DROP TRIGGER IF EXISTS trg_user_phone_oidc_sync ON user_phones;
CREATE TRIGGER trg_user_phone_oidc_sync
AFTER INSERT OR UPDATE OF is_verified ON user_phones
FOR EACH ROW
EXECUTE FUNCTION update_user_oidc_verification();

-- Trigger on users: when primary_email_id or primary_phone_id changes, update OIDC fields
DROP TRIGGER IF EXISTS trg_user_oidc_update ON users;
CREATE TRIGGER trg_user_oidc_update
BEFORE INSERT OR UPDATE OF primary_email_id, primary_phone_id ON users
FOR EACH ROW
EXECUTE FUNCTION update_user_oidc_on_user_change();

-- Initialize existing users' OIDC verification status
UPDATE users
SET
    oidc_email_verified = COALESCE(
        (SELECT is_verified FROM user_emails WHERE id = users.primary_email_id),
        false
    ),
    oidc_phone_verified = COALESCE(
        (SELECT is_verified FROM user_phones WHERE id = users.primary_phone_id),
        false
    );


-- ================================================
-- End of 020_update.sql
-- ================================================
