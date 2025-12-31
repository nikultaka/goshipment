-- =====================================================
-- MODULE 14: SENDGRID WRAPPER
-- =====================================================

-- SendGrid templates (managed templates)
CREATE TABLE sendgrid_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Template Identity
    template_name VARCHAR(100) UNIQUE NOT NULL,
    sendgrid_template_id VARCHAR(100) UNIQUE NOT NULL, -- SendGrid dynamic template ID
    
    -- Template Details
    template_subject VARCHAR(255),
    template_description TEXT,
    
    -- Categorization
    category VARCHAR(50), -- transactional, marketing, notification
    use_case VARCHAR(100), -- welcome, password_reset, invoice, newsletter
    
    -- Localization
    locale VARCHAR(10) DEFAULT 'en_CA',
    
    -- Template Variables
    required_variables JSONB DEFAULT '[]', -- Required substitution variables
    optional_variables JSONB DEFAULT '[]', -- Optional variables
    test_data JSONB DEFAULT '{}', -- Sample data for testing
    
    -- SendGrid Configuration
    generation INTEGER DEFAULT 1, -- dynamic or legacy template generation
    
    -- Version Control
    template_version VARCHAR(20),
    is_active_version BOOLEAN DEFAULT true,
    previous_version_id UUID REFERENCES sendgrid_templates(id),
    
    -- Settings
    is_active BOOLEAN DEFAULT true,
    
    -- Tracking
    track_opens BOOLEAN DEFAULT true,
    track_clicks BOOLEAN DEFAULT true,
    track_subscriptions BOOLEAN DEFAULT false,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_synced_at TIMESTAMP, -- Last sync with SendGrid
    created_by UUID REFERENCES users(id)
);

-- SendGrid sending history and status
CREATE TABLE sendgrid_sends (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- sgr_123abc
    
    -- Message Identity
    sendgrid_message_id VARCHAR(100), -- SendGrid's message ID
    template_id UUID REFERENCES sendgrid_templates(id),
    
    -- Recipient
    recipient_email VARCHAR(255) NOT NULL,
    recipient_name VARCHAR(255),
    from_email VARCHAR(255),
    from_name VARCHAR(255),
    
    -- Content
    subject VARCHAR(255),
    template_data JSONB DEFAULT '{}', -- Variables passed to template
    
    -- SendGrid Request
    request_payload JSONB, -- Full API request payload
    
    -- Sending Status
    send_status VARCHAR(20) DEFAULT 'queued', -- queued, processing, sent, failed, cancelled
    
    -- SendGrid Response
    response_status_code INTEGER,
    response_headers JSONB,
    response_body TEXT,
    
    -- Error Handling
    error_code VARCHAR(50),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Delivery Tracking
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    bounced_at TIMESTAMP,
    unsubscribed_at TIMESTAMP,
    
    -- Bounce/Error Details
    bounce_reason VARCHAR(255),
    bounce_type VARCHAR(20), -- hard, soft, spam
    
    -- Context
    user_id UUID REFERENCES users(id), -- If sent to registered user
    triggered_by_event VARCHAR(100), -- What triggered this email
    campaign_name VARCHAR(255), -- Marketing campaign
    
    -- Priority
    priority VARCHAR(10) DEFAULT 'normal', -- low, normal, high
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- SendGrid webhook events (delivery updates from SendGrid)
CREATE TABLE sendgrid_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event Identity
    sendgrid_event_id VARCHAR(100), -- From webhook payload
    message_id VARCHAR(100), -- Links to sendgrid_sends
    
    -- Event Details
    event_type VARCHAR(30) NOT NULL, -- processed, delivered, opened, clicked, bounced, dropped, etc.
    timestamp TIMESTAMP NOT NULL, -- When event occurred
    
    -- Email Details
    email VARCHAR(255),
    subject VARCHAR(255),
    
    -- Event Specific Data
    ip VARCHAR(45), -- IP address where event occurred
    user_agent TEXT,
    url VARCHAR(500), -- For click events
    reason VARCHAR(500), -- For bounces/drops
    status VARCHAR(100), -- HTTP status for bounces
    
    -- Geographic
    country VARCHAR(3),
    region VARCHAR(100),
    city VARCHAR(100),
    
    -- Categories/Tags
    categories TEXT[], -- SendGrid categories
    unique_args JSONB, -- Custom arguments passed with email
    
    -- Marketing Campaigns
    marketing_campaign_id VARCHAR(100),
    marketing_campaign_name VARCHAR(255),
    
    -- Raw Webhook Data
    raw_payload JSONB, -- Full webhook payload for debugging
    
    -- Processing
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP,
    
    -- Audit
    received_at TIMESTAMP DEFAULT NOW()
);

-- SendGrid suppression management
CREATE TABLE sendgrid_suppressions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Email
    email VARCHAR(255) UNIQUE NOT NULL,
    
    -- Suppression Type
    suppression_type VARCHAR(20) NOT NULL, -- bounce, spam_report, unsubscribe, group_unsubscribe, invalid_email
    
    -- Details
    reason VARCHAR(500),
    created_timestamp TIMESTAMP, -- From SendGrid
    
    -- Group Unsubscribes
    group_id INTEGER, -- SendGrid unsubscribe group ID
    group_name VARCHAR(255),
    
    -- Management
    is_active BOOLEAN DEFAULT true,
    can_resubscribe BOOLEAN DEFAULT true,
    
    -- Sync Status
    synced_from_sendgrid BOOLEAN DEFAULT false,
    last_checked_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- SendGrid API usage tracking
CREATE TABLE sendgrid_api_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- API Call Details
    endpoint VARCHAR(200), -- /v3/mail/send, /v3/templates
    http_method VARCHAR(10),
    
    -- Response
    status_code INTEGER,
    response_time_ms INTEGER,
    
    -- Usage Stats
    emails_sent INTEGER DEFAULT 0, -- For /mail/send endpoint
    
    -- Rate Limiting
    rate_limit_limit INTEGER, -- X-RateLimit-Limit header
    rate_limit_remaining INTEGER, -- X-RateLimit-Remaining header
    rate_limit_reset TIMESTAMP, -- X-RateLimit-Reset header
    
    -- Error Tracking
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Aggregation
    hour_bucket TIMESTAMP, -- For hourly aggregation
    
    called_at TIMESTAMP DEFAULT NOW()
);

-- SendGrid email statistics (daily aggregates)
CREATE TABLE sendgrid_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Time Period
    date DATE NOT NULL,
    
    -- Volume Stats
    emails_sent INTEGER DEFAULT 0,
    emails_delivered INTEGER DEFAULT 0,
    emails_bounced INTEGER DEFAULT 0,
    emails_dropped INTEGER DEFAULT 0,
    
    -- Engagement Stats
    emails_opened INTEGER DEFAULT 0,
    unique_opens INTEGER DEFAULT 0,
    emails_clicked INTEGER DEFAULT 0,
    unique_clicks INTEGER DEFAULT 0,
    
    -- List Management
    unsubscribes INTEGER DEFAULT 0,
    spam_reports INTEGER DEFAULT 0,
    
    -- Calculated Metrics
    delivery_rate DECIMAL(5,2), -- (delivered / sent) * 100
    bounce_rate DECIMAL(5,2), -- (bounced / sent) * 100
    open_rate DECIMAL(5,2), -- (unique_opens / delivered) * 100
    click_rate DECIMAL(5,2), -- (unique_clicks / delivered) * 100
    
    -- Categories
    category_breakdown JSONB DEFAULT '{}',
    
    -- Sync
    synced_from_sendgrid BOOLEAN DEFAULT false,
    synced_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(date)
);

