-- ================================================
-- 027_user_notifications.sql
-- User Notifications
-- 
-- ================================================
-- we'll be using this for in-app notifications
CREATE TABLE user_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Type & content
    type VARCHAR(50) NOT NULL, -- shipment_created, funds_added, pickup_scheduled
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,

    -- Optional linking
    related_resource_type VARCHAR(50), -- shipment, pickup, wallet
    related_resource_id UUID,

    -- State
    is_read BOOLEAN DEFAULT false,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_notifications_user_id_created_at
ON user_notifications (user_id, created_at DESC);