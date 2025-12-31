
-- =====================================================
-- COMPREHENSIVE DATABASE INDEXES
-- =====================================================

-- Users
CREATE INDEX idx_users_avatar_file_id ON users(avatar_file_id);

-- API Versions
CREATE INDEX idx_api_versions_version ON api_versions(version);
CREATE INDEX idx_api_versions_status ON api_versions(status);
CREATE INDEX idx_api_versions_is_default ON api_versions(is_default);

-- OAuth Scopes
CREATE INDEX idx_oauth_scopes_name ON oauth_scopes(name);
CREATE INDEX idx_oauth_scopes_is_active ON oauth_scopes(is_active);

-- OAuth Clients
CREATE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id);
CREATE INDEX idx_oauth_clients_is_active ON oauth_clients(is_active);
CREATE INDEX idx_oauth_clients_client_type ON oauth_clients(client_type);

-- OAuth Client Scopes
CREATE INDEX idx_oauth_client_scopes_client_id ON oauth_client_scopes(client_id);
CREATE INDEX idx_oauth_client_scopes_scope_id ON oauth_client_scopes(scope_id);

-- OAuth Authorization Codes
CREATE INDEX idx_oauth_authorization_codes_code ON oauth_authorization_codes(code);
CREATE INDEX idx_oauth_authorization_codes_client_id ON oauth_authorization_codes(client_id);
CREATE INDEX idx_oauth_authorization_codes_user_id ON oauth_authorization_codes(user_id);
CREATE INDEX idx_oauth_authorization_codes_expires_at ON oauth_authorization_codes(expires_at);
CREATE INDEX idx_oauth_authorization_codes_used_at ON oauth_authorization_codes(used_at);

-- OAuth Access Tokens
CREATE INDEX idx_oauth_access_tokens_jti ON oauth_access_tokens(jti);
CREATE INDEX idx_oauth_access_tokens_client_id ON oauth_access_tokens(client_id);
CREATE INDEX idx_oauth_access_tokens_user_id ON oauth_access_tokens(user_id);
CREATE INDEX idx_oauth_access_tokens_expires_at ON oauth_access_tokens(expires_at);
CREATE INDEX idx_oauth_access_tokens_revoked_at ON oauth_access_tokens(revoked_at);

-- OAuth Refresh Tokens
CREATE INDEX idx_oauth_refresh_tokens_token_hash ON oauth_refresh_tokens(token_hash);
CREATE INDEX idx_oauth_refresh_tokens_client_id ON oauth_refresh_tokens(client_id);
CREATE INDEX idx_oauth_refresh_tokens_user_id ON oauth_refresh_tokens(user_id);
CREATE INDEX idx_oauth_refresh_tokens_token_family_id ON oauth_refresh_tokens(token_family_id);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON oauth_refresh_tokens(expires_at);
CREATE INDEX idx_oauth_refresh_tokens_revoked_at ON oauth_refresh_tokens(revoked_at);

-- JWT Signing Keys
CREATE INDEX idx_jwt_signing_keys_kid ON jwt_signing_keys(kid);
CREATE INDEX idx_jwt_signing_keys_is_active ON jwt_signing_keys(is_active);
CREATE INDEX idx_jwt_signing_keys_is_primary ON jwt_signing_keys(is_primary);

-- Users

CREATE UNIQUE INDEX user_primary_email_unique ON user_emails (user_id) WHERE is_primary = true;
CREATE UNIQUE INDEX user_recovery_email_unique ON user_emails (user_id) WHERE is_recovery = true;
CREATE INDEX idx_user_emails_email_lower ON user_emails(email_lower);
CREATE INDEX idx_user_emails_user_id ON user_emails(user_id);

