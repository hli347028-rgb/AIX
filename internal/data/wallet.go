package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/biz"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type walletRepo struct {
	data *Data
}

func NewWalletRepo(data *Data) biz.WalletRepo {
	return &walletRepo{data: data}
}

func (r *walletRepo) CreateRecharge(ctx context.Context, recharge *biz.Recharge) (*biz.Recharge, error) {
	amount, err := decimal.NewFromString(recharge.Amount)
	if err != nil {
		return nil, err
	}
	asset := strings.ToUpper(strings.TrimSpace(recharge.Asset))
	if asset == "" {
		asset = biz.TokenUSDT
	}
	// unique placeholder until confirm
	placeholder := fmt.Sprintf("pending-%d-%d", recharge.UserID, time.Now().UnixNano())
	expire := recharge.ExpireAt
	po := &RechargePO{
		UserID:      recharge.UserID,
		Asset:       asset,
		Amount:      amount,
		TxHash:      placeholder,
		FromAddress: recharge.FromAddress,
		ToAddress:   recharge.ToAddress,
		Status:      recharge.Status,
		Message:     recharge.Message,
		ExpireAt:    &expire,
	}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return r.rechargeToBiz(po), nil
}

func (r *walletRepo) FindRecharge(ctx context.Context, id int64) (*biz.Recharge, error) {
	var po RechargePO
	err := r.data.db.WithContext(ctx).First(&po, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.rechargeToBiz(&po), nil
}

func (r *walletRepo) FindRechargeByTxHash(ctx context.Context, txHash string) (*biz.Recharge, error) {
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return nil, nil
	}
	var po RechargePO
	err := r.data.db.WithContext(ctx).Where("tx_hash = ?", txHash).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.rechargeToBiz(&po), nil
}

func (r *walletRepo) ConfirmRecharge(ctx context.Context, id int64, txHash string) error {
	_, err := r.ConfirmRechargeCredit(ctx, id, txHash)
	return err
}

func (r *walletRepo) ConfirmRechargeCredit(ctx context.Context, id int64, txHash string) (string, error) {
	var newBal string
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po RechargePO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error; err != nil {
			return err
		}
		if po.Status == biz.RechargeStatusConfirmed {
			return fmt.Errorf("already confirmed")
		}
		now := time.Now()
		if err := tx.Model(&po).Updates(map[string]interface{}{
			"tx_hash":        txHash,
			"status":         biz.RechargeStatusConfirmed,
			"confirmed_time": now,
		}).Error; err != nil {
			return err
		}
		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, po.UserID).Error; err != nil {
			return err
		}
		asset := strings.ToUpper(strings.TrimSpace(po.Asset))
		if asset == biz.TokenWIN {
			user.WinRechargeBalance = user.WinRechargeBalance.Add(po.Amount)
			newBal = user.WinRechargeBalance.String()
			// WIN 充值不产生零号账户/社区补贴奖励（仅 USDT 充值触发）
			return tx.Model(&user).Update("win_recharge_balance", user.WinRechargeBalance).Error
		}
		if asset == biz.TokenWINA {
			return fmt.Errorf("win-a recharge disabled")
		}
		user.UsdtRecharge = user.UsdtRecharge.Add(po.Amount)
		newBal = user.UsdtRecharge.String()
		if err := payUSDTRechargeRoleRewards(tx, user.ID, po.Amount); err != nil {
			return err
		}
		return tx.Model(&user).Update("usdt_recharge", user.UsdtRecharge).Error
	})
	return newBal, err
}

func (r *walletRepo) DeletePendingRecharge(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ? AND status <> ?", id, biz.RechargeStatusConfirmed).
		Delete(&RechargePO{}).Error
}

// AutoCreditChainRecharge records and credits a confirmed on-chain USDT transfer.
// The unique tx_hash index makes this idempotent with both the background scanner
// and the user-triggered recharge confirmation flow.
func (r *walletRepo) AutoCreditChainRecharge(
	ctx context.Context,
	txHash, fromAddress, toAddress, amount string,
	blockNumber uint64,
) (bool, error) {
	txHash = strings.TrimSpace(txHash)
	fromAddress = strings.TrimSpace(fromAddress)
	toAddress = strings.TrimSpace(toAddress)
	amountDec, err := decimal.NewFromString(strings.TrimSpace(amount))
	if txHash == "" || fromAddress == "" || toAddress == "" || err != nil || !amountDec.GreaterThan(decimal.Zero) {
		return false, fmt.Errorf("invalid chain recharge")
	}

	credited := false
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&RechargePO{}).Where("tx_hash = ?", txHash).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}

		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("address = ?", fromAddress).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		now := time.Now()
		recharge := &RechargePO{
			UserID:        user.ID,
			Asset:         biz.TokenUSDT,
			Amount:        amountDec,
			TxHash:        txHash,
			FromAddress:   fromAddress,
			ToAddress:     toAddress,
			Status:        biz.RechargeStatusConfirmed,
			Message:       fmt.Sprintf("chain_monitor:block=%d", blockNumber),
			ConfirmedTime: &now,
		}
		if err := tx.Create(recharge).Error; err != nil {
			// A concurrent manual confirmation may have inserted the same hash.
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return nil
			}
			return err
		}

		user.UsdtRecharge = user.UsdtRecharge.Add(amountDec)
		if err := payUSDTRechargeRoleRewards(tx, user.ID, amountDec); err != nil {
			return err
		}
		if err := tx.Model(&user).Update("usdt_recharge", user.UsdtRecharge).Error; err != nil {
			return err
		}
		credited = true
		return nil
	})
	return credited, err
}

// AutoCreditWinRecharge 确认链上 WIN 转账并入账 win_recharge_balance（tx_hash 幂等）。
func (r *walletRepo) AutoCreditWinRecharge(
	ctx context.Context,
	txHash, fromAddress, toAddress, amount string,
) (bool, string, error) {
	txHash = strings.TrimSpace(txHash)
	fromAddress = strings.TrimSpace(fromAddress)
	toAddress = strings.TrimSpace(toAddress)
	amountDec, err := decimal.NewFromString(strings.TrimSpace(amount))
	if txHash == "" || fromAddress == "" || toAddress == "" || err != nil || !amountDec.GreaterThan(decimal.Zero) {
		return false, "", fmt.Errorf("invalid win recharge")
	}

	credited := false
	var newBal string
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing RechargePO
		findErr := tx.Where("tx_hash = ?", txHash).First(&existing).Error
		if findErr == nil {
			if existing.Status == biz.RechargeStatusConfirmed && strings.EqualFold(existing.Asset, biz.TokenWIN) {
				var user UserPO
				if err := tx.First(&user, existing.UserID).Error; err != nil {
					return err
				}
				newBal = user.WinRechargeBalance.String()
			}
			return nil
		}
		if findErr != nil && findErr != gorm.ErrRecordNotFound {
			return findErr
		}

		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("LOWER(address) = ?", strings.ToLower(fromAddress)).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("user not found")
			}
			return err
		}

		now := time.Now()
		recharge := &RechargePO{
			UserID:        user.ID,
			Asset:         biz.TokenWIN,
			Amount:        amountDec,
			TxHash:        txHash,
			FromAddress:   fromAddress,
			ToAddress:     toAddress,
			Status:        biz.RechargeStatusConfirmed,
			Message:       "win_deposit_only",
			ConfirmedTime: &now,
		}
		if err := tx.Create(recharge).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return nil
			}
			return err
		}

		user.WinRechargeBalance = user.WinRechargeBalance.Add(amountDec)
		if err := tx.Model(&user).Update("win_recharge_balance", user.WinRechargeBalance).Error; err != nil {
			return err
		}
		// WIN 充值不产生零号账户/社区补贴奖励（仅 USDT 充值触发）
		newBal = user.WinRechargeBalance.String()
		credited = true
		return nil
	})
	return credited, newBal, err
}

// AutoCreditWinARecharge WIN-A 充值已关闭（保留接口供历史幂等查询）。
func (r *walletRepo) AutoCreditWinARecharge(
	ctx context.Context,
	txHash, fromAddress, toAddress, amount string,
) (bool, string, error) {
	return false, "", fmt.Errorf("win-a recharge disabled")
}

