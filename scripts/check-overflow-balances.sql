SELECT
  COUNT(*) AS users_with_overflow,
  COALESCE(SUM(overflow_reward), 0) AS overflow_reward_total,
  COALESCE(SUM(overflow_direct), 0) AS overflow_direct_total
FROM users
WHERE overflow_reward > 0 OR overflow_direct > 0;

SELECT type, COUNT(*) AS cnt, COALESCE(SUM(amount), 0) AS amt
FROM reward_logs
GROUP BY type
ORDER BY cnt DESC;

SELECT status, COUNT(*) AS cnt FROM orders GROUP BY status;

SELECT COUNT(*) AS users_all_orders_exited
FROM users u
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.status = 'active');