CREATE UNIQUE INDEX user_primary_phone_unique ON user_phones (user_id) WHERE is_primary = true;
CREATE UNIQUE INDEX user_recovery_phone_unique ON user_phones (user_id) WHERE is_recovery = true;
CREATE INDEX idx_user_phones_e164 ON user_phones(phone_e164);
CREATE INDEX idx_user_phones_user_id ON user_phones(user_id);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_client_number ON users(client_number);
CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_users_referred_by_user_id ON users(referred_by_user_id);
CREATE INDEX idx_users_last_login_at ON users(last_login_at);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Roles
CREATE INDEX idx_roles_name ON roles(name);
CREATE INDEX idx_roles_is_active ON roles(is_active);
CREATE INDEX idx_roles_is_system_role ON roles(is_system_role);
CREATE INDEX idx_roles_is_admin_role ON roles(is_admin_role);
CREATE INDEX idx_roles_hierarchy_level ON roles(hierarchy_level);

-- User Roles
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_user_roles_granted_by ON user_roles(granted_by);
CREATE INDEX idx_user_roles_is_active ON user_roles(is_active);
CREATE INDEX idx_user_roles_expires_at ON user_roles(expires_at);

-- User Sessions
CREATE INDEX idx_user_sessions_session_id ON user_sessions(session_id);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_oauth_client_id ON user_sessions(oauth_client_id);
CREATE INDEX idx_user_sessions_is_active ON user_sessions(is_active);
CREATE INDEX idx_user_sessions_last_activity_at ON user_sessions(last_activity_at);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- User Password History
CREATE INDEX idx_user_password_history_user_id ON user_password_history(user_id);
CREATE INDEX idx_user_password_history_created_at ON user_password_history(created_at);

-- User OAuth Accounts
CREATE INDEX idx_user_oauth_accounts_user_id ON user_oauth_accounts(user_id);
CREATE INDEX idx_user_oauth_accounts_provider ON user_oauth_accounts(provider);
CREATE INDEX idx_user_oauth_accounts_provider_user_id ON user_oauth_accounts(provider_user_id);

-- API Clients
CREATE INDEX idx_api_clients_client_id ON api_clients(client_id);
CREATE INDEX idx_api_clients_is_active ON api_clients(is_active);
CREATE INDEX idx_api_clients_is_sandbox ON api_clients(is_sandbox);
CREATE INDEX idx_api_clients_created_by ON api_clients(created_by);

-- API Client Usage
CREATE INDEX idx_api_client_usage_client_id ON api_client_usage(client_id);
CREATE INDEX idx_api_client_usage_requested_at ON api_client_usage(requested_at);
CREATE INDEX idx_api_client_usage_date_bucket ON api_client_usage(date_bucket);
CREATE INDEX idx_api_client_usage_hour_bucket ON api_client_usage(hour_bucket);
CREATE INDEX idx_api_client_usage_status_code ON api_client_usage(status_code);

-- Audit Logs
CREATE INDEX idx_audit_logs_event_id ON audit_logs(event_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_api_client_id ON audit_logs(api_client_id);
CREATE INDEX idx_audit_logs_actor_type ON audit_logs(actor_type);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_resource_owner_id ON audit_logs(resource_owner_id);
CREATE INDEX idx_audit_logs_occurred_at ON audit_logs(occurred_at);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);

-- System Event Logs
CREATE INDEX idx_system_event_logs_event_id ON system_event_logs(event_id);
CREATE INDEX idx_system_event_logs_event_type ON system_event_logs(event_type);
CREATE INDEX idx_system_event_logs_event_category ON system_event_logs(event_category);
CREATE INDEX idx_system_event_logs_severity ON system_event_logs(severity);
CREATE INDEX idx_system_event_logs_occurred_at ON system_event_logs(occurred_at);
CREATE INDEX idx_system_event_logs_user_id ON system_event_logs(user_id);

-- Security Event Logs
CREATE INDEX idx_security_event_logs_event_id ON security_event_logs(event_id);
CREATE INDEX idx_security_event_logs_event_type ON security_event_logs(event_type);
CREATE INDEX idx_security_event_logs_threat_level ON security_event_logs(threat_level);
CREATE INDEX idx_security_event_logs_user_id ON security_event_logs(user_id);
CREATE INDEX idx_security_event_logs_source_ip ON security_event_logs(source_ip);
CREATE INDEX idx_security_event_logs_occurred_at ON security_event_logs(occurred_at);