func (r *walletRepo) ListRechargesByUser(ctx context.Context, userID int64) ([]*biz.Recharge, error) {
	var list []RechargePO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Recharge, 0, len(list))
	for i := range list {
		out = append(out, r.rechargeToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) ListRechargesByUserAsset(ctx context.Context, userID int64, asset string) ([]*biz.Recharge, error) {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	q := r.data.db.WithContext(ctx).Where("user_id = ?", userID).
		// 交易所划转不在用户充值记录中展示
		Where("tx_hash NOT LIKE ?", "partner:%")
	if asset != "" {
		q = q.Where("asset = ?", asset)
	}
	var list []RechargePO
	if err := q.Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Recharge, 0, len(list))
	for i := range list {
		out = append(out, r.rechargeToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) ListConfirmedUSDTRechargesByUserIDs(
	ctx context.Context, userIDs []int64, offset, limit int,
) ([]*biz.Recharge, int64, error) {
	if len(userIDs) == 0 {
		return nil, 0, nil
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	// 下级充值展示：仅 USDT（空 asset 视为 USDT；不含 WIN / WIN-A）
	assetCond := "(UPPER(asset) = ? OR asset = '' OR asset IS NULL)"
	base := r.data.db.WithContext(ctx).
		Model(&RechargePO{}).
		Where("user_id IN ?", userIDs).
		Where("status = ?", biz.RechargeStatusConfirmed).
		Where(assetCond, biz.TokenUSDT)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type rechargeRow struct {
		RechargePO
		UserAddress string `gorm:"column:user_address"`
	}
	var rows []rechargeRow
	if err := r.data.db.WithContext(ctx).
		Table("recharges r").
		Select("r.*, u.address AS user_address").
		Joins("JOIN users u ON u.id = r.user_id").
		Where("r.user_id IN ?", userIDs).
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		Where("(UPPER(r.asset) = ? OR r.asset = '' OR r.asset IS NULL)", biz.TokenUSDT).
		Order("r.id DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.Recharge, 0, len(rows))
	for i := range rows {
		rec := r.rechargeToBiz(&rows[i].RechargePO)
		if addr := strings.TrimSpace(rows[i].UserAddress); addr != "" {
			rec.Address = addr
		}
		if strings.TrimSpace(rec.Asset) == "" {
			rec.Asset = biz.TokenUSDT
		}
		out = append(out, rec)
	}
	return out, total, nil
}

func (r *walletRepo) ListOrdersByUserIDs(
	ctx context.Context, userIDs []int64, offset, limit int,
) ([]*biz.AdminOrderDetail, int64, error) {
	if len(userIDs) == 0 {
		return nil, 0, nil
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	base := r.data.db.WithContext(ctx).Model(&OrderPO{}).
		Where("user_id IN ?", userIDs).
		Where("status <> ?", biz.OrderStatusCancelled)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ID          int64
		UserID      int64
		Principal   decimal.Decimal
		FromWin     decimal.Decimal
		FundSource  string
		Status      string
		CreatedTime time.Time
		Address     string
	}
	var rows []row
	err := r.data.db.WithContext(ctx).Table("orders AS o").
		Select("o.id, o.user_id, o.principal, o.from_win, o.fund_source, o.status, o.created_time, COALESCE(u.address,'') AS address").
		Joins("LEFT JOIN users AS u ON u.id = o.user_id").
		Where("o.user_id IN ?", userIDs).
		Where("o.status <> ?", biz.OrderStatusCancelled).
		Order("o.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminOrderDetail, 0, len(rows))
	for _, rw := range rows {
		o := &biz.Order{
			ID:          rw.ID,
			UserID:      rw.UserID,
			Principal:   rw.Principal.String(),
			FromWin:     rw.FromWin.String(),
			FundSource:  rw.FundSource,
			Status:      rw.Status,
			CreatedTime: rw.CreatedTime,
		}
		o.SyncCompatFields()
		out = append(out, &biz.AdminOrderDetail{Order: o, UserAddress: rw.Address})
	}
	return out, total, nil
}

func (r *walletRepo) Subscribe(ctx context.Context, userID int64, in biz.SubscribeInput) (*biz.Order, string, error) {
	principal, err := decimal.NewFromString(in.Amount)
	if err != nil {
		return nil, "", err
	}
	exitMul := in.ExitMul
	directRate := in.DirectRate
	if exitMul <= 0 {
		exitMul = 4
	}
	if directRate <= 0 {
		directRate = 0.5
	}
	payFrom := strings.ToLower(strings.TrimSpace(in.PayFrom))
	exitCap := principal.Mul(decimal.NewFromFloat(exitMul))
	var created *biz.Order
	var balOut string

	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		fromRecharge := decimal.Zero
		fromReward := decimal.Zero
		fromWin := decimal.Zero
		fromWinA := decimal.Zero
		directBase := decimal.Zero
		winPriceSnap := decimal.Zero
		updates := map[string]interface{}{}
		fundSource := payFrom
		skipMgmt := payFrom == biz.PayFromReward
		orderPoints := decimal.Zero
		pointsSource := ""
		switch payFrom {
		case biz.PayFromRecharge:
			orderPoints = principal
			pointsSource = biz.PointsSourceRecharge
		case biz.PayFromWin:
			orderPoints = principal
			pointsSource = biz.PointsSourceWin
		case biz.PayFromReward:
			// 仅消耗「邀请链上级划入」额度；额度来自任意层级上级→下级划转，不限直推。
			orderPoints = decimal.Min(principal, user.TransferReinvestCredit)
			if orderPoints.IsPositive() {
				user.TransferReinvestCredit = user.TransferReinvestCredit.Sub(orderPoints)
				updates["transfer_reinvest_credit"] = user.TransferReinvestCredit
				pointsSource = biz.PointsSourceTransferReinvest
			}
		}

		if payFrom != biz.PayFromRecharge && payFrom != biz.PayFromReward && payFrom != biz.PayFromWin {
			return fmt.Errorf("invalid pay_from")
		}
		if payFrom != biz.PayFromReward {
			directBase = principal
		}

		switch payFrom {
		case biz.PayFromRecharge:
			if user.UsdtRecharge.LessThan(principal) {
				return fmt.Errorf("insufficient usdt_recharge")
			}
			user.UsdtRecharge = user.UsdtRecharge.Sub(principal)
			fromRecharge = principal
			updates["usdt_recharge"] = user.UsdtRecharge
			balOut = user.UsdtRecharge.String()
		case biz.PayFromReward:
			if user.UsdtReward.LessThan(principal) {
				return fmt.Errorf("insufficient usdt_reward")
			}
			user.UsdtReward = user.UsdtReward.Sub(principal)
			fromReward = principal
			updates["usdt_reward"] = user.UsdtReward
			balOut = user.UsdtReward.String()
		case biz.PayFromWin:
			winPriceSnap = decimal.NewFromFloat(biz.GetWinPrice())
			if !winPriceSnap.IsPositive() {
				return fmt.Errorf("win price not configured")
			}
			nativeAmt := principal.Div(winPriceSnap).Round(8)
			if !nativeAmt.IsPositive() {
				return fmt.Errorf("win amount too small")
			}
			if user.WinRechargeBalance.LessThan(nativeAmt) {
				return fmt.Errorf("insufficient win_recharge_balance")
			}
			user.WinRechargeBalance = user.WinRechargeBalance.Sub(nativeAmt)
			fromWin = nativeAmt
			updates["win_recharge_balance"] = user.WinRechargeBalance
			balOut = user.WinRechargeBalance.String()
		}

		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		po := &OrderPO{
			UserID:       userID,
			Principal:    principal,
			ExitCap:      exitCap,
			EarnedTotal:  decimal.Zero,
			DirectBase:   directBase,
			FromRecharge: fromRecharge,
			FromReward:   fromReward,
			FromWin:      fromWin,
			FromWinA:     fromWinA,
			Points:       orderPoints,
			PointsSource: pointsSource,
			FundSource:   fundSource,
			Status:       biz.OrderStatusActive,
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		if orderPoints.IsPositive() {
			user.Points = user.Points.Add(orderPoints)
			user.PointsAll = user.PointsAll.Add(orderPoints)
			if err := tx.Model(&user).Updates(map[string]interface{}{
				"points":     user.Points,
				"points_all": user.PointsAll,
			}).Error; err != nil {
				return err
			}
		}
		created = r.orderToBiz(po)
		if fromWin.IsPositive() && winPriceSnap.IsPositive() {
			created.WinPrice = winPriceSnap.String()
		}

		// 直推奖：recharge / win 且有上级（奖励复投不产生直推奖）
		if directBase.IsPositive() && user.InviterID != nil {
			if err := r.payDirectReward(tx, *user.InviterID, userID, po.ID, directBase, directRate); err != nil {
				return err
			}
		}
		// Keep the order and every ancestor's cached performance atomic.
		if err := refreshAncestorPerformance(tx, userID); err != nil {
			return err
		}
		// 管理奖：充值钱包 / WIN 报单产生（大区、小区均可）；奖励复投只增加上级业绩
		if !skipMgmt {
			if err := r.createManagementRewards(tx, &user, po); err != nil {
				return err
			}
		}
		// 本人新认购：只释放直推溢出；管理奖已在下级认购时一次性入账
		if err := r.releasePendingManagementRewards(tx, userID); err != nil {
			return err
		}
		return nil
	})
	return created, balOut, err
}

// createManagementRewards 下级认购时按级差发放管理奖。
// 规则：等级由小区业绩门槛决定（不要求大区达标）；管理奖对大区、小区来源均发放。
// 向上级差：仅正 gap 产生奖励；平级不发。
// 可释放部分进 usdt_reward 并计入出局，超出进 overflow_reward，待本人下次认购释放。
func (r *walletRepo) createManagementRewards(tx *gorm.DB, sourceUser *UserPO, sourceOrder *OrderPO) error {
	if sourceUser == nil || sourceOrder == nil || sourceUser.InviterID == nil || !sourceOrder.Principal.IsPositive() {
		return nil
	}
	currentID := *sourceUser.InviterID
	highestLowerRate := decimal.Zero
	seen := map[int64]bool{sourceUser.ID: true}
	for currentID > 0 {
		if seen[currentID] {
			return fmt.Errorf("invite relationship contains a cycle at user %d", currentID)
		}
		seen[currentID] = true

		var ancestor UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ancestor, currentID).Error; err != nil {
			return err
		}
		rate := decimal.NewFromFloat(biz.MgmtRateForLevel(ancestor.MgmtLevel))
		gap := rate.Sub(highestLowerRate)

		if gap.IsPositive() {
			total := sourceOrder.Principal.Mul(gap).Round(8)
			if total.IsPositive() {
				var existing MgmtRewardPO
				err := tx.Where("user_id = ? AND source_order_id = ?", currentID, sourceOrder.ID).
					First(&existing).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					reward := &MgmtRewardPO{
						UserID: currentID, FromUserID: sourceUser.ID, SourceOrderID: sourceOrder.ID,
						BaseAmount: sourceOrder.Principal, Rate: gap, TotalAmount: total,
					}
					if err := tx.Create(reward).Error; err != nil {
						return err
					}
					if err := r.tryReleaseMgmtAgainstExitCap(tx, currentID, reward); err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
			}
		}
		if rate.GreaterThan(highestLowerRate) {
			highestLowerRate = rate
		}
		if ancestor.InviterID == nil {
			break
		}
		currentID = *ancestor.InviterID
	}
	return nil
}

// tryReleaseMgmtAgainstExitCap 管理奖按出局剩余入账：可释放部分进奖励钱包并计入出局；
// 超出部分进入 overflow_reward，待本人下次认购再释放。同一来源订单只结算一次。
func (r *walletRepo) tryReleaseMgmtAgainstExitCap(tx *gorm.DB, userID int64, reward *MgmtRewardPO) error {
	if reward == nil || !reward.TotalAmount.IsPositive() {
		return nil
	}
	if reward.ReleasedAmount.GreaterThanOrEqual(reward.TotalAmount) {
		return nil
	}
	var logCount int64
	if err := tx.Model(&RewardLogPO{}).
		Where("user_id = ? AND order_id = ? AND type IN ?", userID, reward.SourceOrderID,
			[]string{biz.RewardTypeMgmt, biz.RewardTypeMgmtOverflow}).
		Count(&logCount).Error; err != nil {
		return err
	}
	if logCount > 0 {
		// 已有管理奖流水：视为该来源订单已结算完毕（含当时写入溢出的部分）。
		reward.ReleasedAmount = reward.TotalAmount
		return tx.Model(reward).Update("released_amount", reward.ReleasedAmount).Error
	}
	var orders []OrderPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ?", userID, biz.OrderStatusActive).
		Order("id asc").Find(&orders).Error; err != nil {
		return err
	}
	var user UserPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
		return err
	}

	want := reward.TotalAmount
	remainCap := decimal.Zero
	for _, o := range orders {
		rem := o.ExitCap.Sub(o.EarnedTotal)
		if rem.IsPositive() {
			remainCap = remainCap.Add(rem)
		}
	}
	pay := decimal.Zero
	if remainCap.IsPositive() {
		pay = want
		if pay.GreaterThan(remainCap) {
			pay = remainCap
		}
	}
	overflow := want.Sub(pay)
	exitApplied := decimal.Zero
	if pay.IsPositive() {
		var err error
		exitApplied, err = applyExitToOrders(tx, orders, pay)
		if err != nil {
			return err
		}
		user.UsdtReward = user.UsdtReward.Add(pay)
		fromID := reward.FromUserID
		sourceOrderID := reward.SourceOrderID
		base := reward.BaseAmount
		rate := reward.Rate
		if err := tx.Create(&RewardLogPO{
			UserID:      userID,
			FromUserID:  &fromID,
			OrderID:     &sourceOrderID,
			Type:        biz.RewardTypeMgmt,
			Asset:       biz.TokenUSDT,
			Amount:      pay,
			BaseAmount:  &base,
			Rate:        &rate,
			ExitApplied: exitApplied,
		}).Error; err != nil {
			return err
		}
	}
	if overflow.IsPositive() {
		user.OverflowReward = user.OverflowReward.Add(overflow)
		user.PendingMgmtReward = user.OverflowReward
		// 溢出部分也留流水：出局额度用尽时管理奖同样要在后台可查
		fromID := reward.FromUserID
		sourceOrderID := reward.SourceOrderID
		base := reward.BaseAmount
		rate := reward.Rate
		if err := tx.Create(&RewardLogPO{
			UserID:      userID,
			FromUserID:  &fromID,
			OrderID:     &sourceOrderID,
			Type:        biz.RewardTypeMgmtOverflow,
			Asset:       biz.TokenUSDT,
			Amount:      overflow,
			BaseAmount:  &base,
			Rate:        &rate,
			ExitApplied: decimal.Zero,
		}).Error; err != nil {
			return err
		}
	}
	// 应得已拆入「已入账 + 溢出」；标记 ReleasedAmount=Total，避免同一来源重复结算。
	reward.ReleasedAmount = want
	if err := tx.Model(reward).Update("released_amount", reward.ReleasedAmount).Error; err != nil {
		return err
	}
	return tx.Model(&user).Updates(map[string]interface{}{
		"usdt_reward":         user.UsdtReward,
		"overflow_reward":     user.OverflowReward,
		"pending_mgmt_reward": user.OverflowReward,
	}).Error
}

func applyExitToOrders(tx *gorm.DB, orders []OrderPO, amount decimal.Decimal) (decimal.Decimal, error) {
	applied := decimal.Zero
	if !amount.IsPositive() {
		return applied, nil
	}
	left := amount
	for i := range orders {
		if left.LessThanOrEqual(decimal.Zero) {
			break
		}
		o := &orders[i]
		rem := o.ExitCap.Sub(o.EarnedTotal)
		if rem.LessThanOrEqual(decimal.Zero) {
			continue
		}
		apply := left
		if apply.GreaterThan(rem) {
			apply = rem
		}
		o.EarnedTotal = o.EarnedTotal.Add(apply)
		status := biz.OrderStatusActive
		var exitedAt *time.Time
		if o.EarnedTotal.GreaterThanOrEqual(o.ExitCap) {
			status = biz.OrderStatusExited
			t := time.Now()
			exitedAt = &t
			o.EarnedTotal = o.ExitCap
		}
		if err := tx.Model(o).Updates(map[string]interface{}{
			"earned_total": o.EarnedTotal,
			"status":       status,
			"exited_time":  exitedAt,
		}).Error; err != nil {
			return decimal.Zero, err
		}
		applied = applied.Add(apply)
		left = left.Sub(apply)
	}
	return applied, nil
}

const (
	overflowKindMgmt   = "mgmt"
	overflowKindDirect = "direct"
)

// drainMgmtPool 将管理奖溢出按活跃订单出局容量释放进奖励钱包。
func (r *walletRepo) drainMgmtPool(tx *gorm.DB, userID int64) error {
	return r.drainOverflowPool(tx, userID, overflowKindMgmt)
}

// drainDirectPool 将直推奖溢出按活跃订单出局容量释放，流水类型为直推奖。
func (r *walletRepo) drainDirectPool(tx *gorm.DB, userID int64) error {
	return r.drainOverflowPool(tx, userID, overflowKindDirect)
}

func (r *walletRepo) drainOverflowPool(tx *gorm.DB, userID int64, kind string) error {
	var user UserPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
		return err
	}
	want := user.OverflowDirect
	rewardType := biz.RewardTypeDirectPoolRelease
	if kind == overflowKindMgmt {
		want = user.OverflowReward
		rewardType = biz.RewardTypeMgmtPoolRelease
		if !want.IsPositive() && user.PendingMgmtReward.IsPositive() {
			// 兼容尚未迁移的旧数据
			user.OverflowReward = user.PendingMgmtReward
			want = user.OverflowReward
		}
	}
	if !want.IsPositive() {
		return nil
	}

	var orders []OrderPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ?", userID, biz.OrderStatusActive).
		Order("id asc").Find(&orders).Error; err != nil {
		return err
	}
	remainTotal := decimal.Zero
	for _, o := range orders {
		rem := o.ExitCap.Sub(o.EarnedTotal)
		if rem.IsPositive() {
			remainTotal = remainTotal.Add(rem)
		}
	}
	if remainTotal.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	pay := want
	if pay.GreaterThan(remainTotal) {
		pay = remainTotal
	}
	exitApplied, err := applyExitToOrders(tx, orders, pay)
	if err != nil {
		return err
	}

	user.UsdtReward = user.UsdtReward.Add(pay)
	if kind == overflowKindDirect {
		user.OverflowDirect = user.OverflowDirect.Sub(pay)
		if user.OverflowDirect.IsNegative() {
			user.OverflowDirect = decimal.Zero
		}
	} else {
		user.OverflowReward = user.OverflowReward.Sub(pay)
		if user.OverflowReward.IsNegative() {
			user.OverflowReward = decimal.Zero
		}
		user.PendingMgmtReward = user.OverflowReward
	}
	if err := tx.Create(&RewardLogPO{
		UserID:      userID,
		Type:        rewardType,
		Asset:       biz.TokenUSDT,
		Amount:      pay,
		ExitApplied: exitApplied,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&user).Updates(map[string]interface{}{
		"usdt_reward":         user.UsdtReward,
		"overflow_reward":     user.OverflowReward,
		"pending_mgmt_reward": user.OverflowReward,
		"overflow_direct":     user.OverflowDirect,
	}).Error
}

// releasePendingManagementRewards 本人新认购时释放溢出：
// 直推溢出 + 管理奖溢出（均受出局剩余限制，发不完继续留在溢出池）。
func (r *walletRepo) releasePendingManagementRewards(tx *gorm.DB, userID int64) error {
	if err := r.drainDirectPool(tx, userID); err != nil {
		return err
	}
	return r.drainMgmtPool(tx, userID)
}

func (r *walletRepo) payDirectReward(tx *gorm.DB, inviterID, fromUserID, orderID int64, directBase decimal.Decimal, directRate float64) error {
	want := directBase.Mul(decimal.NewFromFloat(directRate)).Round(8)
	if !want.IsPositive() {
		return nil
	}
	var orders []OrderPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ?", inviterID, biz.OrderStatusActive).
		Order("id asc").Find(&orders).Error; err != nil {
		return err
	}
	remainCap := decimal.Zero
	for _, o := range orders {
		rem := o.ExitCap.Sub(o.EarnedTotal)
		if rem.IsPositive() {
			remainCap = remainCap.Add(rem)
		}
	}

	var inviter UserPO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&inviter, inviterID).Error; err != nil {
		return err
	}

	fromID := fromUserID
	oid := orderID
	base := directBase
	rate := decimal.NewFromFloat(directRate)

	// 直推奖不超过全部认购订单出局剩余：可释放进奖励钱包并计入出局；超出部分进溢出，待本人新认购后再释放。
	pay := decimal.Zero
	if remainCap.IsPositive() {
		pay = want
		if pay.GreaterThan(remainCap) {
			pay = remainCap
		}
	}
	overflow := want.Sub(pay)
	exitApplied := decimal.Zero

	if pay.IsPositive() {
		left := pay
		for i := range orders {
			if left.LessThanOrEqual(decimal.Zero) {
				break
			}
			o := &orders[i]
			remain := o.ExitCap.Sub(o.EarnedTotal)
			if remain.LessThanOrEqual(decimal.Zero) {
				continue
			}
			apply := left
			if apply.GreaterThan(remain) {
				apply = remain
			}
			o.EarnedTotal = o.EarnedTotal.Add(apply)
			updates := map[string]interface{}{"earned_total": o.EarnedTotal}
			if o.EarnedTotal.GreaterThanOrEqual(o.ExitCap) {
				now := time.Now()
				updates["status"] = biz.OrderStatusExited
				updates["exited_time"] = now
				o.EarnedTotal = o.ExitCap
				updates["earned_total"] = o.EarnedTotal
			}
			if err := tx.Model(o).Updates(updates).Error; err != nil {
				return err
			}
			exitApplied = exitApplied.Add(apply)
			left = left.Sub(apply)
		}
		inviter.UsdtReward = inviter.UsdtReward.Add(pay)
	}
	if overflow.IsPositive() {
		inviter.OverflowDirect = inviter.OverflowDirect.Add(overflow)
	}
	if err := tx.Model(&inviter).Updates(map[string]interface{}{
		"usdt_reward":     inviter.UsdtReward,
		"overflow_direct": inviter.OverflowDirect,
	}).Error; err != nil {
		return err
	}
	if !pay.IsPositive() {
		return nil
	}
	return tx.Create(&RewardLogPO{
		UserID:      inviterID,
		FromUserID:  &fromID,
		OrderID:     &oid,
		Type:        biz.RewardTypeDynamicUsdt,
		Asset:       biz.TokenUSDT,
		Amount:      pay,
		BaseAmount:  &base,
		Rate:        &rate,
		ExitApplied: exitApplied,
	}).Error
}

func (r *walletRepo) ListOrdersByUser(ctx context.Context, userID int64) ([]*biz.Order, error) {
	var list []OrderPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Order, 0, len(list))
	for i := range list {
		out = append(out, r.orderToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) ListAllOrders(ctx context.Context) ([]*biz.AdminOrderDetail, error) {
	var list []OrderPO
	if err := r.data.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.AdminOrderDetail, 0, len(list))
	for i := range list {
		addr := ""
		var u UserPO
		if err := r.data.db.WithContext(ctx).Select("address").First(&u, list[i].UserID).Error; err == nil {
			addr = u.Address
		}
		o := r.orderToBiz(&list[i])
		o.SyncCompatFields()
		out = append(out, &biz.AdminOrderDetail{Order: o, UserAddress: addr})
	}
	return out, nil
}

func (r *walletRepo) ListSubscribeOrdersPaged(ctx context.Context, offset, limit int) ([]*biz.AdminOrderDetail, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var total int64
	if err := r.data.db.WithContext(ctx).Model(&OrderPO{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ID          int64
		UserID      int64
		Principal   decimal.Decimal
		Points      decimal.Decimal
		FundSource  string
		Status      string
		CreatedTime time.Time
		Address     string
	}
	var rows []row
	err := r.data.db.WithContext(ctx).Table("orders AS o").
		Select("o.id, o.user_id, o.principal, o.points, o.fund_source, o.status, o.created_time, COALESCE(u.address,'') AS address").
		Joins("LEFT JOIN users AS u ON u.id = o.user_id").
		Order("o.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.AdminOrderDetail, 0, len(rows))
	for _, rw := range rows {
		o := &biz.Order{
			ID: rw.ID, UserID: rw.UserID,
			Principal: rw.Principal.String(),
			Points:    rw.Points.String(),
			FundSource: rw.FundSource,
			Status:    rw.Status,
			CreatedTime: rw.CreatedTime,
		}
		if o.Points == "" || o.Points == "0" {
			o.Points = o.Principal
		}
		o.SyncCompatFields()
		out = append(out, &biz.AdminOrderDetail{Order: o, UserAddress: rw.Address})
	}
	return out, total, nil
}

func (r *walletRepo) FindOrder(ctx context.Context, id int64) (*biz.Order, error) {
	var po OrderPO
	err := r.data.db.WithContext(ctx).First(&po, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o := r.orderToBiz(&po)
	o.SyncCompatFields()
	return o, nil
}

func (r *walletRepo) RemainingExitCapacity(ctx context.Context, userID int64) (string, error) {
	var list []OrderPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, biz.OrderStatusActive).Find(&list).Error; err != nil {
		return "", err
	}
	total := decimal.Zero
	for _, o := range list {
		r := o.ExitCap.Sub(o.EarnedTotal)
		if r.IsPositive() {
			total = total.Add(r)
		}
	}
	return total.String(), nil
}

func (r *walletRepo) CreateTransfer(ctx context.Context, t *biz.Transfer) (*biz.Transfer, error) {
	amount, err := decimal.NewFromString(t.Amount)
	if err != nil {
		return nil, err
	}
	var created *biz.Transfer
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var from, to UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&from, t.FromUserID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&to, t.ToUserID).Error; err != nil {
			return err
		}
		po := &TransferPO{
			FromUserID: t.FromUserID,
			ToUserID:   t.ToUserID,
			Asset:      t.Asset,
			Amount:     amount,
			PayFrom:    t.PayFrom,
			Remark:     t.Remark,
		}
		switch t.Asset {
		case biz.TokenUSDT:
			if t.PayFrom != biz.PayFromReward {
				return fmt.Errorf("invalid pay_from")
			}
			if from.UsdtReward.LessThan(amount) {
				return fmt.Errorf("insufficient usdt_reward")
			}
			from.UsdtReward = from.UsdtReward.Sub(amount)
			po.FromRewardDebit = amount
			to.UsdtReward = to.UsdtReward.Add(amount)
			po.ToCreditReward = amount
		default:
			return fmt.Errorf("invalid asset")
		}
		fromUpdates := map[string]interface{}{
			"usdt_recharge": from.UsdtRecharge,
			"usdt_reward":   from.UsdtReward,
			"aix_balance":   from.AixBalance,
		}
		toUpdates := map[string]interface{}{
			"usdt_reward": to.UsdtReward,
			"aix_balance": to.AixBalance,
		}
		// 额度按邀请链祖先关系处理：任意层级上级均可，不要求直推。
		if uplineToDownline, err := isAncestorInTx(tx, t.FromUserID, t.ToUserID); err != nil {
			return err
		} else if uplineToDownline {
			// 任意上级→下级：无存量额度时全额记给接收方（如 A→B / A→隔代C）；
			// 有存量额度时只传递额度并扣减发送方（如 B→C），避免同一笔资金重复产生 AIX-USDT。
			var creditAdd decimal.Decimal
			if from.TransferReinvestCredit.IsPositive() {
				creditAdd = decimal.Min(amount, from.TransferReinvestCredit)
				from.TransferReinvestCredit = from.TransferReinvestCredit.Sub(creditAdd)
				fromUpdates["transfer_reinvest_credit"] = from.TransferReinvestCredit
			} else {
				creditAdd = amount
			}
			if creditAdd.IsPositive() {
				to.TransferReinvestCredit = to.TransferReinvestCredit.Add(creditAdd)
				toUpdates["transfer_reinvest_credit"] = to.TransferReinvestCredit
			}
		} else if downlineToUpline, err := isAncestorInTx(tx, t.ToUserID, t.FromUserID); err != nil {
			return err
		} else if downlineToUpline {
			deduct := decimal.Min(amount, from.TransferReinvestCredit)
			if deduct.IsPositive() {
				from.TransferReinvestCredit = from.TransferReinvestCredit.Sub(deduct)
				fromUpdates["transfer_reinvest_credit"] = from.TransferReinvestCredit
			}
		}
		if err := tx.Model(&from).Updates(fromUpdates).Error; err != nil {
			return err
		}
		if err := tx.Model(&to).Updates(toUpdates).Error; err != nil {
			return err
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		created = r.transferToBiz(po)
		return nil
	})
	return created, err
}

// isAncestorInTx 判断 ancestorID 是否在 nodeID 的邀请链上方（任意层级，含直推与隔代）。
func isAncestorInTx(tx *gorm.DB, ancestorID, nodeID int64) (bool, error) {
	if ancestorID <= 0 || nodeID <= 0 || ancestorID == nodeID {
		return false, nil
	}
	seen := map[int64]bool{nodeID: true}
	current := nodeID
	for current > 0 {
		var u UserPO
		if err := tx.Select("id", "inviter_id").First(&u, current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		if u.InviterID == nil {
			return false, nil
		}
		parent := *u.InviterID
		if seen[parent] {
			return false, nil
		}
		if parent == ancestorID {
			return true, nil
		}
		seen[parent] = true
		current = parent
	}
	return false, nil
}

func (r *walletRepo) ListTransfersByUser(ctx context.Context, userID int64) ([]*biz.Transfer, error) {
	var list []TransferPO
	if err := r.data.db.WithContext(ctx).
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Transfer, 0, len(list))
	for i := range list {
		out = append(out, r.transferToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) CreateAixWithdrawal(ctx context.Context, userID int64, amount, toAddress string) (*biz.Withdrawal, string, error) {
	return nil, "", fmt.Errorf("AIX withdraw disabled; exchange AIX to USDT withdrawable balance first")
}

// ExchangeAixToWin AIX → 可提 U（USDT）兑换。
// needReview 为 true 时只扣 AIX、不入可提 U，状态 review，待后台审核通过后入账。
func (r *walletRepo) ExchangeAixToWin(ctx context.Context, userID int64, aixAmount string, needReview bool) (*biz.ExchangeRecord, string, string, error) {
	amt, err := decimal.NewFromString(aixAmount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", "", fmt.Errorf("invalid amount")
	}
	price := decimal.NewFromFloat(biz.AixPriceInitial)
	if !price.IsPositive() {
		return nil, "", "", fmt.Errorf("aix price not configured")
	}
	feeRate := decimal.NewFromFloat(biz.GetExchangeFeeRate())
	usdtGross := amt.Mul(price).Round(8)
	if !usdtGross.IsPositive() {
		return nil, "", "", fmt.Errorf("usdt amount too small")
	}
	feeAmount := usdtGross.Mul(feeRate).Round(8)
	usdtNet := usdtGross.Sub(feeAmount).Round(8)
	if !usdtNet.IsPositive() {
		return nil, "", "", fmt.Errorf("usdt net amount too small after fee")
	}

	status := biz.ExchangeStatusCompleted
	remark := fmt.Sprintf("AIX→USDT exchange at price %s USDT/AIX, fee %s%%", price.String(), feeRate.Mul(decimal.NewFromInt(100)).Round(2).String())
	if needReview {
		status = biz.ExchangeStatusReview
		remark = remark + "; pending admin review (daily AIX exchange quota exceeded)"
	}

	var rec *biz.ExchangeRecord
	var aixLeft, usdtWithdrawable string
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
			return err
		}
		if !u.ExchangeEnabled {
			return fmt.Errorf("exchange disabled")
		}
		if u.AixBalance.LessThan(amt) {
			return fmt.Errorf("insufficient aix_balance")
		}
		u.AixBalance = u.AixBalance.Sub(amt)
		updates := map[string]interface{}{
			"aix_balance": u.AixBalance,
		}
		if !needReview {
			u.UsdtWithdrawable = u.UsdtWithdrawable.Add(usdtNet)
			updates["usdt_withdrawable"] = u.UsdtWithdrawable
		}
		if err := tx.Model(&u).Updates(updates).Error; err != nil {
			return err
		}
		po := &ExchangeRecordPO{
			UserID:        userID,
			FromAsset:     biz.TokenAIX,
			FromAmount:    amt,
			ToAsset:       biz.TokenUSDT,
			ToAmount:      usdtNet,
			FeeAmount:     feeAmount,
			ExchangePrice: price,
			FeeRate:       feeRate.Round(6),
			Status:        status,
			Remark:        remark,
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		aixLeft = u.AixBalance.String()
		usdtWithdrawable = u.UsdtWithdrawable.String()
		rec = &biz.ExchangeRecord{
			ID: po.ID, UserID: po.UserID, UserAddress: u.Address,
			FromAsset: po.FromAsset, FromAmount: po.FromAmount.String(),
			ToAsset: po.ToAsset, ToAmount: po.ToAmount.String(),
			FeeAmount: po.FeeAmount.String(), FeeRate: po.FeeRate.String(),
			ExchangePrice: po.ExchangePrice.String(),
			Status:        po.Status, Remark: po.Remark, CreatedTime: po.CreatedTime,
		}
		return nil
	})
	return rec, aixLeft, usdtWithdrawable, err
}

func (r *walletRepo) ApproveExchangeReview(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po ExchangeRecordPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error; err != nil {
			return err
		}
		if po.Status != biz.ExchangeStatusReview {
			return fmt.Errorf("exchange %d not in review status", id)
		}
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, po.UserID).Error; err != nil {
			return err
		}
		u.UsdtWithdrawable = u.UsdtWithdrawable.Add(po.ToAmount)
		if err := tx.Model(&u).Update("usdt_withdrawable", u.UsdtWithdrawable).Error; err != nil {
			return err
		}
		return tx.Model(&po).Updates(map[string]interface{}{
			"status": biz.ExchangeStatusCompleted,
			"remark": po.Remark + "; admin approved",
		}).Error
	})
}

