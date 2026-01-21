-- =====================================================
-- 026_2_add_carrier_rates_json.sql
-- Add carrier_rates_json column to shipment_estimates
-- =====================================================

-- Add carrier_rates_json column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'shipment_estimates'
        AND column_name = 'carrier_rates_json'
    ) THEN
        ALTER TABLE shipment_estimates
        ADD COLUMN carrier_rates_json JSONB DEFAULT '[]'::jsonb;
    END IF;
END $$;
