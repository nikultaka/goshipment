-- Seed API versions

INSERT INTO api_versions (
    version,
    status,
    is_default,
    deprecation_date,
    sunset_date,
    changelog,
    documentation_url,
    created_at,
    updated_at
) VALUES
    (
        'v1',
        'stable',
        TRUE,
        NULL,
        NULL,
        'Initial stable ParcelHub / ShippingGo API.',
        'https://api.example.com/docs/v1',
        NOW(),
        NOW()
    ),
    (
        'v2',
        'development',
        FALSE,
        NULL,
        NULL,
        'Next-generation API (in development).',
        'https://api.example.com/docs/v2',
        NOW(),
        NOW()
    );