func (r *walletRepo) RejectExchangeReview(ctx context.Context, id int64, remark string) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po ExchangeRecordPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error; err != nil {
			return err
		}
		if po.Status != biz.ExchangeStatusReview {
			return fmt.Errorf("exchange %d not in review status", id)
		}
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, po.UserID).Error; err != nil {
			return err
		}
		u.AixBalance = u.AixBalance.Add(po.FromAmount)
		if err := tx.Model(&u).Update("aix_balance", u.AixBalance).Error; err != nil {
			return err
		}
		if strings.TrimSpace(remark) == "" {
			remark = "admin rejected"
		}
		return tx.Model(&po).Updates(map[string]interface{}{
			"status": biz.ExchangeStatusRejected,
			"remark": remark,
		}).Error
	})
}

func (r *walletRepo) SumStaticAixBySettlementDate(ctx context.Context, settlementDate string) (string, error) {
	var total decimal.Decimal
	err := r.data.db.WithContext(ctx).Model(&RewardLogPO{}).
		Where("type = ? AND settlement_date = ?", biz.RewardTypeStaticAix, settlementDate).
		Select("COALESCE(SUM(amount),0)").Scan(&total).Error
	if err != nil {
		return "0", err
	}
	return total.String(), nil
}

func (r *walletRepo) SumExchangedAixSince(ctx context.Context, since time.Time) (string, error) {
	var total decimal.Decimal
	// 仅 completed 计入当日配额；待审核/已拒绝都不计入，隔日也不会占用新一天的阈值。
	err := r.data.db.WithContext(ctx).Model(&ExchangeRecordPO{}).
		Where("UPPER(from_asset) = ? AND created_time >= ? AND status = ?",
			biz.TokenAIX, since, biz.ExchangeStatusCompleted).
		Select("COALESCE(SUM(from_amount),0)").Scan(&total).Error
	if err != nil {
		return "0", err
	}
	return total.String(), nil
}

