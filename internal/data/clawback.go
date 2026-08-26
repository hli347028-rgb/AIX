package data

import (
	"fmt"
	"strings"

	"backend/internal/biz"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const clawbackExitMultiplier = 4

// ClawbackOrderAction describes one order change in a clawback run.
type ClawbackOrderAction struct {
	OrderID       int64
	Action        string // cancel | reduce | skip
	PrincipalBefore decimal.Decimal
	PrincipalAfter  decimal.Decimal
	ExitCapAfter    decimal.Decimal
	ReduceAmount    decimal.Decimal
	Note            string
}

// ClawbackReport is the result of a clawback simulation or apply.
type ClawbackReport struct {
	UserID           int64
	Address          string
	TargetAmount     decimal.Decimal
	RewardBefore     decimal.Decimal
	RewardAfter      decimal.Decimal
	RewardDebit      decimal.Decimal
	OverflowBefore   decimal.Decimal
	OverflowAfter    decimal.Decimal
	OverflowDebit    decimal.Decimal
	OrderPrincipalCut decimal.Decimal
	Unrecovered       decimal.Decimal
	PointsBefore      decimal.Decimal
	PointsAfter       decimal.Decimal
	PointsCutWanted   decimal.Decimal
	PointsCutActual   decimal.Decimal
	PointsShortfall   decimal.Decimal
	Orders            []ClawbackOrderAction
}

// ClawbackOptions controls clawback behavior.
type ClawbackOptions struct {
	Address              string
	TargetAmount         decimal.Decimal
	// TreatAllActiveAsReward: local test mode — every active order is eligible
	// (ignore fund_source / earned_total). Still skips exited.
	TreatAllActiveAsReward bool
	DryRun                 bool
}

// RunMgmtClawback deducts reward wallet, then overflow, then eligible orders.
func RunMgmtClawback(db *gorm.DB, opt ClawbackOptions) (*ClawbackReport, error) {
	addr := strings.ToLower(strings.TrimSpace(opt.Address))
	if addr == "" {
		return nil, fmt.Errorf("address required")
	}
	if !opt.TargetAmount.IsPositive() {
		return nil, fmt.Errorf("target amount must be positive")
	}

	var report *ClawbackReport
	err := db.Transaction(func(tx *gorm.DB) error {
		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("LOWER(address) = ?", addr).First(&user).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		remain := opt.TargetAmount
		rep := &ClawbackReport{
			UserID:         user.ID,
			Address:        user.Address,
			TargetAmount:   opt.TargetAmount,
			RewardBefore:   user.UsdtReward,
			OverflowBefore: user.OverflowReward,
			PointsBefore:   user.Points,
		}

		rewardDebit := decimal.Min(user.UsdtReward, remain)
		user.UsdtReward = user.UsdtReward.Sub(rewardDebit)
		remain = remain.Sub(rewardDebit)
		rep.RewardDebit = rewardDebit
		rep.RewardAfter = user.UsdtReward

		overflowDebit := decimal.Min(user.OverflowReward, remain)
		user.OverflowReward = user.OverflowReward.Sub(overflowDebit)
		remain = remain.Sub(overflowDebit)
		rep.OverflowDebit = overflowDebit
		rep.OverflowAfter = user.OverflowReward

		var orders []OrderPO
		q := tx.Where("user_id = ? AND status = ?", user.ID, biz.OrderStatusActive).Order("id ASC")
		if !opt.DryRun {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if !opt.TreatAllActiveAsReward {
			// Formal: reward-wallet subscriptions only; include ones with
			// partial static release. Still skip exited / non-reward.
			q = q.Where("fund_source = ?", biz.PayFromReward)
		}
		if err := q.Find(&orders).Error; err != nil {
			return err
		}

		pointsCut := decimal.Zero
		orderCut := decimal.Zero
		for i := range orders {
			if !remain.IsPositive() {
				break
			}
			o := &orders[i]
			act := ClawbackOrderAction{
				OrderID:         o.ID,
				PrincipalBefore: o.Principal,
			}
			if !o.Principal.IsPositive() {
				act.Action = "skip"
				act.Note = "zero principal"
				rep.Orders = append(rep.Orders, act)
				continue
			}

			if remain.GreaterThanOrEqual(o.Principal) {
				cut := o.Principal
				remain = remain.Sub(cut)
				orderCut = orderCut.Add(cut)
				pointsCut = pointsCut.Add(o.Points)
				act.Action = "cancel"
				act.ReduceAmount = cut
				act.PrincipalAfter = decimal.Zero
				act.ExitCapAfter = decimal.Zero
				o.Status = biz.OrderStatusCancelled
				if !opt.DryRun {
					if err := tx.Model(o).Updates(map[string]interface{}{
						"status": biz.OrderStatusCancelled,
					}).Error; err != nil {
						return err
					}
				}
			} else {
				cut := remain
				newPrincipal := o.Principal.Sub(cut)
				newExitCap := newPrincipal.Mul(decimal.NewFromInt(clawbackExitMultiplier))
				newPoints := o.Points
				if newPoints.GreaterThan(newPrincipal) {
					pointsCut = pointsCut.Add(newPoints.Sub(newPrincipal))
					newPoints = newPrincipal
				} else if o.Points.Equal(o.Principal) {
					pointsCut = pointsCut.Add(cut)
					newPoints = newPrincipal
				}
				newFromRecharge := o.FromRecharge
				if newFromRecharge.GreaterThan(newPrincipal) {
					newFromRecharge = newPrincipal
				}
				newEarned := o.EarnedTotal
				if newEarned.GreaterThan(newExitCap) {
					newEarned = newExitCap
				}
				status := o.Status
				if newEarned.GreaterThanOrEqual(newExitCap) && newExitCap.IsPositive() {
					status = biz.OrderStatusExited
				}
				remain = decimal.Zero
				orderCut = orderCut.Add(cut)
				act.Action = "reduce"
				act.ReduceAmount = cut
				act.PrincipalAfter = newPrincipal
				act.ExitCapAfter = newExitCap
				o.Principal = newPrincipal
				o.ExitCap = newExitCap
				o.Points = newPoints
				o.FromRecharge = newFromRecharge
				o.EarnedTotal = newEarned
				o.Status = status
				if !opt.DryRun {
					updates := map[string]interface{}{
						"principal":     newPrincipal,
						"exit_cap":      newExitCap,
						"points":        newPoints,
						"from_recharge": newFromRecharge,
						"earned_total":  newEarned,
						"status":        status,
					}
					if err := tx.Model(o).Updates(updates).Error; err != nil {
						return err
					}
				}
			}
			rep.Orders = append(rep.Orders, act)
		}

		rep.OrderPrincipalCut = orderCut
		rep.Unrecovered = remain

		rep.PointsCutWanted = pointsCut
		if pointsCut.IsPositive() {
			actual := pointsCut
			if actual.GreaterThan(user.Points) {
				rep.PointsShortfall = actual.Sub(user.Points)
				actual = user.Points
			}
			rep.PointsCutActual = actual
			user.Points = user.Points.Sub(actual)
			newPointsAll := user.PointsAll.Sub(pointsCut)
			if newPointsAll.IsNegative() {
				newPointsAll = decimal.Zero
			}
			user.PointsAll = newPointsAll
		}
		rep.PointsAfter = user.Points

		if !opt.DryRun {
			if err := tx.Model(&user).Updates(map[string]interface{}{
				"usdt_reward":     user.UsdtReward,
				"overflow_reward": user.OverflowReward,
				"points":          user.Points,
				"points_all":      user.PointsAll,
			}).Error; err != nil {
				return err
			}
			if err := refreshSeedAndAncestors(tx, user.ID); err != nil {
				return err
			}
		}

		report = rep
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

// FormatClawbackReport returns a human-readable clawback summary.
func FormatClawbackReport(r *ClawbackReport) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "address=%s user_id=%d target=%s\n", r.Address, r.UserID, r.TargetAmount.String())
	fmt.Fprintf(&b, "reward: %s -> %s (debit %s)\n", r.RewardBefore, r.RewardAfter, r.RewardDebit)
	fmt.Fprintf(&b, "overflow: %s -> %s (debit %s)\n", r.OverflowBefore, r.OverflowAfter, r.OverflowDebit)
	fmt.Fprintf(&b, "order_principal_cut=%s unrecovered=%s\n", r.OrderPrincipalCut, r.Unrecovered)
	fmt.Fprintf(&b, "points: %s -> %s (wanted %s actual %s shortfall %s)\n",
		r.PointsBefore, r.PointsAfter, r.PointsCutWanted, r.PointsCutActual, r.PointsShortfall)
	for _, o := range r.Orders {
		fmt.Fprintf(&b, "  order#%d %s before=%s after=%s exit_cap=%s cut=%s %s\n",
			o.OrderID, o.Action, o.PrincipalBefore, o.PrincipalAfter, o.ExitCapAfter, o.ReduceAmount, o.Note)
	}
	return b.String()
}
