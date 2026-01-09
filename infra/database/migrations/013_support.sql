-- =====================================================
-- 013_support_tickets.sql
-- Support Ticket System (Org-Scoped, Conversation-Based)
-- Includes Support Ticket Templates (Global + Org Overrides)
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE ticket_category_enum AS ENUM (
    'shipment',
    'wallet',
    'pickup',
    'invoice',
    'billing',
    'account',
    'technical',
    'other'
);

CREATE TYPE ticket_priority_enum AS ENUM (
    'critical',
    'high',
    'normal',
    'low'
);

CREATE TYPE ticket_status_enum AS ENUM (
    'open',
    'resolved',
    'closed'
);

CREATE TYPE ticket_resolution_reason_enum AS ENUM (
    'issue_resolved',
    'no_longer_needed',
    'created_by_mistake',
    'wrong_request_type',
    'will_contact_later',
    'other'
);

CREATE TYPE ticket_message_sender_enum AS ENUM (
    'user',
    'staff',
    'system'
);

-- =====================================================
-- SUPPORT TICKETS (CORE)
-- =====================================================

CREATE TABLE support_tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Human identifier
    ticket_number VARCHAR(20) NOT NULL UNIQUE,
    -- TCK-XXXXXXXX

    -- Scope
    organization_id UUID
        REFERENCES organizations(id) ON DELETE CASCADE,

    created_by_user_id UUID NOT NULL
        REFERENCES users(id),

    assigned_to_user_id UUID
        REFERENCES users(id) ON DELETE SET NULL,

    -- Classification
    category ticket_category_enum NOT NULL,
    priority ticket_priority_enum NOT NULL DEFAULT 'normal',
    status ticket_status_enum NOT NULL DEFAULT 'open',

    -- Content
    subject VARCHAR(255) NOT NULL,
    admin_notes TEXT,

    -- Resolution
    resolution_reason ticket_resolution_reason_enum,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,

    -- Activity
    last_message_at TIMESTAMP NOT NULL DEFAULT NOW(),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_resolution_requires_resolved_status
        CHECK (
            resolution_reason IS NULL
            OR status = 'resolved'
        )
);

-- =====================================================
-- SUPPORT TICKET MESSAGES
-- =====================================================

CREATE TABLE support_ticket_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    ticket_id UUID NOT NULL
        REFERENCES support_tickets(id) ON DELETE CASCADE,

    sender_type ticket_message_sender_enum NOT NULL,
    sender_user_id UUID
        REFERENCES users(id) ON DELETE SET NULL,

    message TEXT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =====================================================
-- SUPPORT TICKET ATTACHMENTS
-- =====================================================

CREATE TABLE support_ticket_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    ticket_id UUID NOT NULL
        REFERENCES support_tickets(id) ON DELETE CASCADE,

    message_id UUID
        REFERENCES support_ticket_messages(id) ON DELETE CASCADE,

    file_id UUID NOT NULL
        REFERENCES files(id) ON DELETE CASCADE,

    uploaded_by_user_id UUID NOT NULL
        REFERENCES users(id),

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =====================================================
-- SUPPORT TICKET TEMPLATES
-- =====================================================
-- NULL organization_id = global system template
-- NOT NULL organization_id = org-specific override
-- =====================================================

CREATE TABLE support_ticket_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    organization_id UUID
        REFERENCES organizations(id) ON DELETE CASCADE,

    template_name VARCHAR(100) NOT NULL,

    subject VARCHAR(255),
    body_template TEXT NOT NULL,

    category ticket_category_enum,

    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    template_variables JSONB NOT NULL DEFAULT '[]',

    usage_count INTEGER NOT NULL DEFAULT 0,
    last_used_at TIMESTAMP,

    created_by_user_id UUID
        REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Prevent duplicate template names per scope
    UNIQUE (organization_id, template_name)
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Tickets
CREATE INDEX idx_support_tickets_org
    ON support_tickets(organization_id);

CREATE INDEX idx_support_tickets_status
    ON support_tickets(status);

CREATE INDEX idx_support_tickets_priority
    ON support_tickets(priority);

CREATE INDEX idx_support_tickets_assigned_to
    ON support_tickets(assigned_to_user_id);

CREATE INDEX idx_support_tickets_last_message_at
    ON support_tickets(last_message_at);

-- Messages
CREATE INDEX idx_support_ticket_messages_ticket
    ON support_ticket_messages(ticket_id);

CREATE INDEX idx_support_ticket_messages_created_at
    ON support_ticket_messages(created_at);

-- Attachments
CREATE INDEX idx_support_ticket_attachments_ticket
    ON support_ticket_attachments(ticket_id);

CREATE INDEX idx_support_ticket_attachments_message
    ON support_ticket_attachments(message_id);

-- Templates
CREATE INDEX idx_support_ticket_templates_org
    ON support_ticket_templates(organization_id);

CREATE INDEX idx_support_ticket_templates_active
    ON support_ticket_templates(is_active);
