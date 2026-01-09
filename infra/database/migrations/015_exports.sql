-- =====================================================
-- 015_exports.sql
-- Unified Export System (PDF, CSV, ZIP)
-- Replaces: 017_pdf_generator.sql, 028_exports.sql
-- =====================================================
--
-- PURPOSE:
-- Single table for all async export jobs.
-- Go workers generate files, store in files table.
--
-- SUPPORTED FORMATS:
-- - PDF: Invoices, statements, reports
-- - CSV: Data exports, transaction lists
-- - ZIP: Contains both PDF and CSV
-- - XLSX: Excel (future)
--
-- EXPORTABLE ENTITIES:
-- - invoice (single)
-- - invoices (batch)
-- - wallet_statement
-- - shipments
-- - shipment_labels (batch PDFs)
-- - commission_statement
-- - address_book
-- - audit_log (admin)
-- =====================================================

-- =====================================================
-- ENUMS
-- =====================================================

CREATE TYPE export_status_enum AS ENUM (
    'queued',       -- Waiting for worker
    'processing',   -- Worker is generating
    'ready',        -- File ready for download
    'failed',       -- Generation failed
    'expired',      -- Past expiration, pending cleanup
    'cancelled'     -- User cancelled
);

CREATE TYPE export_format_enum AS ENUM (
    'pdf',
    'csv',
    'zip'      -- Contains PDF + CSV
);

CREATE TYPE export_entity_enum AS ENUM (
    'invoice',              -- Single invoice
    'invoices',             -- Multiple invoices
    'wallet_statement',     -- Wallet transaction history
    'shipments',            -- Shipment list
    'shipment_labels',      -- Batch shipping labels
    'commission_statement', -- Commission history
    'financial_report',     -- Admin financial report
    'address_book',         -- Contacts export
    'audit_log'             -- Admin audit log
);

CREATE TYPE export_delivery_enum AS ENUM (
    'download',     -- User downloads from UI
    'email'         -- Send via email when ready
);

-- =====================================================
-- EXPORTS
-- =====================================================

CREATE TABLE exports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- -------------------------------------------------
    -- IDENTIFIER
    -- -------------------------------------------------
    export_number VARCHAR(20) UNIQUE NOT NULL,  -- EXP-XXXXXXXX

    -- -------------------------------------------------
    -- OWNERSHIP
    -- -------------------------------------------------
    organization_id UUID NOT NULL
        REFERENCES organizations(id)
        ON DELETE CASCADE,

    requested_by_user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    -- -------------------------------------------------
    -- WHAT TO EXPORT
    -- -------------------------------------------------
    entity_type export_entity_enum NOT NULL,
    format export_format_enum NOT NULL DEFAULT 'csv',

    -- Flexible parameters based on entity_type
    -- Examples:
    --   invoice:    { "invoice_id": "uuid" }
    --   invoices:   { "invoice_ids": [...] } or { "date_range": {...} }
    --   shipments:  { "status": "delivered", "carrier": "fedex" }
    --   wallet:     { "start_date": "2025-01-01", "end_date": "2025-01-31" }
    export_params JSONB NOT NULL DEFAULT '{}',

    -- Human-readable title
    title VARCHAR(255),
    -- e.g., "Invoice INV-ABC123", "January 2025 Wallet Statement"

    -- -------------------------------------------------
    -- DELIVERY
    -- -------------------------------------------------
    delivery_method export_delivery_enum NOT NULL DEFAULT 'download',
    delivery_email VARCHAR(255),
    email_sent_at TIMESTAMP,

    -- -------------------------------------------------
    -- STATUS
    -- -------------------------------------------------
    status export_status_enum NOT NULL DEFAULT 'queued',

    -- Progress (for large exports)
    progress_percent SMALLINT DEFAULT 0,
    progress_message VARCHAR(255),

    -- Error details
    error_code VARCHAR(50),
    error_message TEXT,
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 3,

    -- -------------------------------------------------
    -- OUTPUT FILES
    -- -------------------------------------------------
    -- Primary output file
    file_id UUID
        REFERENCES files(id)
        ON DELETE SET NULL,

    -- For ZIP: individual components
    pdf_file_id UUID
        REFERENCES files(id)
        ON DELETE SET NULL,

    csv_file_id UUID
        REFERENCES files(id)
        ON DELETE SET NULL,

    -- File info (denormalized for quick access)
    file_name VARCHAR(255),
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    row_count INTEGER,          -- For CSV: number of data rows

    -- -------------------------------------------------
    -- DOWNLOAD TRACKING
    -- -------------------------------------------------
    download_count INTEGER NOT NULL DEFAULT 0,
    max_downloads INTEGER,      -- NULL = unlimited
    last_downloaded_at TIMESTAMP,

    -- -------------------------------------------------
    -- EXPIRATION
    -- -------------------------------------------------
    expires_at TIMESTAMP,
    -- Default set by application (e.g., 7 days)

    -- -------------------------------------------------
    -- TIMING
    -- -------------------------------------------------
    queued_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,

    -- -------------------------------------------------
    -- WORKER INFO
    -- -------------------------------------------------
    worker_id VARCHAR(100),
    -- Which worker instance processed this

    -- -------------------------------------------------
    -- METADATA
    -- -------------------------------------------------
    metadata JSONB,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- -------------------------------------------------
    -- CONSTRAINTS
    -- -------------------------------------------------
    CONSTRAINT chk_export_number_format
        CHECK (export_number ~ '^EXP-[A-Z0-9]{8}$'),

    CONSTRAINT chk_export_progress_range
        CHECK (progress_percent >= 0 AND progress_percent <= 100),

    CONSTRAINT chk_export_ready_has_file
        CHECK (status <> 'ready' OR file_id IS NOT NULL),

    CONSTRAINT chk_export_email_delivery
        CHECK (
            delivery_method <> 'email' 
            OR delivery_email IS NOT NULL
        )
);

