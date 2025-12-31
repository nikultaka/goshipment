-- ============================================
-- Create migrations tracking table first
-- ============================================
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(255) PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- Run all migrations
-- ============================================

\i '/docker-entrypoint-initdb.d/migrations/001_enable_uuid.sql';
\i '/docker-entrypoint-initdb.d/migrations/002_core.sql';
\i '/docker-entrypoint-initdb.d/migrations/003_oauth.sql';
\i '/docker-entrypoint-initdb.d/migrations/004_users.sql';
\i '/docker-entrypoint-initdb.d/migrations/005_api_clients.sql';
\i '/docker-entrypoint-initdb.d/migrations/006_audit_logs.sql';
\i '/docker-entrypoint-initdb.d/migrations/007_notifications.sql';
\i '/docker-entrypoint-initdb.d/migrations/008_settings.sql';
\i '/docker-entrypoint-initdb.d/migrations/009_storage.sql';
\i '/docker-entrypoint-initdb.d/migrations/010_reviews.sql';
\i '/docker-entrypoint-initdb.d/migrations/011_testimonials.sql';
\i '/docker-entrypoint-initdb.d/migrations/012_analytics.sql';
\i '/docker-entrypoint-initdb.d/migrations/013_support.sql';
\i '/docker-entrypoint-initdb.d/migrations/014_team_members.sql';
\i '/docker-entrypoint-initdb.d/migrations/015_sendgrid_wrapper.sql';
\i '/docker-entrypoint-initdb.d/migrations/016_twilio_wrapper.sql';
\i '/docker-entrypoint-initdb.d/migrations/017_pdf_generator.sql';
\i '/docker-entrypoint-initdb.d/migrations/018_rate_limits.sql';
\i '/docker-entrypoint-initdb.d/migrations/019_indexes.sql';
\i '/docker-entrypoint-initdb.d/migrations/020_update.sql';
\i '/docker-entrypoint-initdb.d/migrations/021_third_party_oauth.sql';
\i '/docker-entrypoint-initdb.d/migrations/022_auth_method_controls.sql';
\i '/docker-entrypoint-initdb.d/migrations/023_fix_oauth_token_storage.sql';
\i '/docker-entrypoint-initdb.d/migrations/024_oauth_primary_unique.sql';
\i '/docker-entrypoint-initdb.d/migrations/025_oauth_user_fks.sql';
\i '/docker-entrypoint-initdb.d/migrations/026_business.sql';
\i '/docker-entrypoint-initdb.d/migrations/027_increase_oauth_token_size.sql';
\i '/docker-entrypoint-initdb.d/migrations/028_exports.sql';
\i '/docker-entrypoint-initdb.d/migrations/029_auth_methods.sql';
\i '/docker-entrypoint-initdb.d/migrations/030_session_devices.sql';
\i '/docker-entrypoint-initdb.d/migrations/031_user_notifications.sql';

-- ============================================
-- Record all migrations as applied
-- ============================================
INSERT INTO schema_migrations (version) VALUES
  ('001_enable_uuid.sql'),
  ('002_core.sql'),
  ('003_oauth.sql'),
  ('004_users.sql'),
  ('005_api_clients.sql'),
  ('006_audit_logs.sql'),
  ('007_notifications.sql'),
  ('008_settings.sql'),
  ('009_storage.sql'),
  ('010_reviews.sql'),
  ('011_testimonials.sql'),
  ('012_analytics.sql'),
  ('013_support.sql'),
  ('014_team_members.sql'),
  ('015_sendgrid_wrapper.sql'),
  ('016_twilio_wrapper.sql'),
  ('017_pdf_generator.sql'),
  ('018_rate_limits.sql'),
  ('019_indexes.sql'),
  ('020_update.sql'),
  ('021_third_party_oauth.sql'),
  ('022_auth_method_controls.sql'),
  ('023_fix_oauth_token_storage.sql'),
  ('024_oauth_primary_unique.sql'),
  ('025_oauth_user_fks.sql'),
  ('026_business.sql'),
  ('027_increase_oauth_token_size.sql'),
  ('028_exports.sql'),
  ('029_auth_methods.sql'),
  ('030_session_devices.sql'),
  ('031_user_notifications.sql')
ON CONFLICT (version) DO NOTHING;

-- ============================================
-- Run all seeds
-- ============================================

\i '/docker-entrypoint-initdb.d/seeds/002_core_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/003_oauth_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/004_users_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/005_api_clients_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/006_audit_logs_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/007_notifications_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/008_settings_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/009_storage_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/010_reviews_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/011_testimonials_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/012_analytics_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/013_support_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/014_team_members_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/015_sendgrid_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/016_twilio_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/017_pdf_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/018_rate_limits_seed.sql';