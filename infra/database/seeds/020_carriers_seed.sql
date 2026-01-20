-- =====================================================
-- 020_carriers_seed.sql
-- Carrier Master Data Seed
-- =====================================================
-- API credentials are stored in .env (NOT in database)
-- This file seeds: carriers, carrier_settings, carrier_service_levels,
--                  carrier_package_types, carrier_operating_schedule
-- =====================================================

-- =====================================================
-- CARRIERS
-- =====================================================

INSERT INTO carriers (
    id, code, name, display_name, carrier_type,
    supports_domestic, supports_international, supports_pickup, supports_drop_off,
    supports_returns, supports_labels, supports_tracking, supports_insurance,
    supports_signature, supports_cod,
    api_version, api_base_url,
    is_enabled, is_active, sort_order, color_hex
) VALUES
-- FedEx
(
    '11111111-1111-1111-1111-111111111111',
    'fedex', 'FedEx', 'FedEx', 'parcel',
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, TRUE,
    TRUE, FALSE,
    'v1', 'https://apis.fedex.com',
    TRUE, TRUE, 1, '#4D148C'
),
-- UPS
(
    '22222222-2222-2222-2222-222222222222',
    'ups', 'UPS', 'UPS', 'parcel',
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE,
    'v1', 'https://onlinetools.ups.com',
    TRUE, TRUE, 2, '#351C15'
),
-- Purolator
(
    '33333333-3333-3333-3333-333333333333',
    'purolator', 'Purolator', 'Purolator', 'parcel',
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, TRUE,
    TRUE, FALSE,
    'v2', 'https://webservices.purolator.com',
    TRUE, TRUE, 3, '#D71920'
),
-- DHL Express
(
    '44444444-4444-4444-4444-444444444444',
    'dhl_express', 'DHL Express', 'DHL Express', 'parcel',
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, TRUE,
    TRUE, FALSE,
    'v1', 'https://express.api.dhl.com',
    TRUE, TRUE, 4, '#FFCC00'
),
-- Canada Post
(
    '55555555-5555-5555-5555-555555555555',
    'canada_post', 'Canada Post', 'Canada Post', 'parcel',
    TRUE, TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, TRUE,
    TRUE, FALSE,
    'v2', 'https://soa-gw.canadapost.ca',
    TRUE, TRUE, 5, '#DA291C'
);

-- =====================================================
-- CARRIER SETTINGS
-- =====================================================

INSERT INTO carrier_settings (
    carrier_id,
    min_pickup_lead_hours, max_pickup_days_ahead,
    max_packages_per_shipment, max_package_weight_kg, max_package_length_cm, max_declared_value,
    rate_multiplier, flat_markup_amount, pickup_fee_amount,
    carrier_discount_percent, fuel_surcharge_percent,
    is_active
) VALUES
-- FedEx Settings
(
    '11111111-1111-1111-1111-111111111111',
    2, 14,
    25, 68.00, 274.00, 50000.00,
    1.0000, 0.00, 0.00,
    NULL, 16.00,
    TRUE
),
-- UPS Settings
(
    '22222222-2222-2222-2222-222222222222',
    2, 14,
    25, 68.00, 274.00, 50000.00,
    1.0000, 0.00, 0.00,
    NULL, 15.50,
    TRUE
),
-- Purolator Settings
(
    '33333333-3333-3333-3333-333333333333',
    3, 14,
    20, 31.50, 150.00, 25000.00,
    1.0000, 0.00, 0.00,
    NULL, 14.50,
    TRUE
),
-- DHL Express Settings
(
    '44444444-4444-4444-4444-444444444444',
    4, 10,
    20, 70.00, 300.00, 100000.00,
    1.0000, 0.00, 0.00,
    NULL, 18.00,
    TRUE
),
-- Canada Post Settings
(
    '55555555-5555-5555-5555-555555555555',
    4, 10,
    30, 30.00, 200.00, 5000.00,
    1.0000, 0.00, 0.00,
    NULL, 8.50,
    TRUE
);

-- =====================================================
-- CARRIER SERVICE LEVELS
-- =====================================================