-- =====================================================
-- INDEXES
-- =====================================================

-- Ownership lookups
CREATE INDEX idx_exports_org ON exports(organization_id);
CREATE INDEX idx_exports_user ON exports(requested_by_user_id);

-- Status filtering
CREATE INDEX idx_exports_status ON exports(status);
CREATE INDEX idx_exports_entity ON exports(entity_type);
CREATE INDEX idx_exports_format ON exports(format);

-- Worker queue (pending jobs)
CREATE INDEX idx_exports_queue 
    ON exports(queued_at) 
    WHERE status = 'queued';

-- Retry queue (failed but retryable)
CREATE INDEX idx_exports_retry 
    ON exports(updated_at) 
    WHERE status = 'failed' AND retry_count < max_retries;

-- Cleanup job (expired exports)
CREATE INDEX idx_exports_expired 
    ON exports(expires_at) 
    WHERE status = 'ready' AND expires_at IS NOT NULL;

-- User's recent exports
CREATE INDEX idx_exports_user_recent 
    ON exports(requested_by_user_id, created_at DESC);

-- File references
CREATE INDEX idx_exports_file ON exports(file_id);
CREATE INDEX idx_exports_pdf_file ON exports(pdf_file_id);
CREATE INDEX idx_exports_csv_file ON exports(csv_file_id);

-- =====================================================
-- HELPER VIEW: Downloadable Exports
-- =====================================================

CREATE OR REPLACE VIEW v_exports_downloadable AS
SELECT 
    e.*,
    
    -- Can user download?
    (
        e.status = 'ready'
        AND (e.expires_at IS NULL OR e.expires_at > NOW())
        AND (e.max_downloads IS NULL OR e.download_count < e.max_downloads)
    ) AS is_downloadable,
    
    -- Hours until expiry
    CASE 
        WHEN e.expires_at IS NULL THEN NULL
        ELSE GREATEST(0, EXTRACT(EPOCH FROM (e.expires_at - NOW())) / 3600)
    END AS hours_until_expiry

FROM exports e;

-- =====================================================
-- NOTES
-- =====================================================

