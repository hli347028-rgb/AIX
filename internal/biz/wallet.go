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
	PayFromReward   = "reward"
	PayFromWin      = "win" // 用 WIN 充值钱包按 win_price 折算认购（产生直推）

	OrderStatusActive = "active"
	OrderStatusExited = "exited"

	// Legacy aliases
	OrderStatusCompleted    = "exited"
	WithdrawStatusPending   = "pending"
	WithdrawStatusDoing     = "doing"
	WithdrawStatusCompleted = "completed"
	WithdrawStatusFailed    = "failed"

	PayoutStatusSubmitted = "submitted"
	PayoutStatusConfirmed = "confirmed"
	PayoutStatusFailed    = "failed"

	TokenUSDT = "USDT"
	TokenAIX  = "AIX"
	TokenWIN  = "WIN"
	TokenSDT  = "SDT"

	RewardTypeStaticAix         = "static_aix"
	RewardTypeDynamicUsdt       = "dynamic_usdt"
	RewardTypeDirectPoolRelease = "direct_pool_release"
	RewardTypeMgmt              = "mgmt"
	RewardTypeMgmtPoolRelease   = "mgmt_pool_release"
	RewardTypeExitAccel         = "exit_accel"
	RewardTypeTransferIn        = "transfer_in"
	RewardTypeTransferOut       = "transfer_out"
)

// GetWinPrice 返回 WIN 代币价格（USDT/枚）。
// 由 WinPriceOracleJob 每分钟从链上 Pair 储备更新；管理后台亦可手动覆盖。
func GetWinPrice() float64 {
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

// GetMinWinRecharge 返回 WIN 充值最小值（管理端可配，绝对下限 10）。
func GetMinWinRecharge() string {
	if strings.TrimSpace(MinWinRecharge) == "" {
		return conf.DefaultMinWinRecharge
	}
	return MinWinRecharge
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

type SelfTransferRecord struct {
	ID          int64
	Asset       string
	Amount      string
	FromWallet  string
	ToWallet    string
	CreatedTime time.Time
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

// ExchangeRecord AIX → WIN 兑换记录
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
	Points       string // 本单获得积分（= 认购金额）
	WinPrice     string // 认购时 WIN 价格快照（仅 pay_from=win）
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
	ListRechargesByUser(ctx context.Context, userID int64) ([]*Recharge, error)
	ListRechargesByUserAsset(ctx context.Context, userID int64, asset string) ([]*Recharge, error)

	Subscribe(ctx context.Context, userID int64, amount, payFrom string, exitMul, directRate float64) (*Order, string, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]*Order, error)
	ListAllOrders(ctx context.Context) ([]*AdminOrderDetail, error)
	// ListSubscribeOrdersPaged 全平台认购订单分页（含用户地址），供第三方 Open API
	ListSubscribeOrdersPaged(ctx context.Context, offset, limit int) (items []*AdminOrderDetail, total int64, err error)
	FindOrder(ctx context.Context, id int64) (*Order, error)
	RemainingExitCapacity(ctx context.Context, userID int64) (string, error)

	CreateTransfer(ctx context.Context, t *Transfer) (*Transfer, error)
	ListTransfersByUser(ctx context.Context, userID int64) ([]*Transfer, error)
	// MoveRechargeToReward 同用户：充值钱包 USDT → 奖励钱包（不触发直推）
	MoveRechargeToReward(ctx context.Context, userID int64, amount string) (rechargeBal, rewardBal string, err error)

	CreateRewardLog(ctx context.Context, log *RewardLog) error
	ListRewardLogsByUser(ctx context.Context, userID int64) ([]*RewardLog, error)
	GetMgmtRewardSummary(ctx context.Context, userID int64) (*MgmtRewardSummary, error)
	ListMgmtRewardsByUser(ctx context.Context, userID int64) ([]*MgmtReward, error)

	GetAixPrice(ctx context.Context, date string) (string, error)
	UpsertAixPrice(ctx context.Context, date, price, remark string) error
	// GetCurrentWinPrice / UpsertCurrentWinPrice：WIN 现价仅保留一条记录
	GetCurrentWinPrice(ctx context.Context) (string, error)
	UpsertCurrentWinPrice(ctx context.Context, price, source string) error

	// ExchangeAixToWin AIX → WIN 兑换：扣 AixBalance，加 WinBalance（提现钱包），记录 ExchangeRecord
	ExchangeAixToWin(ctx context.Context, userID int64, aixAmount string) (*ExchangeRecord, string, string, error)
	ListExchangeRecordsByUser(ctx context.Context, userID int64) ([]*ExchangeRecord, error)
	ListAllExchangeRecords(ctx context.Context) ([]*ExchangeRecord, error)

	// CreateWinWithdrawal WIN 代币提现
	CreateWinWithdrawal(ctx context.Context, userID int64, amount, toAddress string) (*Withdrawal, string, error)
	// CreateSdtWithdrawal AIX-SDT 提现：扣 points，创建 WithdrawalPO
	CreateSdtWithdrawal(ctx context.Context, userID int64, amount, toAddress string) (*Withdrawal, string, error)
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
