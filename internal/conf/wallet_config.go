package conf

import (
	"os"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	DefaultDepositContract    = "0xa5A438Bb1D0F702c684B4d7bAAE2C520aFb4aE86"
	DefaultWinDepositContract = "0x94db6bb040107ef9a2F1e9DB9d84dD8D6D98997e"
	DefaultRPCURL             = "https://rpc1.eoeo.info"
)

// WalletConfig holds wallet and recharge settings.
type WalletConfig struct {
	DepositAddress                   string   `json:"deposit_address" yaml:"deposit_address"`
	DepositAddresses                 []string `json:"deposit_addresses" yaml:"deposit_addresses"`
	DepositContract                  string   `json:"deposit_contract" yaml:"deposit_contract"`         // USDT BuySomething
	WinDepositContract               string   `json:"win_deposit_contract" yaml:"win_deposit_contract"` // 原生 WIN BuySomething
	UsdtContract                     string   `json:"usdt_contract" yaml:"usdt_contract"`
	UsdtDecimals                     int32    `json:"usdt_decimals" yaml:"usdt_decimals"`
	WinContract                      string   `json:"win_contract" yaml:"win_contract"`
	WinDecimals                      int32    `json:"win_decimals" yaml:"win_decimals"`
	RPCURL                           string   `json:"rpc_url" yaml:"rpc_url"`
	RechargeMonitorEnabled           bool     `json:"recharge_monitor_enabled" yaml:"recharge_monitor_enabled"`
	RechargeScanIntervalSeconds      int64    `json:"recharge_scan_interval_seconds" yaml:"recharge_scan_interval_seconds"`
	RechargeScanQueriesPerCycle      int32    `json:"recharge_scan_queries_per_cycle" yaml:"recharge_scan_queries_per_cycle"`
	RechargeScanQueryIntervalSeconds int64    `json:"recharge_scan_query_interval_seconds" yaml:"recharge_scan_query_interval_seconds"`
	RechargeConfirmations            uint64   `json:"recharge_confirmations" yaml:"recharge_confirmations"`
	RechargeScanStartBlock           uint64   `json:"recharge_scan_start_block" yaml:"recharge_scan_start_block"`
	RechargeScanLookbackBlocks       uint64   `json:"recharge_scan_lookback_blocks" yaml:"recharge_scan_lookback_blocks"`
	RechargeScanBatchBlocks          uint64   `json:"recharge_scan_batch_blocks" yaml:"recharge_scan_batch_blocks"`
	MinSubscribe                     string   `json:"min_subscribe" yaml:"min_subscribe"`
	MinWithdraw                      string   `json:"min_withdraw" yaml:"min_withdraw"`
	WithdrawFeeRate                  float64  `json:"withdraw_fee_rate" yaml:"withdraw_fee_rate"`
	// WinSwap V2 Pair（WWIN/USDT），用于链上轮询 WIN 价格
	WinPair                string `json:"win_pair" yaml:"win_pair"`
	WinPriceOracleEnabled  bool   `json:"win_price_oracle_enabled" yaml:"win_price_oracle_enabled"`
	WinPricePollSeconds    int64  `json:"win_price_poll_seconds" yaml:"win_price_poll_seconds"`                       // 每轮周期（默认 60 秒）
	WinPriceQueriesPerCycle int32 `json:"win_price_queries_per_cycle" yaml:"win_price_queries_per_cycle"`             // 每轮查询次数（默认 10）
	WinPriceQueryIntervalSeconds int64 `json:"win_price_query_interval_seconds" yaml:"win_price_query_interval_seconds"` // 相邻查询间隔（默认 5 秒）
	// WIN 提现链上自动打款（私钥仅部署在服务器，勿提交到 git）
	WithdrawPayoutEnabled              bool   `json:"withdraw_payout_enabled" yaml:"withdraw_payout_enabled"`
	WithdrawPrivateKey                 string `json:"withdraw_private_key" yaml:"withdraw_private_key"`
	WithdrawPrivateKeyFile             string `json:"withdraw_private_key_file" yaml:"withdraw_private_key_file"`
	WithdrawPayoutQueriesPerCycle      int32  `json:"withdraw_payout_queries_per_cycle" yaml:"withdraw_payout_queries_per_cycle"`
	WithdrawPayoutQueryIntervalSeconds int64  `json:"withdraw_payout_query_interval_seconds" yaml:"withdraw_payout_query_interval_seconds"`
}

