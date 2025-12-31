-- =====================================================
-- MODULE 9: STORAGE
-- =====================================================

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL, -- file_123abc
    
    -- File Metadata
    original_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL, -- bytes
    mime_type VARCHAR(100) NOT NULL,
    file_extension VARCHAR(10),
    
    -- Content
    file_hash VARCHAR(64), -- SHA-256 for deduplication
    content_hash VARCHAR(64), -- For content integrity
    
    -- Storage Location
    storage_provider VARCHAR(50) DEFAULT 'local', -- local, s3, gcp, azure
    storage_path VARCHAR(500) NOT NULL, -- Internal storage path
    storage_bucket VARCHAR(100), -- For cloud storage
    storage_region VARCHAR(50), -- For cloud storage
    
    -- URLs
    public_url VARCHAR(500), -- CDN URL if public
    signed_url_expires_at TIMESTAMP, -- For temporary access
    
    -- Ownership
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    owned_by UUID REFERENCES users(id) ON DELETE CASCADE,
    business_id UUID, -- Will reference businesses(id) in Phase 2
    
    -- Access Control
    visibility VARCHAR(20) DEFAULT 'private', -- private, public, protected, internal
    access_token VARCHAR(100), -- For protected files
    password_protected BOOLEAN DEFAULT false,
    password_hash VARCHAR(255), -- For password-protected files
    
    -- Usage
    download_count INTEGER DEFAULT 0,
    last_downloaded_at TIMESTAMP,
    view_count INTEGER DEFAULT 0, -- For images/documents
    last_viewed_at TIMESTAMP,
    
    -- Classification
    file_type VARCHAR(50), -- document, image, video, audio, archive, code
    file_category VARCHAR(50), -- invoice, label, profile_picture, document_upload
    content_classification VARCHAR(20) DEFAULT 'general', -- general, sensitive, confidential, restricted
    
    -- Relationships
    related_resource_type VARCHAR(50), -- shipments, invoices, users, support_tickets
    related_resource_id VARCHAR(100),
    parent_file_id UUID REFERENCES files(id), -- For file versions
    
    -- Virus Scanning
    virus_scan_status VARCHAR(20) DEFAULT 'pending', -- pending, clean, infected, failed
    virus_scan_at TIMESTAMP,
    quarantined BOOLEAN DEFAULT false,
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP, -- For temporary files
    archived_at TIMESTAMP, -- When moved to archive storage
    
    -- Compliance
    gdpr_data_subject_id UUID REFERENCES users(id), -- For GDPR deletion
    retention_policy VARCHAR(50), -- legal_7_years, marketing_3_years, temporary_30_days
    retention_until TIMESTAMP,
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP, -- Soft delete
    purge_after TIMESTAMP -- GDPR request - actually delete the files
);

-- File access logs (for compliance and security)
CREATE TABLE file_access_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    
    -- Accessor
    user_id UUID REFERENCES users(id),
    api_client_id UUID REFERENCES api_clients(id),
    
    -- Access Details
    access_type VARCHAR(20) NOT NULL, -- download, view, metadata, preview
    access_method VARCHAR(50), -- direct_url, api, admin_panel, public_link
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    referrer_url VARCHAR(500),
    
    -- Result
    access_granted BOOLEAN DEFAULT true,
    denial_reason VARCHAR(100),
    bytes_transferred BIGINT,
    
    -- Timing
    access_duration_ms INTEGER,
    accessed_at TIMESTAMP DEFAULT NOW(),
    
    -- Security
    suspicious_activity BOOLEAN DEFAULT false,
    flagged_reason VARCHAR(255)
);

-- File versions (for version control)
CREATE TABLE file_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    
    -- Version Info
    version_number INTEGER NOT NULL,
    version_name VARCHAR(100), -- v1.0, draft, final
    
    -- Version Metadata
    change_description TEXT,
    is_current BOOLEAN DEFAULT false,
    
    -- Storage
    storage_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64),
    
    -- Audit
    uploaded_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(file_id, version_number)
);

-- File shares (for sharing files with external parties)
CREATE TABLE file_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_id VARCHAR(50) UNIQUE NOT NULL, -- share_123abc
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    
    -- Share Configuration
    share_token VARCHAR(100) UNIQUE NOT NULL, -- URL token
    share_name VARCHAR(255), -- Custom name for the share
    
    -- Access Control
    requires_password BOOLEAN DEFAULT false,
    password_hash VARCHAR(255),
    allowed_downloads INTEGER, -- NULL = unlimited
    download_count INTEGER DEFAULT 0,
    
    -- Recipients
    allowed_emails TEXT[], -- Specific email addresses
    
    -- Permissions
    can_download BOOLEAN DEFAULT true,
    can_view_only BOOLEAN DEFAULT false,
    can_preview BOOLEAN DEFAULT true,
    
    -- Lifecycle
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    -- Notifications
    notify_on_access BOOLEAN DEFAULT false,
    
    -- Audit
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_accessed_at TIMESTAMP
);

-- Storage quotas (per user/organization)
CREATE TABLE storage_quotas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Owner
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    business_id UUID, -- Will reference businesses(id) in Phase 2
    
    -- Quota Configuration
    quota_type VARCHAR(20) DEFAULT 'user', -- user, business, system
    storage_limit_bytes BIGINT NOT NULL, -- Total storage allowed
    file_count_limit INTEGER, -- Max number of files
    
    -- Current Usage
    storage_used_bytes BIGINT DEFAULT 0,
    file_count INTEGER DEFAULT 0,
    
    -- Calculation
    last_calculated_at TIMESTAMP DEFAULT NOW(),
    
    -- Overrides
    custom_quota BOOLEAN DEFAULT false, -- Custom quota vs plan-based
    quota_reason VARCHAR(255), -- Why custom quota was set
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT storage_quota_owner CHECK (
        (user_id IS NOT NULL AND business_id IS NULL) OR 
        (user_id IS NULL AND business_id IS NOT NULL) OR
        (quota_type = 'system')
    )
);

CREATE TABLE file_processing_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES files(id),
    operation VARCHAR(50), -- upload, generate_preview, virus_scan
    processor VARCHAR(50), -- api, worker, system
    result VARCHAR(20), -- success, failed
    details TEXT,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE gdpr_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    request_type VARCHAR(20), -- export, delete, restrict
    status VARCHAR(20) DEFAULT 'pending',
    requested_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    notes TEXT
);
