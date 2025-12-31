-- Seed common support ticket templates

INSERT INTO support_ticket_templates (
    id,
    template_name,
    subject,
    body_template,
    category,
    is_public,
    template_variables,
    usage_count,
    last_used_at,
    is_active,
    created_at,
    updated_at,
    created_by
) VALUES
    (
        uuid_generate_v4(),
        'initial_acknowledgment',
        'We have received your support request',
        'Hi {{customer_name}},\n\nWe have received your ticket {{ticket_number}}. Our team will review it and get back to you shortly.\n\nThanks,\n{{app_name}} Support',
        'general',
        TRUE,
        '["customer_name","ticket_number","app_name"]'::jsonb,
        0,
        NULL,
        TRUE,
        NOW(),
        NOW(),
        '00000000-0000-0000-0000-000000000001'
    ),
    (
        uuid_generate_v4(),
        'ticket_resolved',
        'Your support ticket {{ticket_number}} has been resolved',
        'Hi {{customer_name}},\n\nYour ticket {{ticket_number}} has been marked as resolved. If you feel this has not been fully addressed, simply reply to this email and we will reopen it.\n\nThanks,\n{{app_name}} Support',
        'general',
        TRUE,
        '["customer_name","ticket_number","app_name"]'::jsonb,
        0,
        NULL,
        TRUE,
        NOW(),
        NOW(),
        '00000000-0000-0000-0000-000000000001'
    );