func (r *walletRepo) ListExchangeRecordsByUser(ctx context.Context, userID int64) ([]*biz.ExchangeRecord, error) {
	var list []ExchangeRecordPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ExchangeRecord, 0, len(list))
	for _, po := range list {
		out = append(out, exchangeRecordToBiz(&po, ""))
	}
	return out, nil
}

func (r *walletRepo) ListAllExchangeRecords(ctx context.Context) ([]*biz.ExchangeRecord, error) {
	var list []ExchangeRecordPO
	if err := r.data.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []*biz.ExchangeRecord{}, nil
	}
	userIDs := make([]int64, 0, len(list))
	seen := map[int64]bool{}
	for _, po := range list {
		if !seen[po.UserID] {
			seen[po.UserID] = true
			userIDs = append(userIDs, po.UserID)
		}
	}
	var users []UserPO
	if err := r.data.db.WithContext(ctx).Select("id, address").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	addrMap := make(map[int64]string, len(users))
	for _, u := range users {
		addrMap[u.ID] = u.Address
	}
	out := make([]*biz.ExchangeRecord, 0, len(list))
	for i := range list {
		out = append(out, exchangeRecordToBiz(&list[i], addrMap[list[i].UserID]))
	}
	return out, nil
}

// CreateWinWithdrawal WIN 提现已关闭（保留接口供历史查询）。
func (r *walletRepo) CreateWinWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*biz.Withdrawal, string, error) {
	return nil, "", fmt.Errorf("win withdraw disabled")
}

