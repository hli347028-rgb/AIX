package biz

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	// AccountFrozenReason 账户被后台冻结时的错误原因码
	AccountFrozenReason = "ACCOUNT_FROZEN"
	// AccountFrozenMessage 账户被后台冻结时的提示文案
	AccountFrozenMessage = "账户已被冻结，请联系客服"
)

// User represents an AIX wallet user.
type User struct {
	ID              int64
	Address         string
	InviteCode      string
	Username        string // 用户展示名（团队内可见）
	UsdtRecharge      string
	UsdtReward        string
	AixBalance        string // AIX 代币数
	WinBalance         string // WIN 提现钱包（AIX 兑换，可提现）
	WinRechargeBalance string // WIN 充值钱包（链上/后台充值，可认购）
	WinARechargeBalance string // WIN-A 充值钱包（链上充值，可认购）
	UsdtWithdrawable   string // 可提 U 余额（0号账户/社区补贴奖励）
	PendingMgmtReward string // 兼容旧字段 = 管理奖溢出
	OverflowReward    string // 管理奖溢出（USDT）
	OverflowDirect    string // 直推奖溢出（USDT）
	Points            string // 当前积分
	PointsAll         string // 累计总积分
	StaticUsdtTotal   string // 静态总收益（USDT）
	MgmtLevel       int32
	LargeAreaPerf   string
	SmallAreaPerf   string
	TeamPerf        string
	IsZeroAccount          bool
	IsCommunitySubsidy     bool
	CommunitySubsidyRate   int32 // 补贴档位 5/10/15
	ZeroAccountSetAt       *time.Time
	CommunitySubsidySetAt  *time.Time
	ZeroAccountRewardTotal string
	CommunitySubsidyTotal  string
	Status          int32
	IsFrozen        bool
	FrozenAt        *time.Time
	ExchangeEnabled bool // 默认 true；后台关闭后禁止 AIX 兑换
	InviterID       *int64
	InviterAddress  string
	Role            string
	CreatedTime     time.Time
	UpdatedTime     time.Time

	// Compatibility aliases for legacy admin/auth mapping
	Balance              string // = UsdtRecharge
	ReleasedBalance      string // = UsdtReward
	CommunityLevel       string // = A{mgmt_level}
	CommunityStake       string // = SmallAreaPerf
	TeamStake            string // = TeamPerf
	ShareProfitTotal     string
	EcoRewardTotal       string
	CommunityLevelLocked bool
	CreatedAt            time.Time
}

// SyncCompatFields fills legacy alias fields from AIX balances.
func (u *User) SyncCompatFields() {
	if u == nil {
		return
	}
	u.Balance = u.UsdtRecharge
	u.ReleasedBalance = u.UsdtReward
	u.CommunityStake = u.SmallAreaPerf
	u.TeamStake = u.TeamPerf
	u.CreatedAt = u.CreatedTime
	if u.MgmtLevel > 0 {
		u.CommunityLevel = "A" + itoa(int(u.MgmtLevel))
	} else {
		u.CommunityLevel = "A0"
	}
	if u.ShareProfitTotal == "" {
		u.ShareProfitTotal = "0"
	}
	if u.EcoRewardTotal == "" {
		u.EcoRewardTotal = "0"
	}
}

// OverflowTotal 溢出奖励合计（管理奖溢出 + 直推奖溢出），用户端/管理端展示用。
func (u *User) OverflowTotal() string {
	if u == nil {
		return "0"
	}
	a, err := decimal.NewFromString(strings.TrimSpace(u.OverflowReward))
	if err != nil {
		a = decimal.Zero
	}
	b, err := decimal.NewFromString(strings.TrimSpace(u.OverflowDirect))
	if err != nil {
		b = decimal.Zero
	}
	return a.Add(b).String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// DownlineInvitee represents a user in the inviter's downline tree.
type DownlineInvitee struct {
	Address    string
	Generation int32
	CreatedAt  time.Time
}

// DirectInvitee 直推下级节点
type DirectInvitee struct {
	Address          string
	Username         string
	TeamStake        string
	ExitAmount       string
	CommunityLevel   string
	ReleasedBalance  string
	ShareProfitTotal string
	EcoRewardTotal   string
	DirectCount      int32
	TeamDownlineCount int32
	PersonalStake    string
	CreatedAt        time.Time
}

const MaxDownlineGenerations = 10

// Challenge represents a login challenge message.
type Challenge struct {
	Address  string
	Message  string
	ExpireAt time.Time
}

// ChallengeRepo stores temporary login challenges.
type ChallengeRepo interface {
	Save(ctx context.Context, challenge *Challenge) error
	Get(ctx context.Context, address string) (*Challenge, error)
	Delete(ctx context.Context, address string) error
}

// UserRepo handles user persistence.
type UserRepo interface {
	FindByAddress(ctx context.Context, address string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user *User) (*User, error)
	CountInvitees(ctx context.Context, userID int64) (int32, error)
	CountSubscribeOrders(ctx context.Context, userID int64) (int64, error)
	ListDownlineInvitees(ctx context.Context, userID int64, maxDepth int) ([]*DownlineInvitee, error)
	ListAllUsers(ctx context.Context) ([]*User, error)
	ListDirectInvitees(ctx context.Context, userID int64) ([]*User, error)
	ListUsersUnder(ctx context.Context, rootID int64) ([]*User, error)
	CountUsersUnder(ctx context.Context, rootID int64) (int32, error)
	ListUserIDsUnder(ctx context.Context, rootID int64) ([]int64, error)
	SumPrincipalByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error)
	// SumCumulativePrincipalByUserIDs 累计认购本金（active+exited），与 team_perf 口径一致
	SumCumulativePrincipalByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error)
	// SumActivePrincipalUnder 伞下所有未出局认购订单本金合计（含本人及全部下级，status=active）
	SumActivePrincipalUnder(ctx context.Context, rootID int64) (string, error)
	// SumExitAmountByUserIDs 兼容旧接口：活跃订单本金合计（业绩）
	SumExitAmountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error)
	UpdateMgmtStats(ctx context.Context, userID int64, level int32, smallArea, teamPerf string) error
	RefreshPerformance(ctx context.Context) error
	RefreshPerformanceFromUsers(ctx context.Context, userIDs ...int64) error
	AdminUpdateUser(ctx context.Context, update *AdminUserUpdate) error
	UpdateUsername(ctx context.Context, userID int64, username string) error
	SetRole(ctx context.Context, userID int64, role string) error
	GetBalances(ctx context.Context, userID int64) (recharge, reward, aix string, err error)
	AddUsdtRecharge(ctx context.Context, userID int64, amount string) (string, error)
	IsUplineOrDownline(ctx context.Context, a, b int64) (bool, error)

	// Legacy stubs kept for compile of unused paths
	SetWithdrawReset(ctx context.Context, userID int64, reset bool) error
	ClearWithdrawReset(ctx context.Context, userID int64) error
	IsWithdrawReset(ctx context.Context, userID int64) (bool, error)
	UpdateCommunityStats(ctx context.Context, userID int64, level string, communityStake, teamStake string) error
	GetBalance(ctx context.Context, userID int64) (string, error)
	GetReleasedBalance(ctx context.Context, userID int64) (string, error)
	AddBalance(ctx context.Context, userID int64, amount string) (string, error)
	AddReleasedBalance(ctx context.Context, userID int64, amount string) (string, error)
	ClaimReleasedToAccount(ctx context.Context, userID int64, amount string) (string, string, error)
	DeductBalance(ctx context.Context, userID int64, amount string) (string, error)
}
