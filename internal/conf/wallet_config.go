package conf

import (
	"os"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	DefaultDepositContract     = "0xa5A438Bb1D0F702c684B4d7bAAE2C520aFb4aE86"
	DefaultWinDepositContract  = "0x94db6bb040107ef9a2F1e9DB9d84dD8D6D98997e"
	DefaultWinADepositContract = "0xcaa39A8E23F5548AD85d9e2B9B21F63E99505040"
	DefaultSdtContract         = "0x314D550572a0fA001B465a9EBc1dd04D834a0688"
	DefaultRPCURL              = "https://rpc1.eoeo.info"
	DefaultBscRPCURL           = "https://bsc-dataseed.binance.org"
)

// WalletConfig holds wallet and recharge settings.
type WalletConfig struct {
	DepositAddress                   string   `json:"deposit_address" yaml:"deposit_address"`
	DepositAddresses                 []string `json:"deposit_addresses" yaml:"deposit_addresses"`
	DepositContract                  string   `json:"deposit_contract" yaml:"deposit_contract"`             // USDT BuySomething
	WinDepositContract               string   `json:"win_deposit_contract" yaml:"win_deposit_contract"`     // 原生 WIN BuySomething
	WinADepositContract              string   `json:"win_a_deposit_contract" yaml:"win_a_deposit_contract"` // WIN-A BuySomething
	WinARechargeEnabled              *bool    `json:"win_a_recharge_enabled" yaml:"win_a_recharge_enabled"`   // WIN-A 链上充值开关（nil/true=开放）
	UsdtContract                     string   `json:"usdt_contract" yaml:"usdt_contract"`
	UsdtDecimals                     int32    `json:"usdt_decimals" yaml:"usdt_decimals"`
	WinContract                      string   `json:"win_contract" yaml:"win_contract"`
	WinDecimals                      int32    `json:"win_decimals" yaml:"win_decimals"`
	WinAContract                     string   `json:"win_a_contract" yaml:"win_a_contract"` // WIN-A ERC20 代币
	WinADecimals                     int32    `json:"win_a_decimals" yaml:"win_a_decimals"`
	SdtContract                      string   `json:"sdt_contract" yaml:"sdt_contract"`
	SdtDecimals                      int32    `json:"sdt_decimals" yaml:"sdt_decimals"`
	RPCURL                           string   `json:"rpc_url" yaml:"rpc_url"`         // EOEO（WIN 充值 / 价格 / 提现）
	BscRPCURL                        string   `json:"bsc_rpc_url" yaml:"bsc_rpc_url"` // BSC（USDT 充值）
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
	SdtPrivateKey                      string `json:"sdt_private_key" yaml:"sdt_private_key"`
	SdtPrivateKeyFile                  string `json:"sdt_private_key_file" yaml:"sdt_private_key_file"`
	WithdrawPayoutQueriesPerCycle      int32  `json:"withdraw_payout_queries_per_cycle" yaml:"withdraw_payout_queries_per_cycle"`
	WithdrawPayoutQueryIntervalSeconds int64  `json:"withdraw_payout_query_interval_seconds" yaml:"withdraw_payout_query_interval_seconds"`
	// AVE Cloud Data API（首页 K 线代理）
	AveAPIKey          string `json:"ave_api_key" yaml:"ave_api_key"`
	AveAPIKeyFile      string `json:"ave_api_key_file" yaml:"ave_api_key_file"`
	AveKlineBaseURL    string `json:"ave_kline_base_url" yaml:"ave_kline_base_url"`
	AveKlineTokenID    string `json:"ave_kline_token_id" yaml:"ave_kline_token_id"`

	// 向交易所划转 AIX-USDT（WinBit 入金 /v2/winA/aixInbound；enabled=false 或密钥未配齐时不开通）
	ExchangeTransferEnabled        bool     `json:"exchange_transfer_enabled" yaml:"exchange_transfer_enabled"`
	ExchangeTransferAPIURL         string   `json:"exchange_transfer_api_url" yaml:"exchange_transfer_api_url"`
	ExchangeTransferPartnerID      string   `json:"exchange_transfer_partner_id" yaml:"exchange_transfer_partner_id"`
	ExchangeTransferSecretKeys     []string `json:"exchange_transfer_secret_keys" yaml:"exchange_transfer_secret_keys"`
	ExchangeTransferSecretKeysFile string   `json:"exchange_transfer_secret_keys_file" yaml:"exchange_transfer_secret_keys_file"`
}

