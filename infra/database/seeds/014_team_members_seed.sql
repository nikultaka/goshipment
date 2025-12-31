-- Seed team members for admin, core teams, and memberships

-- Team member for admin user
INSERT INTO team_members (
    id,
    user_id,
    display_name,
    job_title,
    department,
    team,
    work_email,
    work_phone,
    office_location,
    profile_picture_file_id,
    timezone,
    work_hours_start,
    work_hours_end,
    work_days,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000001',
    'System Admin',
    'Founder / Administrator',
    'Management',
    'Management',
    'admin@example.com',
    NULL,
    'Remote',
    NULL,
    'America/Toronto',
    '09:00',
    '17:00',
    ARRAY[1,2,3,4,5],
    TRUE,
    NOW(),
    NOW()
);

-- Teams
INSERT INTO teams (
    id,
    team_name,
    description,
    team_lead_id,
    is_active,
    created_at,
    updated_at
) VALUES
    (
        '00000000-0000-0000-0000-000000000501',
        'Management',
        'Leadership and administrative decisions.',
        '00000000-0000-0000-0000-000000000401',
        TRUE,
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0000-000000000502',
        'Engineering',
        'Backend, frontend, and infrastructure engineers.',
        '00000000-0000-0000-0000-000000000401',
        TRUE,
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0000-000000000503',
        'Support',
        'Customer support and ticket handling.',
        '00000000-0000-0000-0000-000000000401',
        TRUE,
        NOW(),
        NOW()
    );

-- Team memberships (admin in Management as lead)
INSERT INTO team_memberships (
    id,
    team_id,
    team_member_id,
    membership_type,
    is_primary_team,
    joined_team_at,
    is_active
) VALUES (
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000401',
    'lead',
    TRUE,
    CURRENT_DATE,
    TRUE
);

-- Admin availability today as "available"
INSERT INTO team_member_availability (
    id,
    team_member_id,
    date,
    status,
    is_all_day,
    created_at,
    updated_at
) VALUES (
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000401',
    CURRENT_DATE,
    'available',
    TRUE,
    NOW(),
    NOW()
);
