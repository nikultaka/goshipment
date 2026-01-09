-- =====================================================
-- 014_suppressions.sql
-- Email & SMS Suppression Lists
-- =====================================================
--
-- PURPOSE:
-- Track emails/phones that should NOT receive messages.
-- Check BEFORE sending any communication.
--
-- PROVIDERS:
-- - Email: SendGrid only
-- - SMS: Twilio only
--
-- REPLACES:
-- - 015_sendgrid_wrapper.sql (DELETE)
-- - 016_twilio_wrapper.sql (DELETE)
-- =====================================================

-- =====================================================
-- EMAIL SUPPRESSIONS (SendGrid)
-- =====================================================

CREATE TABLE email_suppressions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- EMAIL
    -- -------------------------------------------------
    email VARCHAR(255) NOT NULL,

    -- -------------------------------------------------
    -- SUPPRESSION TYPE
    -- -------------------------------------------------
    suppression_type VARCHAR(30) NOT NULL,
    -- hard_bounce    → Permanent failure (invalid email)
    -- soft_bounce    → Temporary failure (mailbox full)
    -- spam_report    → User marked as spam
    -- unsubscribe    → User clicked unsubscribe
    -- invalid        → Invalid email format
    -- manual         → Admin manually blocked

    -- -------------------------------------------------
    -- DETAILS
    -- -------------------------------------------------
    reason TEXT,
    sendgrid_event_id VARCHAR(255),

    -- -------------------------------------------------
    -- USER LINK
    -- -------------------------------------------------
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    suppressed_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    -- For soft_bounce: retry after 72 hours

    reactivated_at TIMESTAMP,
    reactivated_by UUID REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_email_suppression_type
        CHECK (suppression_type IN (
            'hard_bounce', 'soft_bounce', 'spam_report', 
            'unsubscribe', 'invalid', 'manual'
        )),

    CONSTRAINT uq_email_suppression
        UNIQUE (email, suppression_type)
);

-- =====================================================
-- PHONE SUPPRESSIONS (Twilio)
-- =====================================================

CREATE TABLE phone_suppressions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- PHONE NUMBER (E.164 format: +14165551234)
    -- -------------------------------------------------
    phone_number VARCHAR(30) NOT NULL,
    country_code CHAR(2),

    -- -------------------------------------------------
    -- SUPPRESSION TYPE
    -- -------------------------------------------------
    suppression_type VARCHAR(30) NOT NULL,
    -- invalid         → Number doesn't exist (30003, 30005)
    -- landline        → Cannot receive SMS (30006)
    -- opt_out         → User texted STOP (21610)
    -- carrier_block   → Carrier blocked (30007)
    -- unreachable     → Temporary issue
    -- manual          → Admin manually blocked

    -- -------------------------------------------------
    -- DETAILS
    -- -------------------------------------------------
    reason TEXT,
    twilio_error_code VARCHAR(10),
    twilio_message_sid VARCHAR(34),

    -- -------------------------------------------------
    -- USER LINK
    -- -------------------------------------------------
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- -------------------------------------------------
    -- TIMESTAMPS
    -- -------------------------------------------------
    suppressed_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    -- For unreachable: retry after 24 hours

    reactivated_at TIMESTAMP,
    reactivated_by UUID REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_phone_suppression_type
        CHECK (suppression_type IN (
            'invalid', 'landline', 'opt_out', 
            'carrier_block', 'unreachable', 'manual'
        )),

    CONSTRAINT uq_phone_suppression
        UNIQUE (phone_number, suppression_type)
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Email - Primary lookup before sending
CREATE INDEX idx_email_suppressions_lookup 
    ON email_suppressions(email) 
    WHERE is_active = TRUE;

CREATE INDEX idx_email_suppressions_user 
    ON email_suppressions(user_id);

CREATE INDEX idx_email_suppressions_expires 
    ON email_suppressions(expires_at) 
    WHERE expires_at IS NOT NULL AND is_active = TRUE;

-- Phone - Primary lookup before sending
CREATE INDEX idx_phone_suppressions_lookup 
    ON phone_suppressions(phone_number) 
    WHERE is_active = TRUE;

CREATE INDEX idx_phone_suppressions_user 
    ON phone_suppressions(user_id);

CREATE INDEX idx_phone_suppressions_expires 
    ON phone_suppressions(expires_at) 
    WHERE expires_at IS NOT NULL AND is_active = TRUE;

-- =====================================================
-- HELPER FUNCTIONS
-- =====================================================

