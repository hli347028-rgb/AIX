package data

import (
	"encoding/json"
	"time"

	"backend/internal/biz"
	"backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserRepo, NewChallengeRepo, NewWalletRepo, NewStakingRepo, NewSettingsRepo)

// Data .
type Data struct {
	db *gorm.DB
}

// NewData .
func NewData(dbCfg *conf.DatabaseConfig, logger log.Logger) (*Data, func(), error) {
	db, err := gorm.Open(mysql.Open(dbCfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	if err := db.AutoMigrate(
		&UserPO{}, &OrderPO{}, &RechargePO{}, &TransferPO{}, &WithdrawalPO{}, &WithdrawalPayoutPO{},
		&RewardLogPO{}, &MgmtRewardPO{}, &AixPricePO{}, &WinPricePO{}, &SettlementBatchPO{}, &SettingPO{},
		&ExchangeRecordPO{}, &AnnouncementPO{},
	); err != nil {
		return nil, nil, err
	}
	if err := ensureWithdrawalPayoutGuards(db); err != nil {
		return nil, nil, err
	}
	if err := ensureSettlementBatchMultiPerDay(db); err != nil {
		return nil, nil, err
	}
	if err := migrateOverflowReward(db); err != nil {
		return nil, nil, err
	}
	if err := migrateSubscribePoints(db); err != nil {
		return nil, nil, err
	}
	if err := migrateWinRechargeBalance(db); err != nil {
		return nil, nil, err
	}
	if err := ensureUserAdminColumns(db); err != nil {
		return nil, nil, err
	}
	if err := seedDefaults(db); err != nil {
		return nil, nil, err
	}
	if err := ensureZeroAddressAdmin(db); err != nil {
		return nil, nil, err
	}
	if err := refreshAllPerformance(db); err != nil {
		return nil, nil, err
	}
	data := &Data{db: db}
	cleanup := func() {
		sqlDB, err := data.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		log.NewHelper(logger).Info("closing the data resources")
	}
	return data, cleanup, nil
}

// ensureSettlementBatchMultiPerDay removes the historical unique constraint
// on settlement_date so every manual settlement can retain its own batch and
// reward base. Automatic settlement still checks for an existing successful
// date before creating a batch.
func ensureSettlementBatchMultiPerDay(db *gorm.DB) error {
	type indexRow struct {
		IndexName string `gorm:"column:INDEX_NAME"`
	}
	var uniqueIndexes []indexRow
	if err := db.Raw(`
		SELECT INDEX_NAME
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'settlement_batches'
		  AND column_name = 'settlement_date'
		  AND NON_UNIQUE = 0
	`).Scan(&uniqueIndexes).Error; err != nil {
		return err
	}
	for _, index := range uniqueIndexes {
		if index.IndexName == "" || index.IndexName == "PRIMARY" {
			continue
		}
		if err := db.Exec("ALTER TABLE settlement_batches DROP INDEX `" + index.IndexName + "`").Error; err != nil {
			return err
		}
	}

	var normalIndexCount int64
	if err := db.Raw(`
		SELECT COUNT(1)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'settlement_batches'
		  AND column_name = 'settlement_date'
		  AND NON_UNIQUE = 1
	`).Scan(&normalIndexCount).Error; err != nil {
		return err
	}
	if normalIndexCount == 0 {
		return db.Exec("CREATE INDEX idx_settlement_batches_settlement_date ON settlement_batches (settlement_date)").Error
	}
	return nil
}

// migrateOverflowReward 将历史 pending_mgmt_reward 迁入 overflow_reward，并保持两列同步。
func migrateOverflowReward(db *gorm.DB) error {
	return db.Exec(`
		UPDATE users
		SET overflow_reward = pending_mgmt_reward
		WHERE overflow_reward = 0 AND pending_mgmt_reward > 0
	`).Error
}

// migrateSubscribePoints 回填历史订单积分与用户累计积分（仅补零值，不覆盖已有）。
func migrateSubscribePoints(db *gorm.DB) error {
	if err := db.Exec(`
		UPDATE orders
		SET points = principal
		WHERE points = 0 AND principal > 0
	`).Error; err != nil {
		return err
	}
	return db.Exec(`
		UPDATE users u
		SET
			points_all = COALESCE((
				SELECT SUM(o.points) FROM orders o WHERE o.user_id = u.id
			), 0),
			points = COALESCE((
				SELECT SUM(o.points) FROM orders o WHERE o.user_id = u.id
			), 0)
		WHERE u.points_all = 0
	`).Error
}

// migrateWinRechargeBalance 首次启动时将历史 WIN 充值从 win_balance 拆入 win_recharge_balance。
func migrateWinRechargeBalance(db *gorm.DB) error {
	var migrated int64
	if err := db.Raw(`SELECT COUNT(1) FROM users WHERE win_recharge_balance > 0`).Scan(&migrated).Error; err != nil {
		return err
	}
	if migrated > 0 {
		return nil
	}
	var confirmedWin int64
	if err := db.Raw(`SELECT COUNT(1) FROM recharges WHERE UPPER(asset) = 'WIN' AND status = 'confirmed'`).Scan(&confirmedWin).Error; err != nil {
		return err
	}
	if confirmedWin == 0 {
		return nil
	}
	if err := db.Exec(`
		UPDATE users u
		SET win_recharge_balance = COALESCE((
			SELECT SUM(r.amount) FROM recharges r
			WHERE r.user_id = u.id AND UPPER(r.asset) = 'WIN' AND r.status = 'confirmed'
		), 0)
	`).Error; err != nil {
		return err
	}
	return db.Exec(`
		UPDATE users
		SET win_balance = GREATEST(win_balance - win_recharge_balance, 0)
		WHERE win_recharge_balance > 0
	`).Error
}

func ensureUserAdminColumns(db *gorm.DB) error {
	columns := []struct {
		name string
		ddl  string
	}{
		{
			name: "overflow_direct",
			ddl:  "ALTER TABLE users ADD COLUMN overflow_direct decimal(36,18) NOT NULL DEFAULT 0",
		},
		{
			name: "is_zero_account",
			ddl:  "ALTER TABLE users ADD COLUMN is_zero_account tinyint(1) NOT NULL DEFAULT 0",
		},
		{
			name: "is_community_subsidy",
			ddl:  "ALTER TABLE users ADD COLUMN is_community_subsidy tinyint(1) NOT NULL DEFAULT 0",
		},
		{
			name: "zero_account_reward_total",
			ddl:  "ALTER TABLE users ADD COLUMN zero_account_reward_total decimal(36,18) NOT NULL DEFAULT 0",
		},
		{
			name: "community_subsidy_total",
			ddl:  "ALTER TABLE users ADD COLUMN community_subsidy_total decimal(36,18) NOT NULL DEFAULT 0",
		},
		{
			name: "zero_account_set_at",
			ddl:  "ALTER TABLE users ADD COLUMN zero_account_set_at datetime(3) DEFAULT NULL",
		},
		{
			name: "community_subsidy_set_at",
			ddl:  "ALTER TABLE users ADD COLUMN community_subsidy_set_at datetime(3) DEFAULT NULL",
		},
		{
			name: "usdt_withdrawable",
			ddl:  "ALTER TABLE users ADD COLUMN usdt_withdrawable decimal(36,18) NOT NULL DEFAULT 0",
		},
		{
			name: "win_a_recharge_balance",
			ddl:  "ALTER TABLE users ADD COLUMN win_a_recharge_balance decimal(36,18) NOT NULL DEFAULT 0",
		},
	}
	for _, col := range columns {
		var cnt int64
		if err := db.Raw(`
			SELECT COUNT(1)
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name = 'users'
			  AND column_name = ?
		`, col.name).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			if err := db.Exec(col.ddl).Error; err != nil {
				return err
			}
		}
	}
	// 历史已开启角色但无设置时间：用 updated_time 近似回填（仅补空值）
	if err := db.Exec(`
		UPDATE users SET zero_account_set_at = updated_time
		WHERE is_zero_account = 1 AND zero_account_set_at IS NULL
	`).Error; err != nil {
		return err
	}
	if err := db.Exec(`
		UPDATE users SET community_subsidy_set_at = updated_time
		WHERE is_community_subsidy = 1 AND community_subsidy_set_at IS NULL
	`).Error; err != nil {
		return err
	}
	return nil
}

// DB exposes the underlying gorm handle for admin legacy queries.
func (d *Data) DB() *gorm.DB {
	return d.db
}

func seedDefaults(db *gorm.DB) error {
	var cnt int64
	if err := db.Model(&SettingPO{}).Where("`key` = ?", conf.SettingsKeySystemConfig).Count(&cnt).Error; err != nil {
		return err
	}
	snap := conf.SystemConfigSnapshot{
		StaticRate:           conf.DefaultStaticRate,
		ExitMultiplier:       conf.DefaultExitMultiplier,
		DirectRate:           conf.DefaultDirectRate,
		MgmtThresholds:       conf.DefaultMgmtThresholds(),
		MgmtRates:            conf.DefaultMgmtRates(),
		AixPriceInitial:      conf.DefaultAixPrice,
		WinPrice:             conf.DefaultWinPrice,
		WinAPrice:            conf.DefaultWinAPrice,
		MgmtCountsTowardExit: true,
		MinSubscribe:         conf.DefaultMinSubscribe,
		MinUsdtRecharge:      conf.DefaultMinUsdtRecharge,
		MinWinRecharge:       conf.DefaultMinWinRecharge,
		MinWinARecharge:      conf.DefaultMinWinARecharge,
	}
	if cnt == 0 {
		conf.NormalizeBusinessDefaults(&snap)
		raw, _ := json.Marshal(snap)
		if err := db.Create(&SettingPO{Key: conf.SettingsKeySystemConfig, Value: string(raw)}).Error; err != nil {
			return err
		}
	} else {
		var po SettingPO
		if err := db.Where("`key` = ?", conf.SettingsKeySystemConfig).First(&po).Error; err == nil && po.Value != "" {
			_ = json.Unmarshal([]byte(po.Value), &snap)
			conf.NormalizeBusinessDefaults(&snap)
		}
	}
	var totalPriceCnt int64
	if err := db.Model(&AixPricePO{}).Count(&totalPriceCnt).Error; err != nil {
		return err
	}
	if totalPriceCnt == 0 {
		today := time.Now().Format("2006-01-02")
		aixSeed := decimal.NewFromFloat(snap.AixPriceInitial)
		if !aixSeed.IsPositive() {
			aixSeed = decimal.NewFromFloat(conf.DefaultAixPrice)
		}
		if err := db.Create(&AixPricePO{
			Price:         aixSeed.Round(biz.AixPriceDecimals),
			EffectiveDate: today,
			Remark:        "initial",
		}).Error; err != nil {
			return err
		}
	}
	var winCnt int64
	if err := db.Model(&WinPricePO{}).Where("id = ?", WinPriceRowID).Count(&winCnt).Error; err != nil {
		return err
	}
	if winCnt == 0 {
		winSeed := decimal.NewFromFloat(snap.WinPrice)
		if !winSeed.IsPositive() {
			winSeed = decimal.NewFromFloat(conf.DefaultWinPrice)
		}
		return db.Create(&WinPricePO{
			ID:     WinPriceRowID,
			Price:  winSeed,
			Source: "initial",
		}).Error
	}
	return nil
}

func ensureZeroAddressAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&UserPO{}).Where("address = ?", biz.ZeroAddress).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Create(&UserPO{
			Address:    biz.ZeroAddress,
			InviteCode: biz.ZeroAddress,
			Role:       biz.RoleAdmin,
			Status:     1,
		}).Error
	}
	return db.Model(&UserPO{}).
		Where("address = ?", biz.ZeroAddress).
		Update("role", biz.RoleAdmin).Error
}
