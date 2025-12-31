-- =====================================================
-- MODULE 10: REVIEWS
-- =====================================================

-- ============================================
-- 1. reviews
-- ============================================
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL,     -- rev_xxx

    -- Reviewer
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewer_name VARCHAR(255),
    reviewer_email VARCHAR(255),

    -- Rating (0.5 increments)
    rating DECIMAL(2,1) NOT NULL
        CHECK (
            rating >= 1.0 AND 
            rating <= 5.0 AND 
            (rating * 2) = FLOOR(rating * 2)
        ),

    -- Content
    review_title VARCHAR(255),
    review_body TEXT NOT NULL,
    pros TEXT,
    cons TEXT,

    -- Optional media references
    photo_ids UUID[],
    video_ids UUID[],

    -- Review Type
    review_type VARCHAR(20) DEFAULT 'review',
    -- review, testimonial, feedback, comment

    -- Polymorphic target (product, business, shipment, team_member, etc.)
    target_type VARCHAR(50) NOT NULL,
    target_id VARCHAR(100) NOT NULL,

    -- Visibility & status
    is_public BOOLEAN DEFAULT false,
    visible_from TIMESTAMP,
    visible_until TIMESTAMP,

    status VARCHAR(20) DEFAULT 'pending',
    -- pending, approved, rejected, flagged, removed

    moderation_reason VARCHAR(255),
    moderated_by UUID REFERENCES users(id),
    moderated_at TIMESTAMP,

    -- Flags & analytics
    flagged_count INTEGER DEFAULT 0,
    last_flagged_at TIMESTAMP,

    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,

    view_count INTEGER DEFAULT 0,

    -- Lifecycle
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);



-- ============================================
-- 2. review_helpfulness
-- ============================================
CREATE TABLE review_helpfulness (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote BOOLEAN NOT NULL,      -- true = helpful, false = not helpful
    voted_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(review_id, user_id)
);

-- ============================================
-- 3. review_flags
-- ============================================
CREATE TABLE review_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    flagged_by UUID REFERENCES users(id),

    flag_type VARCHAR(50) NOT NULL,
    -- spam, abuse, fake, profanity, irrelevant, discrimination

    flag_reason TEXT,
    resolved BOOLEAN DEFAULT false,
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- 4. review_responses
-- ============================================
CREATE TABLE review_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,

    responder_id UUID REFERENCES users(id),
    responder_role VARCHAR(50),       -- admin, support, business_owner

    response_body TEXT NOT NULL,
    is_public BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- 5. review_rating_summary
-- ============================================
CREATE TABLE review_rating_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    target_type VARCHAR(50) NOT NULL,
    target_id VARCHAR(100) NOT NULL,

    rating_avg DECIMAL(3,2) DEFAULT 0.00,
    rating_count INTEGER DEFAULT 0,
    rating_total INTEGER DEFAULT 0,

    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(target_type, target_id)
);
