-- =====================================================
-- 016_indexes.sql
-- Consolidated Indexes for Tables 001-015
-- =====================================================


-- =====================================================
-- OAUTH MODULE (003_oauth.sql)
-- =====================================================

-- OAuth Clients
CREATE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id);
CREATE INDEX idx_oauth_clients_active ON oauth_clients(id) WHERE is_active = TRUE;
CREATE INDEX idx_oauth_clients_type ON oauth_clients(client_type);

-- OAuth Client Scopes
CREATE INDEX idx_oauth_client_scopes_client ON oauth_client_scopes(client_id);
CREATE INDEX idx_oauth_client_scopes_scope ON oauth_client_scopes(scope_id);

-- OAuth Authorization Codes
CREATE INDEX idx_oauth_auth_codes_client ON oauth_authorization_codes(client_id);
CREATE INDEX idx_oauth_auth_codes_user ON oauth_authorization_codes(user_id);
CREATE INDEX idx_oauth_auth_codes_expires ON oauth_authorization_codes(expires_at);
CREATE INDEX idx_oauth_auth_codes_unused 
    ON oauth_authorization_codes(code) 
    WHERE used_at IS NULL;

-- OAuth Access Tokens
CREATE INDEX idx_oauth_access_tokens_client ON oauth_access_tokens(client_id);
CREATE INDEX idx_oauth_access_tokens_user ON oauth_access_tokens(user_id);
CREATE INDEX idx_oauth_access_tokens_jti ON oauth_access_tokens(jti);
CREATE INDEX idx_oauth_access_tokens_active 
    ON oauth_access_tokens(expires_at) 
    WHERE revoked_at IS NULL;
CREATE INDEX idx_oauth_access_tokens_last_used ON oauth_access_tokens(last_used_at);

-- OAuth Refresh Tokens
CREATE INDEX idx_oauth_refresh_tokens_client ON oauth_refresh_tokens(client_id);
CREATE INDEX idx_oauth_refresh_tokens_user ON oauth_refresh_tokens(user_id);
CREATE INDEX idx_oauth_refresh_tokens_family ON oauth_refresh_tokens(token_family_id);
CREATE INDEX idx_oauth_refresh_tokens_active 
    ON oauth_refresh_tokens(expires_at) 
    WHERE revoked_at IS NULL;
CREATE INDEX idx_oauth_refresh_tokens_parent ON oauth_refresh_tokens(parent_token_id);

-- JWT Signing Keys
CREATE INDEX idx_jwt_signing_keys_active ON jwt_signing_keys(id) WHERE is_active = TRUE;
CREATE INDEX idx_jwt_signing_keys_primary ON jwt_signing_keys(id) WHERE is_primary = TRUE;


-- =====================================================
-- USERS MODULE (004_users.sql)
-- =====================================================

-- Users
CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_client_number ON users(client_number);
CREATE INDEX idx_users_username ON users(username_lower);
CREATE INDEX idx_users_active ON users(id) WHERE is_active = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_users_verified ON users(is_verified);
CREATE INDEX idx_users_last_login_at ON users(last_login_at);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_users_referred_by ON users(referred_by_user_id);
CREATE INDEX idx_users_deletion_requested 
    ON users(account_deletion_requested_at) 
    WHERE account_deletion_requested_at IS NOT NULL;
CREATE INDEX idx_users_locked ON users(locked_until) WHERE locked_until IS NOT NULL;

-- User Emails
CREATE INDEX idx_user_emails_user ON user_emails(user_id);
CREATE INDEX idx_user_emails_email ON user_emails(email_lower);
CREATE INDEX idx_user_emails_primary ON user_emails(user_id) WHERE is_primary = TRUE;
CREATE INDEX idx_user_emails_verified ON user_emails(user_id) WHERE is_verified = TRUE;
CREATE INDEX idx_user_emails_active ON user_emails(user_id) WHERE is_active = TRUE;

-- User Phones
CREATE INDEX idx_user_phones_user ON user_phones(user_id);
CREATE INDEX idx_user_phones_number ON user_phones(phone_e164);
CREATE INDEX idx_user_phones_primary ON user_phones(user_id) WHERE is_primary = TRUE;
CREATE INDEX idx_user_phones_verified ON user_phones(user_id) WHERE is_verified = TRUE;

-- Roles
CREATE INDEX idx_roles_name ON roles(name);
CREATE INDEX idx_roles_active ON roles(id) WHERE is_active = TRUE;

-- User Roles
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_user_roles_granted_by ON user_roles(granted_by);

-- User Password History
CREATE INDEX idx_user_password_history_user ON user_password_history(user_id);
CREATE INDEX idx_user_password_history_created ON user_password_history(created_at);

-- User OAuth Accounts
CREATE INDEX idx_user_oauth_accounts_user ON user_oauth_accounts(user_id);
CREATE INDEX idx_user_oauth_accounts_provider ON user_oauth_accounts(provider);
CREATE INDEX idx_user_oauth_accounts_provider_user ON user_oauth_accounts(provider, provider_user_id);
CREATE INDEX idx_user_oauth_accounts_email ON user_oauth_accounts(email);
CREATE INDEX idx_user_oauth_accounts_last_login ON user_oauth_accounts(last_login_at);