// CreateSdtWithdrawal AIX-USDT 提现：扣 points，创建 WithdrawalPO
func (r *walletRepo) CreateSdtWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*biz.Withdrawal, string, error) {
	amt, err := decimal.NewFromString(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", fmt.Errorf("invalid amount")
	}
	var created *biz.Withdrawal
	var left string
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
			return err
		}
		if u.Points.LessThan(amt) {
			return fmt.Errorf("insufficient points")
		}
		u.Points = u.Points.Sub(amt)
		if err := tx.Model(&u).Update("points", u.Points).Error; err != nil {
			return err
		}
		if strings.TrimSpace(status) == "" {
			status = biz.WithdrawStatusPending
		}
		po := &WithdrawalPO{
			UserID:    userID,
			Asset:     biz.TokenSDT,
			Amount:    amt,
			Fee:       decimal.Zero,
			PayAmount: amt,
			ToAddress: toAddress,
			Status:    status,
			Remark:    "AIX-USDT withdraw; ERC20 payout pending",
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		left = u.Points.String()
		created = &biz.Withdrawal{
			ID: po.ID, UserID: po.UserID, Address: u.Address,
			ToAddress: po.ToAddress, Amount: po.Amount.String(),
			Fee: po.Fee.String(), NetAmount: po.PayAmount.String(),
			Status: po.Status, TxHash: po.TxHash, Asset: po.Asset,
			CreatedAt: po.CreatedTime,
		}
		return nil
	})
	return created, left, err
}

