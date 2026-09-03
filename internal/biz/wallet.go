package biz

import (
	"context"
	"strings"
	"time"

	"backend/internal/conf"

	"github.com/shopspring/decimal"
)

const (
	RechargeStatusPending   = "pending"
	RechargeStatusConfirmed = "confirmed"
	RechargeStatusRejected  = "rejected"

	PayFromRecharge = "recharge"
	PayFromReward   = "reward" // 奖励钱包复投：默认不产生 AIX-USDT；邀请链上级划转额度内复投可产生
	PayFromWin      = "win"    // 用 WIN 充值钱包按 win_price 折算认购（产生直推/管理奖）
	PayFromWinA     = "win_a"  // 认购已关闭；保留供历史订单 fund_source 识别

	// PointsSource AIX-USDT 产生来源（写入 orders.points_source）
	PointsSourceRecharge         = "recharge"          // USDT 认购
	PointsSourceWin              = "win"               // WIN 认购
	PointsSourceTransferReinvest = "transfer_reinvest" // 任意层级上级→下级划转后，下级复投产生（不限直推）

	OrderStatusActive    = "active"
	OrderStatusExited    = "exited"
	OrderStatusCancelled = "cancelled"

	// Legacy aliases
	OrderStatusCompleted    = "exited"
	WithdrawStatusReview    = "review"
	WithdrawStatusPending   = "pending"
	WithdrawStatusDoing     = "doing"
	WithdrawStatusCompleted = "completed"
	WithdrawStatusFailed    = "failed"
	WithdrawStatusRejected  = "rejected"

	ExchangeStatusCompleted = "completed"
	ExchangeStatusReview    = "review"
	ExchangeStatusRejected  = "rejected"

	PayoutStatusSubmitted = "submitted"
	PayoutStatusConfirmed = "confirmed"
	PayoutStatusFailed    = "failed"

	TokenUSDT = "USDT"
	TokenAIX  = "AIX"
	TokenWIN  = "WIN"
	TokenWINA = "WIN-A"
	TokenSDT  = "SDT"

	// 下级 USDT 充值时，上级角色奖励比例（入账可提 U 余额；仅 USDT，不含 WIN/WIN-A）
	ZeroAccountRechargeRate      = 0.10
	CommunitySubsidyRechargeRate = 0.05

	RewardTypeStaticAix         = "static_aix"
	RewardTypeDynamicUsdt       = "dynamic_usdt"
	RewardTypeDirectPoolRelease = "direct_pool_release"
	RewardTypeMgmt              = "mgmt"
	RewardTypeMgmtPoolRelease   = "mgmt_pool_release"
	RewardTypeMgmtOverflow      = "mgmt_overflow" // 管理奖溢出：应得但受出局额度限制未入账，留档待释放
	RewardTypeExitAccel         = "exit_accel"
	RewardTypeTransferIn        = "transfer_in"
	RewardTypeTransferOut       = "transfer_out"
	RewardTypeZeroAccount       = "zero_account"       // 零号账户：下级 USDT 充值奖励
	RewardTypeCommunitySubsidy  = "community_subsidy"  // 社区补贴：下级 USDT 充值奖励
)

// GetWinPrice 返回 WIN 代币价格（USDT/枚）。
// 由 WinPriceOracleJob 每分钟从链上 Pair 储备更新；管理后台亦可手动覆盖。
func GetWinPrice() float64 {
	return WinPrice
}

// GetWinAPrice 返回 WIN-A 代币价格（USDT/枚）。
// 业务要求：WIN-A 价格始终与 WIN 价格相等。
func GetWinAPrice() float64 {
	return WinPrice
}

// GetExchangeFeeRate 返回兑换手续费率（如 0.05 = 5%）。
func GetExchangeFeeRate() float64 {
	return ExchangeFeeRate
}

// GetMinUsdtRecharge 返回 USDT 充值最小值（管理端可配，绝对下限 10）。
func GetMinUsdtRecharge() string {
	if strings.TrimSpace(MinUsdtRecharge) == "" {
		return conf.DefaultMinUsdtRecharge
	}
	return MinUsdtRecharge
}

