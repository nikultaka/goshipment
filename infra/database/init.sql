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
\i '/docker-entrypoint-initdb.d/migrations/012_organizations.sql';
\i '/docker-entrypoint-initdb.d/migrations/013_support.sql';
\i '/docker-entrypoint-initdb.d/migrations/014_suppressions.sql';
\i '/docker-entrypoint-initdb.d/migrations/015_exports.sql';
\i '/docker-entrypoint-initdb.d/migrations/016_indexes.sql';
\i '/docker-entrypoint-initdb.d/migrations/017_auth_methods.sql';
\i '/docker-entrypoint-initdb.d/migrations/018_business.sql';
\i '/docker-entrypoint-initdb.d/migrations/019_address_book.sql';
\i '/docker-entrypoint-initdb.d/migrations/020_carriers.sql';
\i '/docker-entrypoint-initdb.d/migrations/021_staff_profiles.sql';
\i '/docker-entrypoint-initdb.d/migrations/022_organization_preferences.sql';
\i '/docker-entrypoint-initdb.d/migrations/023_wallet.sql';
\i '/docker-entrypoint-initdb.d/migrations/024_geo_reference.sql';
\i '/docker-entrypoint-initdb.d/migrations/025_hs_code.sql';
\i '/docker-entrypoint-initdb.d/migrations/026_shipment_estimates.sql';
\i '/docker-entrypoint-initdb.d/migrations/027_shipment_packages.sql';
\i '/docker-entrypoint-initdb.d/migrations/028_shipments.sql';
\i '/docker-entrypoint-initdb.d/migrations/029_shipment_events.sql';
\i '/docker-entrypoint-initdb.d/migrations/030_pickups.sql';
\i '/docker-entrypoint-initdb.d/migrations/031_discounts.sql';
\i '/docker-entrypoint-initdb.d/migrations/032_commissions.sql';
\i '/docker-entrypoint-initdb.d/migrations/033_referrals.sql';
\i '/docker-entrypoint-initdb.d/migrations/034_invoices.sql';
\i '/docker-entrypoint-initdb.d/migrations/035_indexes.sql';
\i '/docker-entrypoint-initdb.d/migrations/036_shipment_financials.sql';
\i '/docker-entrypoint-initdb.d/migrations/037_financial_reports.sql';
\i '/docker-entrypoint-initdb.d/migrations/038_exchange_rates.sql';

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
    ('012_organizations.sql'),
    ('013_support.sql'),
    ('014_suppressions.sql'),
    ('015_exports.sql'),
    ('016_indexes.sql'),
    ('017_auth_methods.sql'),
    ('018_business.sql'),
    ('019_address_book.sql'),
    ('020_carriers.sql'),
    ('021_staff_profiles.sql'),
    ('022_organization_preferences.sql'),
    ('023_wallet.sql'),
    ('024_geo_reference.sql'),
    ('025_hs_code.sql'),
    ('026_shipment_estimates.sql'),
    ('027_shipment_packages.sql'),
    ('028_shipments.sql'),
    ('029_shipment_events.sql'),
    ('030_pickups.sql'),
    ('031_discounts.sql'),
    ('032_commissions.sql'),
    ('033_referrals.sql'),
    ('034_invoices.sql'),
    ('035_indexes.sql'),
    ('036_shipment_financials.sql'),
    ('037_financial_reports.sql'),
    ('038_exchange_rates.sql')
ON CONFLICT (version) DO NOTHING;

-- ============================================
-- Run seeds (same order as migrations)
-- ============================================

\i '/docker-entrypoint-initdb.d/seeds/001_enable_uuid.sql';
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
\i '/docker-entrypoint-initdb.d/seeds/012_organizations_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/013_support_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/014_suppressions_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/015_exports_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/016_indexes_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/017_auth_methods_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/018_business_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/019_address_book_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/020_carriers_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/021_staff_profiles_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/022_organization_preferences_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/023_wallet_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/024_geo_reference_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/025_hs_code_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/026_shipment_estimates_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/027_shipment_packages_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/028_shipments_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/029_shipment_events_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/030_pickups_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/031_discounts_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/032_commissions_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/033_referrals_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/034_invoices_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/035_indexes_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/036_shipment_financials_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/037_financial_reports_seed.sql';
\i '/docker-entrypoint-initdb.d/seeds/038_exchange_rates_seed.sql';
