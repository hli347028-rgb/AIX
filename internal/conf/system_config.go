package conf

import (
	"strconv"
	"strings"
)

const SettingsKeySystemConfig = "system_config"

// SystemConfigSnapshot 可热更新的系统参数（AIX）
type SystemConfigSnapshot struct {
	JwtSecret        string   `json:"jwt_secret"`
	ChallengeTTL     string   `json:"challenge_ttl"`
	AdminAddresses   []string `json:"admin_addresses"`
	DepositAddress   string   `json:"deposit_address"`
	DepositAddresses []string `json:"deposit_addresses"`
	UsdtContract     string   `json:"usdt_contract"`
	UsdtDecimals     int32    `json:"usdt_decimals"`
	RPCURL           string   `json:"rpc_url"`
	MinSubscribe     string   `json:"min_subscribe"`

	// AIX 业务参数
	StaticRate            float64   `json:"static_rate"`             // 日静态利率（%），默认 0.5
	ExitMultiplier        float64   `json:"exit_multiplier"`         // 出局倍数，默认 4
	DirectRate            float64   `json:"direct_rate"`             // 直推比例，默认 0.5
	MgmtThresholds        []float64 `json:"mgmt_thresholds"`         // W1–W10 小区业绩门槛
	MgmtRates             []float64 `json:"mgmt_rates"`              // W1–W10 管理奖比例
	AixPriceInitial       float64   `json:"aix_price_initial"`       // 初始 AIX 价格
	WinPrice              float64   `json:"win_price"`               // WIN 代币价格（USDT/枚）
	WinAPrice             float64   `json:"win_a_price"`             // WIN-A 代币价格（USDT/枚）
	ExchangeFeeRate       float64   `json:"exchange_fee_rate"`       // 兑换手续费率（0.05 = 5%）
	MinUsdtRecharge       string    `json:"min_usdt_recharge"`       // USDT 充值最小值（≥10）
	MinWinRecharge        string    `json:"min_win_recharge"`        // WIN 充值最小值（≥10）
	MinWinARecharge       string    `json:"min_win_a_recharge"`      // WIN-A 充值最小值（≥10）
	RegisterBonus         string    `json:"register_bonus"`          // 注册赠送金额，入充值钱包（0=不赠送）
	WinWithdrawReviewThreshold string `json:"win_withdraw_review_threshold"` // WIN 提现审核阈值（≥此金额需审核，0=不审核）
	SdtWithdrawReviewThreshold string `json:"sdt_withdraw_review_threshold"` // AIX-USDT 提现审核阈值
	UsdtWithdrawReviewThreshold string `json:"usdt_withdraw_review_threshold"` // 可提 U 提现审核阈值

	// 交易所划转（合作方转账加款接口 /v1/transfer/credit）限额
	PartnerMinAmount  string `json:"partner_min_amount"`  // 单笔下限
	PartnerMaxAmount  string `json:"partner_max_amount"`  // 单笔上限
	PartnerDailyLimit string `json:"partner_daily_limit"` // 单日累计上限

	// AIX 兑换审核：全网当日已兑换 AIX 超过「今日AIX数量 × 阈值%」后，后续兑换进待审核
	ExchangeReviewThresholdPercent string `json:"exchange_review_threshold_percent"`

	// 用户端「向交易所划转」AIX-USDT 单笔最低额（管理端可配）
	ExchangeTransferMinAmount string `json:"exchange_transfer_min_amount"`

	MgmtCountsTowardExit  bool      `json:"mgmt_counts_toward_exit"` // 管理奖是否计入出局
	MgmtCountsTowardExitP *bool     `json:"-"`                       // 内部：区分 JSON 缺省与 false

	// 子账户（密码与可访问模块，主账户在配置项维护；覆盖 yaml 默认值）
	AdminSubAccounts []AdminSubAccount `json:"admin_sub_accounts,omitempty"`

	// 兼容旧 admin 字段（忽略）
	WithdrawFeeRate float64   `json:"withdraw_fee_rate,omitempty"`
	ReleaseMinRate  float64   `json:"release_min_rate,omitempty"`
	ReleaseMaxRate  float64   `json:"release_max_rate,omitempty"`
	MaxReferralGen  int32     `json:"max_referral_gen,omitempty"`
	ReferralRates   []float64 `json:"referral_rates,omitempty"`
	EcoThresholds   []float64 `json:"eco_thresholds,omitempty"`
	EcoRates        []float64 `json:"eco_rates,omitempty"`
}