-- Notification Templates
CREATE INDEX idx_notification_templates_template_id ON notification_templates(template_id);
CREATE INDEX idx_notification_templates_channel ON notification_templates(channel);
CREATE INDEX idx_notification_templates_is_active ON notification_templates(is_active);
CREATE INDEX idx_notification_templates_locale ON notification_templates(locale);

-- Notification Queue
CREATE INDEX idx_notification_queue_queue_id ON notification_queue(queue_id);
CREATE INDEX idx_notification_queue_template_id ON notification_queue(template_id);
CREATE INDEX idx_notification_queue_user_id ON notification_queue(user_id);
CREATE INDEX idx_notification_queue_status ON notification_queue(status);
CREATE INDEX idx_notification_queue_channel ON notification_queue(channel);
CREATE INDEX idx_notification_queue_scheduled_at ON notification_queue(scheduled_at);
CREATE INDEX idx_notification_queue_next_retry_at ON notification_queue(next_retry_at);

-- Notification Delivery Events
CREATE INDEX idx_notification_delivery_events_notification_id ON notification_delivery_events(notification_id);
CREATE INDEX idx_notification_delivery_events_event_type ON notification_delivery_events(event_type);
CREATE INDEX idx_notification_delivery_events_event_timestamp ON notification_delivery_events(event_timestamp);

-- User Notification Preferences
CREATE INDEX idx_user_notification_preferences_user_id ON user_notification_preferences(user_id);

-- Settings Audit Log
CREATE INDEX idx_settings_audit_log_user_id ON settings_audit_log(user_id);
CREATE INDEX idx_settings_audit_log_changed_by ON settings_audit_log(changed_by);
CREATE INDEX idx_settings_audit_log_changed_at ON settings_audit_log(changed_at);

-- Files
CREATE INDEX idx_files_external_id ON files(external_id);
CREATE INDEX idx_files_uploaded_by ON files(uploaded_by);
CREATE INDEX idx_files_owned_by ON files(owned_by);
CREATE INDEX idx_files_file_type ON files(file_type);
CREATE INDEX idx_files_file_category ON files(file_category);
CREATE INDEX idx_files_visibility ON files(visibility);
CREATE INDEX idx_files_related_resource_type ON files(related_resource_type);
CREATE INDEX idx_files_related_resource_id ON files(related_resource_id);
CREATE INDEX idx_files_parent_file_id ON files(parent_file_id);
CREATE INDEX idx_files_virus_scan_status ON files(virus_scan_status);
CREATE INDEX idx_files_is_active ON files(is_active);
CREATE INDEX idx_files_expires_at ON files(expires_at);
CREATE INDEX idx_files_created_at ON files(created_at);
CREATE INDEX idx_files_file_hash ON files(file_hash);

-- File Access Logs
CREATE INDEX idx_file_access_logs_file_id ON file_access_logs(file_id);
CREATE INDEX idx_file_access_logs_user_id ON file_access_logs(user_id);
CREATE INDEX idx_file_access_logs_api_client_id ON file_access_logs(api_client_id);
CREATE INDEX idx_file_access_logs_access_type ON file_access_logs(access_type);
CREATE INDEX idx_file_access_logs_accessed_at ON file_access_logs(accessed_at);

-- File Versions
CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
CREATE INDEX idx_file_versions_version_number ON file_versions(version_number);
CREATE INDEX idx_file_versions_is_current ON file_versions(is_current);

-- File Shares
CREATE INDEX idx_file_shares_share_id ON file_shares(share_id);
CREATE INDEX idx_file_shares_file_id ON file_shares(file_id);
CREATE INDEX idx_file_shares_share_token ON file_shares(share_token);
CREATE INDEX idx_file_shares_is_active ON file_shares(is_active);
CREATE INDEX idx_file_shares_expires_at ON file_shares(expires_at);

