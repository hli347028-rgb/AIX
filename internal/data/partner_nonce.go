package data

import (
	"context"
	"strings"
	"sync"
	"time"

	"backend/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type partnerNonceRepo struct {
	data *Data
	log  *log.Helper

	mu          sync.Mutex
	lastCleanup time.Time
}

// NewPartnerNonceRepo .
func NewPartnerNonceRepo(data *Data, logger log.Logger) biz.PartnerNonceRepo {
	return &partnerNonceRepo{data: data, log: log.NewHelper(logger)}
}

// cleanupInterval 限制清理频率，避免每个请求都跑一次 DELETE。
const cleanupInterval = time.Minute

// Occupy 原子占用 (partnerID, nonce)。返回 true 表示首次出现（放行），
// false 表示重放。
//
// 原子性来自 partner_nonces 的唯一索引，而不是「先查后插」——后者在并发下
// 两个请求可能都查到不存在，从而让同一个 nonce 被受理两次。
func (r *partnerNonceRepo) Occupy(ctx context.Context, partnerID, nonce string, retain time.Duration) (bool, error) {
	partnerID = strings.TrimSpace(partnerID)
	nonce = strings.TrimSpace(nonce)
	if partnerID == "" || nonce == "" {
		return false, nil
	}

	err := r.data.db.WithContext(ctx).Create(&PartnerNoncePO{
		PartnerID: partnerID,
		Nonce:     nonce,
	}).Error
	if err != nil {
		if isDuplicateKeyErr(err) {
			return false, nil
		}
		return false, err
	}

	r.cleanupExpired(ctx, retain)
	return true, nil
}

// cleanupExpired 删除超出保留窗口的记录。清理失败只记日志：
// 去重表长胖不影响正确性，但让加款请求失败会影响资金流程。
func (r *partnerNonceRepo) cleanupExpired(ctx context.Context, retain time.Duration) {
	if retain <= 0 {
		return
	}
	r.mu.Lock()
	if time.Since(r.lastCleanup) < cleanupInterval {
		r.mu.Unlock()
		return
	}
	r.lastCleanup = time.Now()
	r.mu.Unlock()

	cutoff := time.Now().Add(-retain)
	if err := r.data.db.WithContext(ctx).
		Where("created_time < ?", cutoff).
		Delete(&PartnerNoncePO{}).Error; err != nil {
		r.log.Warnf("partner nonce cleanup failed: %v", err)
	}
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062")
}