const (
	DefaultStaticRate      = 0.5
	DefaultExitMultiplier  = 4.0
	DefaultDirectRate      = 0.5
	DefaultAixPrice              = 1.0
	DefaultAixPriceDailyGrowth   = 0.02 // AIX 每日上涨 2%
	DefaultWinPrice          = 1.0
	DefaultWinAPrice         = 1.0
	DefaultExchangeFeeRate   = 0.05
	DefaultMinSubscribe      = "100"
	DefaultMinUsdtRecharge   = "10"
	DefaultMinWinRecharge    = "10"
	DefaultMinWinARecharge   = "10"
	DefaultRegisterBonus     = "0" // 注册赠送默认关闭，各环境自行开启
	FloorMinRechargeAmount   = 10.0 // 管理端与业务校验的绝对下限

	// 交易所划转限额默认值
	DefaultPartnerMinAmount  = "10"
	DefaultPartnerMaxAmount  = "100000"
	DefaultPartnerDailyLimit = "1000000"

	// 兑换审核阈值（%）：当日已兑换 AIX 超过「今日AIX × 该百分比」后，后续兑换需审核。默认 100=不提前触发。
	DefaultExchangeReviewThresholdPercent = "100"

	// 向交易所划转 AIX-USDT 单笔最低额
	DefaultExchangeTransferMinAmount = "500"
)

// DefaultMgmtThresholds W1→W10 小区业绩门槛（USDT）
func DefaultMgmtThresholds() []float64 {
	return []float64{5000, 20000, 50000, 200000, 500000, 1500000, 4000000, 8000000, 15000000, 30000000}
}

// DefaultMgmtRates W1→W10 管理奖比例（如 0.2 = 20%）
func DefaultMgmtRates() []float64 {
	return []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1}
}

// NormalizeBusinessDefaults 补齐 AIX 业务参数默认值
func NormalizeBusinessDefaults(s *SystemConfigSnapshot) {
	if s == nil {
		return
	}
	if s.StaticRate <= 0 {
		s.StaticRate = DefaultStaticRate
	}
	if s.ExitMultiplier <= 0 {
		s.ExitMultiplier = DefaultExitMultiplier
	}
	if s.DirectRate <= 0 {
		s.DirectRate = DefaultDirectRate
	}
	if s.AixPriceInitial <= 0 {
		s.AixPriceInitial = DefaultAixPrice
	}
	if s.WinPrice <= 0 {
		s.WinPrice = DefaultWinPrice
	}
	// WIN-A 价格始终等于 WIN 价格
	s.WinAPrice = s.WinPrice
	if s.ExchangeFeeRate <= 0 {
		s.ExchangeFeeRate = DefaultExchangeFeeRate
	}
	if s.MinSubscribe == "" {
		s.MinSubscribe = DefaultMinSubscribe
	}
	s.MinUsdtRecharge = normalizeMinRecharge(s.MinUsdtRecharge, DefaultMinUsdtRecharge)
	s.MinWinRecharge = normalizeMinRecharge(s.MinWinRecharge, DefaultMinWinRecharge)
	s.MinWinARecharge = normalizeMinRecharge(s.MinWinARecharge, DefaultMinWinARecharge)
	if strings.TrimSpace(s.RegisterBonus) == "" {
		s.RegisterBonus = DefaultRegisterBonus
	}
	s.PartnerMinAmount = normalizePositiveAmount(s.PartnerMinAmount, DefaultPartnerMinAmount)
	s.PartnerMaxAmount = normalizePositiveAmount(s.PartnerMaxAmount, DefaultPartnerMaxAmount)
	s.PartnerDailyLimit = normalizePositiveAmount(s.PartnerDailyLimit, DefaultPartnerDailyLimit)
	s.ExchangeReviewThresholdPercent = normalizePercent(s.ExchangeReviewThresholdPercent, DefaultExchangeReviewThresholdPercent)
	s.ExchangeTransferMinAmount = normalizePositiveAmount(s.ExchangeTransferMinAmount, DefaultExchangeTransferMinAmount)
	if len(s.MgmtThresholds) != 10 {
		s.MgmtThresholds = DefaultMgmtThresholds()
	}
	if len(s.MgmtRates) != 10 {
		s.MgmtRates = DefaultMgmtRates()
	}
	// 缺省时默认计入出局
	if s.MgmtCountsTowardExitP == nil && !s.MgmtCountsTowardExit {
		// JSON 反序列化后若显式 false 会保留；首次 Normalize 时若两者皆零则置 true
		// 使用：若从未设置过业务开关，默认 true
		s.MgmtCountsTowardExit = true
	}
}

// normalizePercent 空值或非法值回落默认；允许 0（表示一兑换就审）。
func normalizePercent(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return fallback
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// normalizePositiveAmount 空值或非正数回落到默认值。
// 与 normalizeMinRecharge 的区别：不强制抬到 FloorMinRechargeAmount，
// 因为上限/日限额没有那条 ≥10 的业务下限。
func normalizePositiveAmount(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// normalizeMinRecharge 空值回落默认；若解析失败或 < 绝对下限，则抬到下限。
func normalizeMinRecharge(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < FloorMinRechargeAmount {
		return strconv.FormatFloat(FloorMinRechargeAmount, 'f', -1, 64)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
