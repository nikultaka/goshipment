-- -- =====================================================
-- -- 035_indexes.sql
-- -- Then ADD after 033-049 tables (in a new migration or end of 049):
-- -- =====================================================
-- ALTER TABLE shipment_packages
--     ADD CONSTRAINT fk_shipment_packages_shipment
--     FOREIGN KEY (shipment_id)
--     REFERENCES shipments(id)
--     ON DELETE CASCADE;
-- -- =====================================================
-- -- DEFERRED FK: shipment_events → pickups
-- -- =====================================================
-- ALTER TABLE shipment_events
--     ADD CONSTRAINT fk_shipment_events_pickup
--     FOREIGN KEY (pickup_id)
--     REFERENCES pickups(id)
--     ON DELETE CASCADE;

-- ALTER TABLE support_tickets
--     ADD CONSTRAINT fk_support_tickets_shipment
--     FOREIGN KEY (shipment_id)
--     REFERENCES shipments(id)
--     ON DELETE SET NULL;

-- ALTER TABLE support_tickets
--     ADD CONSTRAINT fk_support_tickets_invoice
--     FOREIGN KEY (invoice_id)
--     REFERENCES invoices(id)
--     ON DELETE SET NULL;

-- ALTER TABLE support_tickets
--     ADD CONSTRAINT fk_support_tickets_pickup
--     FOREIGN KEY (pickup_id)
--     REFERENCES pickups(id)
--     ON DELETE SET NULL;

-- ALTER TABLE system_settings
--     ADD CONSTRAINT fk_system_settings_logo
--     FOREIGN KEY (company_logo_file_id)
--     REFERENCES files(id)
--     ON DELETE SET NULL;

-- -- In 051_deferred_fks.sql:
-- ALTER TABLE notifications
--     ADD CONSTRAINT fk_notifications_org
--     FOREIGN KEY (organization_id)
--     REFERENCES organizations(id)
--     ON DELETE CASCADE;

-- ALTER TABLE notification_queue
--     ADD CONSTRAINT fk_notification_queue_org
--     FOREIGN KEY (organization_id)
--     REFERENCES organizations(id)
--     ON DELETE SET NULL;

-- -- Deferred FKs (tables created in different order)

-- ALTER TABLE oauth_authorization_codes
--     ADD CONSTRAINT fk_oauth_auth_codes_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE oauth_access_tokens
--     ADD CONSTRAINT fk_oauth_access_tokens_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- ALTER TABLE oauth_refresh_tokens
--     ADD CONSTRAINT fk_oauth_refresh_tokens_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- ALTER TABLE storage_quotas
--     ADD CONSTRAINT fk_storage_quotas_org
--     FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

-- ALTER TABLE discount_redemptions
--     ADD CONSTRAINT fk_discount_redemptions_invoice
--     FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL;

-- CREATE INDEX idx_discount_redemptions_invoice_id
--     ON discount_redemptions(invoice_id);

-- CREATE UNIQUE INDEX uq_discount_redemptions_discount_invoice
--     ON discount_redemptions(discount_id, invoice_id)
--     WHERE invoice_id IS NOT NULL;

-- -- Deferred FK: commission_transactions -> invoices
-- ALTER TABLE commission_transactions
--     ADD CONSTRAINT fk_commission_tx_invoice
--     FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL;


-- Migration: 035_indexes.sql
-- ============================================================

-- =====================================================
-- 035_indexes.sql
-- Deferred Foreign Keys (tables created in different order)
-- =====================================================

-- =====================================================
-- DEFERRED FK: shipment_packages → shipments
-- =====================================================
ALTER TABLE shipment_packages
    ADD CONSTRAINT fk_shipment_packages_shipment
    FOREIGN KEY (shipment_id)
    REFERENCES shipments(id)
    ON DELETE CASCADE;

-- =====================================================
-- DEFERRED FK: shipment_events → pickups
-- =====================================================
ALTER TABLE shipment_events
    ADD CONSTRAINT fk_shipment_events_pickup
    FOREIGN KEY (pickup_id)
    REFERENCES pickups(id)
    ON DELETE CASCADE;

-- =====================================================
-- DEFERRED FK: system_settings → files
-- =====================================================
ALTER TABLE system_settings
    ADD CONSTRAINT fk_system_settings_logo
    FOREIGN KEY (company_logo_file_id)
    REFERENCES files(id)
    ON DELETE SET NULL;

-- =====================================================
-- DEFERRED FK: notifications → organizations
-- =====================================================
ALTER TABLE notifications
    ADD CONSTRAINT fk_notifications_org
    FOREIGN KEY (organization_id)
    REFERENCES organizations(id)
    ON DELETE CASCADE;

ALTER TABLE notification_queue
    ADD CONSTRAINT fk_notification_queue_org
    FOREIGN KEY (organization_id)
    REFERENCES organizations(id)
    ON DELETE SET NULL;

-- =====================================================
-- DEFERRED FK: oauth tables → users
-- (oauth tables created before users table)
-- =====================================================
-- ALTER TABLE oauth_authorization_codes
--     ADD CONSTRAINT fk_oauth_auth_codes_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE oauth_access_tokens
--     ADD CONSTRAINT fk_oauth_access_tokens_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- ALTER TABLE oauth_refresh_tokens
--     ADD CONSTRAINT fk_oauth_refresh_tokens_user
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- =====================================================
-- DEFERRED FK: storage_quotas → organizations
-- =====================================================
ALTER TABLE storage_quotas
    ADD CONSTRAINT fk_storage_quotas_org
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

-- =====================================================
-- DEFERRED FK: discount_redemptions → invoices
-- (discounts created before invoices)
-- =====================================================
ALTER TABLE discount_redemptions
    ADD CONSTRAINT fk_discount_redemptions_invoice
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL;

CREATE INDEX idx_discount_redemptions_invoice_id
    ON discount_redemptions(invoice_id);

CREATE UNIQUE INDEX uq_discount_redemptions_discount_invoice
    ON discount_redemptions(discount_id, invoice_id)
    WHERE invoice_id IS NOT NULL;

-- =====================================================
-- DEFERRED FK: commission_transactions → invoices
-- (commissions created before invoices)
-- =====================================================
ALTER TABLE commission_transactions
    ADD CONSTRAINT fk_commission_tx_invoice
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL;

CREATE INDEX idx_commission_tx_invoice 
    ON commission_transactions(invoice_id);