// GetPartnerMinAmount 返回交易所划转单笔下限（管理端可配）。
func GetPartnerMinAmount() string {
	if strings.TrimSpace(PartnerMinAmount) == "" {
		return conf.DefaultPartnerMinAmount
	}
	return PartnerMinAmount
}

// GetPartnerMaxAmount 返回交易所划转单笔上限（管理端可配）。
func GetPartnerMaxAmount() string {
	if strings.TrimSpace(PartnerMaxAmount) == "" {
		return conf.DefaultPartnerMaxAmount
	}
	return PartnerMaxAmount
}

// GetExchangeReviewThresholdPercent 兑换审核阈值百分比（管理端可配，默认 100）。
func GetExchangeReviewThresholdPercent() string {
	if strings.TrimSpace(ExchangeReviewThresholdPercent) == "" {
		return conf.DefaultExchangeReviewThresholdPercent
	}
	return ExchangeReviewThresholdPercent
}

// GetPartnerDailyLimit 返回交易所划转单日累计上限（管理端可配）。
func GetPartnerDailyLimit() string {
	if strings.TrimSpace(PartnerDailyLimit) == "" {
		return conf.DefaultPartnerDailyLimit
	}
	return PartnerDailyLimit
}

// GetRegisterBonus 返回注册赠送金额，非法或非正数一律视为不赠送。
func GetRegisterBonus() decimal.Decimal {
	amount, err := decimal.NewFromString(strings.TrimSpace(RegisterBonus))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return amount
}

// GetMinWinRecharge 返回 WIN 充值最小值（管理端可配，绝对下限 10）。
func GetMinWinRecharge() string {
	if strings.TrimSpace(MinWinRecharge) == "" {
		return conf.DefaultMinWinRecharge
	}
	return MinWinRecharge
}

// GetMinWinARecharge 返回 WIN-A 充值最小值（管理端可配，绝对下限 10）。
func GetMinWinARecharge() string {
	if strings.TrimSpace(MinWinARecharge) == "" {
		return conf.DefaultMinWinARecharge
	}
	return MinWinARecharge
}

func GetWinWithdrawReviewThreshold() string {
	if strings.TrimSpace(WinWithdrawReviewThreshold) == "" {
		return "0"
	}
	return WinWithdrawReviewThreshold
}

func GetSdtWithdrawReviewThreshold() string {
	if strings.TrimSpace(SdtWithdrawReviewThreshold) == "" {
		return "0"
	}
	return SdtWithdrawReviewThreshold
}

func GetUsdtWithdrawReviewThreshold() string {
	if strings.TrimSpace(UsdtWithdrawReviewThreshold) == "" {
		return "0"
	}
	return UsdtWithdrawReviewThreshold
}

// NeedsWithdrawReview 提现金额超过阈值时需人工审核；阈值为 0 表示不审核。
func NeedsWithdrawReview(asset string, amount decimal.Decimal) bool {
	thresholdStr := "0"
	switch strings.ToUpper(strings.TrimSpace(asset)) {
	case TokenWIN:
		thresholdStr = GetWinWithdrawReviewThreshold()
	case TokenSDT:
		thresholdStr = GetSdtWithdrawReviewThreshold()
	case TokenUSDT:
		thresholdStr = GetUsdtWithdrawReviewThreshold()
	default:
		return false
	}
	threshold, err := decimal.NewFromString(strings.TrimSpace(thresholdStr))
	if err != nil || !threshold.IsPositive() {
		return false
	}
	return amount.GreaterThan(threshold)
}

// Recharge represents a USDT/WIN recharge order.
type Recharge struct {
	ID            int64
	UserID        int64
	Address       string // from_address / user address for display
	Asset         string // USDT | WIN
	Amount        string
	Message       string
	TxHash        string
	FromAddress   string
	ToAddress     string
	Status        string
	ExpireAt      time.Time
	CreatedAt     time.Time
	ConfirmedAt   *time.Time
	CreatedTime   time.Time
	ConfirmedTime *time.Time
}

// Transfer internal transfer record.
type Transfer struct {
	ID                int64
	FromUserID        int64
	ToUserID          int64
	Asset             string
	Amount            string
	PayFrom           string
	FromRechargeDebit string
	FromRewardDebit   string
	ToCreditReward    string
	ToCreditAix       string
	Remark            string
	CreatedTime       time.Time
}

