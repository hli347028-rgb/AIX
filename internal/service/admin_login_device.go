package service

import (
	"strconv"
	"strings"

	"backend/internal/pkg/eth"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HandleLoginDeviceList GET /api/admin_dhb/login_device_list
// Aggregates login devices by linked account count for multi-account checks.
func (s *AdminLegacyService) HandleLoginDeviceList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if s.loginDevices == nil {
		return ctx.Result(200, map[string]interface{}{"list": []any{}, "count": 0})
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	deviceID := strings.TrimSpace(q.Get("device_id"))
	minAccounts := int64(2)
	if v := strings.TrimSpace(q.Get("min_accounts")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 1 {
			minAccounts = n
		}
	}
	list, total, err := s.loginDevices.ListDeviceStats(ctx, deviceID, minAccounts, offset, pageSize)
	if err != nil {
		return err
	}
	rows := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		rows = append(rows, map[string]interface{}{
			"device_id":     item.DeviceID,
			"account_count": item.AccountCount,
			"login_count":   item.LoginCount,
			"last_login_at": formatLegacyTime(item.LastLoginAt),
			"last_ip":       item.LastClientIP,
			"last_ua":       item.LastUA,
			"last_client":   item.LastClient,
			"device_label":  item.DeviceLabel,
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":     rows,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// HandleLoginDeviceUsers GET /api/admin_dhb/login_device_users
// Lists all accounts linked to one device_id.
func (s *AdminLegacyService) HandleLoginDeviceUsers(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if s.loginDevices == nil {
		return ctx.Result(200, map[string]interface{}{"list": []any{}, "count": 0})
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	deviceID := strings.TrimSpace(q.Get("device_id"))
	if deviceID == "" {
		return ctx.Result(200, map[string]interface{}{"list": []any{}, "count": 0})
	}
	list, total, err := s.loginDevices.ListUsersByDevice(ctx, deviceID, offset, pageSize)
	if err != nil {
		return err
	}
	rows := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		rows = append(rows, map[string]interface{}{
			"user_id":        item.UserID,
			"address":        item.Address,
			"username":       item.Username,
			"login_count":    item.LoginCount,
			"first_login_at": formatLegacyTime(item.FirstLoginAt),
			"last_login_at":  formatLegacyTime(item.LastLoginAt),
			"last_ip":        item.LastClientIP,
			"last_ua":        item.LastUA,
			"last_client":    item.LastClient,
			"is_frozen":      item.IsFrozen,
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":      rows,
		"count":     total,
		"page":      page,
		"pageSize":  pageSize,
		"device_id": deviceID,
	})
}

// HandleUserLoginDevices GET /api/admin_dhb/user_login_devices
// Lists recent login device logs for one user.
func (s *AdminLegacyService) HandleUserLoginDevices(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if s.loginDevices == nil {
		return ctx.Result(200, map[string]interface{}{"list": []any{}, "count": 0})
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	userID, _ := strconv.ParseInt(strings.TrimSpace(q.Get("user_id")), 10, 64)
	address := strings.TrimSpace(q.Get("address"))
	if userID <= 0 && address != "" {
		norm, err := eth.NormalizeAddress(address)
		if err == nil {
			u, err := s.userRepo.FindByAddress(ctx, norm)
			if err != nil {
				return err
			}
			if u != nil {
				userID = u.ID
			}
		}
	}
	if userID <= 0 {
		return ctx.Result(200, map[string]interface{}{"list": []any{}, "count": 0})
	}
	list, total, err := s.loginDevices.ListLogsByUser(ctx, userID, offset, pageSize)
	if err != nil {
		return err
	}
	rows := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		rows = append(rows, map[string]interface{}{
			"id":           item.ID,
			"user_id":      item.UserID,
			"device_id":    item.DeviceID,
			"client_ip":    item.ClientIP,
			"user_agent":   item.UserAgent,
			"client":       item.Client,
			"device_label": item.DeviceLabel,
			"created_at":   formatLegacyTime(item.CreatedAt),
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":     rows,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
		"user_id":  userID,
	})
}
