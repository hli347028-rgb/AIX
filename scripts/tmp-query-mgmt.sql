SELECT rl.id, rl.type, rl.amount, rl.base_amount, rl.rate, rl.order_id,
  u.address AS beneficiary, fu.address AS source_addr, u.mgmt_level,
  rl.created_time
FROM reward_logs rl
JOIN users u ON u.id = rl.user_id
LEFT JOIN users fu ON fu.id = rl.from_user_id
WHERE fu.address = '0xbcff838406f650b5e195d45f9cf4ea50a21fee5e'
  AND rl.type IN ('mgmt', 'mgmt_pool_release')
ORDER BY rl.id DESC
LIMIT 10;

SELECT u.address, u.mgmt_level, u.inviter_id, inv.address AS inviter_addr
FROM users u
LEFT JOIN users inv ON inv.id = u.inviter_id
WHERE u.address IN (
  '0xbcff838406f650b5e195d45f9cf4ea50a21fee5e',
  '0xdb5f8a5fc4557e35e5edbdc50b332f2fb87cb342',
  '0x4af454317e5cd97853daaa2940b3cc1e6cfbc74e'
);
