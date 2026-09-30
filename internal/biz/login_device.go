package biz

import (
	"context"
	"time"
)

// LoginDeviceInfo 一次登录的设备画像。
type LoginDeviceInfo struct {
	DeviceID    string
	ClientIP    string
	UserAgent   string
	Client      string
	DeviceLabel string
}

// UserLoginLog 登录流水（管理端列表用）。
type UserLoginLog struct {
	ID          int64
	UserID      int64
	Address     string
	DeviceID    string
	ClientIP    string
	UserAgent   string
	Client      string
	DeviceLabel string
	CreatedAt   time.Time
}

// DeviceAccountStat 同设备关联账号统计。
type DeviceAccountStat struct {
	DeviceID     string
	AccountCount int64
	LoginCount   int64
	LastLoginAt  time.Time
	LastClientIP string
	LastUA       string
	LastClient   string
	DeviceLabel  string
}

// DeviceLinkedUser 某设备下的关联用户。
type DeviceLinkedUser struct {
	UserID       int64
	Address      string
	Username     string
	LoginCount   int64
	FirstLoginAt time.Time
	LastLoginAt  time.Time
	LastClientIP string
	LastUA       string
	LastClient   string
	IsFrozen     bool
}

// LoginDeviceRepo 登录设备记录仓储。
type LoginDeviceRepo interface {
	RecordLogin(ctx context.Context, userID int64, info LoginDeviceInfo) error
	ListLogsByUser(ctx context.Context, userID int64, offset, limit int) ([]*UserLoginLog, int64, error)
	ListDeviceStats(ctx context.Context, deviceID string, minAccounts int64, offset, limit int) ([]*DeviceAccountStat, int64, error)
	ListUsersByDevice(ctx context.Context, deviceID string, offset, limit int) ([]*DeviceLinkedUser, int64, error)
}
