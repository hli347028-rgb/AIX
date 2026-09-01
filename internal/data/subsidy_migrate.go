package data

import (
	"backend/internal/biz"

	"gorm.io/gorm"
)

// migrateZeroAccountIntoSubsidy 删除 0 号账户时，将原 10% 并入社区补贴档位。
// 仅 0 号 → 补贴 10%；0 号+补贴 → 原档位 +10% 或双开无档位 → 15%（上限 15%）。
func migrateZeroAccountIntoSubsidy(db *gorm.DB) error {
	// 无档位且非 0 号：不应保留「已开通补贴」标记，避免被默认成 5%。
	if err := db.Exec(`
		UPDATE users SET is_community_subsidy = 0
		WHERE is_community_subsidy = 1 AND community_subsidy_rate = 0 AND is_zero_account = 0
	`).Error; err != nil {
		return err
	}
	// 历史 0 号已获收益并入社区补贴累计，清零 0 号累计（可重复执行）。
	if err := db.Exec(`
		UPDATE users
		SET community_subsidy_total = community_subsidy_total + zero_account_reward_total,
		    zero_account_reward_total = 0
		WHERE zero_account_reward_total > 0
	`).Error; err != nil {
		return err
	}
	var rows []UserPO
	if err := db.Where("is_zero_account = ?", true).Find(&rows).Error; err != nil {
		return err
	}
	for _, u := range rows {
		newRate := biz.MergeZeroAccountIntoSubsidyRate(u.CommunitySubsidyRate, u.IsCommunitySubsidy)
		if err := db.Model(&UserPO{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
			"is_community_subsidy":     true,
			"community_subsidy_rate":   newRate,
			"community_subsidy_set_at": gorm.Expr("COALESCE(community_subsidy_set_at, NOW(3))"),
			"is_zero_account":          false,
		}).Error; err != nil {
			return err
		}
	}
	// 修正已迁移用户：0 号 + 社区补贴双开却被算成 10% → 应为 15%。
	return fixDualRoleMergedToTenPercent(db)
}

// fixDualRoleMergedToTenPercent 双开用户合并后应为 15%，不应停留在 10%。
func fixDualRoleMergedToTenPercent(db *gorm.DB) error {
	return db.Exec(`
		UPDATE users u
		SET community_subsidy_rate = 15
		WHERE u.community_subsidy_rate = 10
		  AND u.is_community_subsidy = 1
		  AND u.zero_account_set_at IS NOT NULL
		  AND (
		    EXISTS (
		      SELECT 1 FROM reward_logs rl
		      WHERE rl.user_id = u.id AND rl.type = ?
		    )
		    OR (
		      u.community_subsidy_set_at IS NOT NULL
		      AND u.community_subsidy_set_at <= u.zero_account_set_at
		    )
		    OR (
		      u.community_subsidy_set_at IS NOT NULL
		      AND u.community_subsidy_set_at < DATE_SUB(NOW(3), INTERVAL 30 MINUTE)
		    )
		  )
	`, biz.RewardTypeCommunitySubsidy).Error
}