-- Password Reset Tokens
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);
CREATE INDEX idx_password_reset_tokens_active 
    ON password_reset_tokens(user_id, expires_at) 
    WHERE used_at IS NULL;


-- =====================================================
-- API CLIENTS MODULE (005_api_clients.sql)
-- =====================================================

-- API Clients
CREATE INDEX idx_api_clients_active ON api_clients(id) WHERE is_active = TRUE;
CREATE INDEX idx_api_clients_sandbox ON api_clients(id) WHERE is_sandbox = TRUE;
CREATE INDEX idx_api_clients_created_by ON api_clients(created_by);

-- API Client Usage
CREATE INDEX idx_api_client_usage_client ON api_client_usage(client_id);
CREATE INDEX idx_api_client_usage_date ON api_client_usage(date_bucket);
CREATE INDEX idx_api_client_usage_client_date ON api_client_usage(client_id, date_bucket);
CREATE INDEX idx_api_client_usage_requested ON api_client_usage(requested_at);


-- =====================================================
-- STORAGE MODULE (009_storage.sql)
-- =====================================================

-- Files
CREATE INDEX idx_files_external ON files(external_id);
CREATE INDEX idx_files_number ON files(file_number);
CREATE INDEX idx_files_uploaded_by ON files(uploaded_by);
CREATE INDEX idx_files_owned_by ON files(owned_by);
CREATE INDEX idx_files_org ON files(organization_id);
CREATE INDEX idx_files_business ON files(business_id);
CREATE INDEX idx_files_hash ON files(file_hash);
CREATE INDEX idx_files_content_hash ON files(content_hash);
CREATE INDEX idx_files_mime ON files(mime_type);
CREATE INDEX idx_files_type ON files(file_type);
CREATE INDEX idx_files_category ON files(file_category);
CREATE INDEX idx_files_visibility ON files(visibility);
CREATE INDEX idx_files_related ON files(related_resource_type, related_resource_id);
CREATE INDEX idx_files_parent ON files(parent_file_id);
CREATE INDEX idx_files_virus_status ON files(virus_scan_status);
CREATE INDEX idx_files_created ON files(created_at);
CREATE INDEX idx_files_deleted ON files(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_files_active ON files(id) WHERE is_active = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_files_quarantined ON files(id) WHERE quarantined = TRUE;
CREATE INDEX idx_files_expiring ON files(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_gdpr ON files(gdpr_data_subject_id) WHERE gdpr_data_subject_id IS NOT NULL;

-- File Versions
CREATE INDEX idx_file_versions_file ON file_versions(file_id);
CREATE INDEX idx_file_versions_uploaded_by ON file_versions(uploaded_by);
CREATE INDEX idx_file_versions_version ON file_versions(file_id, version_number);
CREATE INDEX idx_file_versions_current ON file_versions(file_id) WHERE is_current = TRUE;

-- File Shares
CREATE INDEX idx_file_shares_share_id ON file_shares(share_id);
CREATE INDEX idx_file_shares_file ON file_shares(file_id);
CREATE INDEX idx_file_shares_token ON file_shares(share_token);
CREATE INDEX idx_file_shares_created_by ON file_shares(created_by);
CREATE INDEX idx_file_shares_expires ON file_shares(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_file_shares_active ON file_shares(id) WHERE is_active = TRUE;
CREATE INDEX idx_file_shares_last_access ON file_shares(last_accessed_at);

-- GDPR Requests
CREATE INDEX idx_gdpr_requests_user ON gdpr_requests(user_id);
CREATE INDEX idx_gdpr_requests_type ON gdpr_requests(request_type);
CREATE INDEX idx_gdpr_requests_status ON gdpr_requests(status);
CREATE INDEX idx_gdpr_requests_requested ON gdpr_requests(requested_at);
CREATE INDEX idx_gdpr_requests_pending 
    ON gdpr_requests(requested_at) 
    WHERE status = 'pending';


-- =====================================================
-- DEFERRED FOREIGN KEYS
-- =====================================================

-- Users → Files (avatar)
ALTER TABLE users 
    ADD CONSTRAINT fk_users_avatar_file 
    FOREIGN KEY (avatar_file_id) 
    REFERENCES files(id) 
    ON DELETE SET NULL;


-- =====================================================
-- DEFERRED TO 035_indexes.sql (after organizations created):
-- =====================================================
-- Storage Quotas indexes (table references organizations)
-- CREATE INDEX idx_storage_quotas_org ON storage_quotas(organization_id);
-- CREATE INDEX idx_storage_quotas_custom ON storage_quotas(id) WHERE is_custom_quota = TRUE;
-- =====================================================


-- =====================================================
-- NOTES
-- =====================================================
-- 
-- system_settings is a SINGLETON table (id BOOLEAN = TRUE)
-- No indexes needed for single-row table.
--
-- Tables with indexes defined INLINE (not here):
-- - oauth_providers (003)
-- - user_sessions (004)
-- - audit_logs (006)
-- - notifications, notification_queue, notification_preferences (007)
-- - reviews, review_* (010)
-- - testimonials (011)
-- - support_tickets, support_ticket_* (012)
-- - organizations, organization_* (013)
-- - email_suppressions, phone_suppressions (014)
-- - exports (015)
--
-- =====================================================