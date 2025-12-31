-- =====================================================
-- MODULE 16: TWILIO WRAPPER
-- =====================================================

-- Twilio messaging services configuration
CREATE TABLE twilio_messaging_services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Service Identity
    service_name VARCHAR(100) UNIQUE NOT NULL,
    twilio_service_sid VARCHAR(34) UNIQUE, -- Twilio messaging service SID
    
    -- Configuration
    use_case VARCHAR(50) DEFAULT 'notifications', -- notifications, marketing, verification, alerts
    
    -- Sender Configuration
    sender_pool TEXT[], -- Array of phone numbers/sender IDs
    use_alphanumeric_sender BOOLEAN DEFAULT false,
    default_sender VARCHAR(20), -- Default phone number or sender ID
    
    -- Message Settings
    validity_period INTEGER DEFAULT 14400, -- Message validity in seconds (4 hours)
    delivery_callback_url VARCHAR(500), -- Webhook for delivery status
    
    -- Features
    smart_encoding BOOLEAN DEFAULT true, -- Twilio Smart Encoding
    sticky_sender BOOLEAN DEFAULT false, -- Use same sender for conversation
    
    -- Fallback Configuration
    fallback_to_long_code BOOLEAN DEFAULT true,
    
    -- Geographic
    supported_countries TEXT[] DEFAULT ARRAY['CA', 'US'], -- ISO country codes
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_synced_at TIMESTAMP,
    created_by UUID REFERENCES users(id)
);

-- SMS/messaging queue and history
CREATE TABLE twilio_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- twm_123abc
    
    -- Message Identity
    twilio_message_sid VARCHAR(34) UNIQUE, -- Twilio message SID
    messaging_service_id UUID REFERENCES twilio_messaging_services(id),
    
    -- Sender/Recipient
    from_number VARCHAR(20),
    to_number VARCHAR(20) NOT NULL,
    
    -- Content
    message_body TEXT NOT NULL,
    message_type VARCHAR(20) DEFAULT 'sms', -- sms, mms, whatsapp
    
    -- Media (for MMS)
    media_urls TEXT[], -- Array of media URLs
    media_count INTEGER DEFAULT 0,
    
    -- Message Configuration
    max_price DECIMAL(10,4), -- Maximum price willing to pay
    provide_feedback BOOLEAN DEFAULT false, -- Request delivery feedback
    attempt INTEGER DEFAULT 1,
    validity_period INTEGER, -- Override default validity period
    
    -- Twilio Request
    request_payload JSONB, -- Full API request
    
    -- Status Tracking
    status VARCHAR(20) DEFAULT 'queued', -- queued, sending, sent, delivered, undelivered, failed
    twilio_status VARCHAR(20), -- Raw Twilio status
    
    -- Error Handling
    error_code VARCHAR(10), -- Twilio error code
    error_message TEXT,
    
    -- Delivery Details
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    
    -- Pricing
    price DECIMAL(10,4), -- Actual price charged
    price_unit VARCHAR(3), -- Currency
    
    -- Context
    user_id UUID REFERENCES users(id),
    triggered_by_event VARCHAR(100),
    campaign_name VARCHAR(255),
    
    -- Priority
    priority VARCHAR(10) DEFAULT 'normal',
    
    -- Retry Logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Twilio webhook events (status updates)
CREATE TABLE twilio_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event Identity
    message_sid VARCHAR(34), -- Links to twilio_messages
    account_sid VARCHAR(34),
    
    -- Event Details
    message_status VARCHAR(20) NOT NULL, -- queued, sending, sent, delivered, undelivered, failed
    status_timestamp TIMESTAMP,
    
    -- Message Details
    from_number VARCHAR(20),
    to_number VARCHAR(20),
    message_body TEXT,
    
    -- Error Information
    error_code VARCHAR(10),
    error_message TEXT,
    
    -- Pricing (if available)
    price DECIMAL(10,4),
    price_unit VARCHAR(3),
    
    -- Geographic/Carrier Info
    from_country VARCHAR(3),
    to_country VARCHAR(3),
    from_state VARCHAR(50),
    to_state VARCHAR(50),
    from_city VARCHAR(100),
    to_city VARCHAR(100),
    from_zip VARCHAR(20),
    to_zip VARCHAR(20),
    
    -- Raw Webhook Data
    raw_payload JSONB,
    
    -- Processing
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP,
    
    received_at TIMESTAMP DEFAULT NOW()
);

