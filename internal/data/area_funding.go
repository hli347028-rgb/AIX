package data

import (
	"context"

	"backend/internal/biz"

	"github.com/shopspring/decimal"
)

// SumAreaFundingBreakdown 按大区/小区汇总：
//   - Usdt：认购 from_recharge（USDT）
//   - Win：认购 from_win（原生 WIN）
//   - ExchangeWin：下级已确认 WIN 充值且 tx_hash 以 partner: 开头（仅明细）
//   - Reward：认购 from_reward（USDT）
//   - TotalUsdt：SUM(principal)，即 USDT认购 + WIN认购×当时价 + 复投；不含交易所划转
//
// 区域划分与 team_perf 一致：直推各枝累计本金（active+exited），最大枝=大区，其余=小区；不含本人订单。
// 仅遍历 root 伞下用户，避免全表扫边在生产超时。
func (r *userRepo) SumAreaFundingBreakdown(ctx context.Context, rootID int64) (large, small biz.AreaFundingBreakdown, err error) {
	zero := biz.AreaFundingBreakdown{Usdt: "0", Win: "0", ExchangeWin: "0", Reward: "0", TotalUsdt: "0"}
	large, small = zero, zero
	if rootID <= 0 {
		return large, small, nil
	}

	var direct []int64
	if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
		Where("inviter_id = ?", rootID).
		Pluck("id", &direct).Error; err != nil {
		return zero, zero, err
	}
	if len(direct) == 0 {
		return large, small, nil
	}

	collectSubtree := func(branchRoot int64) ([]int64, error) {
		var ids []int64
		frontier := []int64{branchRoot}
		seen := map[int64]struct{}{branchRoot: {}}
		for len(frontier) > 0 {
			ids = append(ids, frontier...)
			var kids []int64
			if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
				Where("inviter_id IN ?", frontier).
				Pluck("id", &kids).Error; err != nil {
				return nil, err
			}
			next := make([]int64, 0, len(kids))
			for _, id := range kids {
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				next = append(next, id)
			}
			frontier = next
		}
		return ids, nil
	}

	sumPrincipal := func(userIDs []int64) (decimal.Decimal, error) {
		if len(userIDs) == 0 {
			return decimal.Zero, nil
		}
		var total decimal.Decimal
		if err := r.data.db.WithContext(ctx).Model(&OrderPO{}).
			Select("COALESCE(SUM(principal),0)").
			Where("user_id IN ?", userIDs).
			Where("status IN ?", []string{biz.OrderStatusActive, biz.OrderStatusExited}).
			Scan(&total).Error; err != nil {
			return decimal.Zero, err
		}
		return total, nil
	}

	branchIDs := make(map[int64][]int64, len(direct))
	branchStakes := make(map[int64]decimal.Decimal, len(direct))
	for _, childID := range direct {
		ids, cErr := collectSubtree(childID)
		if cErr != nil {
			return zero, zero, cErr
		}
		branchIDs[childID] = ids
		stake, sErr := sumPrincipal(ids)
		if sErr != nil {
			return zero, zero, sErr
		}
		branchStakes[childID] = stake
	}
	largeRoot := biz.PickLargeAreaChild(branchStakes)

	var largeIDs, smallIDs []int64
	for _, childID := range direct {
		ids := branchIDs[childID]
		if childID == largeRoot {
			largeIDs = append(largeIDs, ids...)
		} else {
			smallIDs = append(smallIDs, ids...)
		}
	}

	sumSide := func(userIDs []int64) (biz.AreaFundingBreakdown, error) {
		out := biz.AreaFundingBreakdown{Usdt: "0", Win: "0", ExchangeWin: "0", Reward: "0", TotalUsdt: "0"}
		if len(userIDs) == 0 {
			return out, nil
		}
		type fundRow struct {
			Usdt      decimal.Decimal `gorm:"column:usdt"`
			Win       decimal.Decimal `gorm:"column:win"`
			Reward    decimal.Decimal `gorm:"column:reward"`
			Principal decimal.Decimal `gorm:"column:principal"`
		}
		var funds fundRow
		if err := r.data.db.WithContext(ctx).Model(&OrderPO{}).
			Select(`COALESCE(SUM(from_recharge),0) AS usdt,
				COALESCE(SUM(from_win),0) AS win,
				COALESCE(SUM(from_reward),0) AS reward,
				COALESCE(SUM(principal),0) AS principal`).
			Where("user_id IN ?", userIDs).
			Where("status IN ?", []string{biz.OrderStatusActive, biz.OrderStatusExited}).
			Scan(&funds).Error; err != nil {
			return out, err
		}
		var exchangeWin decimal.Decimal
		if err := r.data.db.WithContext(ctx).Model(&RechargePO{}).
			Select("COALESCE(SUM(amount),0)").
			Where("user_id IN ?", userIDs).
			Where("status = ?", biz.RechargeStatusConfirmed).
			Where("UPPER(asset) IN ?", []string{biz.TokenWIN, biz.TokenWINA}).
			Where("LOWER(tx_hash) LIKE ?", "partner:%").
			Scan(&exchangeWin).Error; err != nil {
			return out, err
		}
		out.Usdt = funds.Usdt.String()
		out.Win = funds.Win.String()
		out.Reward = funds.Reward.String()
		out.ExchangeWin = exchangeWin.String()
		out.TotalUsdt = funds.Principal.String()
		return out, nil
	}

	large, err = sumSide(largeIDs)
	if err != nil {
		return zero, zero, err
	}
	small, err = sumSide(smallIDs)
	if err != nil {
		return zero, zero, err
	}
	return large, small, nil
}
