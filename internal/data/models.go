package data

import (
	"time"

	"github.com/shopspring/decimal"
)

const decimalType = "decimal(36,18)"

type UserPO struct {
	ID                int64           `gorm:"primaryKey;autoIncrement"`
	Address           string          `gorm:"uniqueIndex;size:42;not null"`
	InviterID         *int64          `gorm:"index"`
	InviteCode        string          `gorm:"uniqueIndex;size:64;not null"`
	Username          string          `gorm:"column:username;size:64;default:'';not null"` // 用户端「我的团队」名称设置
	UsdtRecharge      decimal.Decimal `gorm:"column:usdt_recharge;type:decimal(36,18);default:0;not null"`
	UsdtReward        decimal.Decimal `gorm:"column:usdt_reward;type:decimal(36,18);default:0;not null"`
	AixBalance        decimal.Decimal `gorm:"column:aix_balance;type:decimal(36,18);default:0;not null"`         // AIX 代币数（静态换算入账）
	WinBalance          decimal.Decimal `gorm:"column:win_balance;type:decimal(36,18);default:0;not null"`                 // WIN 提现钱包（AIX 兑换）
	WinRechargeBalance  decimal.Decimal `gorm:"column:win_recharge_balance;type:decimal(36,18);default:0;not null"`       // WIN 充值钱包
	WinARechargeBalance decimal.Decimal `gorm:"column:win_a_recharge_balance;type:decimal(36,18);default:0;not null"`     // WIN-A 充值钱包
	UsdtWithdrawable    decimal.Decimal `gorm:"column:usdt_withdrawable;type:decimal(36,18);default:0;not null"`           // 可提 U 余额（0号/社区补贴）
	PendingMgmtReward decimal.Decimal `gorm:"column:pending_mgmt_reward;type:decimal(36,18);default:0;not null"` // 兼容旧列；= OverflowReward
	OverflowReward    decimal.Decimal `gorm:"column:overflow_reward;type:decimal(36,18);default:0;not null"`     // 管理奖溢出
	OverflowDirect    decimal.Decimal `gorm:"column:overflow_direct;type:decimal(36,18);default:0;not null"`     // 直推奖溢出
	Points                  decimal.Decimal `gorm:"column:points;type:decimal(36,18);default:0;not null"`                         // 当前 AIX-USDT
	PointsAll               decimal.Decimal `gorm:"column:points_all;type:decimal(36,18);default:0;not null"`                     // 累计 AIX-USDT
	TransferReinvestCredit  decimal.Decimal `gorm:"column:transfer_reinvest_credit;type:decimal(36,18);default:0;not null"`       // 上级划入、可用于复投产生 AIX-USDT 的额度
	StaticUsdtTotal   decimal.Decimal `gorm:"column:static_usdt_total;type:decimal(36,18);default:0;not null"`   // 静态总收益（USDT 金本位累计）
	MgmtLevel         int32           `gorm:"column:mgmt_level;default:0;not null"`
	MgmtLevelLocked   bool            `gorm:"column:mgmt_level_locked;default:false;not null"`
	LargeAreaPerf     decimal.Decimal `gorm:"column:large_area_perf;type:decimal(36,18);default:0;not null"`
	SmallAreaPerf     decimal.Decimal `gorm:"column:small_area_perf;type:decimal(36,18);default:0;not null"`
	TeamPerf          decimal.Decimal `gorm:"column:team_perf;type:decimal(36,18);default:0;not null"`
	IsZeroAccount          bool            `gorm:"column:is_zero_account;default:false;not null"` // 已废弃，补贴合并后仅作历史迁移
	IsCommunitySubsidy     bool            `gorm:"column:is_community_subsidy;default:false;not null"`
	CommunitySubsidyRate   int32           `gorm:"column:community_subsidy_rate;default:0;not null"` // 补贴档位：5 / 10 / 15
	ZeroAccountSetAt       *time.Time      `gorm:"column:zero_account_set_at"`
	CommunitySubsidySetAt  *time.Time      `gorm:"column:community_subsidy_set_at"`
	ZeroAccountRewardTotal decimal.Decimal `gorm:"column:zero_account_reward_total;type:decimal(36,18);default:0;not null"`
	CommunitySubsidyTotal  decimal.Decimal `gorm:"column:community_subsidy_total;type:decimal(36,18);default:0;not null"`
	Status            int32           `gorm:"default:1;not null"`
	IsFrozen          bool            `gorm:"column:is_frozen;default:false;not null"` // 冻结后禁止登录/充值/报单/提现
	FrozenAt          *time.Time      `gorm:"column:frozen_at"`
	Role              string          `gorm:"size:16;default:user;not null"` // app admin helper, not in business DDL
	CreatedTime       time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime       time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (UserPO) TableName() string { return "users" }

type OrderPO struct {
	ID           int64           `gorm:"primaryKey;autoIncrement"`
	UserID       int64           `gorm:"index;not null"`
	Principal    decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	ExitCap      decimal.Decimal `gorm:"column:exit_cap;type:decimal(36,18);not null"`
	EarnedTotal  decimal.Decimal `gorm:"column:earned_total;type:decimal(36,18);default:0;not null"`
	DirectBase   decimal.Decimal `gorm:"column:direct_base;type:decimal(36,18);default:0;not null"`
	FromRecharge decimal.Decimal `gorm:"column:from_recharge;type:decimal(36,18);default:0;not null"`
	FromReward   decimal.Decimal `gorm:"column:from_reward;type:decimal(36,18);default:0;not null"`
	FromWin      decimal.Decimal `gorm:"column:from_win;type:decimal(36,18);default:0;not null"`       // WIN 扣款数量（按认购时 win_price 折算）
	FromWinA     decimal.Decimal `gorm:"column:from_win_a;type:decimal(36,18);default:0;not null"`     // WIN-A 扣款数量（按认购时 win_a_price 折算）
	Points       decimal.Decimal `gorm:"column:points;type:decimal(36,18);default:0;not null"`         // 本单获得 AIX-USDT
	PointsSource string          `gorm:"column:points_source;size:32;default:'';not null"`             // recharge | win | transfer_reinvest
	FundSource   string          `gorm:"column:fund_source;size:16;not null"`
	Status       string          `gorm:"size:16;default:active;not null"`
	ExitedTime   *time.Time      `gorm:"column:exited_time"`
	CreatedTime  time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime  time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (OrderPO) TableName() string { return "orders" }

type RechargePO struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	UserID        int64           `gorm:"index;not null"`
	Asset         string          `gorm:"size:16;default:USDT;not null;index"` // USDT | WIN | WIN-A
	Amount        decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	TxHash        string          `gorm:"size:66;uniqueIndex;not null"`
	FromAddress   string          `gorm:"column:from_address;size:42"`
	ToAddress     string          `gorm:"column:to_address;size:42"`
	Status        string          `gorm:"size:16;default:pending;not null"`
	Message       string          `gorm:"type:text"` // signing message for confirm flow
	ExpireAt      *time.Time      `gorm:"column:expire_at"`
	ConfirmedTime *time.Time      `gorm:"column:confirmed_time"`
	CreatedTime   time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime   time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (RechargePO) TableName() string { return "recharges" }

type TransferPO struct {
	ID                int64           `gorm:"primaryKey;autoIncrement"`
	FromUserID        int64           `gorm:"column:from_user_id;index;not null"`
	ToUserID          int64           `gorm:"column:to_user_id;index;not null"`
	Asset             string          `gorm:"size:16;not null"`
	Amount            decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	PayFrom           string          `gorm:"column:pay_from;size:16"`
	FromRechargeDebit decimal.Decimal `gorm:"column:from_recharge_debit;type:decimal(36,18);default:0;not null"`
	FromRewardDebit   decimal.Decimal `gorm:"column:from_reward_debit;type:decimal(36,18);default:0;not null"`
	ToCreditReward    decimal.Decimal `gorm:"column:to_credit_reward;type:decimal(36,18);default:0;not null"`
	ToCreditAix       decimal.Decimal `gorm:"column:to_credit_aix;type:decimal(36,18);default:0;not null"`
	Remark            string          `gorm:"size:255"`
	CreatedTime       time.Time       `gorm:"column:created_time;autoCreateTime"`
}

func (TransferPO) TableName() string { return "transfers" }

// WithdrawalPO 支持 AIX 与 WIN 代币提现（AIX 当前禁止提现，仅 WIN 可提现）
type WithdrawalPO struct {
	ID          int64           `gorm:"primaryKey;autoIncrement"`
	UserID      int64           `gorm:"index;not null"`
	Asset       string          `gorm:"size:16;default:AIX;not null"` // AIX 或 WIN
	Amount      decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	Fee         decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null"`
	PayAmount   decimal.Decimal `gorm:"column:pay_amount;type:decimal(36,18);not null"`
	ToAddress   string          `gorm:"column:to_address;size:42;not null"`
	TxHash      string          `gorm:"column:tx_hash;size:66"`
	Status      string          `gorm:"size:16;default:pending;not null"`
	Remark      string          `gorm:"size:255"`
	PayoutNonce *uint64         `gorm:"column:payout_nonce"`
	CreatedTime time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (WithdrawalPO) TableName() string { return "withdrawals" }

// WithdrawalPayoutPO tracks each on-chain payout attempt for idempotency.
type WithdrawalPayoutPO struct {
	ID           int64           `gorm:"primaryKey;autoIncrement"`
	WithdrawID   int64           `gorm:"column:withdraw_id;index;not null"`
	TxHash       string          `gorm:"column:tx_hash;uniqueIndex;size:66;not null"`
	Nonce        uint64          `gorm:"not null"`
	FromAddress  string          `gorm:"column:from_address;size:42;not null"`
	ToAddress    string          `gorm:"column:to_address;size:42;not null"`
	Amount       decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	Status       string          `gorm:"size:16;not null"` // submitted, confirmed, failed
	CreatedTime  time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime  time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (WithdrawalPayoutPO) TableName() string { return "withdrawal_payouts" }

// ExchangeRecordPO AIX → WIN 兑换记录
type ExchangeRecordPO struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	UserID        int64           `gorm:"index;not null"`
	FromAsset     string          `gorm:"column:from_asset;size:16;not null"` // 固定 AIX
	FromAmount    decimal.Decimal `gorm:"column:from_amount;type:decimal(36,18);not null"`
	ToAsset       string          `gorm:"column:to_asset;size:16;not null"` // 固定 WIN
	ToAmount      decimal.Decimal `gorm:"column:to_amount;type:decimal(36,18);not null"`
	FeeAmount     decimal.Decimal `gorm:"column:fee_amount;type:decimal(36,18);not null;default:0"`
	ExchangePrice decimal.Decimal `gorm:"column:exchange_price;type:decimal(36,18);not null"` // 兑换时的 WIN 价格（USDT/枚）
	FeeRate       decimal.Decimal `gorm:"column:fee_rate;type:decimal(12,6);not null;default:0"` // 兑换时的手续费率
	Status        string          `gorm:"size:16;default:completed;not null"`                 // completed
	Remark        string          `gorm:"size:255"`
	CreatedTime   time.Time       `gorm:"column:created_time;autoCreateTime"`
}

func (ExchangeRecordPO) TableName() string { return "exchange_records" }

type RewardLogPO struct {
	ID             int64            `gorm:"primaryKey;autoIncrement"`
	UserID         int64            `gorm:"index;not null"`
	FromUserID     *int64           `gorm:"column:from_user_id"`
	OrderID        *int64           `gorm:"column:order_id;index"`
	BatchID        *int64           `gorm:"column:batch_id;index"`
	Type           string           `gorm:"size:32;not null"`
	Asset          string           `gorm:"size:16;not null"`
	Amount         decimal.Decimal  `gorm:"type:decimal(36,18);not null"`
	BaseAmount     *decimal.Decimal `gorm:"column:base_amount;type:decimal(36,18)"`
	Rate           *decimal.Decimal `gorm:"type:decimal(36,18)"`
	ExitApplied    decimal.Decimal  `gorm:"column:exit_applied;type:decimal(36,18);default:0;not null"`
	Meta           *string          `gorm:"type:json"`
	SettlementDate *string          `gorm:"column:settlement_date;type:date;index"`
	CreatedTime    time.Time        `gorm:"column:created_time;autoCreateTime"`
}

func (RewardLogPO) TableName() string { return "reward_logs" }

// MgmtRewardPO stores the one-time differential entitlement generated by a
// downline subscription. Each (user_id, source_order_id) is settled once into
// the beneficiary's reward wallet at creation time.
type MgmtRewardPO struct {
	ID             int64           `gorm:"primaryKey;autoIncrement"`
	UserID         int64           `gorm:"column:user_id;index;not null;uniqueIndex:uk_mgmt_source"`
	FromUserID     int64           `gorm:"column:from_user_id;index;not null"`
	SourceOrderID  int64           `gorm:"column:source_order_id;index;not null;uniqueIndex:uk_mgmt_source"`
	BaseAmount     decimal.Decimal `gorm:"column:base_amount;type:decimal(36,18);not null"`
	Rate           decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	TotalAmount    decimal.Decimal `gorm:"column:total_amount;type:decimal(36,18);not null"`
	ReleasedAmount decimal.Decimal `gorm:"column:released_amount;type:decimal(36,18);default:0;not null"`
	CreatedTime    time.Time       `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime    time.Time       `gorm:"column:updated_time;autoUpdateTime"`
}

func (MgmtRewardPO) TableName() string { return "mgmt_rewards" }

type AixPricePO struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	Price         decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	EffectiveDate string          `gorm:"column:effective_date;type:date;uniqueIndex;not null"`
	Remark        string          `gorm:"size:255"`
	CreatedTime   time.Time       `gorm:"column:created_time;autoCreateTime"`
}

func (AixPricePO) TableName() string { return "aix_prices" }

// WinPricePO 当前 WIN 价格（全表仅保留 1 条，固定 ID=1，预言机/后台均覆盖更新）。
type WinPricePO struct {
	ID          int64           `gorm:"primaryKey"`
	Price       decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	Source      string          `gorm:"size:32;default:oracle;not null"`
	UpdatedTime time.Time       `gorm:"column:updated_time;autoUpdateTime"`
	CreatedTime time.Time       `gorm:"column:created_time;autoCreateTime"`
}

const WinPriceRowID int64 = 1

func (WinPricePO) TableName() string { return "win_prices" }

type SettlementBatchPO struct {
	ID             int64           `gorm:"primaryKey;autoIncrement"`
	SettlementDate string          `gorm:"column:settlement_date;type:date;index;not null"`
	AixPrice       decimal.Decimal `gorm:"column:aix_price;type:decimal(36,18);not null"`
	Status         string          `gorm:"size:16;default:running;not null"`
	StaticCount    int32           `gorm:"column:static_count;default:0;not null"`
	StaticAmount   decimal.Decimal `gorm:"column:static_amount;type:decimal(36,18);default:0;not null"`
	MgmtCount      int32           `gorm:"column:mgmt_count;default:0;not null"`
	MgmtAmount     decimal.Decimal `gorm:"column:mgmt_amount;type:decimal(36,18);default:0;not null"`
	StartedTime    *time.Time      `gorm:"column:started_time"`
	FinishedTime   *time.Time      `gorm:"column:finished_time"`
	ErrorMsg       string          `gorm:"column:error_msg;size:512"`
	CreatedTime    time.Time       `gorm:"column:created_time;autoCreateTime"`
}

func (SettlementBatchPO) TableName() string { return "settlement_batches" }

type SettingPO struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Key         string    `gorm:"uniqueIndex;size:64;column:key;not null"`
	Value       string    `gorm:"type:json;not null"`
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime"`
}

func (SettingPO) TableName() string { return "settings" }

// AnnouncementPO 系统公告（管理端编辑，用户端列表展示）
type AnnouncementPO struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Title       string    `gorm:"size:256;not null"`
	Content     string    `gorm:"type:longtext;not null"`
	Status      int32     `gorm:"default:1;not null"` // 1=发布 0=下架
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime"`
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime"`
}

func (AnnouncementPO) TableName() string { return "announcements" }

// FeedbackPO 用户问题反馈（用户端提交，管理端查看）
type FeedbackPO struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UserID      *int64    `gorm:"column:user_id;index"`
	Address     string    `gorm:"column:address;size:128;not null;index"`
	Content     string    `gorm:"column:content;type:text;not null"`
	Status      int32     `gorm:"column:status;default:0;not null;index"` // 0=待处理 1=已读 2=已处理
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime;index"`
}

func (FeedbackPO) TableName() string { return "feedbacks" }

// AdminOperationLogPO 管理后台操作审计日志。
type AdminOperationLogPO struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Operator     string    `gorm:"column:operator;size:64;not null;index"`     // 主账户名或子账户名
	OperatorType string    `gorm:"column:operator_type;size:16;not null"`      // main | sub
	Action       string    `gorm:"column:action;size:128;not null;index"`      // 请求路径
	ActionLabel  string    `gorm:"column:action_label;size:128;not null"`      // 操作说明
	Method       string    `gorm:"column:method;size:16;not null"`
	Params       string    `gorm:"column:params;size:2048"`                    // 请求参数摘要
	ClientIP     string    `gorm:"column:client_ip;size:64"`
	CreatedTime  time.Time `gorm:"column:created_time;autoCreateTime;index"`
}

func (AdminOperationLogPO) TableName() string { return "admin_operation_logs" }

// PartnerNoncePO 合作方转账加款接口的 nonce 去重记录。
//
// 唯一索引 (partner_id, nonce) 就是防重放的原子操作本身：插入成功=首次出现，
// 插入冲突=重放。等价于文档 §5 建议的 Redis SET NX EX 300，但不引入新的运行时依赖。
// 过期行由定时清理删除，保留窗口不得小于时间戳允许偏差，否则会出现
// 「nonce 已被清理但时间戳仍在有效期内」的重放空档。
type PartnerNoncePO struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	PartnerID   string    `gorm:"column:partner_id;size:64;not null;uniqueIndex:uniq_partner_nonce,priority:1"`
	Nonce       string    `gorm:"column:nonce;size:64;not null;uniqueIndex:uniq_partner_nonce,priority:2"`
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime;index"`
}

func (PartnerNoncePO) TableName() string { return "partner_nonces" }
