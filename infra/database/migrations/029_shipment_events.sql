-- =====================================================
-- 029_shipment_events.sql
-- Shipment Events (Timeline + Audit)
--
-- PURPOSE:
-- - One unified event stream for shipment lifecycle
-- - Supports shipment-level, package-level, and pickup-level events
-- - Designed for:
--     - Timeline UI (user + admin)
--     - Notifications (email/SMS/web)
--     - Webhook processing audit
--     - Dispute resolution / debugging
--
-- DESIGN PRINCIPLES:
-- 1) A shipment event ALWAYS belongs to a shipment (shipment_id is required)
-- 2) It may optionally target ONE sub-entity:
--       - package_id OR pickup_id
--    (but not both at the same time)
-- 3) Provider payloads can be stored for audit/debug without affecting business logic
-- =====================================================

-- =====================================================
-- ENUM: shipment_event_source_enum
-- Where the event came from
-- =====================================================

CREATE TYPE shipment_event_source_enum AS ENUM (
    'system',       -- internal automation / scheduled jobs
    'user',         -- created by user action (UI/API)
    'admin',        -- created by admin action
    'provider'      -- carrier/provider webhook or API response
);

-- =====================================================
-- ENUM: shipment_event_severity_enum
-- How important the event is (UI + alerting)
-- =====================================================

CREATE TYPE shipment_event_severity_enum AS ENUM (
    'info',
    'warning',
    'error'
);

-- =====================================================
-- ENUM: shipment_event_entity_enum
-- Which entity this event targets
-- =====================================================

CREATE TYPE shipment_event_entity_enum AS ENUM (
    'shipment',
    'package',
    'pickup'
);

-- =====================================================
-- TABLE: shipment_events
-- =====================================================

CREATE TABLE shipment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- REQUIRED PARENT
    -- -------------------------------------------------
    shipment_id UUID NOT NULL,

    -- -------------------------------------------------
    -- OWNERSHIP
    -- -------------------------------------------------
    organization_id UUID NOT NULL,

    -- -------------------------------------------------
    -- OPTIONAL SUB-ENTITY TARGET
    -- -------------------------------------------------
    -- Exactly one of these can be set per event:
    -- - package_id for per-package tracking/label events
    -- - pickup_id  for pickup scheduling events
    package_id UUID,
    pickup_id UUID,

    entity_type shipment_event_entity_enum NOT NULL DEFAULT 'shipment',
    -- shipment | package | pickup

    -- -------------------------------------------------
    -- EVENT IDENTITY (STABLE KEYS)
    -- -------------------------------------------------
    event_key VARCHAR(100) NOT NULL,
    -- Examples:
    -- shipment.created
    -- shipment.rates_requested
    -- shipment.rate_selected
    -- shipment.payment_succeeded
    -- shipment.payment_failed
    -- shipment.cancelled
    --
    -- package.label_purchased
    -- package.tracking_updated
    -- package.delivered
    -- package.delivery_failed
    --
    -- pickup.scheduled
    -- pickup.cancelled
    -- pickup.failed
    --
    -- Keep these short, stable, and code-driven.

    source shipment_event_source_enum NOT NULL DEFAULT 'system',
    severity shipment_event_severity_enum NOT NULL DEFAULT 'info',

    -- -------------------------------------------------
    -- HUMAN-FRIENDLY DISPLAY (UI)
    -- -------------------------------------------------
    title VARCHAR(200),
    message TEXT,
    -- These are optional. If null, UI can derive from event_key.

    -- -------------------------------------------------
    -- PROVIDER METADATA (OPTIONAL)
    -- -------------------------------------------------
    provider VARCHAR(50),
    -- e.g. shippo | ups | fedex | canadapost

    provider_event_id VARCHAR(150),
    -- provider's unique event id (if available)
    -- used for deduplication

    provider_reference VARCHAR(150),
    -- optional reference id (shipment/transaction/etc. from provider)

    raw_payload JSONB,
    -- store raw provider payload for audit/debug
    -- do not rely on it for business logic

    -- -------------------------------------------------
    -- EVENT TIMING
    -- -------------------------------------------------
    occurred_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    -- When the event actually occurred (provider time / system time)

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- AUDIT
    -- -------------------------------------------------
    actor_user_id UUID,
    -- user/admin who caused the event (if source=user/admin)
    -- null for pure system/provider events

    actor_ip INET,
    actor_user_agent TEXT,

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT fk_shipment_events_org
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id)
        ON DELETE CASCADE,
        
    CONSTRAINT chk_shipment_events_entity_target
        CHECK (
            -- Shipment events have no sub-entity
            (entity_type = 'shipment' AND package_id IS NULL AND pickup_id IS NULL)
            OR
            -- Package events target exactly one package
            (entity_type = 'package' AND package_id IS NOT NULL AND pickup_id IS NULL)
            OR
            -- Pickup events target exactly one pickup
            (entity_type = 'pickup' AND pickup_id IS NOT NULL AND package_id IS NULL)
        ),

    -- -------------------------------------------------
    -- FOREIGN KEYS
    -- -------------------------------------------------
    CONSTRAINT fk_shipment_events_shipment
        FOREIGN KEY (shipment_id)
        REFERENCES shipments(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_shipment_events_package
        FOREIGN KEY (package_id)
        REFERENCES shipment_packages(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_shipment_events_actor_user
        FOREIGN KEY (actor_user_id)
        REFERENCES users(id)
);

-- =====================================================
-- DEDUPLICATION (PROVIDER EVENTS)
-- =====================================================
-- Providers may deliver the same webhook/event multiple times.
-- If provider_event_id exists, enforce uniqueness per provider.

CREATE UNIQUE INDEX uq_shipment_events_provider_event
    ON shipment_events(provider, provider_event_id)
    WHERE provider IS NOT NULL
      AND provider_event_id IS NOT NULL;


-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_shipment_events_shipment_id
    ON shipment_events(shipment_id);

CREATE INDEX idx_shipment_events_package_id
    ON shipment_events(package_id);

CREATE INDEX idx_shipment_events_pickup_id
    ON shipment_events(pickup_id);

CREATE INDEX idx_shipment_events_entity_type
    ON shipment_events(entity_type);

CREATE INDEX idx_shipment_events_event_key
    ON shipment_events(event_key);

CREATE INDEX idx_shipment_events_occurred_at
    ON shipment_events(occurred_at);

CREATE INDEX idx_shipment_events_source
    ON shipment_events(source);

CREATE INDEX idx_shipment_events_severity
    ON shipment_events(severity);

CREATE INDEX idx_shipment_events_org_id
    ON shipment_events(organization_id);

-- =====================================================
-- NOTES
-- =====================================================
-- WHY shipment_id IS REQUIRED:
-- -----------------------------------------------------
-- Even package/pickup events must roll up into one shipment timeline.
-- This makes UI queries and notifications fast and simple:
--   SELECT * FROM shipment_events WHERE shipment_id = ? ORDER BY occurred_at;
--
-- WHY event_key EXISTS:
-- -----------------------------------------------------
-- UI and notification templates should key off stable event_key values,
-- not fragile provider messages or raw payload shapes.
--
-- PROVIDER PAYLOAD STORAGE:
-- -----------------------------------------------------
-- raw_payload is for audit/debug only.
-- If you need structured values (tracking, status, eta), store them in:
-- - shipment_packages (tracking, label fields)
-- - shipments (shipment_status, payment_status)
-- - pickups (pickup_status, provider_pickup_id)
-- =====================================================
