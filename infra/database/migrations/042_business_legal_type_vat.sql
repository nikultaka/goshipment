-- Migration: Add new business types and VAT field to businesses table
-- Description: Adds corporation, llc, sole_proprietor business types and vat_number field

-- Add new values to business_type_enum
-- Note: PostgreSQL doesn't allow removing enum values, so we keep existing ones for backward compatibility
DO $$
BEGIN
    -- Add new business type values if they don't exist
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'corporation' AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'business_type_enum')) THEN
        ALTER TYPE business_type_enum ADD VALUE 'corporation';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'llc' AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'business_type_enum')) THEN
        ALTER TYPE business_type_enum ADD VALUE 'llc';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'sole_proprietor' AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'business_type_enum')) THEN
        ALTER TYPE business_type_enum ADD VALUE 'sole_proprietor';
    END IF;
END$$;

-- Add vat_number column to businesses table
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS vat_number VARCHAR(50);

-- Add comment for documentation
COMMENT ON COLUMN businesses.vat_number IS 'VAT registration number for the business';
