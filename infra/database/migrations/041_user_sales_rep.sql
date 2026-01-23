-- Migration: Add is_sales_rep column to users table
-- Description: Adds a flag to identify users who are sales representatives

-- Add is_sales_rep column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_sales_rep BOOLEAN NOT NULL DEFAULT false;

-- Create index for efficient querying of sales reps
CREATE INDEX IF NOT EXISTS idx_users_is_sales_rep ON users(is_sales_rep) WHERE is_sales_rep = true AND deleted_at IS NULL;

-- Add comment for documentation
COMMENT ON COLUMN users.is_sales_rep IS 'Flag indicating if the user is a sales representative';
