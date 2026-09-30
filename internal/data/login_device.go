package data

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/biz"
)

type loginDeviceRepo struct {
	data *Data
}

// NewLoginDeviceRepo creates LoginDeviceRepo.
func NewLoginDeviceRepo(d *Data) biz.LoginDeviceRepo {
	return &loginDeviceRepo{data: d}
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func deriveDeviceLabel(ua, client string) string {
	uaLower := strings.ToLower(ua)
	client = strings.TrimSpace(client)
	var parts []string
	switch {
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ios"):
		parts = append(parts, "iOS")
	case strings.Contains(uaLower, "android"):
		parts = append(parts, "Android")
	case strings.Contains(uaLower, "windows"):
		parts = append(parts, "Windows")
	case strings.Contains(uaLower, "mac os") || strings.Contains(uaLower, "macintosh"):
		parts = append(parts, "macOS")
	case strings.Contains(uaLower, "linux"):
		parts = append(parts, "Linux")
	default:
		if ua != "" {
			parts = append(parts, "UnknownOS")
		}
	}
	switch {
	case strings.Contains(uaLower, "tokenpocket") || strings.EqualFold(client, "tokenpocket"):
		parts = append(parts, "TokenPocket")
	case strings.Contains(uaLower, "metamask") || strings.EqualFold(client, "metamask"):
		parts = append(parts, "MetaMask")
	case client != "" && !strings.EqualFold(client, "web"):
		parts = append(parts, client)
	case strings.Contains(uaLower, "mobile"):
		parts = append(parts, "MobileBrowser")
	case ua != "":
		parts = append(parts, "Browser")
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "/")
}

func (r *loginDeviceRepo) RecordLogin(ctx context.Context, userID int64, info biz.LoginDeviceInfo) error {
	if userID <= 0 {
		return nil
	}
	deviceID := truncateRunes(info.DeviceID, 64)
	if deviceID == "" {
		deviceID = "unknown"
	}
	label := strings.TrimSpace(info.DeviceLabel)
	if label == "" {
		label = deriveDeviceLabel(info.UserAgent, info.Client)
	}
	po := &UserLoginLogPO{
		UserID:      userID,
		DeviceID:    deviceID,
		ClientIP:    truncateRunes(info.ClientIP, 64),
		UserAgent:   truncateRunes(info.UserAgent, 512),
		Client:      truncateRunes(info.Client, 64),
		DeviceLabel: truncateRunes(label, 128),
		CreatedTime: time.Now(),
	}
	return r.data.db.WithContext(ctx).Create(po).Error
}

func (r *loginDeviceRepo) ListLogsByUser(ctx context.Context, userID int64, offset, limit int) ([]*biz.UserLoginLog, int64, error) {
	if userID <= 0 {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	db := r.data.db.WithContext(ctx).Model(&UserLoginLogPO{}).Where("user_id = ?", userID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []UserLoginLogPO
	if err := db.Order("id desc").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.UserLoginLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, &biz.UserLoginLog{
			ID:          row.ID,
			UserID:      row.UserID,
			DeviceID:    row.DeviceID,
			ClientIP:    row.ClientIP,
			UserAgent:   row.UserAgent,
			Client:      row.Client,
			DeviceLabel: row.DeviceLabel,
			CreatedAt:   row.CreatedTime,
		})
	}
	return out, total, nil
}

func (r *loginDeviceRepo) ListDeviceStats(ctx context.Context, deviceID string, minAccounts int64, offset, limit int) ([]*biz.DeviceAccountStat, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if minAccounts < 1 {
		minAccounts = 1
	}
	deviceID = strings.TrimSpace(deviceID)

	type aggRow struct {
		DeviceID     string    `gorm:"column:device_id"`
		AccountCount int64     `gorm:"column:account_count"`
		LoginCount   int64     `gorm:"column:login_count"`
		LastLoginAt  time.Time `gorm:"column:last_login_at"`
	}

	base := r.data.db.WithContext(ctx).Table("user_login_logs").
		Select("device_id, COUNT(DISTINCT user_id) AS account_count, COUNT(*) AS login_count, MAX(created_time) AS last_login_at").
		Group("device_id").
		Having("COUNT(DISTINCT user_id) >= ?", minAccounts)
	if deviceID != "" {
		base = base.Where("device_id = ?", deviceID)
	}

	var total int64
	countSQL := r.data.db.WithContext(ctx).Table("(?) AS t", base).Count(&total)
	if err := countSQL.Error; err != nil {
		return nil, 0, err
	}

	var aggs []aggRow
	if err := base.Order("account_count desc, last_login_at desc").Offset(offset).Limit(limit).Scan(&aggs).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.DeviceAccountStat, 0, len(aggs))
	for _, a := range aggs {
		stat := &biz.DeviceAccountStat{
			DeviceID:     a.DeviceID,
			AccountCount: a.AccountCount,
			LoginCount:   a.LoginCount,
			LastLoginAt:  a.LastLoginAt,
		}
		var latest UserLoginLogPO
		if err := r.data.db.WithContext(ctx).
			Where("device_id = ?", a.DeviceID).
			Order("id desc").
			First(&latest).Error; err == nil {
			stat.LastClientIP = latest.ClientIP
			stat.LastUA = latest.UserAgent
			stat.LastClient = latest.Client
			stat.DeviceLabel = latest.DeviceLabel
		}
		out = append(out, stat)
	}
	return out, total, nil
}

