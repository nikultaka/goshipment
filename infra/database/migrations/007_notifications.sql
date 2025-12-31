-- =====================================================
-- MODULE 7: NOTIFICATIONS
-- =====================================================

-- Notification templates
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id VARCHAR(50) UNIQUE NOT NULL, -- welcome_email, password_reset, invoice_ready
    name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Channel
    channel VARCHAR(20) NOT NULL, -- email, sms, push, webhook
    
    -- Template Content
    subject VARCHAR(255), -- For email
    body_template TEXT NOT NULL, -- Template with {{variables}}
    html_template TEXT, -- HTML version for email
    
    -- Template Variables
    required_variables JSONB DEFAULT '[]', -- Required template variables
    optional_variables JSONB DEFAULT '[]', -- Optional variables
    
    -- Localization
    locale VARCHAR(10) DEFAULT 'en_CA',
    
    -- Settings
    is_active BOOLEAN DEFAULT true,
    is_transactional BOOLEAN DEFAULT true, -- Transactional vs marketing
    auto_send BOOLEAN DEFAULT false, -- Auto-send or require trigger
    
    -- Rate Limiting
    max_sends_per_user_per_hour INTEGER,
    max_sends_per_user_per_day INTEGER,
    
    -- Delivery Settings
    priority VARCHAR(10) DEFAULT 'normal', -- low, normal, high, urgent
    retry_attempts INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 60,
    
    -- Tracking
    track_opens BOOLEAN DEFAULT true, -- Email opens
    track_clicks BOOLEAN DEFAULT true, -- Email clicks
    
    -- External Provider Settings
    sendgrid_template_id VARCHAR(100), -- SendGrid dynamic template ID
    twilio_messaging_service_sid VARCHAR(100), -- Twilio messaging service
    
    -- Version Control
    version INTEGER DEFAULT 1,
    previous_version_id UUID REFERENCES notification_templates(id),
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    last_modified_by UUID REFERENCES users(id)
);

-- Notification queue (for NATS processing)
CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    queue_id VARCHAR(50) UNIQUE NOT NULL, -- nq_123abc
    
    -- Template
    template_id UUID NOT NULL REFERENCES notification_templates(id),
    
    -- Recipient
    user_id UUID REFERENCES users(id), -- If registered user
    recipient_email VARCHAR(255),
    recipient_phone VARCHAR(20),
    recipient_name VARCHAR(255),
    
    -- Content
    subject VARCHAR(255),
    body TEXT NOT NULL,
    html_body TEXT,
    template_variables JSONB DEFAULT '{}',
    
    -- Delivery
    channel VARCHAR(20) NOT NULL,
    priority VARCHAR(10) DEFAULT 'normal',
    
    -- Scheduling
    scheduled_at TIMESTAMP DEFAULT NOW(),
    send_after TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    -- Status
    status VARCHAR(20) DEFAULT 'queued', -- queued, processing, sent, failed, expired, cancelled
    
    -- Processing
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    last_attempt_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    
    -- Provider Details
    provider_used VARCHAR(50), -- sendgrid, twilio
    provider_message_id VARCHAR(255),
    provider_response JSONB DEFAULT '{}',
    
    -- Results
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Context
    triggered_by_event VARCHAR(100), -- user_registration, payment_success
    related_resource_type VARCHAR(50),
    related_resource_id VARCHAR(100),
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Notification delivery tracking
CREATE TABLE notification_delivery_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES notification_queue(id) ON DELETE CASCADE,
    
    -- Event
    event_type VARCHAR(30) NOT NULL, -- sent, delivered, opened, clicked, bounced, unsubscribed
    event_timestamp TIMESTAMP DEFAULT NOW(),
    
    -- Provider Event
    provider_event_id VARCHAR(255), -- External provider's event ID
    provider_event_data JSONB DEFAULT '{}',
    
    -- Email Specific
    email_subject VARCHAR(255),
    email_from VARCHAR(255),
    email_to VARCHAR(255),
    
    -- Click/Open Tracking
    clicked_url VARCHAR(500),
    ip_address INET,
    user_agent TEXT,
    location VARCHAR(100),
    
    -- SMS Specific
    sms_to VARCHAR(20),
    sms_from VARCHAR(20),
    sms_segments INTEGER, -- Number of SMS segments
    
    -- Error Details
    bounce_type VARCHAR(50), -- hard, soft
    bounce_reason VARCHAR(255),
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- User notification preferences
CREATE TABLE user_notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Channel Preferences
    email_enabled BOOLEAN DEFAULT true,
    sms_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT true,
    
    -- Notification Types
    marketing_emails BOOLEAN DEFAULT false,
    promotional_sms BOOLEAN DEFAULT false,
    security_alerts BOOLEAN DEFAULT true,
    order_updates BOOLEAN DEFAULT true,
    billing_notifications BOOLEAN DEFAULT true,
    system_announcements BOOLEAN DEFAULT true,
    
    -- Frequency Settings
    digest_frequency VARCHAR(20) DEFAULT 'immediate', -- immediate, daily, weekly, never
    quiet_hours_enabled BOOLEAN DEFAULT false,
    quiet_hours_start TIME, -- 22:00
    quiet_hours_end TIME, -- 08:00
    quiet_hours_timezone VARCHAR(50),
    
    -- Unsubscribe Tokens (for one-click unsubscribe)
    email_unsubscribe_token VARCHAR(100) UNIQUE,
    sms_unsubscribe_token VARCHAR(100) UNIQUE,
    
    -- Tracking
    preferences_updated_at TIMESTAMP DEFAULT NOW(),
    unsubscribed_from_all_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id)
);