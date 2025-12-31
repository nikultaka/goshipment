ALTER TABLE oauth_authorization_codes
  ADD CONSTRAINT fk_oauth_authorization_codes_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE oauth_access_tokens
  ADD CONSTRAINT fk_oauth_access_tokens_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE oauth_refresh_tokens
  ADD CONSTRAINT fk_oauth_refresh_tokens_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
