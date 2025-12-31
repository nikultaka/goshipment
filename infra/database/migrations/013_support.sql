-- =====================================================
-- MODULE 13: SUPPORT (TICKETS)
-- =====================================================

CREATE TABLE support_tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- tick_123abc
    ticket_number VARCHAR(20) UNIQUE NOT NULL, -- SUP-2024-001
    
    -- Customer
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    customer_name VARCHAR(255),
    customer_email VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),
    
    -- Ticket Details
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50), -- technical, billing, general, bug_report, feature_request
    priority VARCHAR(10) DEFAULT 'medium', -- low, medium, high, urgent
    
    -- Classification
    ticket_type VARCHAR(50) DEFAULT 'question', -- question, issue, request, complaint
    source VARCHAR(50) DEFAULT 'web_form', -- web_form, email, phone, chat, api
    
    -- Assignment
    assigned_to UUID REFERENCES users(id),
    assigned_at TIMESTAMP,
    department VARCHAR(50), -- support, technical, billing, sales
    
    -- Status Tracking
    status VARCHAR(20) DEFAULT 'open', -- open, in_progress, pending_customer, resolved, closed
    resolution VARCHAR(20), -- resolved, not_resolved, duplicate, invalid
    
    -- Workflow
    first_response_at TIMESTAMP,
    last_response_at TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    
    -- SLA Tracking
    sla_priority VARCHAR(10), -- Calculated SLA priority
    sla_due_at TIMESTAMP, -- When response is due
    sla_breached BOOLEAN DEFAULT false,
    sla_breach_reason VARCHAR(255),
    
    -- Satisfaction
    satisfaction_rating INTEGER, -- 1-5 stars
    satisfaction_feedback TEXT,
    satisfaction_submitted_at TIMESTAMP,
    
    -- Related Resources
    related_resource_type VARCHAR(50), -- shipment, invoice, user_account
    related_resource_id VARCHAR(100),
    
    -- Tags & Labels
    tags TEXT[], -- For categorization and filtering
    labels TEXT[], -- Custom labels
    
    -- Escalation
    escalated BOOLEAN DEFAULT false,
    escalated_to UUID REFERENCES users(id),
    escalated_at TIMESTAMP,
    escalation_reason VARCHAR(255),
    
    -- Internal Notes
    internal_notes TEXT, -- Private notes for support team
    
    -- Metadata
    customer_ip_address INET,
    customer_user_agent TEXT,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Ticket messages/responses
CREATE TABLE support_ticket_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- msg_123abc
    ticket_id UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    
    -- Message Details
    message_text TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT false, -- Internal note vs customer-visible
    is_auto_generated BOOLEAN DEFAULT false, -- System-generated message
    
    -- Sender
    sender_type VARCHAR(20) NOT NULL, -- customer, agent, system
    sender_user_id UUID REFERENCES users(id), -- If sent by user/agent
    sender_name VARCHAR(255),
    sender_email VARCHAR(255),
    
    -- Message Type
    message_type VARCHAR(20) DEFAULT 'reply', -- reply, note, status_change, escalation
    
    -- Delivery
    delivery_method VARCHAR(20) DEFAULT 'web', -- web, email, sms
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    
    -- Attachments
    has_attachments BOOLEAN DEFAULT false,
    attachment_count INTEGER DEFAULT 0,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Ticket attachments
CREATE TABLE support_ticket_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    message_id UUID REFERENCES support_ticket_messages(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    
    -- Attachment Context
    attachment_type VARCHAR(50) DEFAULT 'document', -- document, screenshot, log_file
    description VARCHAR(255),
    
    -- Access Control
    visible_to_customer BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT NOW(),
    uploaded_by UUID REFERENCES users(id)
);

-- Support ticket templates (for common responses)
CREATE TABLE support_ticket_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_name VARCHAR(100) UNIQUE NOT NULL,
    
    -- Template Content
    subject VARCHAR(255),
    body_template TEXT NOT NULL,
    
    -- Usage
    category VARCHAR(50), -- Common category this template applies to
    is_public BOOLEAN DEFAULT true, -- Available to all agents
    
    -- Variables
    template_variables JSONB DEFAULT '[]', -- Required variables
    
    -- Statistics
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);