type LinealTransferRecord struct {
	ID                  int64
	Direction           string
	Relationship        string
	CounterpartyUserID  int64
	CounterpartyAddress string
	Asset               string
	Amount              string
	FromWallet          string
	ToWallet            string
	CreatedTime         time.Time
}

// RewardLog AIX reward ledger row.
type RewardLog struct {
	ID             int64
	UserID         int64
	FromUserID     *int64
	OrderID        *int64
	BatchID        *int64
	Type           string
	Asset          string
	Amount         string
	BaseAmount     string
	Rate           string
	ExitApplied    string
	Meta           string
	SettlementDate string
	CreatedTime    time.Time
}

type MgmtRewardSummary struct {
	Released string
	Pending  string
	Total    string
}

type MgmtReward struct {
	ID             int64
	UserID         int64
	FromUserID     int64
	SourceOrderID  int64
	BaseAmount     string
	Rate           string
	TotalAmount    string
	ReleasedAmount string
	PendingAmount  string
	CreatedTime    time.Time
}

// OrderReleaseSummary compatibility for GetBalance mapping
type OrderReleaseSummary struct {
	ExitTotal     string
	ReleasedTotal string
	PendingTotal  string
	UnexitedTotal string
	TotalNodes    int32
}

// Withdrawal legacy stub / 提现模型（含未完成的 USDT/AIX 提现路径）
type Withdrawal struct {
	ID        int64
	UserID    int64
	Address   string
	ToAddress string
	Amount    string
	Fee       string
	NetAmount string
	Status    string
	TxHash      string
	PayoutNonce *uint64
	Asset       string
	Remark      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WithdrawalPayoutAttempt tracks one on-chain payout broadcast.
type WithdrawalPayoutAttempt struct {
	ID          int64
	WithdrawID  int64
	TxHash      string
	Nonce       uint64
	FromAddress string
	ToAddress   string
	Amount      string
	Status      string
	CreatedAt   time.Time
}

// ExchangeRecord AIX → 可提 U（USDT）兑换记录
type ExchangeRecord struct {
	ID            int64
	UserID        int64
	UserAddress   string
	FromAsset     string
	FromAmount    string
	ToAsset       string
	ToAmount      string
	FeeAmount     string
	FeeRate       string
	ExchangePrice string
	Status        string
	Remark        string
	CreatedTime   time.Time
}

// SubscribeInput 报单入参（单源：recharge / reward / win）。
type SubscribeInput struct {
	Amount     string  // 总本金 USDT
	PayFrom    string  // recharge | reward | win
	ExitMul    float64
	DirectRate float64
}

// Order represents an AIX subscribe order.
type Order struct {
	ID           int64
	UserID       int64
	Principal    string
	ExitCap      string
	EarnedTotal  string
	DirectBase   string
	FromRecharge string
	FromReward   string
	FromWin      string
	FromWinA     string
	Points       string // 本单获得 AIX-USDT
	PointsSource string // AIX-USDT 产生来源：recharge | win | transfer_reinvest
	WinPrice     string // 认购时 WIN 价格快照（仅 pay_from=win）
	WinAPrice    string // 认购时 WIN-A 价格快照（仅 pay_from=win_a）
	FundSource   string
	Status       string
	ExitedTime   *time.Time
	CreatedTime  time.Time
	UpdatedTime  time.Time

	// Legacy field aliases for proto/admin mapping
	ProductID      int64
	ProductName    string
	Quantity       int32
	TotalAmount    string
	ExitMultiplier string
	ExitTarget     string
	ReleasedAmount string
	ReleaseDay     int32
	CycleDay       int32
	RateGoingUp    bool
	CreatedAt      time.Time
}

// SyncCompatFields fills legacy order aliases.
func (o *Order) SyncCompatFields() {
	if o == nil {
		return
	}
	o.TotalAmount = o.Principal
	o.ExitTarget = o.ExitCap
	o.ReleasedAmount = o.EarnedTotal
	o.ProductName = o.FundSource
	o.ExitMultiplier = "4"
	o.Quantity = 1
	o.CreatedAt = o.CreatedTime
}

// WalletRepo handles wallet persistence.
type WalletRepo interface {
	CreateRecharge(ctx context.Context, recharge *Recharge) (*Recharge, error)
	FindRecharge(ctx context.Context, id int64) (*Recharge, error)
	FindRechargeByTxHash(ctx context.Context, txHash string) (*Recharge, error)
	ConfirmRecharge(ctx context.Context, id int64, txHash string) error
	ConfirmRechargeCredit(ctx context.Context, id int64, txHash string) (newRechargeBalance string, err error)
	DeletePendingRecharge(ctx context.Context, id int64) error
	AutoCreditChainRecharge(ctx context.Context, txHash, fromAddress, toAddress, amount string, blockNumber uint64) (credited bool, err error)
	// AutoCreditWinRecharge 链上/确认后入账 WIN → win_recharge_balance（按 tx_hash 幂等）
	AutoCreditWinRecharge(ctx context.Context, txHash, fromAddress, toAddress, amount string) (credited bool, newWinRechargeBalance string, err error)
	// AutoCreditWinARecharge 链上入账 WIN-A → win_a_recharge_balance（按 tx_hash 幂等）
	AutoCreditWinARecharge(ctx context.Context, txHash, fromAddress, toAddress, amount string) (credited bool, newWinARechargeBalance string, err error)
	// CreditPartnerWin 合作方转账加款：单事务完成「查用户 → 加 win_recharge_balance → 落流水」。
	// txHash 传合成幂等键（partner:{partner_id}:{nonce}），复用 recharges.tx_hash 唯一索引。
	CreditPartnerWin(ctx context.Context, in PartnerCreditInput) (*PartnerCreditResult, error)
	// SumPartnerCreditedSince 统计某合作方自 since 起已成功加款的总额，用于单日限额。
	SumPartnerCreditedSince(ctx context.Context, partnerID string, since time.Time) (string, error)
	ListRechargesByUser(ctx context.Context, userID int64) ([]*Recharge, error)
	ListRechargesByUserAsset(ctx context.Context, userID int64, asset string) ([]*Recharge, error)
	ListConfirmedUSDTRechargesByUserIDs(ctx context.Context, userIDs []int64, offset, limit int) ([]*Recharge, int64, error)
	// ListOrdersByUserIDs 按用户 ID 集合分页认购订单（含地址），供下级认购列表。
	ListOrdersByUserIDs(ctx context.Context, userIDs []int64, offset, limit int) ([]*AdminOrderDetail, int64, error)

	// Subscribe 单源报单（recharge / reward / win）。
	Subscribe(ctx context.Context, userID int64, in SubscribeInput) (*Order, string, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]*Order, error)
	ListAllOrders(ctx context.Context) ([]*AdminOrderDetail, error)
	// ListSubscribeOrdersPaged 全平台认购订单分页（含用户地址），供第三方 Open API
	ListSubscribeOrdersPaged(ctx context.Context, offset, limit int) (items []*AdminOrderDetail, total int64, err error)
	FindOrder(ctx context.Context, id int64) (*Order, error)
	RemainingExitCapacity(ctx context.Context, userID int64) (string, error)

	CreateTransfer(ctx context.Context, t *Transfer) (*Transfer, error)
	ListTransfersByUser(ctx context.Context, userID int64) ([]*Transfer, error)

	CreateRewardLog(ctx context.Context, log *RewardLog) error
	ListRewardLogsByUser(ctx context.Context, userID int64) ([]*RewardLog, error)
	GetMgmtRewardSummary(ctx context.Context, userID int64) (*MgmtRewardSummary, error)
	GetDirectRewardTotal(ctx context.Context, userID int64) (string, error)
	ListMgmtRewardsByUser(ctx context.Context, userID int64) ([]*MgmtReward, error)

	GetAixPrice(ctx context.Context, date string) (string, error)
	GetLatestAixPriceBefore(ctx context.Context, date string) (string, error)
	UpsertAixPrice(ctx context.Context, date, price, remark string) error
	// GetCurrentWinPrice / UpsertCurrentWinPrice：WIN 现价仅保留一条记录
	GetCurrentWinPrice(ctx context.Context) (string, error)
	UpsertCurrentWinPrice(ctx context.Context, price, source string) error

	// ExchangeAixToWin AIX → 可提 U（USDT）：扣 AixBalance；needReview 时不入账 U，状态为 review。
	ExchangeAixToWin(ctx context.Context, userID int64, aixAmount string, needReview bool) (*ExchangeRecord, string, string, error)
	ApproveExchangeReview(ctx context.Context, id int64) error
	RejectExchangeReview(ctx context.Context, id int64, remark string) error
	// SumStaticAixBySettlementDate 某结算日静态发放的 AIX 枚数合计（reward_logs.amount）。
	SumStaticAixBySettlementDate(ctx context.Context, settlementDate string) (string, error)
	// SumExchangedAixSince 自 since 起已占用当日配额的兑换 AIX 量。
	// 只计 completed：待审核不占配额，因此也不会带到次日阈值。
	SumExchangedAixSince(ctx context.Context, since time.Time) (string, error)
	ListExchangeRecordsByUser(ctx context.Context, userID int64) ([]*ExchangeRecord, error)
	ListAllExchangeRecords(ctx context.Context) ([]*ExchangeRecord, error)

	// CreateWinWithdrawal WIN 代币提现
	CreateWinWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*Withdrawal, string, error)
	// CreateSdtWithdrawal AIX-USDT 提现：扣 points，创建 WithdrawalPO
	CreateSdtWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*Withdrawal, string, error)
	// CreateUsdtWithdrawal 可提 U 余额提现：扣 usdt_withdrawable，创建 WithdrawalPO
	CreateUsdtWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*Withdrawal, string, error)
	ApproveWithdrawalReview(ctx context.Context, id int64) error
	RejectWithdrawalReview(ctx context.Context, id int64, remark string) error
	ListWithdrawalsByUser(ctx context.Context, userID int64) ([]*Withdrawal, error)
	// 提现相关（USDT 审批路径未完成，保留接口）
	CreateWithdrawal(ctx context.Context, userID int64, amount, fee, netAmount, toAddress string) (*Withdrawal, string, error)
	SumWithdrawnByUser(ctx context.Context, userID int64) (string, error)
	ListAllWithdrawals(ctx context.Context) ([]*Withdrawal, error)
	ApproveWithdrawal(ctx context.Context, id int64) error
	// WIN 链上自动打款
	ClaimNextPendingWinWithdrawal(ctx context.Context) (*Withdrawal, error)
	SetWithdrawalTxHash(ctx context.Context, id int64, txHash string) error
	CompleteWithdrawalPayout(ctx context.Context, id int64, txHash string) error
	ReleaseWithdrawalPayout(ctx context.Context, id int64, remark string) error
	ResetWithdrawalForRetry(ctx context.Context, id int64, remark string) error
	HasConfirmedWithdrawalPayout(ctx context.Context, withdrawID int64) (bool, error)
	CreateWithdrawalPayoutAttempt(ctx context.Context, withdrawID int64, txHash, fromAddress, toAddress, amount string, nonce uint64) error
	ListWithdrawalPayoutAttempts(ctx context.Context, withdrawID int64) ([]*WithdrawalPayoutAttempt, error)
	MarkWithdrawalPayoutFailed(ctx context.Context, withdrawID int64, txHash string) error
	SetWithdrawalPayoutNonce(ctx context.Context, id int64, nonce uint64) error
	ListDoingWithdrawalsWithoutTxHash(ctx context.Context) ([]*Withdrawal, error)
	ListStaleDoingWinWithdrawals(ctx context.Context, staleBefore time.Time) ([]*Withdrawal, error)
	ListDoingWithdrawalsWithTxHash(ctx context.Context) ([]*Withdrawal, error)
	AdminUpdateOrder(ctx context.Context, update *AdminOrderUpdate) (*Order, error)
}

// AdminOrderDetail 管理员可见的订单详情
type AdminOrderDetail struct {
	Order       *Order
	UserAddress string
}

// AdminOrderUpdate 管理员修改订单字段
type AdminOrderUpdate struct {
	OrderID        int64
	Quantity       int32
	TotalAmount    string
	Status         string
	ExitMultiplier string
	ExitTarget     string
	ReleasedAmount string
	ReleaseDay     int32
	CycleDay       int32
}

func ParseAmount(amount string) (decimal.Decimal, error) {
	return decimal.NewFromString(amount)
}