// CreateUsdtWithdrawal 可提 U 余额提现：扣 UsdtWithdrawable，创建 WithdrawalPO
func (r *walletRepo) CreateUsdtWithdrawal(ctx context.Context, userID int64, amount, toAddress, status string) (*biz.Withdrawal, string, error) {
	amt, err := decimal.NewFromString(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", fmt.Errorf("invalid amount")
	}
	var created *biz.Withdrawal
	var left string
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
			return err
		}
		if u.UsdtWithdrawable.LessThan(amt) {
			return fmt.Errorf("insufficient usdt_withdrawable")
		}
		u.UsdtWithdrawable = u.UsdtWithdrawable.Sub(amt)
		if err := tx.Model(&u).Update("usdt_withdrawable", u.UsdtWithdrawable).Error; err != nil {
			return err
		}
		if strings.TrimSpace(status) == "" {
			status = biz.WithdrawStatusPending
		}
		po := &WithdrawalPO{
			UserID:    userID,
			Asset:     biz.TokenUSDT,
			Amount:    amt,
			Fee:       decimal.Zero,
			PayAmount: amt,
			ToAddress: toAddress,
			Status:    status,
			Remark:    "USDT withdrawable withdraw; ERC20 payout pending",
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		left = u.UsdtWithdrawable.String()
		created = &biz.Withdrawal{
			ID: po.ID, UserID: po.UserID, Address: u.Address,
			ToAddress: po.ToAddress, Amount: po.Amount.String(),
			Fee: po.Fee.String(), NetAmount: po.PayAmount.String(),
			Status: po.Status, TxHash: po.TxHash, Asset: po.Asset,
			CreatedAt: po.CreatedTime,
		}
		return nil
	})
	return created, left, err
}

func exchangeRecordToBiz(po *ExchangeRecordPO, userAddr string) *biz.ExchangeRecord {
	return &biz.ExchangeRecord{
		ID: po.ID, UserID: po.UserID, UserAddress: userAddr,
		FromAsset: po.FromAsset, FromAmount: po.FromAmount.String(),
		ToAsset: po.ToAsset, ToAmount: po.ToAmount.String(),
		FeeAmount:     po.FeeAmount.String(),
		FeeRate:       po.FeeRate.String(),
		ExchangePrice: po.ExchangePrice.String(),
		Status:        po.Status, Remark: po.Remark, CreatedTime: po.CreatedTime,
	}
}

func (r *walletRepo) CreateRewardLog(ctx context.Context, log *biz.RewardLog) error {
	amount, _ := decimal.NewFromString(log.Amount)
	po := &RewardLogPO{
		UserID:      log.UserID,
		FromUserID:  log.FromUserID,
		OrderID:     log.OrderID,
		BatchID:     log.BatchID,
		Type:        log.Type,
		Asset:       log.Asset,
		Amount:      amount,
		ExitApplied: decimal.Zero,
	}
	if log.BaseAmount != "" {
		b, _ := decimal.NewFromString(log.BaseAmount)
		po.BaseAmount = &b
	}
	if log.Rate != "" {
		rt, _ := decimal.NewFromString(log.Rate)
		po.Rate = &rt
	}
	if log.ExitApplied != "" {
		ea, _ := decimal.NewFromString(log.ExitApplied)
		po.ExitApplied = ea
	}
	if log.SettlementDate != "" {
		d := log.SettlementDate
		po.SettlementDate = &d
	}
	if log.Meta != "" {
		m := log.Meta
		po.Meta = &m
	}
	return r.data.db.WithContext(ctx).Create(po).Error
}

func (r *walletRepo) ListRewardLogsByUser(ctx context.Context, userID int64) ([]*biz.RewardLog, error) {
	var list []RewardLogPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.RewardLog, 0, len(list))
	for i := range list {
		out = append(out, rewardLogToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) GetMgmtRewardSummary(ctx context.Context, userID int64) (*biz.MgmtRewardSummary, error) {
	type row struct {
		Total    decimal.Decimal
		Released decimal.Decimal
	}
	var result row
	if err := r.data.db.WithContext(ctx).Model(&MgmtRewardPO{}).
		Select("COALESCE(SUM(total_amount),0) AS total, COALESCE(SUM(released_amount),0) AS released").
		Where("user_id = ?", userID).Scan(&result).Error; err != nil {
		return nil, err
	}
	pending := result.Total.Sub(result.Released)
	if pending.IsNegative() {
		pending = decimal.Zero
	}
	return &biz.MgmtRewardSummary{
		Released: result.Released.Round(8).String(),
		Pending:  pending.Round(8).String(),
		Total:    result.Total.Round(8).String(),
	}, nil
}

func (r *walletRepo) GetDirectRewardTotal(ctx context.Context, userID int64) (string, error) {
	var total decimal.Decimal
	if err := r.data.db.WithContext(ctx).Model(&RewardLogPO{}).
		Select("COALESCE(SUM(amount),0)").
		Where("user_id = ? AND type IN ?", userID, []string{biz.RewardTypeDynamicUsdt, biz.RewardTypeDirectPoolRelease}).
		Scan(&total).Error; err != nil {
		return "0", err
	}
	return total.Round(8).String(), nil
}

func (r *walletRepo) ListMgmtRewardsByUser(ctx context.Context, userID int64) ([]*biz.MgmtReward, error) {
	var list []MgmtRewardPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.MgmtReward, 0, len(list))
	for _, item := range list {
		pending := item.TotalAmount.Sub(item.ReleasedAmount)
		if pending.IsNegative() {
			pending = decimal.Zero
		}
		out = append(out, &biz.MgmtReward{
			ID: item.ID, UserID: item.UserID, FromUserID: item.FromUserID,
			SourceOrderID: item.SourceOrderID, BaseAmount: item.BaseAmount.String(),
			Rate: item.Rate.String(), TotalAmount: item.TotalAmount.String(),
			ReleasedAmount: item.ReleasedAmount.String(), PendingAmount: pending.String(),
			CreatedTime: item.CreatedTime,
		})
	}
	return out, nil
}

func (r *walletRepo) GetAixPrice(ctx context.Context, date string) (string, error) {
	var po AixPricePO
	err := r.data.db.WithContext(ctx).Where("effective_date = ?", date).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return po.Price.StringFixed(biz.AixPriceDecimals), nil
}

func (r *walletRepo) GetLatestAixPriceBefore(ctx context.Context, date string) (string, error) {
	var po AixPricePO
	err := r.data.db.WithContext(ctx).
		Where("effective_date < ?", date).
		Order("effective_date DESC").
		First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return po.Price.StringFixed(biz.AixPriceDecimals), nil
}

func (r *walletRepo) UpsertAixPrice(ctx context.Context, date, price, remark string) error {
	p, err := decimal.NewFromString(price)
	if err != nil {
		return err
	}
	p = p.Round(biz.AixPriceDecimals)
	var po AixPricePO
	err = r.data.db.WithContext(ctx).Where("effective_date = ?", date).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return r.data.db.WithContext(ctx).Create(&AixPricePO{Price: p, EffectiveDate: date, Remark: remark}).Error
	}
	if err != nil {
		return err
	}
	return r.data.db.WithContext(ctx).Model(&po).Updates(map[string]interface{}{"price": p, "remark": remark}).Error
}

