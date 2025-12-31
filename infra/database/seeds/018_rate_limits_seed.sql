-- 018_rate_limits_seed.sql
INSERT INTO rate_limits (target_type, target_id, capacity, refill_rate)
VALUES 
('user', 'default', 100, 150);

INSERT INTO rate_limits (target_type, target_id, capacity, refill_rate)
VALUES 
('client', 'web', 1000, 1000);

INSERT INTO rate_limits (target_type, target_id, capacity, refill_rate)
VALUES 
('client', 'android', 1000, 1000),
('client', 'ios', 1000, 1000);

INSERT INTO rate_limits (target_type, target_id, capacity, refill_rate)
VALUES
('client', 'partner_shopify', 300, 300),
('client', 'partner_etsy', 300, 300);

INSERT INTO rate_limits (target_type, target_id, capacity, refill_rate)
VALUES
('client', 'system', 5000, 5000),
('client', 'worker', 5000, 5000);

