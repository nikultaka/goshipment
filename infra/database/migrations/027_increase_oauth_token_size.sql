-- 026_increase_oauth_token_size.sql
-- Increase OAuth token column sizes to support encrypted tokens

-- Google/Facebook access tokens can be 200+ characters
-- When encrypted with AES-256-GCM and base64 encoded, they exceed 255 characters
-- Encrypted format: base64(IV + ciphertext + auth_tag) can be 300-500+ characters

ALTER TABLE user_oauth_accounts
  ALTER COLUMN access_token_encrypted TYPE TEXT,
  ALTER COLUMN refresh_token_encrypted TYPE TEXT;

-- Optional: Add comment to document why TEXT is needed
COMMENT ON COLUMN user_oauth_accounts.access_token_encrypted IS 'Encrypted OAuth access token (AES-256-GCM, base64 encoded). Uses TEXT to support tokens up to 500+ characters after encryption.';
COMMENT ON COLUMN user_oauth_accounts.refresh_token_encrypted IS 'Encrypted OAuth refresh token (AES-256-GCM, base64 encoded). Uses TEXT to support tokens up to 500+ characters after encryption.';
