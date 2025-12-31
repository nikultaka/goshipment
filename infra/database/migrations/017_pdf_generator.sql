-- =====================================================
-- MODULE 17: PDF GENERATOR
-- =====================================================

-- PDF templates
CREATE TABLE pdf_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- pdf_123abc
    
    -- Template Identity
    template_name VARCHAR(100) UNIQUE NOT NULL,
    template_type VARCHAR(50) NOT NULL, -- invoice, label, report, certificate, document
    
    -- Template Content
    template_html TEXT NOT NULL, -- HTML template
    template_css TEXT, -- CSS styles
    header_html TEXT, -- Page header
    footer_html TEXT, -- Page footer
    
    -- Template Variables
    required_variables JSONB DEFAULT '[]', -- Required data variables
    optional_variables JSONB DEFAULT '[]', -- Optional variables
    sample_data JSONB DEFAULT '{}', -- Test data for preview
    
    -- Page Configuration
    page_size VARCHAR(20) DEFAULT 'A4', -- A4, Letter, Legal, A3
    page_orientation VARCHAR(20) DEFAULT 'portrait', -- portrait, landscape
    page_margins JSONB DEFAULT '{"top": 20, "right": 20, "bottom": 20, "left": 20}', -- mm
    
    -- PDF Options
    enable_header BOOLEAN DEFAULT false,
    enable_footer BOOLEAN DEFAULT false,
    enable_background BOOLEAN DEFAULT true,
    print_background BOOLEAN DEFAULT true,
    
    -- Quality Settings
    dpi INTEGER DEFAULT 300, -- Resolution
    jpeg_quality INTEGER DEFAULT 90, -- 1-100
    compress_pdf BOOLEAN DEFAULT true,
    
    -- Security
    enable_password BOOLEAN DEFAULT false,
    owner_password VARCHAR(255), -- Encrypted
    user_password VARCHAR(255), -- Encrypted
    permissions JSONB DEFAULT '{}', -- PDF permissions
    
    -- Versioning
    version INTEGER DEFAULT 1,
    is_active_version BOOLEAN DEFAULT true,
    previous_version_id UUID REFERENCES pdf_templates(id),
    
    -- Usage
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    last_modified_by UUID REFERENCES users(id)
);

-- PDF generation queue
CREATE TABLE pdf_generation_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- pdfq_123abc
    
    -- Template
    template_id UUID NOT NULL REFERENCES pdf_templates(id),
    
    -- Generation Request
    request_data JSONB NOT NULL, -- Data to merge with template
    
    -- Output Configuration
    output_filename VARCHAR(255),
    output_format VARCHAR(10) DEFAULT 'pdf', -- pdf, png, jpeg
    
    -- Custom Options (override template defaults)
    custom_page_size VARCHAR(20),
    custom_orientation VARCHAR(20),
    custom_margins JSONB,
    
    -- Security Override
    custom_password VARCHAR(255), -- Encrypted
    custom_permissions JSONB,
    
    -- Processing Status
    status VARCHAR(20) DEFAULT 'queued', -- queued, processing, completed, failed, cancelled
    
    -- Priority
    priority VARCHAR(10) DEFAULT 'normal', -- low, normal, high, urgent
    
    -- Context
    user_id UUID REFERENCES users(id),
    related_resource_type VARCHAR(50), -- invoice, shipment, report
    related_resource_id VARCHAR(100),
    
    -- Processing Details
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_time_ms INTEGER,
    
    -- Output
    generated_file_id UUID REFERENCES files(id), -- Generated PDF file
    output_file_size BIGINT, -- File size in bytes
    page_count INTEGER,
    
    -- Error Handling
    error_code VARCHAR(50),
    error_message TEXT,
    error_details JSONB,
    
    -- Retry Logic
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    retry_after TIMESTAMP,
    
    -- Worker Information
    processed_by_worker VARCHAR(100), -- Worker instance that processed
    worker_version VARCHAR(50),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- PDF generation statistics
CREATE TABLE pdf_generation_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Time Period
    date DATE NOT NULL,
    template_id UUID REFERENCES pdf_templates(id),
    
    -- Volume Stats
    total_requests INTEGER DEFAULT 0,
    successful_generations INTEGER DEFAULT 0,
    failed_generations INTEGER DEFAULT 0,
    cancelled_generations INTEGER DEFAULT 0,
    
    -- Performance Stats
    avg_processing_time_ms INTEGER,
    min_processing_time_ms INTEGER,
    max_processing_time_ms INTEGER,
    avg_file_size_bytes BIGINT,
    
    -- Queue Stats
    avg_queue_time_ms INTEGER,
    max_queue_time_ms INTEGER,
    
    -- Error Stats
    error_breakdown JSONB DEFAULT '{}', -- Errors by type
    
    -- Quality Stats
    avg_page_count DECIMAL(5,2),
    
    calculated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(date, template_id)
);