-- Phone number validation and formatting
CREATE TABLE phone_number_validation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Original Number
    original_number VARCHAR(50) NOT NULL,
    
    -- Formatted Number
    formatted_number VARCHAR(20),
    e164_format VARCHAR(20), -- International format
    national_format VARCHAR(30),
    
    -- Geographic Information
    country_code VARCHAR(3), -- ISO country code
    country_calling_code VARCHAR(5), -- +1, +44, etc.
    region VARCHAR(100),
    
    -- Carrier Information
    carrier_name VARCHAR(100),
    carrier_type VARCHAR(20), -- mobile, landline, voip
    
    -- Validation Results
    is_valid BOOLEAN,
    is_mobile BOOLEAN,
    is_landline BOOLEAN,
    is_voip BOOLEAN,
    can_receive_sms BOOLEAN,
    
    -- Quality Score
    validity_score DECIMAL(3,2), -- 0.00 to 1.00
    
    -- Twilio Lookup API Response
    twilio_lookup_response JSONB,
    
    -- Cache Control
    validated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP, -- Cache expiration
    
    -- Usage
    validation_count INTEGER DEFAULT 1,
    last_validated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(original_number)
);

-- SMS opt-out management
CREATE TABLE sms_opt_outs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Phone Number
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    formatted_phone VARCHAR(20), -- E.164 format
    
    -- Opt-out Details
    opt_out_method VARCHAR(20) NOT NULL, -- keyword, web_form, api, support_request
    opt_out_keyword VARCHAR(20), -- STOP, UNSUBSCRIBE, etc.
    opt_out_message TEXT, -- User's opt-out message
    
    -- Context
    user_id UUID REFERENCES users(id),
    messaging_service_id UUID REFERENCES twilio_messaging_services(id),
    
    -- Scope
    opt_out_scope VARCHAR(20) DEFAULT 'all', -- all, service_specific, campaign_specific
    campaign_name VARCHAR(255), -- For campaign-specific opt-outs
    
    -- Management
    is_active BOOLEAN DEFAULT true,
    can_opt_in BOOLEAN DEFAULT true,
    
    -- Compliance
    compliance_notes TEXT, -- Regulatory compliance notes
    
    opted_out_at TIMESTAMP DEFAULT NOW(),
    opted_in_at TIMESTAMP, -- If they opt back in
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Twilio API usage and rate limiting
CREATE TABLE twilio_api_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- API Call Details
    api_endpoint VARCHAR(200), -- /Messages, /Accounts/ACxxx/Messages
    http_method VARCHAR(10),
    
    -- Response
    status_code INTEGER,
    response_time_ms INTEGER,
    
    -- Usage Metrics
    messages_sent INTEGER DEFAULT 0, -- For messaging endpoints
    lookups_performed INTEGER DEFAULT 0, -- For lookup endpoints
    
    -- Cost Tracking
    total_cost DECIMAL(10,4) DEFAULT 0,
    cost_currency VARCHAR(3) DEFAULT 'USD',
    
    -- Error Tracking
    error_code VARCHAR(10),
    error_message TEXT,
    
    -- Rate Limiting (if applicable)
    rate_limit_remaining INTEGER,
    rate_limit_reset TIMESTAMP,
    
    -- Aggregation
    hour_bucket TIMESTAMP GENERATED ALWAYS AS (date_trunc('hour', called_at)) STORED,
    
    called_at TIMESTAMP DEFAULT NOW()
);

-- Twilio account statistics
CREATE TABLE twilio_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Time Period
    date DATE NOT NULL,
    
    -- Volume Stats
    sms_sent INTEGER DEFAULT 0,
    sms_delivered INTEGER DEFAULT 0,
    sms_failed INTEGER DEFAULT 0,
    mms_sent INTEGER DEFAULT 0,
    mms_delivered INTEGER DEFAULT 0,
    
    -- Cost Stats
    total_cost DECIMAL(10,4) DEFAULT 0,
    sms_cost DECIMAL(10,4) DEFAULT 0,
    mms_cost DECIMAL(10,4) DEFAULT 0,
    lookup_cost DECIMAL(10,4) DEFAULT 0,
    
    -- Quality Metrics
    delivery_rate DECIMAL(5,2), -- (delivered / sent) * 100
    failure_rate DECIMAL(5,2), -- (failed / sent) * 100
    
    -- Carrier Breakdown
    carrier_breakdown JSONB DEFAULT '{}',
    
    -- Country Breakdown
    country_breakdown JSONB DEFAULT '{}',
    
    -- Sync Status
    synced_from_twilio BOOLEAN DEFAULT false,
    synced_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(date)
);