const (
	DefaultWinPair                     = "0x15ad085fc866370b59936575565434b14d22281d"
	DefaultWinPricePollSeconds         = int64(60)
	DefaultWinPriceQueriesPerCycle     = int32(10)
	DefaultWinPriceQueryIntervalSeconds = int64(5)
	DefaultWithdrawPayoutQueriesPerCycle     = int32(10)
	DefaultWithdrawPayoutQueryIntervalSeconds = int64(5)
)

func (w *WalletConfig) GetDepositContract() string {
	if w == nil || strings.TrimSpace(w.DepositContract) == "" {
		return DefaultDepositContract
	}
	return strings.TrimSpace(w.DepositContract)
}

func (w *WalletConfig) GetWinDepositContract() string {
	if w == nil || strings.TrimSpace(w.WinDepositContract) == "" {
		return DefaultWinDepositContract
	}
	return strings.TrimSpace(w.WinDepositContract)
}

func (w *WalletConfig) GetRechargeScanIntervalSeconds() int64 {
	if w == nil || w.RechargeScanIntervalSeconds <= 0 {
		return 60
	}
	return w.RechargeScanIntervalSeconds
}

func (w *WalletConfig) GetRechargeScanQueriesPerCycle() int32 {
	if w == nil || w.RechargeScanQueriesPerCycle <= 0 {
		return 10
	}
	return w.RechargeScanQueriesPerCycle
}

func (w *WalletConfig) GetRechargeScanQueryIntervalSeconds() int64 {
	if w == nil || w.RechargeScanQueryIntervalSeconds <= 0 {
		return 5
	}
	return w.RechargeScanQueryIntervalSeconds
}

func (w *WalletConfig) GetRechargeConfirmations() uint64 {
	if w == nil || w.RechargeConfirmations == 0 {
		return 3
	}
	return w.RechargeConfirmations
}

func (w *WalletConfig) GetRechargeScanLookbackBlocks() uint64 {
	if w == nil || w.RechargeScanLookbackBlocks == 0 {
		return 5000
	}
	return w.RechargeScanLookbackBlocks
}

func (w *WalletConfig) GetRechargeScanBatchBlocks() uint64 {
	if w == nil || w.RechargeScanBatchBlocks == 0 {
		return 1000
	}
	return w.RechargeScanBatchBlocks
}

// GetDepositAddresses 返回全部收款地址（去重，顺序：deposit_addresses 再 deposit_address）。
// deposit_address 支持逗号/空白分隔多个地址。
func (w *WalletConfig) GetDepositAddresses() []string {
	if w == nil {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(w.DepositAddresses)+1)
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			key := strings.ToLower(p)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, p)
		}
	}
	for _, a := range w.DepositAddresses {
		add(a)
	}
	add(w.DepositAddress)
	return out
}