-- PDF watermarks (for branded documents)
CREATE TABLE pdf_watermarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Watermark Identity
    watermark_name VARCHAR(100) UNIQUE NOT NULL,
    
    -- Watermark Type
    watermark_type VARCHAR(20) NOT NULL, -- text, image, logo
    
    -- Text Watermark
    watermark_text VARCHAR(255), -- Text to display
    font_family VARCHAR(50) DEFAULT 'Arial',
    font_size INTEGER DEFAULT 12,
    font_color VARCHAR(7) DEFAULT '#000000', -- Hex color
    opacity DECIMAL(3,2) DEFAULT 0.50, -- 0.00 to 1.00
    
    -- Image Watermark
    watermark_image_file_id UUID REFERENCES files(id), -- Image file
    image_scale DECIMAL(3,2) DEFAULT 1.00, -- Scale factor
    
    -- Position
    position VARCHAR(20) DEFAULT 'center', -- center, top-left, top-right, bottom-left, bottom-right
    offset_x INTEGER DEFAULT 0, -- Horizontal offset in pixels
    offset_y INTEGER DEFAULT 0, -- Vertical offset in pixels
    rotation_degrees INTEGER DEFAULT 0, -- Rotation angle
    
    -- Application
    apply_to_all_pages BOOLEAN DEFAULT true,
    apply_to_first_page BOOLEAN DEFAULT true,
    apply_behind_content BOOLEAN DEFAULT true, -- Background vs foreground
    
    -- Usage
    is_default BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

-- PDF digital signatures (for document authenticity)
CREATE TABLE pdf_digital_signatures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Signature Identity
    signature_name VARCHAR(100) UNIQUE NOT NULL,
    
    -- Certificate Information
    certificate_file_id UUID REFERENCES files(id), -- PFX/P12 certificate
    certificate_password VARCHAR(255), -- Encrypted
    certificate_alias VARCHAR(100),
    
    -- Signer Information
    signer_name VARCHAR(255) NOT NULL,
    signer_organization VARCHAR(255),
    signer_email VARCHAR(255),
    signer_location VARCHAR(255),
    
    -- Signature Appearance
    signature_image_file_id UUID REFERENCES files(id), -- Signature image
    show_signature_image BOOLEAN DEFAULT true,
    signature_text TEXT, -- Text to display in signature
    
    -- Positioning
    signature_page INTEGER DEFAULT 1, -- Which page to sign (1-based)
    signature_x INTEGER, -- X position in pixels
    signature_y INTEGER, -- Y position in pixels
    signature_width INTEGER DEFAULT 200,
    signature_height INTEGER DEFAULT 100,
    
    -- Certification Level
    certification_level INTEGER DEFAULT 0, -- 0=signature, 1=certified, 2=approval
    
    -- Timestamp Authority
    timestamp_server_url VARCHAR(500), -- TSA server URL
    use_timestamp BOOLEAN DEFAULT false,
    
    -- Usage Tracking
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

-- PDF batch operations (for bulk generation)
CREATE TABLE pdf_batch_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- batch_123abc
    
    -- Batch Details
    batch_name VARCHAR(255),
    operation_type VARCHAR(50) NOT NULL, -- generate, merge, split, compress, watermark
    
    -- Template (for generation)
    template_id UUID REFERENCES pdf_templates(id),
    
    -- Input Data
    input_data JSONB, -- Array of data objects for generation
    input_files JSONB, -- Array of file IDs for merging/processing
    
    -- Configuration
    batch_config JSONB DEFAULT '{}', -- Operation-specific configuration
    
    -- Output Settings
    output_format VARCHAR(20) DEFAULT 'individual', -- individual, merged, zip_archive
    output_prefix VARCHAR(100), -- Filename prefix
    
    -- Processing Status
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, completed, failed, cancelled
    
    -- Progress Tracking
    total_items INTEGER NOT NULL,
    processed_items INTEGER DEFAULT 0,
    successful_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    
    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    estimated_completion TIMESTAMP,
    
    -- Results
    output_files JSONB DEFAULT '[]', -- Generated file IDs
    result_summary JSONB DEFAULT '{}',
    
    -- Error Tracking
    errors JSONB DEFAULT '[]', -- Array of errors encountered
    
    -- Resource Usage
    memory_usage_mb INTEGER,
    cpu_time_seconds INTEGER,
    
    -- Context
    created_by UUID REFERENCES users(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);