-- /*
-- EXPORT FLOW:
-- ────────────────────────────────────────────────────────

-- 1. USER REQUESTS EXPORT (API)
   
--    POST /api/v1/exports
--    {
--      "entity_type": "wallet_statement",
--      "format": "pdf",
--      "params": {
--        "start_date": "2025-01-01",
--        "end_date": "2025-01-31"
--      },
--      "delivery_method": "download"
--    }

--    → INSERT INTO exports (status='queued')
--    → Return export_number immediately
--    → User sees "Preparing export..." in UI


-- 2. GO WORKER PICKS UP JOB

--    SELECT * FROM exports 
--    WHERE status = 'queued' 
--    ORDER BY queued_at 
--    FOR UPDATE SKIP LOCKED 
--    LIMIT 1;

--    → UPDATE status = 'processing', started_at = NOW()


-- 3. WORKER GENERATES FILE(S)

--    Based on entity_type + format:
   
--    PDF only:
--    - Generate PDF → Upload to files table → Set file_id
   
--    CSV only:
--    - Generate CSV → Upload to files table → Set file_id
   
--    ZIP (both):
--    - Generate PDF → Upload → Set pdf_file_id
--    - Generate CSV → Upload → Set csv_file_id
--    - Create ZIP containing both → Upload → Set file_id


-- 4. WORKER COMPLETES

--    UPDATE exports SET
--      status = 'ready',
--      file_id = :file_uuid,
--      file_name = 'wallet-statement-2025-01.pdf',
--      file_size_bytes = 45678,
--      mime_type = 'application/pdf',
--      completed_at = NOW(),
--      duration_ms = 1234,
--      expires_at = NOW() + INTERVAL '7 days'
--    WHERE id = :export_id;


-- 5. USER DOWNLOADS

--    GET /api/v1/exports/:export_number/download
   
--    → Check is_downloadable
--    → Generate signed URL from files table
--    → Increment download_count
--    → Return redirect or file stream


-- 6. CLEANUP JOB (Daily cron)

--    -- Mark expired
--    UPDATE exports 
--    SET status = 'expired', updated_at = NOW()
--    WHERE status = 'ready' 
--      AND expires_at < NOW();

--    -- Delete old files (files table cleanup handles actual deletion)
--    -- Or: DELETE FROM exports WHERE status = 'expired' AND expires_at < NOW() - INTERVAL '30 days';


-- ENTITY TYPE EXAMPLES:
-- ────────────────────────────────────────────────────────

-- invoice (single PDF):
--   params: { "invoice_id": "uuid-here" }
--   output: INV-ABC123.pdf

-- invoices (batch):
--   params: { 
--     "start_date": "2025-01-01", 
--     "end_date": "2025-01-31",
--     "status": "paid"  // optional filter
--   }
--   output ZIP contains:
--     - invoices-summary.csv
--     - invoices/INV-001.pdf
--     - invoices/INV-002.pdf
--     - ...

-- wallet_statement:
--   params: { "start_date": "2025-01-01", "end_date": "2025-01-31" }
--   PDF: Formatted statement with header, transactions, totals
--   CSV: Raw transaction data

-- shipments:
--   params: { 
--     "status": "delivered",
--     "carrier": "fedex",
--     "date_range": { "start": "2025-01-01", "end": "2025-01-31" }
--   }
--   CSV: Shipment list with all details
--   PDF: Formatted report

-- shipment_labels (batch):
--   params: { "shipment_ids": ["uuid1", "uuid2", ...] }
--   output: ZIP of label PDFs

-- commission_statement:
--   params: { "start_date": "2025-01-01", "end_date": "2025-01-31" }
--   PDF: Commission report
--   CSV: Transaction details


-- FORMAT OUTPUTS:
-- ────────────────────────────────────────────────────────

-- format='pdf':
--   file_id → PDF file
--   pdf_file_id = NULL
--   csv_file_id = NULL

-- format='csv':
--   file_id → CSV file
--   pdf_file_id = NULL
--   csv_file_id = NULL

-- format='zip':
--   file_id → ZIP file (primary download)
--   pdf_file_id → PDF inside ZIP (optional individual access)
--   csv_file_id → CSV inside ZIP (optional individual access)

-- format='xlsx':
--   file_id → Excel file
--   (future implementation)


-- MIME TYPES:
-- ────────────────────────────────────────────────────────

-- PDF:  application/pdf
-- CSV:  text/csv
-- ZIP:  application/zip
-- XLSX: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet


-- FILE NAMING:
-- ────────────────────────────────────────────────────────

-- Pattern: {entity}-{identifier}-{date}.{ext}

-- Examples:
--   invoice-INV-ABC123.pdf
--   invoices-2025-01.zip
--   wallet-statement-2025-01.pdf
--   shipments-2025-01-01-to-2025-01-31.csv
--   commission-statement-2025-Q1.pdf
-- */