func (w *WalletConfig) GetDepositAddress() string {
	addrs := w.GetDepositAddresses()
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// SetDepositAddresses 写入收款地址列表，并同步主展示地址。
func (w *WalletConfig) SetDepositAddresses(addrs []string) {
	if w == nil {
		return
	}
	w.DepositAddresses = nil
	w.DepositAddress = ""
	normalized := (&WalletConfig{DepositAddresses: addrs}).GetDepositAddresses()
	w.DepositAddresses = normalized
	if len(normalized) > 0 {
		w.DepositAddress = normalized[0]
	}
}

func (w *WalletConfig) GetUsdtContract() string {
	if w == nil {
		return ""
	}
	return w.UsdtContract
}

func (w *WalletConfig) GetUsdtDecimals() int32 {
	if w == nil || w.UsdtDecimals <= 0 {
		return 6
	}
	return w.UsdtDecimals
}

func (w *WalletConfig) GetWinContract() string {
	if w == nil {
		return ""
	}
	return strings.TrimSpace(w.WinContract)
}

func (w *WalletConfig) GetWinDecimals() int32 {
	if w == nil || w.WinDecimals <= 0 {
		return 18
	}
	return w.WinDecimals
}

func (w *WalletConfig) GetRPCURL() string {
	if w == nil || strings.TrimSpace(w.RPCURL) == "" {
		return DefaultRPCURL
	}
	return strings.TrimSpace(w.RPCURL)
}

// IsRechargeMonitorEnabled controls the in-process depositOnly ticker.
// When false, only cron/HTTP may trigger USDT/WIN deposit sync (avoids double-scan with crontab).
func (w *WalletConfig) IsRechargeMonitorEnabled() bool {
	if w == nil {
		return true
	}
	return w.RechargeMonitorEnabled
}

func (w *WalletConfig) GetMinSubscribe() string {
	if w == nil || w.MinSubscribe == "" {
		return "100"
	}
	return w.MinSubscribe
}

func (w *WalletConfig) GetMinWithdraw() string {
	if w == nil || w.MinWithdraw == "" {
		return "10"
	}
	return w.MinWithdraw
}

func (w *WalletConfig) GetWithdrawFeeRate() decimal.Decimal {
	if w == nil || w.WithdrawFeeRate <= 0 {
		return decimal.NewFromFloat(0.06)
	}
	return decimal.NewFromFloat(w.WithdrawFeeRate)
}

func (w *WalletConfig) GetWinPair() string {
	if w == nil || strings.TrimSpace(w.WinPair) == "" {
		return DefaultWinPair
	}
	return strings.TrimSpace(w.WinPair)
}

func (w *WalletConfig) IsWinPriceOracleEnabled() bool {
	if w == nil {
		return true
	}
	return w.WinPriceOracleEnabled
}

func (w *WalletConfig) GetWinPricePollSeconds() int64 {
	if w == nil || w.WinPricePollSeconds <= 0 {
		return DefaultWinPricePollSeconds
	}
	return w.WinPricePollSeconds
}

func (w *WalletConfig) GetWinPriceQueriesPerCycle() int32 {
	if w == nil || w.WinPriceQueriesPerCycle <= 0 {
		return DefaultWinPriceQueriesPerCycle
	}
	return w.WinPriceQueriesPerCycle
}

func (w *WalletConfig) GetWinPriceQueryIntervalSeconds() int64 {
	if w == nil || w.WinPriceQueryIntervalSeconds <= 0 {
		return DefaultWinPriceQueryIntervalSeconds
	}
	return w.WinPriceQueryIntervalSeconds
}

func (w *WalletConfig) IsWithdrawPayoutEnabled() bool {
	if w == nil {
		return false
	}
	return w.WithdrawPayoutEnabled
}

func (w *WalletConfig) GetWithdrawPayoutQueriesPerCycle() int32 {
	if w == nil || w.WithdrawPayoutQueriesPerCycle <= 0 {
		return DefaultWithdrawPayoutQueriesPerCycle
	}
	return w.WithdrawPayoutQueriesPerCycle
}

func (w *WalletConfig) GetWithdrawPayoutQueryIntervalSeconds() int64 {
	if w == nil || w.WithdrawPayoutQueryIntervalSeconds <= 0 {
		return DefaultWithdrawPayoutQueryIntervalSeconds
	}
	return w.WithdrawPayoutQueryIntervalSeconds
}

// GetWithdrawPrivateKey 优先读环境变量 AIX_WITHDRAW_PRIVATE_KEY，其次 key 文件，最后 config 字段。
func (w *WalletConfig) GetWithdrawPrivateKey() string {
	if env := strings.TrimSpace(os.Getenv("AIX_WITHDRAW_PRIVATE_KEY")); env != "" {
		return strings.TrimPrefix(env, "0x")
	}
	if w != nil {
		if path := strings.TrimSpace(w.WithdrawPrivateKeyFile); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				key := strings.TrimSpace(string(b))
				if key != "" {
					return strings.TrimPrefix(key, "0x")
				}
			}
		}
		if key := strings.TrimSpace(w.WithdrawPrivateKey); key != "" {
			return strings.TrimPrefix(key, "0x")
		}
	}
	return ""
}
