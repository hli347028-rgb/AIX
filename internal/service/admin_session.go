package service

import (
	"strings"

	"backend/internal/conf"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AdminSession 管理后台登录会话（主账户 / 子账户 / 链上管理员）。
type AdminSession struct {
	Operator string // 展示用账号名
	IsMain   bool   // 是否主账户（仅主账户可查看操作记录）
}

func (s *AdminLegacyService) isConfiguredSubAccount(account string) bool {
	account = strings.TrimSpace(account)
	if account == "" {
		return false
	}
	for _, item := range s.effectiveSubAccounts() {
		if strings.TrimSpace(item.Account) == account {
			return true
		}
	}
	return false
}

func (s *AdminLegacyService) effectiveSubAccounts() []conf.AdminSubAccount {
	defaults := s.authCfg.GetAdminSubAccounts()
	snap := s.admin.GetPersistedConfigSnapshot()
	if snap == nil || len(snap.AdminSubAccounts) == 0 {
		return defaults
	}
	out := make([]conf.AdminSubAccount, len(defaults))
	for i, d := range defaults {
		out[i] = d
	}
	for _, item := range snap.AdminSubAccounts {
		acc := strings.TrimSpace(item.Account)
		if acc == "" {
			continue
		}
		for i := range out {
			if strings.TrimSpace(out[i].Account) == acc {
				if strings.TrimSpace(item.Password) != "" {
					out[i].Password = item.Password
				}
				if len(item.Modules) > 0 {
					out[i].Modules = append([]string(nil), item.Modules...)
				}
				break
			}
		}
	}
	return out
}

func (s *AdminLegacyService) subAccountModules(account string) []string {
	account = strings.TrimSpace(account)
	for _, item := range s.effectiveSubAccounts() {
		if strings.TrimSpace(item.Account) == account {
			return append([]string(nil), item.Modules...)
		}
	}
	return nil
}

func (s *AdminLegacyService) validateSubLogin(account, password string) bool {
	account = strings.TrimSpace(account)
	password = strings.TrimSpace(password)
	if account == "" || password == "" {
		return false
	}
	for _, sub := range s.effectiveSubAccounts() {
		if strings.TrimSpace(sub.Account) == account && strings.TrimSpace(sub.Password) == password {
			return true
		}
	}
	return false
}

func (s *AdminLegacyService) resolveAdminSession(ctx khttp.Context, tokenString string) (*AdminSession, error) {
	session, err := s.sessionFromToken(ctx, tokenString)
	if err != nil {
		return nil, errors.Unauthorized("UNAUTHORIZED", "token 无效或已过期")
	}
	return session, nil
}

func (s *AdminLegacyService) requireAnyAdmin(ctx khttp.Context) (*AdminSession, error) {
	return s.resolveAdminSession(ctx, s.token(ctx))
}

func (s *AdminLegacyService) requireMainAdmin(ctx khttp.Context) (*AdminSession, error) {
	session, err := s.requireAnyAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if !session.IsMain {
		return nil, errors.Forbidden("FORBIDDEN", "仅主账户可查看操作记录")
	}
	return session, nil
}

func (s *AdminLegacyService) requireAdmin(ctx khttp.Context) error {
	_, err := s.requireAnyAdmin(ctx)
	return err
}
