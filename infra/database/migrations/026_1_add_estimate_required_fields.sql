-- =====================================================
-- 026_1_add_estimate_required_fields.sql
-- Add required fields for Shipment Estimates API
-- =====================================================

-- Add estimate number for human-readable reference
ALTER TABLE shipment_estimates
ADD COLUMN estimate_number VARCHAR(20) UNIQUE;

-- Add carrier rates snapshot (JSONB)
ALTER TABLE shipment_estimates
ADD COLUMN carrier_rates_snapshot JSONB;

-- Add check constraint for estimate number format
ALTER TABLE shipment_estimates
ADD CONSTRAINT chk_estimate_number_format
    CHECK (estimate_number IS NULL OR estimate_number ~ '^EST-[A-Z0-9]{8}$');

-- Add index for estimate number lookups
CREATE INDEX idx_estimate_number ON shipment_estimates(estimate_number);

-- Add GIN index for JSONB queries on rates
CREATE INDEX idx_carrier_rates_snapshot_gin ON shipment_estimates USING gin(carrier_rates_snapshot);

-- Add comments
COMMENT ON COLUMN shipment_estimates.estimate_number IS 'Format: EST-{8 alphanumeric}, e.g., EST-A1B2C3D4';
COMMENT ON COLUMN shipment_estimates.carrier_rates_snapshot IS 'JSONB snapshot of all carrier rate quotes returned at estimate creation time';
