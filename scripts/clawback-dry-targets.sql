SELECT COALESCE(SUM(m.total_amount),0) AS total_due
FROM mgmt_rewards m
JOIN orders o ON o.id = m.source_order_id
WHERE o.fund_source = 'reward';

SELECT u.id, u.address, SUM(m.total_amount) AS due,
       u.usdt_reward, u.overflow_reward, u.points, u.points_all
FROM mgmt_rewards m
JOIN orders o ON o.id = m.source_order_id
JOIN users u ON u.id = m.user_id
WHERE o.fund_source = 'reward'
GROUP BY u.id, u.address, u.usdt_reward, u.overflow_reward, u.points, u.points_all
ORDER BY due DESC;
