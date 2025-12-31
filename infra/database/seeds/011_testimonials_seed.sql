-- =====================================================
-- SEED: TESTIMONIALS MODULE
-- =====================================================

INSERT INTO testimonials (
    id,
    external_id,
    author_name,
    author_title,
    author_company,
    avatar_file_id,
    testimonial_text,
    rating,
    is_featured,
    display_order,
    is_active,
    created_at,
    updated_at
) VALUES
(
    uuid_generate_v4(),
    'tml_0001',
    'Sarah Thompson',
    'CTO',
    'BlueTech Innovations',
    NULL,
    'The API is incredibly well-architected. The documentation and OAuth implementation are world-class.',
    5.0,
    TRUE,
    1,
    TRUE,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'tml_0002',
    'Michael Reyes',
    'Founder',
    'ParcelPro Logistics',
    NULL,
    'Integrating with this system saved us thousands of dollars. The webhook handling and PDF generator are flawless.',
    4.5,
    TRUE,
    2,
    TRUE,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'tml_0003',
    'Alicia Carter',
    'Project Manager',
    'NovaPrint Studios',
    NULL,
    'The design module and storage system were exactly what our team needed. Fast, stable, and secure.',
    4.0,
    FALSE,
    3,
    TRUE,
    NOW(),
    NOW()
);
