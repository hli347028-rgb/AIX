package conf

import (
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// TransferPartnerConfig 合作方转账加款接口配置（POST /v1/transfer/credit）。
//
// 与 AuthConfig.OpenAPIKeys 是两套独立的认证体系，不可混用：
// Open API 是 API Key 查询只读数据，本接口是 HMAC 双向签名的资金加款。
type TransferPartnerConfig struct {
	// Partners 已开通的合作方；未列出的 partner_id 一律拒绝。
	Partners []TransferPartner `json:"partners" yaml:"partners"`
	// TimestampSkew 允许的时间戳偏差，默认 300s（文档 §5 第 4 步）。
	TimestampSkew string `json:"timestamp_skew" yaml:"timestamp_skew"`
	// IPRateLimitPerSec 验签前按客户端 IP 的限速，防未认证流量消耗资源。
	IPRateLimitPerSec int `json:"ip_rate_limit_per_sec" yaml:"ip_rate_limit_per_sec"`
}

// TransferPartner 单个合作方。
type TransferPartner struct {
	PartnerID string `json:"partner_id" yaml:"partner_id"`
	// SecretKeys 支持多把密钥并行，用于零停机轮换：过渡窗口内新旧密钥都能验签通过。
	// 密钥不应写在这里（config.yaml 受 git 跟踪），生产请用 SecretKeysFile。
	SecretKeys []string `json:"secret_keys" yaml:"secret_keys"`
	// SecretKeysFile 密钥文件路径，每行一把（如 /opt/aix/secrets/partner-AIX10001.key）。
	// 与 withdraw_private_key_file 同样的外置做法，避免密钥进仓库。
	SecretKeysFile string `json:"secret_keys_file" yaml:"secret_keys_file"`
	// Enabled 为 false 时返回 1006（合作方被禁用），与 1002（不存在）区分开。
	Enabled bool `json:"enabled" yaml:"enabled"`
	// 以下三个限额是该合作方的单独覆盖，留空即使用管理端配置项里的全局值。
	MinAmount       string `json:"min_amount" yaml:"min_amount"`
	MaxAmount       string `json:"max_amount" yaml:"max_amount"`
	DailyLimit      string `json:"daily_limit" yaml:"daily_limit"`
	RateLimitPerSec int    `json:"rate_limit_per_sec" yaml:"rate_limit_per_sec"`
}

const (
	defaultTimestampSkew    = 300 * time.Second
	defaultIPRateLimit      = 20
	defaultPartnerRateLimit = 10
)

// SkewDuration 返回允许的时间戳偏差。
// 该值同时决定 nonce 的保留窗口：nonce TTL 不得小于它，否则会出现
// 「nonce 已过期但时间戳仍有效」的重放空档（文档 §5）。
func (c *TransferPartnerConfig) SkewDuration() time.Duration {
	if c == nil || strings.TrimSpace(c.TimestampSkew) == "" {
		return defaultTimestampSkew
	}
	d, err := time.ParseDuration(strings.TrimSpace(c.TimestampSkew))
	if err != nil || d <= 0 {
		return defaultTimestampSkew
	}
	return d
}

func (c *TransferPartnerConfig) IPRateLimit() int {
	if c == nil || c.IPRateLimitPerSec <= 0 {
		return defaultIPRateLimit
	}
	return c.IPRateLimitPerSec
}

// FindPartner 按 partner_id 精确查找（大小写敏感，与签名字段保持一致）。
func (c *TransferPartnerConfig) FindPartner(partnerID string) *TransferPartner {
	partnerID = strings.TrimSpace(partnerID)
	if c == nil || partnerID == "" {
		return nil
	}
	for i := range c.Partners {
		if strings.TrimSpace(c.Partners[i].PartnerID) == partnerID {
			return &c.Partners[i]
		}
	}
	return nil
}

// ActiveSecrets 返回该合作方当前可用的密钥，为空表示无法通过验签。
//
// 来源优先级与 withdraw 私钥一致：环境变量 → 密钥文件 → config 内联字段。
// 环境变量名为 AIX_PARTNER_SECRET_<PARTNER_ID>，多把用逗号分隔。
func (p *TransferPartner) ActiveSecrets() []string {
	if p == nil {
		return nil
	}
	envName := "AIX_PARTNER_SECRET_" + strings.ToUpper(strings.TrimSpace(p.PartnerID))
	if env := strings.TrimSpace(os.Getenv(envName)); env != "" {
		return nonEmptyTrimmed(strings.Split(env, ","))
	}
	if path := strings.TrimSpace(p.SecretKeysFile); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			if keys := nonEmptyTrimmed(strings.Split(string(b), "\n")); len(keys) > 0 {
				return keys
			}
		}
	}
	return nonEmptyTrimmed(p.SecretKeys)
}

func nonEmptyTrimmed(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		// 支持密钥文件里用 # 写注释标注轮换日期
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (p *TransferPartner) RateLimit() int {
	if p == nil || p.RateLimitPerSec <= 0 {
		return defaultPartnerRateLimit
	}
	return p.RateLimitPerSec
}

// 三个限额均为「合作方级覆盖」：留空返回零值，由调用方回落到管理端的全局配置。
// 因此这里刻意不带默认值兜底——若在这里兜底，管理端改了也不会生效。

// MinAmountDecimal 单笔下限；未配置返回零值。
func (p *TransferPartner) MinAmountDecimal() decimal.Decimal {
	return optionalPositiveDecimal(p.minAmountRaw())
}

// MaxAmountDecimal 单笔上限；未配置返回零值。
func (p *TransferPartner) MaxAmountDecimal() decimal.Decimal {
	return optionalPositiveDecimal(p.maxAmountRaw())
}

// DailyLimitDecimal 单日累计上限；未配置返回零值。
func (p *TransferPartner) DailyLimitDecimal() decimal.Decimal {
	return optionalPositiveDecimal(p.dailyLimitRaw())
}

func (p *TransferPartner) minAmountRaw() string {
	if p == nil {
		return ""
	}
	return p.MinAmount
}

func (p *TransferPartner) maxAmountRaw() string {
	if p == nil {
		return ""
	}
	return p.MaxAmount
}

func (p *TransferPartner) dailyLimitRaw() string {
	if p == nil {
		return ""
	}
	return p.DailyLimit
}

func optionalPositiveDecimal(raw string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || !d.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}
	return d
}
