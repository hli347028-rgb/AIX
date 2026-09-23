package data

import (
	"backend/internal/biz"

	"gorm.io/gorm"
)

// migrateZeroAccountIntoSubsidy 删除 0 号账户时，将原 10% 并入社区补贴档位。
// 仅 0 号 → 补贴 10%；0 号+补贴 → 原档位 +10% 或双开无档位 → 15%（上限 15%）。
//
// 不再在每次启动时把「已是 10%」的用户强制改成 15%：人工后台设成 10% 是合法档位，
// 旧版 fixDualRoleMergedToTenPercent（含「超过 30 分钟就 10→15」）会误伤。
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
	return nil
}
