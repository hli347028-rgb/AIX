package job

import (
	"context"
	"time"

	"backend/internal/biz"
	"backend/internal/pkg/token"

	"github.com/go-kratos/kratos/v2/log"
)

// SettlementJob 仅在中国时间每日 0 点触发：先锁定当日兑换额度，再跑系统日结。
// 启动/部署不会触发结算；管理端也无法手动触发。
type SettlementJob struct {
	uc     *biz.SettlementUsecase
	log    *log.Helper
	stopCh chan struct{}
}

func NewSettlementJob(uc *biz.SettlementUsecase, logger log.Logger) *SettlementJob {
	return &SettlementJob{
		uc:     uc,
		log:    log.NewHelper(logger),
		stopCh: make(chan struct{}),
	}
}

func (j *SettlementJob) Start() {
	go j.run()
	j.log.Info("settlement job started: China midnight only (no boot run)")
}

func (j *SettlementJob) Stop() {
	close(j.stopCh)
}

func (j *SettlementJob) run() {
	for {
		delay := durationUntilNextChinaMidnight(time.Now())
		j.log.Infof("next system settlement/quota lock in %s", delay.Round(time.Second))
		select {
		case <-time.After(delay):
			j.runOnce()
		case <-j.stopCh:
			return
		}
	}
}

func (j *SettlementJob) runOnce() {
	ctx := context.Background()
	now := time.Now()
	quotaDate := now.In(token.ChinaLocation()).Format("2006-01-02")
	settlementDate := biz.TodaySettlementDate(now)

	// 1) 自然日兑换额度：与结算无关，一天只锁一次
	j.log.Infof("locking daily exchange quota for %s", quotaDate)
	if err := j.uc.EnsureDailyExchangeQuota(ctx, quotaDate); err != nil {
		j.log.Errorf("exchange quota lock %s failed: %v", quotaDate, err)
	} else {
		j.log.Infof("exchange quota lock %s ok", quotaDate)
	}

	// 2) 系统日结（结算日为「昨日」）
	j.log.Infof("running daily settlement for %s", settlementDate)
	if err := j.uc.RunDailySettlement(ctx, settlementDate); err != nil {
		j.log.Errorf("settlement %s failed: %v", settlementDate, err)
		return
	}
	j.log.Infof("settlement %s completed", settlementDate)

	if err := j.uc.BackfillMissingEcoRewards(ctx); err != nil {
		j.log.Errorf("eco backfill failed: %v", err)
	}
}

func durationUntilNextChinaMidnight(now time.Time) time.Duration {
	loc := token.ChinaLocation()
	now = now.In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	return next.Sub(now)
}
