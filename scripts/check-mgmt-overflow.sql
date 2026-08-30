SELECT COUNT(*) AS overflow_rows FROM reward_logs WHERE type = 'mgmt_overflow';

SELECT
  COUNT(*) AS total_count,
  COALESCE(SUM(rl.amount), 0) AS amount_total,
  COALESCE(SUM(CASE WHEN NOT EXISTS (
    SELECT 1 FROM reward_logs m
    WHERE m.user_id = rl.user_id AND m.order_id = rl.order_id AND m.type = 'mgmt'
  ) THEN 1 ELSE 0 END), 0) AS fully_exited_count,
  COALESCE(SUM(CASE WHEN NOT EXISTS (
    SELECT 1 FROM reward_logs m
    WHERE m.user_id = rl.user_id AND m.order_id = rl.order_id AND m.type = 'mgmt'
  ) THEN rl.amount ELSE 0 END), 0) AS fully_exited_total
FROM reward_logs rl
JOIN users u ON u.id = rl.user_id
LEFT JOIN users fu ON fu.id = rl.from_user_id
WHERE rl.type = 'mgmt_overflow';

SELECT
  rl.id,
  u.address,
  COALESCE(fu.address, '') AS from_address,
  rl.amount,
  rl.order_id,
  NOT EXISTS (
    SELECT 1 FROM reward_logs m
    WHERE m.user_id = rl.user_id AND m.order_id = rl.order_id AND m.type = 'mgmt'
  ) AS fully_exited,
  rl.created_time
FROM reward_logs rl
JOIN users u ON u.id = rl.user_id
LEFT JOIN users fu ON fu.id = rl.from_user_id
WHERE rl.type = 'mgmt_overflow'
ORDER BY rl.id DESC
LIMIT 10;
