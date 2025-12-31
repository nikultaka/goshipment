CREATE UNIQUE INDEX IF NOT EXISTS user_oauth_accounts_one_primary_per_user
ON user_oauth_accounts(user_id)
WHERE is_primary = true;