// GetCurrentWinPrice 读取唯一一条 WIN 现价。
func (r *walletRepo) GetCurrentWinPrice(ctx context.Context) (string, error) {
	var po WinPricePO
	err := r.data.db.WithContext(ctx).Where("id = ?", WinPriceRowID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return po.Price.String(), nil
}

// UpsertCurrentWinPrice 覆盖更新唯一一条 WIN 现价（不新增历史行）。
func (r *walletRepo) UpsertCurrentWinPrice(ctx context.Context, price, source string) error {
	p, err := decimal.NewFromString(price)
	if err != nil {
		return err
	}
	if !p.IsPositive() {
		return fmt.Errorf("invalid win price")
	}
	if source == "" {
		source = "oracle"
	}
	var po WinPricePO
	err = r.data.db.WithContext(ctx).Where("id = ?", WinPriceRowID).First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return r.data.db.WithContext(ctx).Create(&WinPricePO{
			ID:     WinPriceRowID,
			Price:  p,
			Source: source,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.data.db.WithContext(ctx).Model(&po).Updates(map[string]interface{}{
		"price":  p,
		"source": source,
	}).Error
}

func (r *walletRepo) rechargeToBiz(po *RechargePO) *biz.Recharge {
	asset := po.Asset
	if asset == "" {
		asset = biz.TokenUSDT
	}
	rec := &biz.Recharge{
		ID:          po.ID,
		UserID:      po.UserID,
		Address:     po.FromAddress,
		Asset:       asset,
		Amount:      po.Amount.String(),
		Message:     po.Message,
		TxHash:      po.TxHash,
		FromAddress: po.FromAddress,
		ToAddress:   po.ToAddress,
		Status:      po.Status,
		CreatedAt:   po.CreatedTime,
		CreatedTime: po.CreatedTime,
	}
	if po.ExpireAt != nil {
		rec.ExpireAt = *po.ExpireAt
	}
	if po.ConfirmedTime != nil {
		rec.ConfirmedAt = po.ConfirmedTime
		rec.ConfirmedTime = po.ConfirmedTime
	}
	return rec
}

func (r *walletRepo) orderToBiz(po *OrderPO) *biz.Order {
	o := &biz.Order{
		ID:           po.ID,
		UserID:       po.UserID,
		Principal:    po.Principal.String(),
		ExitCap:      po.ExitCap.String(),
		EarnedTotal:  po.EarnedTotal.String(),
		DirectBase:   po.DirectBase.String(),
		FromRecharge: po.FromRecharge.String(),
		FromReward:   po.FromReward.String(),
		FromWin:      po.FromWin.String(),
		FromWinA:     po.FromWinA.String(),
		Points:       po.Points.String(),
		PointsSource: po.PointsSource,
		FundSource:   po.FundSource,
		Status:       po.Status,
		ExitedTime:   po.ExitedTime,
		CreatedTime:  po.CreatedTime,
		UpdatedTime:  po.UpdatedTime,
	}
	o.SyncCompatFields()
	return o
}

func (r *walletRepo) transferToBiz(po *TransferPO) *biz.Transfer {
	return &biz.Transfer{
		ID:                po.ID,
		FromUserID:        po.FromUserID,
		ToUserID:          po.ToUserID,
		Asset:             po.Asset,
		Amount:            po.Amount.String(),
		PayFrom:           po.PayFrom,
		FromRechargeDebit: po.FromRechargeDebit.String(),
		FromRewardDebit:   po.FromRewardDebit.String(),
		ToCreditReward:    po.ToCreditReward.String(),
		ToCreditAix:       po.ToCreditAix.String(),
		Remark:            po.Remark,
		CreatedTime:       po.CreatedTime,
	}
}

func rewardLogToBiz(po *RewardLogPO) *biz.RewardLog {
	log := &biz.RewardLog{
		ID:          po.ID,
		UserID:      po.UserID,
		FromUserID:  po.FromUserID,
		OrderID:     po.OrderID,
		BatchID:     po.BatchID,
		Type:        po.Type,
		Asset:       po.Asset,
		Amount:      po.Amount.String(),
		ExitApplied: po.ExitApplied.String(),
		CreatedTime: po.CreatedTime,
	}
	if po.BaseAmount != nil {
		log.BaseAmount = po.BaseAmount.String()
	}
	if po.Rate != nil {
		log.Rate = po.Rate.String()
	}
	if po.SettlementDate != nil {
		log.SettlementDate = *po.SettlementDate
	}
	if po.Meta != nil {
		log.Meta = *po.Meta
	}
	return log
}

// --- legacy stubs ---

func (r *walletRepo) CreateWithdrawal(ctx context.Context, userID int64, amount, fee, netAmount, toAddress string) (*biz.Withdrawal, string, error) {
	return nil, "", fmt.Errorf("USDT withdraw not supported; use AIX withdraw")
}
func (r *walletRepo) ListWithdrawalsByUser(ctx context.Context, userID int64) ([]*biz.Withdrawal, error) {
	var list []WithdrawalPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.Withdrawal, 0, len(list))
	for _, po := range list {
		out = append(out, &biz.Withdrawal{
			ID: po.ID, UserID: po.UserID, ToAddress: po.ToAddress,
			Amount: po.Amount.String(), Fee: po.Fee.String(), NetAmount: po.PayAmount.String(),
			Status: po.Status, TxHash: po.TxHash, Asset: po.Asset, Remark: po.Remark,
			CreatedAt: po.CreatedTime, UpdatedAt: po.UpdatedTime,
		})
	}
	return out, nil
}
func (r *walletRepo) SumWithdrawnByUser(ctx context.Context, userID int64) (string, error) {
	return "0", nil
}
func (r *walletRepo) ListAllWithdrawals(ctx context.Context) ([]*biz.Withdrawal, error) {
	var list []WithdrawalPO
	if err := r.data.db.WithContext(ctx).
		Where("asset IN ?", []string{biz.TokenWIN, biz.TokenSDT, biz.TokenUSDT}).
		Order("id desc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []*biz.Withdrawal{}, nil
	}
	userIDs := make([]int64, 0, len(list))
	seen := map[int64]bool{}
	for _, po := range list {
		if !seen[po.UserID] {
			seen[po.UserID] = true
			userIDs = append(userIDs, po.UserID)
		}
	}
	var users []UserPO
	if err := r.data.db.WithContext(ctx).Select("id, address").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	addrMap := make(map[int64]string, len(users))
	for _, u := range users {
		addrMap[u.ID] = u.Address
	}
	out := make([]*biz.Withdrawal, 0, len(list))
	for _, po := range list {
		out = append(out, &biz.Withdrawal{
			ID: po.ID, UserID: po.UserID, Address: addrMap[po.UserID],
			ToAddress: po.ToAddress, Amount: po.Amount.String(),
			Fee: po.Fee.String(), NetAmount: po.PayAmount.String(),
			Status: po.Status, TxHash: po.TxHash, Asset: po.Asset,
			CreatedAt: po.CreatedTime,
		})
	}
	return out, nil
}

func withdrawalPOToBiz(po *WithdrawalPO, userAddr string) *biz.Withdrawal {
	return &biz.Withdrawal{
		ID: po.ID, UserID: po.UserID, Address: userAddr,
		ToAddress: po.ToAddress, Amount: po.Amount.String(),
		Fee: po.Fee.String(), NetAmount: po.PayAmount.String(),
		Status: po.Status, TxHash: po.TxHash, Asset: po.Asset,
		PayoutNonce: po.PayoutNonce,
		Remark:      po.Remark,
		CreatedAt: po.CreatedTime,
		UpdatedAt: po.UpdatedTime,
	}
}

// ClaimNextPendingWinWithdrawal 原子领取一条待打款提现（pending → doing，SDT / USDT）。
func (r *walletRepo) ClaimNextPendingWinWithdrawal(ctx context.Context) (*biz.Withdrawal, error) {
	var claimed *biz.Withdrawal
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po WithdrawalPO
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("asset IN ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '')",
				[]string{biz.TokenSDT, biz.TokenUSDT}, biz.WithdrawStatusPending).
			Order("id asc").
			First(&po).Error
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&po).Updates(map[string]interface{}{
			"status": biz.WithdrawStatusDoing,
			"remark": "chain payout in progress",
		}).Error; err != nil {
			return err
		}
		var u UserPO
		if err := tx.Select("address").First(&u, po.UserID).Error; err != nil {
			return err
		}
		po.Status = biz.WithdrawStatusDoing
		claimed = withdrawalPOToBiz(&po, u.Address)
		return nil
	})
	return claimed, err
}

func (r *walletRepo) SetWithdrawalTxHash(ctx context.Context, id int64, txHash string) error {
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return fmt.Errorf("tx hash is empty")
	}
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPO{}).
		Where("id = ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '')", id, biz.WithdrawStatusDoing).
		Update("tx_hash", txHash)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("withdrawal %d not in doing state or tx_hash already set", id)
	}
	return nil
}

func (r *walletRepo) CompleteWithdrawalPayout(ctx context.Context, id int64, txHash string) error {
	txHash = strings.TrimSpace(txHash)
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var confirmedCount int64
		if err := tx.Model(&WithdrawalPayoutPO{}).
			Where("withdraw_id = ? AND status = ?", id, biz.PayoutStatusConfirmed).
			Count(&confirmedCount).Error; err != nil {
			return err
		}
		if confirmedCount == 0 {
			res := tx.Model(&WithdrawalPayoutPO{}).
				Where("withdraw_id = ? AND tx_hash = ?", id, txHash).
				Update("status", biz.PayoutStatusConfirmed)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				var po WithdrawalPO
				if err := tx.Select("pay_amount", "to_address").First(&po, id).Error; err != nil {
					return fmt.Errorf("payout attempt not found for tx %s", txHash)
				}
				if err := tx.Create(&WithdrawalPayoutPO{
					WithdrawID: id,
					TxHash:     txHash,
					ToAddress:  strings.ToLower(po.ToAddress),
					Amount:     po.PayAmount,
					Status:     biz.PayoutStatusConfirmed,
				}).Error; err != nil {
					return err
				}
			}
		}

		res := tx.Model(&WithdrawalPO{}).
			Where("id = ? AND status != ?", id, biz.WithdrawStatusCompleted).
			Updates(map[string]interface{}{
				"status":  biz.WithdrawStatusCompleted,
				"tx_hash": txHash,
				"remark":  "chain payout completed",
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("withdrawal %d already completed or not found", id)
		}
		return nil
	})
}

