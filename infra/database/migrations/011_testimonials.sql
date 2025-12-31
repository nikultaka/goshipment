-- =====================================================
-- MODULE 11: TESTIMONIALS
-- =====================================================

CREATE TABLE testimonials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(50) UNIQUE NOT NULL,    -- tml_xxx

    -- Author
    author_name VARCHAR(255) NOT NULL,
    author_title VARCHAR(255),
    author_company VARCHAR(255),
    avatar_file_id UUID REFERENCES files(id),

    -- Display Content
    testimonial_text TEXT NOT NULL,

    -- Optional rating (0.5 increments)
    rating DECIMAL(2,1)
        CHECK (
            rating IS NULL OR (
                rating >= 1.0 AND 
                rating <= 5.0 AND 
                (rating * 2) = FLOOR(rating * 2)
            )
        ),

    -- Display Settings
    is_featured BOOLEAN DEFAULT false,
    display_order INTEGER DEFAULT 0,

    -- Visibility
    is_active BOOLEAN DEFAULT true,
    visible_from TIMESTAMP,
    visible_until TIMESTAMP,

    -- Lifecycle
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