-- FedEx Service Levels
INSERT INTO carrier_service_levels (
    carrier_id, code, carrier_code, name, display_name, category,
    min_transit_days, max_transit_days, delivery_time_description,
    is_domestic, is_international,
    supports_pickup, supports_saturday_delivery, supports_signature, supports_insurance,
    max_weight_kg, max_length_cm, max_declared_value,
    is_enabled, is_active, sort_order
) VALUES
('11111111-1111-1111-1111-111111111111', 'FEDEX_GROUND', 'FEDEX_GROUND', 'FedEx Ground', 'Ground', 'ground',
    3, 7, '3-7 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 1),
('11111111-1111-1111-1111-111111111111', 'FEDEX_EXPRESS_SAVER', 'FEDEX_EXPRESS_SAVER', 'FedEx Express Saver', 'Express Saver', 'express',
    2, 3, '2-3 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 2),
('11111111-1111-1111-1111-111111111111', 'FEDEX_2DAY', 'FEDEX_2_DAY', 'FedEx 2Day', '2Day', 'express',
    2, 2, '2 business days', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 3),
('11111111-1111-1111-1111-111111111111', 'FEDEX_PRIORITY_OVERNIGHT', 'PRIORITY_OVERNIGHT', 'FedEx Priority Overnight', 'Priority Overnight', 'priority',
    1, 1, 'Next business day by 10:30 AM', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 4),
('11111111-1111-1111-1111-111111111111', 'FEDEX_STANDARD_OVERNIGHT', 'STANDARD_OVERNIGHT', 'FedEx Standard Overnight', 'Standard Overnight', 'priority',
    1, 1, 'Next business day by 3:00 PM', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 5),
('11111111-1111-1111-1111-111111111111', 'FEDEX_INTERNATIONAL_PRIORITY', 'INTERNATIONAL_PRIORITY', 'FedEx International Priority', 'International Priority', 'international',
    1, 3, '1-3 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 6),
('11111111-1111-1111-1111-111111111111', 'FEDEX_INTERNATIONAL_ECONOMY', 'INTERNATIONAL_ECONOMY', 'FedEx International Economy', 'International Economy', 'international',
    4, 6, '4-6 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 7);

-- UPS Service Levels
INSERT INTO carrier_service_levels (
    carrier_id, code, carrier_code, name, display_name, category,
    min_transit_days, max_transit_days, delivery_time_description,
    is_domestic, is_international,
    supports_pickup, supports_saturday_delivery, supports_signature, supports_insurance,
    max_weight_kg, max_length_cm, max_declared_value,
    is_enabled, is_active, sort_order
) VALUES
('22222222-2222-2222-2222-222222222222', 'UPS_STANDARD', '11', 'UPS Standard', 'Standard', 'ground',
    3, 7, '3-7 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 1),
('22222222-2222-2222-2222-222222222222', 'UPS_GROUND', '03', 'UPS Ground', 'Ground', 'ground',
    1, 5, '1-5 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 2),
('22222222-2222-2222-2222-222222222222', 'UPS_3DAY_SELECT', '12', 'UPS 3 Day Select', '3 Day Select', 'express',
    3, 3, '3 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 3),
('22222222-2222-2222-2222-222222222222', 'UPS_2ND_DAY_AIR', '02', 'UPS 2nd Day Air', '2nd Day Air', 'express',
    2, 2, '2 business days', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 4),
('22222222-2222-2222-2222-222222222222', 'UPS_NEXT_DAY_AIR_SAVER', '13', 'UPS Next Day Air Saver', 'Next Day Air Saver', 'priority',
    1, 1, 'Next business day by end of day', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 5),
('22222222-2222-2222-2222-222222222222', 'UPS_NEXT_DAY_AIR', '01', 'UPS Next Day Air', 'Next Day Air', 'priority',
    1, 1, 'Next business day by 10:30 AM', TRUE, FALSE,
    TRUE, TRUE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 6),
('22222222-2222-2222-2222-222222222222', 'UPS_WORLDWIDE_EXPRESS', '07', 'UPS Worldwide Express', 'Worldwide Express', 'international',
    1, 3, '1-3 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 7),
('22222222-2222-2222-2222-222222222222', 'UPS_WORLDWIDE_EXPEDITED', '08', 'UPS Worldwide Expedited', 'Worldwide Expedited', 'international',
    2, 5, '2-5 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 68.00, 274.00, 50000.00, TRUE, TRUE, 8);

-- Purolator Service Levels
INSERT INTO carrier_service_levels (
    carrier_id, code, carrier_code, name, display_name, category,
    min_transit_days, max_transit_days, delivery_time_description,
    is_domestic, is_international,
    supports_pickup, supports_saturday_delivery, supports_signature, supports_insurance,
    max_weight_kg, max_length_cm, max_declared_value,
    is_enabled, is_active, sort_order
) VALUES
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_GROUND', 'PurolatorGround', 'Purolator Ground', 'Ground', 'ground',
    3, 7, '3-7 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 31.50, 150.00, 25000.00, TRUE, TRUE, 1),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS', 'PurolatorExpress', 'Purolator Express', 'Express', 'express',
    1, 2, '1-2 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 31.50, 150.00, 25000.00, TRUE, TRUE, 2),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_9AM', 'PurolatorExpress9AM', 'Purolator Express 9AM', 'Express 9AM', 'priority',
    1, 1, 'Next business day by 9:00 AM', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 31.50, 150.00, 25000.00, TRUE, TRUE, 3),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_10_30AM', 'PurolatorExpress10:30AM', 'Purolator Express 10:30AM', 'Express 10:30AM', 'priority',
    1, 1, 'Next business day by 10:30 AM', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 31.50, 150.00, 25000.00, TRUE, TRUE, 4),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_INTL', 'PurolatorExpressInternational', 'Purolator Express International', 'Express International', 'international',
    2, 5, '2-5 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 31.50, 150.00, 25000.00, TRUE, TRUE, 5);

-- DHL Express Service Levels
INSERT INTO carrier_service_levels (
    carrier_id, code, carrier_code, name, display_name, category,
    min_transit_days, max_transit_days, delivery_time_description,
    is_domestic, is_international,
    supports_pickup, supports_saturday_delivery, supports_signature, supports_insurance,
    max_weight_kg, max_length_cm, max_declared_value,
    is_enabled, is_active, sort_order
) VALUES
('44444444-4444-4444-4444-444444444444', 'DHL_EXPRESS_DOMESTIC', 'N', 'DHL Express Domestic', 'Express Domestic', 'express',
    1, 2, '1-2 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 70.00, 300.00, 100000.00, TRUE, TRUE, 1),
('44444444-4444-4444-4444-444444444444', 'DHL_EXPRESS_WORLDWIDE', 'P', 'DHL Express Worldwide', 'Express Worldwide', 'international',
    1, 3, '1-3 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 70.00, 300.00, 100000.00, TRUE, TRUE, 2),
('44444444-4444-4444-4444-444444444444', 'DHL_EXPRESS_12', 'Y', 'DHL Express 12:00', 'Express 12:00', 'priority',
    1, 1, 'Next business day by 12:00 PM', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 70.00, 300.00, 100000.00, TRUE, TRUE, 3),
('44444444-4444-4444-4444-444444444444', 'DHL_EXPRESS_9', 'K', 'DHL Express 9:00', 'Express 9:00', 'priority',
    1, 1, 'Next business day by 9:00 AM', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 70.00, 300.00, 100000.00, TRUE, TRUE, 4),
('44444444-4444-4444-4444-444444444444', 'DHL_ECONOMY_SELECT', 'H', 'DHL Economy Select', 'Economy Select', 'economy',
    4, 8, '4-8 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 70.00, 300.00, 100000.00, TRUE, TRUE, 5);

-- Canada Post Service Levels
INSERT INTO carrier_service_levels (
    carrier_id, code, carrier_code, name, display_name, category,
    min_transit_days, max_transit_days, delivery_time_description,
    is_domestic, is_international,
    supports_pickup, supports_saturday_delivery, supports_signature, supports_insurance,
    max_weight_kg, max_length_cm, max_declared_value,
    is_enabled, is_active, sort_order
) VALUES
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_REGULAR', 'DOM.RP', 'Canada Post Regular Parcel', 'Regular Parcel', 'ground',
    4, 9, '4-9 business days', TRUE, FALSE,
    FALSE, FALSE, FALSE, TRUE, 30.00, 200.00, 5000.00, TRUE, TRUE, 1),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_EXPEDITED', 'DOM.EP', 'Canada Post Expedited Parcel', 'Expedited Parcel', 'express',
    2, 4, '2-4 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 30.00, 200.00, 5000.00, TRUE, TRUE, 2),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_XPRESSPOST', 'DOM.XP', 'Canada Post Xpresspost', 'Xpresspost', 'priority',
    1, 2, '1-2 business days', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 30.00, 200.00, 5000.00, TRUE, TRUE, 3),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_PRIORITY', 'DOM.PC', 'Canada Post Priority', 'Priority', 'priority',
    1, 1, 'Next business day', TRUE, FALSE,
    TRUE, FALSE, TRUE, TRUE, 30.00, 200.00, 5000.00, TRUE, TRUE, 4),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_INTL_TRACKED', 'INT.TP', 'Canada Post Tracked Packet - International', 'International Tracked', 'international',
    6, 12, '6-12 business days', FALSE, TRUE,
    FALSE, FALSE, TRUE, TRUE, 2.00, 90.00, 500.00, TRUE, TRUE, 5),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_INTL_XPRESSPOST', 'INT.XP', 'Canada Post Xpresspost - International', 'International Xpresspost', 'international',
    4, 7, '4-7 business days', FALSE, TRUE,
    TRUE, FALSE, TRUE, TRUE, 30.00, 200.00, 5000.00, TRUE, TRUE, 6);

-- =====================================================
-- CARRIER PACKAGE TYPES
-- =====================================================

-- FedEx Package Types
INSERT INTO carrier_package_types (
    carrier_id, code, carrier_code, name, display_name,
    is_custom, is_carrier_provided,
    length_cm, width_cm, height_cm, max_weight_kg,
    is_enabled, sort_order
) VALUES
('11111111-1111-1111-1111-111111111111', 'YOUR_PACKAGING', 'YOUR_PACKAGING', 'Your Packaging', 'Your Packaging',
    TRUE, FALSE, NULL, NULL, NULL, 68.00, TRUE, 1),
('11111111-1111-1111-1111-111111111111', 'FEDEX_ENVELOPE', 'FEDEX_ENVELOPE', 'FedEx Envelope', 'FedEx Envelope',
    FALSE, TRUE, 32.00, 24.00, 1.00, 0.50, TRUE, 2),
('11111111-1111-1111-1111-111111111111', 'FEDEX_PAK', 'FEDEX_PAK', 'FedEx Pak', 'FedEx Pak',
    FALSE, TRUE, 40.00, 31.00, 5.00, 2.50, TRUE, 3),
('11111111-1111-1111-1111-111111111111', 'FEDEX_BOX', 'FEDEX_BOX', 'FedEx Box', 'FedEx Box',
    FALSE, TRUE, 45.00, 35.00, 30.00, 20.00, TRUE, 4),
('11111111-1111-1111-1111-111111111111', 'FEDEX_TUBE', 'FEDEX_TUBE', 'FedEx Tube', 'FedEx Tube',
    FALSE, TRUE, 96.50, 15.00, 15.00, 9.00, TRUE, 5);

-- UPS Package Types
INSERT INTO carrier_package_types (
    carrier_id, code, carrier_code, name, display_name,
    is_custom, is_carrier_provided,
    length_cm, width_cm, height_cm, max_weight_kg,
    is_enabled, sort_order
) VALUES
('22222222-2222-2222-2222-222222222222', 'YOUR_PACKAGING', '02', 'Your Packaging', 'Your Packaging',
    TRUE, FALSE, NULL, NULL, NULL, 68.00, TRUE, 1),
('22222222-2222-2222-2222-222222222222', 'UPS_LETTER', '01', 'UPS Letter', 'UPS Letter',
    FALSE, TRUE, 32.00, 24.00, 1.00, 0.50, TRUE, 2),
('22222222-2222-2222-2222-222222222222', 'UPS_EXPRESS_PAK', '04', 'UPS Express Pak', 'Express Pak',
    FALSE, TRUE, 40.00, 31.00, 5.00, 2.50, TRUE, 3),
('22222222-2222-2222-2222-222222222222', 'UPS_EXPRESS_BOX_SMALL', '2a', 'UPS Express Box Small', 'Express Box Small',
    FALSE, TRUE, 33.00, 27.00, 5.00, 4.50, TRUE, 4),
('22222222-2222-2222-2222-222222222222', 'UPS_EXPRESS_BOX_MEDIUM', '2b', 'UPS Express Box Medium', 'Express Box Medium',
    FALSE, TRUE, 40.00, 31.00, 8.00, 9.00, TRUE, 5),
('22222222-2222-2222-2222-222222222222', 'UPS_EXPRESS_BOX_LARGE', '2c', 'UPS Express Box Large', 'Express Box Large',
    FALSE, TRUE, 48.00, 36.00, 9.00, 13.50, TRUE, 6),
('22222222-2222-2222-2222-222222222222', 'UPS_TUBE', '03', 'UPS Tube', 'UPS Tube',
    FALSE, TRUE, 96.50, 15.00, 15.00, 9.00, TRUE, 7);

-- Purolator Package Types
INSERT INTO carrier_package_types (
    carrier_id, code, carrier_code, name, display_name,
    is_custom, is_carrier_provided,
    length_cm, width_cm, height_cm, max_weight_kg,
    is_enabled, sort_order
) VALUES
('33333333-3333-3333-3333-333333333333', 'YOUR_PACKAGING', 'CustomerPackaging', 'Your Packaging', 'Your Packaging',
    TRUE, FALSE, NULL, NULL, NULL, 31.50, TRUE, 1),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_ENVELOPE', 'PurolatorExpressEnvelope', 'Purolator Express Envelope', 'Express Envelope',
    FALSE, TRUE, 39.00, 27.00, 1.00, 0.50, TRUE, 2),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_PACK', 'PurolatorExpressPack', 'Purolator Express Pack', 'Express Pack',
    FALSE, TRUE, 42.00, 32.00, 5.00, 3.00, TRUE, 3),
('33333333-3333-3333-3333-333333333333', 'PUROLATOR_EXPRESS_BOX', 'PurolatorExpressBox', 'Purolator Express Box', 'Express Box',
    FALSE, TRUE, 46.00, 35.00, 27.00, 10.00, TRUE, 4);

-- DHL Express Package Types
INSERT INTO carrier_package_types (
    carrier_id, code, carrier_code, name, display_name,
    is_custom, is_carrier_provided,
    length_cm, width_cm, height_cm, max_weight_kg,
    is_enabled, sort_order
) VALUES
('44444444-4444-4444-4444-444444444444', 'YOUR_PACKAGING', 'YP', 'Your Packaging', 'Your Packaging',
    TRUE, FALSE, NULL, NULL, NULL, 70.00, TRUE, 1),
('44444444-4444-4444-4444-444444444444', 'DHL_EXPRESS_ENVELOPE', 'EE', 'DHL Express Envelope', 'Express Envelope',
    FALSE, TRUE, 35.00, 27.50, 1.00, 0.30, TRUE, 2),
('44444444-4444-4444-4444-444444444444', 'DHL_FLYER', 'FLY', 'DHL Flyer', 'DHL Flyer',
    FALSE, TRUE, 40.00, 30.00, 3.00, 2.00, TRUE, 3),
('44444444-4444-4444-4444-444444444444', 'DHL_BOX_2', 'BOX2', 'DHL Box 2', 'DHL Box 2',
    FALSE, TRUE, 33.70, 18.20, 10.00, 1.00, TRUE, 4),
('44444444-4444-4444-4444-444444444444', 'DHL_BOX_3', 'BOX3', 'DHL Box 3', 'DHL Box 3',
    FALSE, TRUE, 33.70, 32.20, 10.00, 2.00, TRUE, 5),
('44444444-4444-4444-4444-444444444444', 'DHL_BOX_5', 'BOX5', 'DHL Box 5', 'DHL Box 5',
    FALSE, TRUE, 33.70, 32.20, 34.50, 10.00, TRUE, 6);

-- Canada Post Package Types
INSERT INTO carrier_package_types (
    carrier_id, code, carrier_code, name, display_name,
    is_custom, is_carrier_provided,
    length_cm, width_cm, height_cm, max_weight_kg,
    is_enabled, sort_order
) VALUES
('55555555-5555-5555-5555-555555555555', 'YOUR_PACKAGING', 'PARCEL', 'Your Packaging', 'Your Packaging',
    TRUE, FALSE, NULL, NULL, NULL, 30.00, TRUE, 1),
('55555555-5555-5555-5555-555555555555', 'CANADA_POST_ENVELOPE', 'ENVELOPE', 'Canada Post Envelope', 'Envelope',
    FALSE, TRUE, 31.75, 24.00, 1.00, 0.50, TRUE, 2),
('55555555-5555-5555-5555-555555555555', 'MAILING_BOX_CUBE', 'MAILING_BOX', 'Mailing Box - Cube', 'Mailing Box Cube',
    FALSE, TRUE, 15.00, 15.00, 15.00, 5.00, TRUE, 3);

-- =====================================================
-- CARRIER OPERATING SCHEDULE
-- =====================================================
-- Standard weekday operating hours for all carriers

-- FedEx Operating Schedule
INSERT INTO carrier_operating_schedule (carrier_id, day_of_week, is_operating, pickup_start_time, pickup_end_time, same_day_cutoff_time, next_day_cutoff_time) VALUES
('11111111-1111-1111-1111-111111111111', 'monday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('11111111-1111-1111-1111-111111111111', 'tuesday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('11111111-1111-1111-1111-111111111111', 'wednesday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('11111111-1111-1111-1111-111111111111', 'thursday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('11111111-1111-1111-1111-111111111111', 'friday', TRUE, '08:00', '17:00', '13:00', '16:00'),
('11111111-1111-1111-1111-111111111111', 'saturday', TRUE, '09:00', '14:00', NULL, NULL),
('11111111-1111-1111-1111-111111111111', 'sunday', FALSE, NULL, NULL, NULL, NULL);

-- UPS Operating Schedule
INSERT INTO carrier_operating_schedule (carrier_id, day_of_week, is_operating, pickup_start_time, pickup_end_time, same_day_cutoff_time, next_day_cutoff_time) VALUES
('22222222-2222-2222-2222-222222222222', 'monday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('22222222-2222-2222-2222-222222222222', 'tuesday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('22222222-2222-2222-2222-222222222222', 'wednesday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('22222222-2222-2222-2222-222222222222', 'thursday', TRUE, '08:00', '18:00', '14:00', '17:00'),
('22222222-2222-2222-2222-222222222222', 'friday', TRUE, '08:00', '17:00', '13:00', '16:00'),
('22222222-2222-2222-2222-222222222222', 'saturday', TRUE, '09:00', '14:00', NULL, NULL),
('22222222-2222-2222-2222-222222222222', 'sunday', FALSE, NULL, NULL, NULL, NULL);

-- Purolator Operating Schedule
INSERT INTO carrier_operating_schedule (carrier_id, day_of_week, is_operating, pickup_start_time, pickup_end_time, same_day_cutoff_time, next_day_cutoff_time) VALUES
('33333333-3333-3333-3333-333333333333', 'monday', TRUE, '08:30', '17:30', '13:00', '16:00'),
('33333333-3333-3333-3333-333333333333', 'tuesday', TRUE, '08:30', '17:30', '13:00', '16:00'),
('33333333-3333-3333-3333-333333333333', 'wednesday', TRUE, '08:30', '17:30', '13:00', '16:00'),
('33333333-3333-3333-3333-333333333333', 'thursday', TRUE, '08:30', '17:30', '13:00', '16:00'),
('33333333-3333-3333-3333-333333333333', 'friday', TRUE, '08:30', '17:00', '12:00', '15:00'),
('33333333-3333-3333-3333-333333333333', 'saturday', FALSE, NULL, NULL, NULL, NULL),
('33333333-3333-3333-3333-333333333333', 'sunday', FALSE, NULL, NULL, NULL, NULL);

-- DHL Express Operating Schedule
INSERT INTO carrier_operating_schedule (carrier_id, day_of_week, is_operating, pickup_start_time, pickup_end_time, same_day_cutoff_time, next_day_cutoff_time) VALUES
('44444444-4444-4444-4444-444444444444', 'monday', TRUE, '09:00', '18:00', '13:00', '16:00'),
('44444444-4444-4444-4444-444444444444', 'tuesday', TRUE, '09:00', '18:00', '13:00', '16:00'),
('44444444-4444-4444-4444-444444444444', 'wednesday', TRUE, '09:00', '18:00', '13:00', '16:00'),
('44444444-4444-4444-4444-444444444444', 'thursday', TRUE, '09:00', '18:00', '13:00', '16:00'),
('44444444-4444-4444-4444-444444444444', 'friday', TRUE, '09:00', '17:00', '12:00', '15:00'),
('44444444-4444-4444-4444-444444444444', 'saturday', FALSE, NULL, NULL, NULL, NULL),
('44444444-4444-4444-4444-444444444444', 'sunday', FALSE, NULL, NULL, NULL, NULL);

-- Canada Post Operating Schedule
INSERT INTO carrier_operating_schedule (carrier_id, day_of_week, is_operating, pickup_start_time, pickup_end_time, same_day_cutoff_time, next_day_cutoff_time) VALUES
('55555555-5555-5555-5555-555555555555', 'monday', TRUE, '08:00', '17:00', '12:00', '15:00'),
('55555555-5555-5555-5555-555555555555', 'tuesday', TRUE, '08:00', '17:00', '12:00', '15:00'),
('55555555-5555-5555-5555-555555555555', 'wednesday', TRUE, '08:00', '17:00', '12:00', '15:00'),
('55555555-5555-5555-5555-555555555555', 'thursday', TRUE, '08:00', '17:00', '12:00', '15:00'),
('55555555-5555-5555-5555-555555555555', 'friday', TRUE, '08:00', '17:00', '12:00', '15:00'),
('55555555-5555-5555-5555-555555555555', 'saturday', FALSE, NULL, NULL, NULL, NULL),
('55555555-5555-5555-5555-555555555555', 'sunday', FALSE, NULL, NULL, NULL, NULL);

-- =====================================================
-- CARRIER SURCHARGES
-- =====================================================

-- FedEx Surcharges
INSERT INTO carrier_surcharges (carrier_id, code, name, description, surcharge_type, amount, percent, applies_to_domestic, applies_to_international, is_mandatory, is_active) VALUES
('11111111-1111-1111-1111-111111111111', 'FUEL', 'Fuel Surcharge', 'Variable fuel surcharge', 'percent', NULL, 16.00, TRUE, TRUE, TRUE, TRUE),
('11111111-1111-1111-1111-111111111111', 'RESIDENTIAL', 'Residential Delivery', 'Delivery to residential address', 'fixed', 5.30, NULL, TRUE, FALSE, FALSE, TRUE),
('11111111-1111-1111-1111-111111111111', 'SIGNATURE', 'Signature Required', 'Adult signature required', 'fixed', 6.75, NULL, TRUE, TRUE, FALSE, TRUE);

-- UPS Surcharges
INSERT INTO carrier_surcharges (carrier_id, code, name, description, surcharge_type, amount, percent, applies_to_domestic, applies_to_international, is_mandatory, is_active) VALUES
('22222222-2222-2222-2222-222222222222', 'FUEL', 'Fuel Surcharge', 'Variable fuel surcharge', 'percent', NULL, 15.50, TRUE, TRUE, TRUE, TRUE),
('22222222-2222-2222-2222-222222222222', 'RESIDENTIAL', 'Residential Delivery', 'Delivery to residential address', 'fixed', 5.15, NULL, TRUE, FALSE, FALSE, TRUE),
('22222222-2222-2222-2222-222222222222', 'SIGNATURE', 'Signature Required', 'Adult signature required', 'fixed', 6.50, NULL, TRUE, TRUE, FALSE, TRUE);

-- Purolator Surcharges
INSERT INTO carrier_surcharges (carrier_id, code, name, description, surcharge_type, amount, percent, applies_to_domestic, applies_to_international, is_mandatory, is_active) VALUES
('33333333-3333-3333-3333-333333333333', 'FUEL', 'Fuel Surcharge', 'Variable fuel surcharge', 'percent', NULL, 14.50, TRUE, TRUE, TRUE, TRUE),
('33333333-3333-3333-3333-333333333333', 'RESIDENTIAL', 'Residential Delivery', 'Delivery to residential address', 'fixed', 4.50, NULL, TRUE, FALSE, FALSE, TRUE);

-- DHL Express Surcharges
INSERT INTO carrier_surcharges (carrier_id, code, name, description, surcharge_type, amount, percent, applies_to_domestic, applies_to_international, is_mandatory, is_active) VALUES
('44444444-4444-4444-4444-444444444444', 'FUEL', 'Fuel Surcharge', 'Variable fuel surcharge', 'percent', NULL, 18.00, TRUE, TRUE, TRUE, TRUE),
('44444444-4444-4444-4444-444444444444', 'REMOTE_AREA', 'Remote Area Surcharge', 'Delivery to remote area', 'fixed', 25.00, NULL, TRUE, TRUE, FALSE, TRUE);

-- Canada Post Surcharges
INSERT INTO carrier_surcharges (carrier_id, code, name, description, surcharge_type, amount, percent, applies_to_domestic, applies_to_international, is_mandatory, is_active) VALUES
('55555555-5555-5555-5555-555555555555', 'FUEL', 'Fuel Surcharge', 'Variable fuel surcharge', 'percent', NULL, 8.50, TRUE, TRUE, TRUE, TRUE),
('55555555-5555-5555-5555-555555555555', 'SIGNATURE', 'Signature Required', 'Signature on delivery', 'fixed', 3.50, NULL, TRUE, TRUE, FALSE, TRUE);