func (r *walletRepo) ReleaseWithdrawalPayout(ctx context.Context, id int64, remark string) error {
	if len(remark) > 255 {
		remark = remark[:255]
	}
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPO{}).
		Where("id = ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '')", id, biz.WithdrawStatusDoing).
		Updates(map[string]interface{}{
			"status": biz.WithdrawStatusPending,
			"remark": remark,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return nil
}

func (r *walletRepo) ResetWithdrawalForRetry(ctx context.Context, id int64, remark string) error {
	return r.safeResetWithdrawalForRetry(ctx, id, remark)
}

func (r *walletRepo) ListDoingWithdrawalsWithTxHash(ctx context.Context) ([]*biz.Withdrawal, error) {
	var list []WithdrawalPO
	if err := r.data.db.WithContext(ctx).
		Where("asset IN ? AND status = ? AND tx_hash IS NOT NULL AND tx_hash != ''",
			[]string{biz.TokenWIN, biz.TokenSDT, biz.TokenUSDT}, biz.WithdrawStatusDoing).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return r.withdrawalsWithUserAddress(ctx, list)
}

func (r *walletRepo) ReleaseStaleDoingWithdrawals(ctx context.Context, staleBefore time.Time) (int64, error) {
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPO{}).
		Where("asset IN ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '') AND updated_time < ?",
			[]string{biz.TokenWIN, biz.TokenSDT, biz.TokenUSDT}, biz.WithdrawStatusDoing, staleBefore).
		Updates(map[string]interface{}{
			"status": biz.WithdrawStatusPending,
			"remark": "payout timed out; retry pending",
		})
	return res.RowsAffected, res.Error
}

func (r *walletRepo) withdrawalsWithUserAddress(ctx context.Context, list []WithdrawalPO) ([]*biz.Withdrawal, error) {
	if len(list) == 0 {
		return []*biz.Withdrawal{}, nil
	}
	userIDs := make([]int64, 0, len(list))
	seen := map[int64]bool{}
	for _, po := range list {
		if !seen[po.UserID] {
			seen[po.UserID] = true
			userIDs = append(userIDs, po.UserID)
		}
	}
	var users []UserPO
	if err := r.data.db.WithContext(ctx).Select("id, address").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	addrMap := make(map[int64]string, len(users))
	for _, u := range users {
		addrMap[u.ID] = u.Address
	}
	out := make([]*biz.Withdrawal, 0, len(list))
	for i := range list {
		out = append(out, withdrawalPOToBiz(&list[i], addrMap[list[i].UserID]))
	}
	return out, nil
}

func (r *walletRepo) ApproveWithdrawalReview(ctx context.Context, id int64) error {
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPO{}).
		Where("id = ? AND status = ?", id, biz.WithdrawStatusReview).
		Updates(map[string]interface{}{
			"status": biz.WithdrawStatusPending,
			"remark": "admin approved; awaiting chain payout",
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("withdrawal %d not in review status", id)
	}
	return nil
}

func (r *walletRepo) RejectWithdrawalReview(ctx context.Context, id int64, remark string) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po WithdrawalPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error; err != nil {
			return err
		}
		if po.Status != biz.WithdrawStatusReview {
			return fmt.Errorf("withdrawal %d not in review status", id)
		}
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, po.UserID).Error; err != nil {
			return err
		}
		switch strings.ToUpper(strings.TrimSpace(po.Asset)) {
		case biz.TokenWIN:
			u.WinBalance = u.WinBalance.Add(po.Amount)
			if err := tx.Model(&u).Update("win_balance", u.WinBalance).Error; err != nil {
				return err
			}
		case biz.TokenSDT:
			u.Points = u.Points.Add(po.Amount)
			if err := tx.Model(&u).Update("points", u.Points).Error; err != nil {
				return err
			}
		case biz.TokenUSDT:
			u.UsdtWithdrawable = u.UsdtWithdrawable.Add(po.Amount)
			if err := tx.Model(&u).Update("usdt_withdrawable", u.UsdtWithdrawable).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported withdraw asset %s", po.Asset)
		}
		if strings.TrimSpace(remark) == "" {
			remark = "admin rejected"
		}
		return tx.Model(&po).Updates(map[string]interface{}{
			"status": biz.WithdrawStatusRejected,
			"remark": remark,
		}).Error
	})
}

func (r *walletRepo) ApproveWithdrawal(ctx context.Context, id int64) error {
	return r.ApproveWithdrawalReview(ctx, id)
}
func (r *walletRepo) AdminUpdateOrder(ctx context.Context, update *biz.AdminOrderUpdate) (*biz.Order, error) {
	updates := map[string]interface{}{}
	if update.TotalAmount != "" {
		v, err := decimal.NewFromString(update.TotalAmount)
		if err != nil {
			return nil, err
		}
		updates["principal"] = v
	}
	if update.ExitTarget != "" {
		v, err := decimal.NewFromString(update.ExitTarget)
		if err != nil {
			return nil, err
		}
		updates["exit_cap"] = v
	}
	if update.ReleasedAmount != "" {
		v, err := decimal.NewFromString(update.ReleasedAmount)
		if err != nil {
			return nil, err
		}
		updates["earned_total"] = v
	}
	if update.Status != "" {
		st := update.Status
		if st == "completed" {
			st = biz.OrderStatusExited
		}
		updates["status"] = st
	}
	if len(updates) == 0 {
		return r.FindOrder(ctx, update.OrderID)
	}
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order OrderPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, update.OrderID).Error; err != nil {
			return err
		}
		if err := tx.Model(&order).Updates(updates).Error; err != nil {
			return err
		}
		return refreshAncestorPerformance(tx, order.UserID)
	})
	if err != nil {
		return nil, err
	}
	return r.FindOrder(ctx, update.OrderID)
}

// payUSDTRechargeRoleRewards 下级 USDT 充值时，按社区补贴级差向上发放（5%/10%/15%）。
// 仅 USDT 充值触发；WIN / WIN-A 充值不发放。
// 充值者自身档位作为起点：若下级已设 15%，则其上方所有人拿不到该笔补贴。
// 平级不发：上级档位 ≤ 下级已占用档位时不发；更高档位发差额。
func payUSDTRechargeRoleRewards(tx *gorm.DB, fromUserID int64, amount decimal.Decimal) error {
	if !amount.GreaterThan(decimal.Zero) {
		return nil
	}
	var recharger UserPO
	if err := tx.Select("id", "inviter_id", "is_community_subsidy", "community_subsidy_rate").
		First(&recharger, fromUserID).Error; err != nil {
		return err
	}
	if recharger.InviterID == nil {
		return nil
	}

	rechargerPct := biz.EffectiveSubsidyRatePercent(recharger.IsCommunitySubsidy, recharger.CommunitySubsidyRate)
	highestLowerPct := rechargerPct

	currentID := *recharger.InviterID
	seen := map[int64]bool{fromUserID: true}
	for currentID > 0 {
		if seen[currentID] {
			return fmt.Errorf("invite relationship contains a cycle at user %d", currentID)
		}
		seen[currentID] = true

		var ancestor UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "inviter_id", "is_community_subsidy", "community_subsidy_rate",
				"usdt_withdrawable", "community_subsidy_total").
			First(&ancestor, currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return err
		}

		ancestorPct := biz.EffectiveSubsidyRatePercent(ancestor.IsCommunitySubsidy, ancestor.CommunitySubsidyRate)
		gap := biz.SubsidyGapRate(ancestorPct, highestLowerPct)
		if gap.IsPositive() {
			reward := amount.Mul(gap).Round(8)
			if reward.IsPositive() {
				ancestor.UsdtWithdrawable = ancestor.UsdtWithdrawable.Add(reward)
				ancestor.CommunitySubsidyTotal = ancestor.CommunitySubsidyTotal.Add(reward)
				if err := tx.Model(&ancestor).Updates(map[string]interface{}{
					"usdt_withdrawable":       ancestor.UsdtWithdrawable,
					"community_subsidy_total": ancestor.CommunitySubsidyTotal,
				}).Error; err != nil {
					return err
				}
				base := amount
				rate := gap
				fromID := fromUserID
				if err := tx.Create(&RewardLogPO{
					UserID:     ancestor.ID,
					FromUserID: &fromID,
					Type:       biz.RewardTypeCommunitySubsidy,
					Asset:      biz.TokenUSDT,
					Amount:     reward,
					BaseAmount: &base,
					Rate:       &rate,
				}).Error; err != nil {
					return err
				}
			}
		}
		if ancestorPct > highestLowerPct {
			highestLowerPct = ancestorPct
		}
		if ancestor.InviterID == nil {
			break
		}
		currentID = *ancestor.InviterID
	}
	return nil
}