-- Files for users
-- Add FK for users.avatar_file_id (moved from table definition)
ALTER TABLE users ADD CONSTRAINT fk_users_avatar_file FOREIGN KEY (avatar_file_id) REFERENCES files(id) ON DELETE SET NULL;

-- Storage Quotas
CREATE INDEX idx_storage_quotas_user_id ON storage_quotas(user_id);
CREATE INDEX idx_storage_quotas_business_id ON storage_quotas(business_id);
CREATE INDEX idx_storage_quotas_quota_type ON storage_quotas(quota_type);

-- File Processing Logs
CREATE INDEX idx_file_processing_logs_file_id ON file_processing_logs(file_id);
CREATE INDEX idx_file_processing_logs_operation ON file_processing_logs(operation);
CREATE INDEX idx_file_processing_logs_result ON file_processing_logs(result);
CREATE INDEX idx_file_processing_logs_processed_at ON file_processing_logs(processed_at);

-- GDPR Requests
CREATE INDEX idx_gdpr_requests_user_id ON gdpr_requests(user_id);
CREATE INDEX idx_gdpr_requests_request_type ON gdpr_requests(request_type);
CREATE INDEX idx_gdpr_requests_status ON gdpr_requests(status);
CREATE INDEX idx_gdpr_requests_requested_at ON gdpr_requests(requested_at);

-- Support Tickets
CREATE INDEX idx_support_tickets_external_id ON support_tickets(external_id);
CREATE INDEX idx_support_tickets_ticket_number ON support_tickets(ticket_number);
CREATE INDEX idx_support_tickets_user_id ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_assigned_to ON support_tickets(assigned_to);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_category ON support_tickets(category);
CREATE INDEX idx_support_tickets_created_at ON support_tickets(created_at);

-- Support Ticket Messages
CREATE INDEX idx_support_ticket_messages_external_id ON support_ticket_messages(external_id);
CREATE INDEX idx_support_ticket_messages_ticket_id ON support_ticket_messages(ticket_id);
CREATE INDEX idx_support_ticket_messages_sender_user_id ON support_ticket_messages(sender_user_id);
CREATE INDEX idx_support_ticket_messages_sender_type ON support_ticket_messages(sender_type);
CREATE INDEX idx_support_ticket_messages_created_at ON support_ticket_messages(created_at);

-- Support Ticket Attachments
CREATE INDEX idx_support_ticket_attachments_ticket_id ON support_ticket_attachments(ticket_id);
CREATE INDEX idx_support_ticket_attachments_message_id ON support_ticket_attachments(message_id);
CREATE INDEX idx_support_ticket_attachments_file_id ON support_ticket_attachments(file_id);

-- Support Ticket Templates
CREATE INDEX idx_support_ticket_templates_template_name ON support_ticket_templates(template_name);
CREATE INDEX idx_support_ticket_templates_category ON support_ticket_templates(category);
CREATE INDEX idx_support_ticket_templates_is_active ON support_ticket_templates(is_active);

-- Team Members
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
CREATE INDEX idx_team_members_department ON team_members(department);
CREATE INDEX idx_team_members_team ON team_members(team);
CREATE INDEX idx_team_members_is_active ON team_members(is_active);

-- Teams
CREATE INDEX idx_teams_team_name ON teams(team_name);
CREATE INDEX idx_teams_team_lead_id ON teams(team_lead_id);
CREATE INDEX idx_teams_is_active ON teams(is_active);

-- Team Memberships
CREATE INDEX idx_team_memberships_team_id ON team_memberships(team_id);
CREATE INDEX idx_team_memberships_team_member_id ON team_memberships(team_member_id);
CREATE INDEX idx_team_memberships_is_active ON team_memberships(is_active);

-- Team Member Availability
CREATE INDEX idx_team_member_availability_team_member_id ON team_member_availability(team_member_id);
CREATE INDEX idx_team_member_availability_date ON team_member_availability(date);
CREATE INDEX idx_team_member_availability_status ON team_member_availability(status);

