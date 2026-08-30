package conf

import (
	"crypto/subtle"
	"strings"
	"time"
)

// AuthConfig holds Web3 auth settings loaded from config.yaml.
type AuthConfig struct {
	JwtSecret          string            `json:"jwt_secret" yaml:"jwt_secret"`
	BootstrapAddresses []string          `json:"bootstrap_addresses" yaml:"bootstrap_addresses"`
	AdminAddresses     []string          `json:"admin_addresses" yaml:"admin_addresses"`
	AdminAccount       string            `json:"admin_account" yaml:"admin_account"`
	AdminPassword      string            `json:"admin_password" yaml:"admin_password"`
	AdminSubAccounts   []AdminSubAccount `json:"admin_sub_accounts" yaml:"admin_sub_accounts"`
	ChallengeTTL         string            `json:"challenge_ttl" yaml:"challenge_ttl"`
	// OpenAPIKeys 第三方开放接口密钥（请求头 X-API-Key 或 Authorization: Bearer <key>）
	OpenAPIKeys []string `json:"open_api_keys" yaml:"open_api_keys"`
}

// AdminSubAccount 管理后台子账户（主账户可在配置项设置密码与可访问模块）。
type AdminSubAccount struct {
	Account  string   `json:"account" yaml:"account"`
	Password string   `json:"password" yaml:"password"`
	Modules  []string `json:"modules" yaml:"modules"` // 可访问路由 path，如 /home、/member
}

func (a *AuthConfig) GetAdminAddresses() []string {
	if a == nil {
		return nil
	}
	return a.AdminAddresses
}

func (a *AuthConfig) GetBootstrapAddresses() []string {
	if a == nil {
		return nil
	}
	return a.BootstrapAddresses
}

func (a *AuthConfig) ChallengeDuration() time.Duration {
	if a == nil || a.ChallengeTTL == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(a.ChallengeTTL)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}

func (a *AuthConfig) GetJwtSecret() string {
	if a == nil || a.JwtSecret == "" {
		return "change-me-in-production"
	}
	return a.JwtSecret
}

func (a *AuthConfig) GetAdminAccount() string {
	if a == nil || a.AdminAccount == "" {
		return "admin"
	}
	return a.AdminAccount
}

func (a *AuthConfig) GetAdminPassword() string {
	if a == nil || a.AdminPassword == "" {
		return "admin123"
	}
	return a.AdminPassword
}

// DefaultAdminSubAccounts 内置子账户（配置未填写时使用）。
func DefaultAdminSubAccounts() []AdminSubAccount {
	return []AdminSubAccount{
		{Account: "user1", Password: "user1"},
		{Account: "user2", Password: "user2"},
		{Account: "user3", Password: "user3"},
	}
}

func (a *AuthConfig) GetAdminSubAccounts() []AdminSubAccount {
	if a != nil && len(a.AdminSubAccounts) > 0 {
		return a.AdminSubAccounts
	}
	return DefaultAdminSubAccounts()
}

func (a *AuthConfig) ValidateSubLogin(account, password string) bool {
	account = strings.TrimSpace(account)
	password = strings.TrimSpace(password)
	if account == "" || password == "" {
		return false
	}
	for _, sub := range a.GetAdminSubAccounts() {
		if strings.TrimSpace(sub.Account) == account && strings.TrimSpace(sub.Password) == password {
			return true
		}
	}
	return false
}

const AdminSubJWTPrefix = "admin:sub:"

func (a *AuthConfig) SubAccountJWTAddress(account string) string {
	return AdminSubJWTPrefix + strings.TrimSpace(account)
}

func SubAccountFromJWTAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if !strings.HasPrefix(addr, AdminSubJWTPrefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(addr, AdminSubJWTPrefix))
}

// IsOpenAPIKey reports whether key is configured for third-party Open API access.
func (a *AuthConfig) IsOpenAPIKey(key string) bool {
	key = strings.TrimSpace(key)
	if a == nil || key == "" {
		return false
	}
	kb := []byte(key)
	for _, k := range a.OpenAPIKeys {
		k = strings.TrimSpace(k)
		if k == "" || len(k) != len(kb) {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(k), kb) == 1 {
			return true
		}
	}
	return false
}
