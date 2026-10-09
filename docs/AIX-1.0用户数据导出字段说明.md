# AIX 1.0 用户数据导出字段说明

来源模板：`1.0数据导出模板.csv`。一行一个用户。金额按实际小数导出，不乘出局倍数。

| 模板列 | 口径 | 数据来源 |
|---|---|---|
| 会员ID | 用户编号 | `users.id` |
| 钱包地址 | 用户钱包地址 | `users.address` |
| 上级钱包地址 | 邀请人钱包地址。没有上级时留空 | `users.inviter_id` 对应的 `users.address` |
| 用户投资额 | 全部订单的认购本金合计，含已出局订单 | `orders.principal` 按 `user_id` 求和 |
| 待领取资产 | 溢出奖励，管理奖溢出加直推奖溢出 | `users.overflow_reward` + `users.overflow_direct` |
| 算力 | 当前进行中订单的认购本金合计。已出局订单不计入 | `orders.principal`，且 `status = active` |
| 已产收益 | 已计入出局的收益合计 | `orders.earned_total` 按 `user_id` 求和 |
| 待产收益 | 剩余出局额度。出局上限减去已计入出局的收益，负数按 0 | `orders.exit_cap - orders.earned_total` |
| 卖出收益 | 已转账的 USDT 提现合计 | `withdrawals.amount`，`asset = USDT` 且 `status = completed` |
| AIX余额 | 当前 AIX 代币余额 | `users.aix_balance` |
| USDT余额 | 充值余额、奖励余额、可提 U 三项合计 | `usdt_recharge` + `usdt_reward` + `usdt_withdrawable` |
| WIN余额 | 充值钱包与提现钱包合计 | `win_recharge_balance` + `win_balance` |
| AIX-USDT余额 | 当前 AIX-USDT | `users.points` |
| 团队业绩 | 累计团队业绩 | `users.team_perf` |
| 等级 | 管理等级，写成 A0–A10 | `users.mgmt_level`，0 为 A0 |
| 注册时间 | 用户注册时间 | `users.created_time` |

补充约定：

- 用户投资额和算力都用认购本金，不用 4 倍出局上限。
- 卖出收益只含已完成的 USDT 提现。AIX-USDT 提现、WIN 提现，以及处理中、待审核的 USDT 提现不计入。
- 待领取资产只含尚未释放进奖励余额的溢出，不含奖励余额本身。
- 等级按现有 A 级导出，不写成模板示例里的 V1、V3、V5。