-- SendGrid Templates
CREATE INDEX idx_sendgrid_templates_template_name ON sendgrid_templates(template_name);
CREATE INDEX idx_sendgrid_templates_sendgrid_template_id ON sendgrid_templates(sendgrid_template_id);
CREATE INDEX idx_sendgrid_templates_category ON sendgrid_templates(category);
CREATE INDEX idx_sendgrid_templates_is_active ON sendgrid_templates(is_active);

-- SendGrid Sends
CREATE INDEX idx_sendgrid_sends_external_id ON sendgrid_sends(external_id);
CREATE INDEX idx_sendgrid_sends_sendgrid_message_id ON sendgrid_sends(sendgrid_message_id);
CREATE INDEX idx_sendgrid_sends_template_id ON sendgrid_sends(template_id);
CREATE INDEX idx_sendgrid_sends_recipient_email ON sendgrid_sends(recipient_email);
CREATE INDEX idx_sendgrid_sends_user_id ON sendgrid_sends(user_id);
CREATE INDEX idx_sendgrid_sends_send_status ON sendgrid_sends(send_status);
CREATE INDEX idx_sendgrid_sends_created_at ON sendgrid_sends(created_at);

-- SendGrid Webhook Events
CREATE INDEX idx_sendgrid_webhook_events_sendgrid_event_id ON sendgrid_webhook_events(sendgrid_event_id);
CREATE INDEX idx_sendgrid_webhook_events_message_id ON sendgrid_webhook_events(message_id);
CREATE INDEX idx_sendgrid_webhook_events_event_type ON sendgrid_webhook_events(event_type);
CREATE INDEX idx_sendgrid_webhook_events_timestamp ON sendgrid_webhook_events(timestamp);
CREATE INDEX idx_sendgrid_webhook_events_email ON sendgrid_webhook_events(email);

-- SendGrid Suppressions
CREATE INDEX idx_sendgrid_suppressions_email ON sendgrid_suppressions(email);
CREATE INDEX idx_sendgrid_suppressions_suppression_type ON sendgrid_suppressions(suppression_type);
CREATE INDEX idx_sendgrid_suppressions_is_active ON sendgrid_suppressions(is_active);

-- SendGrid API Usage
CREATE INDEX idx_sendgrid_api_usage_endpoint ON sendgrid_api_usage(endpoint);
CREATE INDEX idx_sendgrid_api_usage_status_code ON sendgrid_api_usage(status_code);
CREATE INDEX idx_sendgrid_api_usage_hour_bucket ON sendgrid_api_usage(hour_bucket);
CREATE INDEX idx_sendgrid_api_usage_called_at ON sendgrid_api_usage(called_at);

-- SendGrid Stats
CREATE INDEX idx_sendgrid_stats_date ON sendgrid_stats(date);

-- Twilio Messaging Services
CREATE INDEX idx_twilio_messaging_services_service_name ON twilio_messaging_services(service_name);
CREATE INDEX idx_twilio_messaging_services_twilio_service_sid ON twilio_messaging_services(twilio_service_sid);
CREATE INDEX idx_twilio_messaging_services_is_active ON twilio_messaging_services(is_active);

-- Twilio Messages
CREATE INDEX idx_twilio_messages_external_id ON twilio_messages(external_id);
CREATE INDEX idx_twilio_messages_twilio_message_sid ON twilio_messages(twilio_message_sid);
CREATE INDEX idx_twilio_messages_messaging_service_id ON twilio_messages(messaging_service_id);
CREATE INDEX idx_twilio_messages_to_number ON twilio_messages(to_number);
CREATE INDEX idx_twilio_messages_user_id ON twilio_messages(user_id);
CREATE INDEX idx_twilio_messages_status ON twilio_messages(status);
CREATE INDEX idx_twilio_messages_created_at ON twilio_messages(created_at);

-- Twilio Webhook Events
CREATE INDEX idx_twilio_webhook_events_message_sid ON twilio_webhook_events(message_sid);
CREATE INDEX idx_twilio_webhook_events_message_status ON twilio_webhook_events(message_status);
CREATE INDEX idx_twilio_webhook_events_status_timestamp ON twilio_webhook_events(status_timestamp);
CREATE INDEX idx_twilio_webhook_events_to_number ON twilio_webhook_events(to_number);

