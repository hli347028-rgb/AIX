SELECT type, COUNT(*) AS cnt, MIN(created_time) AS first_at, MAX(created_time) AS last_at
FROM reward_logs
WHERE type IN ('zero_account', 'community_subsidy')
GROUP BY type;
