SET @addr = '0x1ce5843b7dd5ee65ea7c249e78b3f4cd1b1c72a8';
SET @root_id = (
  SELECT id FROM users
  WHERE address COLLATE utf8mb4_general_ci = @addr COLLATE utf8mb4_general_ci
  LIMIT 1
);

SELECT @root_id AS root_user_id, @addr AS root_address;

WITH RECURSIVE downline AS (
  SELECT id FROM users WHERE inviter_id = @root_id
  UNION ALL
  SELECT u.id FROM users u INNER JOIN downline d ON u.inviter_id = d.id
)
SELECT COUNT(*) AS downline_count FROM downline;

-- 8月24日（中国时区 UTC+8）
WITH RECURSIVE downline AS (
  SELECT id FROM users WHERE inviter_id = @root_id
  UNION ALL
  SELECT u.id FROM users u INNER JOIN downline d ON u.inviter_id = d.id
)
SELECT
  'china_2026-08-24' AS period,
  COALESCE(SUM(r.amount), 0) AS total_usdt,
  COUNT(*) AS recharge_count
FROM recharges r
WHERE r.user_id IN (SELECT id FROM downline)
  AND UPPER(r.asset) = 'USDT'
  AND r.status = 'confirmed'
  AND COALESCE(r.confirmed_time, r.created_time) >= '2026-08-23 16:00:00'
  AND COALESCE(r.confirmed_time, r.created_time) < '2026-08-24 16:00:00';

WITH RECURSIVE downline AS (
  SELECT id, address FROM users WHERE inviter_id = @root_id
  UNION ALL
  SELECT u.id, u.address FROM users u INNER JOIN downline d ON u.inviter_id = d.id
)
SELECT
  d.address,
  r.amount,
  r.tx_hash,
  COALESCE(r.confirmed_time, r.created_time) AS recharge_time
FROM recharges r
JOIN downline d ON d.id = r.user_id
WHERE UPPER(r.asset) = 'USDT'
  AND r.status = 'confirmed'
  AND COALESCE(r.confirmed_time, r.created_time) >= '2026-08-23 16:00:00'
  AND COALESCE(r.confirmed_time, r.created_time) < '2026-08-24 16:00:00'
ORDER BY recharge_time, r.id;
