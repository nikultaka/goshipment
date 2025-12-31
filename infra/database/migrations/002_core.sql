-- =====================================================
-- MODULE 2: CORE
-- =====================================================

-- ============================
-- 1. API VERSIONING
-- ============================
-- Used only for your dashboard and internal structure.
-- Allows showing warnings about deprecated API versions.

CREATE TABLE api_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version VARCHAR(20) UNIQUE NOT NULL,          -- v1, v2

    status VARCHAR(20) DEFAULT 'stable',          -- development, beta, stable, deprecated
    is_default BOOLEAN DEFAULT FALSE,

    deprecation_date DATE,
    sunset_date DATE,

    changelog TEXT,
    documentation_url VARCHAR(500),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