const (
	DefaultAveKlineBaseURL = "https://prod.ave-api.com"
	// WIN on WIN Chain (same link as futurefi.vue AVE page)
	DefaultAveKlineTokenID = "0x193013574dacbd38bf26ecb654b3fd787b94d216-winchain"
	DefaultWinPair         = "0x15ad085fc866370b59936575565434b14d22281d"
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

func (w *WalletConfig) GetWinADepositContract() string {
	if w == nil || strings.TrimSpace(w.WinADepositContract) == "" {
		return DefaultWinADepositContract
	}
	return strings.TrimSpace(w.WinADepositContract)
}

func (w *WalletConfig) IsWinARechargeEnabled() bool {
	return false // WIN-A 充值已关闭
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

func (w *WalletConfig) GetWinAContract() string {
	if w == nil {
		return ""
	}
	return strings.TrimSpace(w.WinAContract)
}

func (w *WalletConfig) GetWinADecimals() int32 {
	if w == nil || w.WinADecimals <= 0 {
		return 18
	}
	return w.WinADecimals
}

func (w *WalletConfig) GetSdtContract() string {
	if w == nil || strings.TrimSpace(w.SdtContract) == "" {
		return DefaultSdtContract
	}
	return strings.TrimSpace(w.SdtContract)
}

func (w *WalletConfig) GetSdtDecimals() int32 {
	if w == nil || w.SdtDecimals <= 0 {
		return 18
	}
	return w.SdtDecimals
}

func (w *WalletConfig) GetRPCURL() string {
	if w == nil || strings.TrimSpace(w.RPCURL) == "" {
		return DefaultRPCURL
	}
	return strings.TrimSpace(w.RPCURL)
}

// GetBscRPCURL is used to scan the USDT BuySomething contract on BSC.
func (w *WalletConfig) GetBscRPCURL() string {
	if w == nil || strings.TrimSpace(w.BscRPCURL) == "" {
		return DefaultBscRPCURL
	}
	return strings.TrimSpace(w.BscRPCURL)
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

// GetSdtPrivateKey reads AIX-USDT payout key; falls back to WIN withdraw key when unset.
func (w *WalletConfig) GetSdtPrivateKey() string {
	if env := strings.TrimSpace(os.Getenv("AIX_SDT_WITHDRAW_PRIVATE_KEY")); env != "" {
		return strings.TrimPrefix(env, "0x")
	}
	if w != nil {
		if path := strings.TrimSpace(w.SdtPrivateKeyFile); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				key := strings.TrimSpace(string(b))
				if key != "" {
					return strings.TrimPrefix(key, "0x")
				}
			}
		}
		if key := strings.TrimSpace(w.SdtPrivateKey); key != "" {
			return strings.TrimPrefix(key, "0x")
		}
	}
	return w.GetWithdrawPrivateKey()
}

// GetAveAPIKey reads AVE_API_KEY env, then ave_api_key_file, then config field.
func (w *WalletConfig) GetAveAPIKey() string {
	if env := strings.TrimSpace(os.Getenv("AVE_API_KEY")); env != "" {
		return env
	}
	if w != nil {
		if path := strings.TrimSpace(w.AveAPIKeyFile); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				if key := strings.TrimSpace(string(b)); key != "" {
					return key
				}
			}
		}
		if key := strings.TrimSpace(w.AveAPIKey); key != "" {
			return key
		}
	}
	return ""
}

func (w *WalletConfig) GetAveKlineBaseURL() string {
	if w != nil {
		if u := strings.TrimSpace(w.AveKlineBaseURL); u != "" {
			return u
		}
	}
	return DefaultAveKlineBaseURL
}

func (w *WalletConfig) GetAveKlineTokenID() string {
	if w != nil {
		if id := strings.TrimSpace(w.AveKlineTokenID); id != "" {
			return id
		}
	}
	return DefaultAveKlineTokenID
}

// ExchangeTransferConfigured reports whether outbound exchange transfer can be attempted.
func (w *WalletConfig) ExchangeTransferConfigured() bool {
	if w == nil || !w.ExchangeTransferEnabled {
		return false
	}
	if strings.TrimSpace(w.ExchangeTransferAPIURL) == "" {
		return false
	}
	if strings.TrimSpace(w.ExchangeTransferPartnerID) == "" {
		return false
	}
	return len(w.ExchangeTransferActiveSecrets()) > 0
}

// ExchangeTransferActiveSecrets returns signing secrets for outbound exchange transfer.
// Priority: AIX_EXCHANGE_TRANSFER_SECRET → secret file → inline keys.
func (w *WalletConfig) ExchangeTransferActiveSecrets() []string {
	if w == nil {
		return nil
	}
	if env := strings.TrimSpace(os.Getenv("AIX_EXCHANGE_TRANSFER_SECRET")); env != "" {
		return nonEmptyLines(strings.Split(env, ","))
	}
	if path := strings.TrimSpace(w.ExchangeTransferSecretKeysFile); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			if keys := nonEmptyLines(strings.Split(string(b), "\n")); len(keys) > 0 {
				return keys
			}
		}
	}
	return nonEmptyLines(w.ExchangeTransferSecretKeys)
}

func nonEmptyLines(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, s)
	}
	return out
}