func (r *loginDeviceRepo) ListUsersByDevice(ctx context.Context, deviceID string, offset, limit int) ([]*biz.DeviceLinkedUser, int64, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 20
	}

	type userAgg struct {
		UserID       int64     `gorm:"column:user_id"`
		LoginCount   int64     `gorm:"column:login_count"`
		FirstLoginAt time.Time `gorm:"column:first_login_at"`
		LastLoginAt  time.Time `gorm:"column:last_login_at"`
	}

	base := r.data.db.WithContext(ctx).Table("user_login_logs").
		Select("user_id, COUNT(*) AS login_count, MIN(created_time) AS first_login_at, MAX(created_time) AS last_login_at").
		Where("device_id = ?", deviceID).
		Group("user_id")

	var total int64
	if err := r.data.db.WithContext(ctx).Table("(?) AS t", base).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var aggs []userAgg
	if err := base.Order("last_login_at desc").Offset(offset).Limit(limit).Scan(&aggs).Error; err != nil {
		return nil, 0, err
	}
	if len(aggs) == 0 {
		return []*biz.DeviceLinkedUser{}, total, nil
	}

	userIDs := make([]int64, 0, len(aggs))
	for _, a := range aggs {
		userIDs = append(userIDs, a.UserID)
	}
	var users []UserPO
	_ = r.data.db.WithContext(ctx).Select("id", "address", "username", "is_frozen").Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[int64]UserPO, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	out := make([]*biz.DeviceLinkedUser, 0, len(aggs))
	for _, a := range aggs {
		item := &biz.DeviceLinkedUser{
			UserID:       a.UserID,
			LoginCount:   a.LoginCount,
			FirstLoginAt: a.FirstLoginAt,
			LastLoginAt:  a.LastLoginAt,
		}
		if u, ok := userMap[a.UserID]; ok {
			item.Address = u.Address
			item.Username = u.Username
			item.IsFrozen = u.IsFrozen
		}
		var latest UserLoginLogPO
		if err := r.data.db.WithContext(ctx).
			Where("device_id = ? AND user_id = ?", deviceID, a.UserID).
			Order("id desc").
			First(&latest).Error; err == nil {
			item.LastClientIP = latest.ClientIP
			item.LastUA = latest.UserAgent
			item.LastClient = latest.Client
		}
		out = append(out, item)
	}
	return out, total, nil
}
