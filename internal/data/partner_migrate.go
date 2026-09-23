package data

import (
	"log"

	"gorm.io/gorm"
)

const partnerWinAToWinMigrateKey = "partner_wina_to_win_v1"

// migratePartnerWinABalanceToWin 一次性：把历史「交易所划转记入 WIN-A 钱包」的余额挪回 WIN 充值钱包。
//
// 规则：
//   - 仅处理 tx_hash 以 partner: 开头且 asset=WIN-A 的已确认流水
//   - 每人迁移额 = min(该用户 partner WIN-A 流水合计, 当前 win_a_recharge_balance)
//     （已花掉的部分不再虚增 WIN）
//   - recharges.asset 保持 WIN-A，供管理端区分币种
//   - 用 settings 标记幂等，避免重复迁移
func migratePartnerWinABalanceToWin(db *gorm.DB) error {
	var done int64
	if err := db.Model(&SettingPO{}).Where("`key` = ?", partnerWinAToWinMigrateKey).Count(&done).Error; err != nil {
		return err
	}
	if done > 0 {
		return nil
	}

	var partnerWinARows int64
	if err := db.Raw(`
		SELECT COUNT(1) FROM recharges
		WHERE status = 'confirmed'
		  AND UPPER(asset) = 'WIN-A'
		  AND tx_hash LIKE 'partner:%'
	`).Scan(&partnerWinARows).Error; err != nil {
		return err
	}
	if partnerWinARows == 0 {
		return db.Create(&SettingPO{
			Key:   partnerWinAToWinMigrateKey,
			Value: `{"done":true,"moved_users":0,"note":"no partner WIN-A rows"}`,
		}).Error
	}

	res := db.Exec(`
		UPDATE users u
		JOIN (
			SELECT user_id, COALESCE(SUM(amount), 0) AS partner_wina
			FROM recharges
			WHERE status = 'confirmed'
			  AND UPPER(asset) = 'WIN-A'
			  AND tx_hash LIKE 'partner:%'
			GROUP BY user_id
		) p ON p.user_id = u.id
		SET
			u.win_recharge_balance = u.win_recharge_balance + LEAST(u.win_a_recharge_balance, p.partner_wina),
			u.win_a_recharge_balance = u.win_a_recharge_balance - LEAST(u.win_a_recharge_balance, p.partner_wina)
		WHERE u.win_a_recharge_balance > 0
		  AND p.partner_wina > 0
	`)
	if res.Error != nil {
		return res.Error
	}
	log.Printf("migratePartnerWinABalanceToWin: moved rows=%d partner_wina_recharges=%d", res.RowsAffected, partnerWinARows)

	return db.Create(&SettingPO{
		Key:   partnerWinAToWinMigrateKey,
		Value: `{"done":true}`,
	}).Error
}
