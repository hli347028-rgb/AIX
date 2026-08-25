SHOW COLUMNS FROM users LIKE '%_set_at';
SELECT id, is_zero_account, zero_account_set_at, is_community_subsidy, community_subsidy_set_at
FROM users
WHERE is_zero_account = 1 OR is_community_subsidy = 1
ORDER BY id
LIMIT 5;
