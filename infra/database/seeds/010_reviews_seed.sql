-- =====================================================
-- SEED: REVIEWS MODULE
-- =====================================================

-- ----------------------------------------------
-- 1. Insert Sample Reviews
-- ----------------------------------------------

INSERT INTO reviews (
    id, external_id, user_id,
    reviewer_name, reviewer_email,
    rating, review_title, review_body,
    target_type, target_id,
    is_public, status, created_at
) VALUES
(
    uuid_generate_v4(), 'rev_0001',
    '00000000-0000-0000-0000-000000000002',          -- demo user
    'Demo User', 'user@example.com',
    4.5,
    'Great platform!',
    'The API is fast, clean, and well-documented. Loving the experience so far.',
    'platform', 'core',
    TRUE, 'approved', NOW()
),
(
    uuid_generate_v4(), 'rev_0002',
    '00000000-0000-0000-0000-000000000001',          -- admin user
    'System Admin', 'admin@example.com',
    5.0,
    'Exceptional Support',
    'Support tickets were handled quickly and professionally.',
    'support_ticket', 'SUP-2024-0001',
    TRUE, 'approved', NOW()
),
(
    uuid_generate_v4(), 'rev_0003',
    '00000000-0000-0000-0000-000000000002',          -- demo user
    'Demo User', 'user@example.com',
    3.5,
    'Good but could be better',
    'Features are solid but I would like to see more integrations.',
    'platform', 'core',
    TRUE, 'approved', NOW()
);

-- ----------------------------------------------
-- 2. Helpful Votes
-- ----------------------------------------------

INSERT INTO review_helpfulness (id, review_id, user_id, vote)
SELECT uuid_generate_v4(), r.id, '00000000-0000-0000-0000-000000000001', true
FROM reviews r WHERE external_id = 'rev_0001';

INSERT INTO review_helpfulness (id, review_id, user_id, vote)
SELECT uuid_generate_v4(), r.id, '00000000-0000-0000-0000-000000000002', false
FROM reviews r WHERE external_id = 'rev_0003';

-- ----------------------------------------------
-- 3. Flags
-- ----------------------------------------------

INSERT INTO review_flags (id, review_id, flagged_by, flag_type, flag_reason)
SELECT uuid_generate_v4(), r.id, '00000000-0000-0000-0000-000000000001',
       'irrelevant', 'Flagging as test moderation action'
FROM reviews r WHERE external_id = 'rev_0003';

-- ----------------------------------------------
-- 4. Admin Responses
-- ----------------------------------------------

INSERT INTO review_responses (
    id, review_id, responder_id, responder_role, response_body, is_public
)
SELECT uuid_generate_v4(), r.id,
       '00000000-0000-0000-0000-000000000001',      -- system admin
       'admin',
       'Thank you for your feedback! We are working on additional integrations.',
       TRUE
FROM reviews r WHERE external_id = 'rev_0003';

-- ----------------------------------------------
-- 5. Rating Summary
-- (platform:core → average rating across reviews)
-- ----------------------------------------------

INSERT INTO review_rating_summary (
    id, target_type, target_id, rating_avg, rating_count, rating_total, updated_at
) VALUES (
    uuid_generate_v4(),
    'platform', 'core',
    4.00,         -- (4.5 + 3.5) / 2
    2,
    8,            -- sum of ratings
    NOW()
);

INSERT INTO review_rating_summary (
    id, target_type, target_id, rating_avg, rating_count, rating_total, updated_at
) VALUES (
    uuid_generate_v4(),
    'support_ticket', 'SUP-2024-0001',
    5.00,
    1,
    5,
    NOW()
);