-- Phone Number Validation
CREATE INDEX idx_phone_number_validation_original_number ON phone_number_validation(original_number);
CREATE INDEX idx_phone_number_validation_e164_format ON phone_number_validation(e164_format);
CREATE INDEX idx_phone_number_validation_country_code ON phone_number_validation(country_code);

-- SMS Opt Outs
CREATE INDEX idx_sms_opt_outs_phone_number ON sms_opt_outs(phone_number);
CREATE INDEX idx_sms_opt_outs_user_id ON sms_opt_outs(user_id);
CREATE INDEX idx_sms_opt_outs_is_active ON sms_opt_outs(is_active);

-- Twilio API Usage
CREATE INDEX idx_twilio_api_usage_api_endpoint ON twilio_api_usage(api_endpoint);
CREATE INDEX idx_twilio_api_usage_status_code ON twilio_api_usage(status_code);
CREATE INDEX idx_twilio_api_usage_hour_bucket ON twilio_api_usage(hour_bucket);
CREATE INDEX idx_twilio_api_usage_called_at ON twilio_api_usage(called_at);

-- Twilio Stats
CREATE INDEX idx_twilio_stats_date ON twilio_stats(date);

-- PDF Templates
CREATE INDEX idx_pdf_templates_external_id ON pdf_templates(external_id);
CREATE INDEX idx_pdf_templates_template_name ON pdf_templates(template_name);
CREATE INDEX idx_pdf_templates_template_type ON pdf_templates(template_type);
CREATE INDEX idx_pdf_templates_is_active ON pdf_templates(is_active);

-- PDF Generation Queue
CREATE INDEX idx_pdf_generation_queue_external_id ON pdf_generation_queue(external_id);
CREATE INDEX idx_pdf_generation_queue_template_id ON pdf_generation_queue(template_id);
CREATE INDEX idx_pdf_generation_queue_user_id ON pdf_generation_queue(user_id);
CREATE INDEX idx_pdf_generation_queue_status ON pdf_generation_queue(status);
CREATE INDEX idx_pdf_generation_queue_priority ON pdf_generation_queue(priority);
CREATE INDEX idx_pdf_generation_queue_created_at ON pdf_generation_queue(created_at);

-- PDF Generation Stats
CREATE INDEX idx_pdf_generation_stats_date ON pdf_generation_stats(date);
CREATE INDEX idx_pdf_generation_stats_template_id ON pdf_generation_stats(template_id);

-- PDF Watermarks
CREATE INDEX idx_pdf_watermarks_watermark_name ON pdf_watermarks(watermark_name);
CREATE INDEX idx_pdf_watermarks_watermark_type ON pdf_watermarks(watermark_type);
CREATE INDEX idx_pdf_watermarks_is_active ON pdf_watermarks(is_active);

-- PDF Digital Signatures
CREATE INDEX idx_pdf_digital_signatures_signature_name ON pdf_digital_signatures(signature_name);
CREATE INDEX idx_pdf_digital_signatures_is_active ON pdf_digital_signatures(is_active);

-- PDF Batch Operations
CREATE INDEX idx_pdf_batch_operations_external_id ON pdf_batch_operations(external_id);
CREATE INDEX idx_pdf_batch_operations_template_id ON pdf_batch_operations(template_id);
CREATE INDEX idx_pdf_batch_operations_status ON pdf_batch_operations(status);
CREATE INDEX idx_pdf_batch_operations_created_by ON pdf_batch_operations(created_by);
CREATE INDEX idx_pdf_batch_operations_created_at ON pdf_batch_operations(created_at);

-- REVIEWS
CREATE INDEX idx_reviews_target ON reviews (target_type, target_id);
CREATE INDEX idx_reviews_status ON reviews (status);
CREATE INDEX idx_testimonials_active ON testimonials (is_active);
CREATE INDEX idx_testimonials_featured ON testimonials (is_featured, display_order);

