SELECT
  u.id,
  u.address,
  u.is_zero_account,
  u.is_community_subsidy,
  u.created_time,
  u.updated_time,
  (SELECT MIN(rl.created_time) FROM reward_logs rl WHERE rl.user_id = u.id AND rl.type = 'zero_account') AS first_zero_reward,
  (SELECT MIN(rl.created_time) FROM reward_logs rl WHERE rl.user_id = u.id AND rl.type = 'community_subsidy') AS first_subsidy_reward
FROM users u
WHERE u.is_zero_account = 1 OR u.is_community_subsidy = 1
ORDER BY u.id;
