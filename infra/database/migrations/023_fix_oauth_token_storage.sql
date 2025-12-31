-- Provider tokens must be usable for refresh; store encrypted (or store nothing)
ALTER TABLE user_oauth_accounts
  RENAME COLUMN access_token_hash TO access_token_encrypted;

ALTER TABLE user_oauth_accounts
  RENAME COLUMN refresh_token_hash TO refresh_token_encrypted;

-- Optional: useful metadata
ALTER TABLE user_oauth_accounts
  ADD COLUMN token_type VARCHAR(30),
  ADD COLUMN scopes TEXT[],
  ADD COLUMN refresh_expires_at TIMESTAMP;