CREATE OR REPLACE FUNCTION is_email_suppressed(check_email TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM email_suppressions
        WHERE email = LOWER(TRIM(check_email))
          AND is_active = TRUE
          AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION is_phone_suppressed(check_phone TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM phone_suppressions
        WHERE phone_number = check_phone
          AND is_active = TRUE
          AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- =====================================================
-- NOTES
-- =====================================================

-- /*
-- USAGE IN GO:
-- ────────────────────────────────────────────────────────

-- // Before sending
-- if suppressed, _ := db.QueryRow("SELECT is_email_suppressed($1)", email); suppressed {
--     return ErrEmailSuppressed
-- }


-- SENDGRID WEBHOOK → email_suppressions:
-- ────────────────────────────────────────────────────────

-- POST /webhooks/sendgrid

-- Event types to handle:
-- - bounce      → hard_bounce or soft_bounce
-- - dropped     → Check reason
-- - spamreport  → spam_report  
-- - unsubscribe → unsubscribe

-- INSERT INTO email_suppressions (email, suppression_type, reason, sendgrid_event_id)
-- VALUES ($1, $2, $3, $4)
-- ON CONFLICT (email, suppression_type) 
-- DO UPDATE SET reason = EXCLUDED.reason, suppressed_at = NOW(), is_active = TRUE;


-- TWILIO WEBHOOK → phone_suppressions:
-- ────────────────────────────────────────────────────────

-- POST /webhooks/twilio/status

-- Error codes to handle:
-- - 30003 → invalid (Unreachable destination)
-- - 30005 → invalid (Unknown destination)
-- - 30006 → landline (Landline or unreachable)
-- - 30007 → carrier_block (Carrier violation)
-- - 21610 → opt_out (STOP keyword received)

-- INSERT INTO phone_suppressions (phone_number, suppression_type, reason, twilio_error_code)
-- VALUES ($1, $2, $3, $4)
-- ON CONFLICT (phone_number, suppression_type)
-- DO UPDATE SET reason = EXCLUDED.reason, suppressed_at = NOW(), is_active = TRUE;


-- NOTIFICATION CATEGORIES (notification_queue.category):
-- ────────────────────────────────────────────────────────

-- Email categories:
-- - welcome
-- - email_verification
-- - password_reset
-- - password_changed
-- - security_alert
-- - login_new_device
-- - shipment_created
-- - shipment_label_ready
-- - shipment_picked_up
-- - shipment_in_transit
-- - shipment_delivered
-- - shipment_exception
-- - invoice_created
-- - payment_received
-- - payment_failed
-- - wallet_low_balance
-- - wallet_topped_up
-- - refund_processed
-- - export_ready
-- - export_failed
-- - commission_earned
-- - commission_paid
-- - org_invite
-- - org_member_joined
-- - system_maintenance
-- - system_announcement

-- SMS categories (subset - high priority only):
-- - phone_verification (2FA code)
-- - security_alert (account compromise)
-- - login_new_device (2FA)
-- - shipment_delivered
-- - shipment_exception
-- - payment_failed


-- SENDGRID TEMPLATES (Go constants):
-- ────────────────────────────────────────────────────────

-- // internal/config/sendgrid.go
-- const (
--     // Account
--     SGTemplateWelcome           = "d-xxx"
--     SGTemplateEmailVerification = "d-xxx"
--     SGTemplatePasswordReset     = "d-xxx"
--     SGTemplatePasswordChanged   = "d-xxx"
--     SGTemplateSecurityAlert     = "d-xxx"
--     SGTemplateLoginNewDevice    = "d-xxx"
    
--     // Shipments
--     SGTemplateShipmentCreated   = "d-xxx"
--     SGTemplateLabelReady        = "d-xxx"
--     SGTemplateShipmentPickedUp  = "d-xxx"
--     SGTemplateShipmentInTransit = "d-xxx"
--     SGTemplateShipmentDelivered = "d-xxx"
--     SGTemplateShipmentException = "d-xxx"
    
--     // Billing
--     SGTemplateInvoiceCreated    = "d-xxx"
--     SGTemplatePaymentReceived   = "d-xxx"
--     SGTemplatePaymentFailed     = "d-xxx"
--     SGTemplateWalletLowBalance  = "d-xxx"
--     SGTemplateWalletToppedUp    = "d-xxx"
--     SGTemplateRefundProcessed   = "d-xxx"
    
--     // Exports
--     SGTemplateExportReady       = "d-xxx"
--     SGTemplateExportFailed      = "d-xxx"
    
--     // Commission
--     SGTemplateCommissionEarned  = "d-xxx"
--     SGTemplateCommissionPaid    = "d-xxx"
    
--     // Organization
--     SGTemplateOrgInvite         = "d-xxx"
--     SGTemplateOrgMemberJoined   = "d-xxx"
    
--     // System
--     SGTemplateSystemMaintenance = "d-xxx"
-- )


-- TWILIO CONFIG (.env):
-- ────────────────────────────────────────────────────────

-- TWILIO_ACCOUNT_SID=ACxxxxx
-- TWILIO_AUTH_TOKEN=xxxxx
-- TWILIO_MESSAGING_SERVICE_SID=MGxxxxx
-- TWILIO_FROM_NUMBER=+14165551234
-- TWILIO_VERIFY_SERVICE_SID=VAxxxxx  # For 2FA codes


-- 2FA SMS FORMAT (Twilio Verify or custom):
-- ────────────────────────────────────────────────────────

-- Option 1: Twilio Verify Service (recommended)
-- - Handles rate limiting, fraud detection
-- - Use TWILIO_VERIFY_SERVICE_SID

-- Option 2: Custom SMS via notification_queue
-- - Template: "Your ShipInAFlash code is: {{code}}. Valid for 10 minutes."
-- - category: 'phone_verification'
-- - priority: 'urgent'


-- CLEANUP JOB (Daily):
-- ────────────────────────────────────────────────────────

-- -- Expire soft bounces after 72 hours
-- UPDATE email_suppressions 
-- SET is_active = FALSE, reactivated_at = NOW()
-- WHERE suppression_type = 'soft_bounce'
--   AND expires_at < NOW()
--   AND is_active = TRUE;

-- -- Expire unreachable after 24 hours
-- UPDATE phone_suppressions 
-- SET is_active = FALSE, reactivated_at = NOW()
-- WHERE suppression_type = 'unreachable'
--   AND expires_at < NOW()
--   AND is_active = TRUE;
-- */