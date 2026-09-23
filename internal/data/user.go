package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/pkg/eth"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userRepo struct {
	data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{data: data}
}

func (r *userRepo) FindByAddress(ctx context.Context, address string) (*biz.User, error) {
	var po UserPO
	addr := strings.TrimSpace(address)
	err := r.data.db.WithContext(ctx).
		Where("LOWER(address) = ?", strings.ToLower(addr)).
		First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBiz(ctx, &po), nil
}

func (r *userRepo) FindByID(ctx context.Context, id int64) (*biz.User, error) {
	var po UserPO
	err := r.data.db.WithContext(ctx).First(&po, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toBiz(ctx, &po), nil
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	invite := user.InviteCode
	if invite == "" {
		invite = user.Address
	}
	po := &UserPO{
		Address:    user.Address,
		InviterID:  user.InviterID,
		InviteCode: invite,
		Role:       biz.RoleUser,
		Status:     1,
	}
	var bonus decimal.Decimal
	if parsed, err := decimal.NewFromString(user.UsdtRecharge); err == nil && parsed.IsPositive() {
		bonus = parsed
		po.UsdtRecharge = parsed
	}
	// 注册赠送与上级的零号账户/社区补贴奖励必须同成同败
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		if !bonus.IsPositive() {
			return nil
		}
		return payUSDTRechargeRoleRewards(tx, po.ID, bonus)
	})
	if err != nil {
		return nil, err
	}
	return r.toBiz(ctx, po), nil
}

func (r *userRepo) CountInvitees(ctx context.Context, userID int64) (int32, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&UserPO{}).Where("inviter_id = ?", userID).Count(&count).Error
	return int32(count), err
}

func (r *userRepo) CountSubscribeOrders(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&OrderPO{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *userRepo) ListDownlineInvitees(ctx context.Context, userID int64, maxDepth int) ([]*biz.DownlineInvitee, error) {
	if maxDepth <= 0 {
		maxDepth = 1000
	}
	var result []*biz.DownlineInvitee
	currentLevelIDs := []int64{userID}
	for generation := 1; generation <= maxDepth; generation++ {
		if len(currentLevelIDs) == 0 {
			break
		}
		var pos []UserPO
		if err := r.data.db.WithContext(ctx).
			Where("inviter_id IN ?", currentLevelIDs).
			Order("created_time ASC").
			Find(&pos).Error; err != nil {
			return nil, err
		}
		if len(pos) == 0 {
			break
		}
		next := make([]int64, 0, len(pos))
		for _, po := range pos {
			result = append(result, &biz.DownlineInvitee{
				Address:    po.Address,
				Generation: int32(generation),
				CreatedAt:  po.CreatedTime,
			})
			next = append(next, po.ID)
		}
		currentLevelIDs = next
	}
	return result, nil
}

func (r *userRepo) ListUsersUnder(ctx context.Context, rootID int64) ([]*biz.User, error) {
	pos, err := r.listUsersUnder(ctx, rootID)
	if err != nil {
		return nil, err
	}
	// 逐行调用 toBiz 会为每个成员单独查一次上级地址，整棵树可达数千次往返。
	inviterAddrs, err := r.inviterAddresses(ctx, pos)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.User, 0, len(pos))
	for i := range pos {
		inviterAddress := ""
		if pos[i].InviterID != nil {
			inviterAddress = inviterAddrs[*pos[i].InviterID]
		}
		out = append(out, r.toBizWithInviter(&pos[i], inviterAddress))
	}
	return out, nil
}

// inviterAddresses 一次性取回给定成员集合所需的全部上级地址。
func (r *userRepo) inviterAddresses(ctx context.Context, pos []UserPO) (map[int64]string, error) {
	ids := make([]int64, 0, len(pos))
	seen := make(map[int64]struct{}, len(pos))
	for i := range pos {
		if pos[i].InviterID == nil {
			continue
		}
		if _, ok := seen[*pos[i].InviterID]; ok {
			continue
		}
		seen[*pos[i].InviterID] = struct{}{}
		ids = append(ids, *pos[i].InviterID)
	}
	out := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ID      int64
		Address string
	}
	var rows []row
	if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
		Select("id", "address").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.ID] = item.Address
	}
	return out, nil
}

func (r *userRepo) CountUsersUnder(ctx context.Context, rootID int64) (int32, error) {
	ids, err := r.listUserIDsUnder(ctx, rootID)
	if err != nil {
		return 0, err
	}
	return int32(len(ids)), nil
}

func (r *userRepo) ListUserIDsUnder(ctx context.Context, rootID int64) ([]int64, error) {
	return r.listUserIDsUnder(ctx, rootID)
}

// listUserIDsUnder 收集 rootID 之下的全部后代（不含本人）。
// 优先用 MySQL 递归 CTE（单次往返）；失败时退回按层 BFS。
// 禁止再全表拉 id→inviter：用户量大时会拖垮团队页 / profile。
func (r *userRepo) listUserIDsUnder(ctx context.Context, rootID int64) ([]int64, error) {
	if rootID <= 0 {
		return nil, nil
	}
	var ids []int64
	err := r.data.db.WithContext(ctx).Raw(`
		WITH RECURSIVE under AS (
			SELECT id FROM users WHERE inviter_id = ?
			UNION ALL
			SELECT u.id FROM users u INNER JOIN under ON u.inviter_id = under.id
		)
		SELECT id FROM under
	`, rootID).Scan(&ids).Error
	if err == nil {
		return ids, nil
	}
	return r.listUserIDsUnderBFS(ctx, rootID)
}

func (r *userRepo) listUserIDsUnderBFS(ctx context.Context, rootID int64) ([]int64, error) {
	var all []int64
	frontier := []int64{rootID}
	seen := map[int64]struct{}{rootID: {}}
	for len(frontier) > 0 {
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
			all = append(all, id)
			next = append(next, id)
		}
		frontier = next
	}
	return all, nil
}

func (r *userRepo) listUsersUnder(ctx context.Context, rootID int64) ([]UserPO, error) {
	ids, err := r.listUserIDsUnder(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var all []UserPO
	if err := r.data.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("id asc").
		Find(&all).Error; err != nil {
		return nil, err
	}
	return all, nil
}

func (r *userRepo) ListAllUsers(ctx context.Context) ([]*biz.User, error) {
	var list []UserPO
	if err := r.data.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	inviterIDs := make([]int64, 0, len(list))
	for i := range list {
		if list[i].InviterID != nil {
			inviterIDs = append(inviterIDs, *list[i].InviterID)
		}
	}
	inviterAddr := r.batchUserAddresses(ctx, inviterIDs)
	out := make([]*biz.User, 0, len(list))
	for i := range list {
		addr := ""
		if list[i].InviterID != nil {
			addr = inviterAddr[*list[i].InviterID]
		}
		out = append(out, r.toBizWithInviter(&list[i], addr))
	}
	return out, nil
}

func (r *userRepo) ListUsersPaged(ctx context.Context, addressFilter string, offset, limit int) ([]*biz.User, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	addressFilter = strings.TrimSpace(addressFilter)
	build := func() *gorm.DB {
		q := r.data.db.WithContext(ctx).Model(&UserPO{})
		if addressFilter != "" {
			q = q.Where("LOWER(address) LIKE ?", "%"+strings.ToLower(addressFilter)+"%")
		}
		return q
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []UserPO
	if err := build().Order("id desc").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	inviterIDs := make([]int64, 0, len(list))
	for i := range list {
		if list[i].InviterID != nil {
			inviterIDs = append(inviterIDs, *list[i].InviterID)
		}
	}
	inviterAddr := r.batchUserAddresses(ctx, inviterIDs)
	out := make([]*biz.User, 0, len(list))
	for i := range list {
		addr := ""
		if list[i].InviterID != nil {
			addr = inviterAddr[*list[i].InviterID]
		}
		out = append(out, r.toBizWithInviter(&list[i], addr))
	}
	return out, total, nil
}

func (r *userRepo) CountDirectInviteesByUserIDs(ctx context.Context, userIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(userIDs))
	for _, id := range userIDs {
		out[id] = 0
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		InviterID int64
		Cnt       int
	}
	var rows []row
	if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
		Select("inviter_id AS inviter_id, COUNT(*) AS cnt").
		Where("inviter_id IN ?", userIDs).
		Group("inviter_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.InviterID] = item.Cnt
	}
	return out, nil
}

func (r *userRepo) batchUserAddresses(ctx context.Context, ids []int64) map[int64]string {
	out := make(map[int64]string)
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ID      int64
		Address string
	}
	var rows []row
	if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
		Select("id, address").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out
	}
	for _, item := range rows {
		out[item.ID] = item.Address
	}
	return out
}

func (r *userRepo) ListDirectInvitees(ctx context.Context, userID int64) ([]*biz.User, error) {
	var list []UserPO
	if err := r.data.db.WithContext(ctx).Where("inviter_id = ?", userID).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.User, 0, len(list))
	for i := range list {
		out = append(out, r.toBiz(ctx, &list[i]))
	}
	return out, nil
}

func (r *userRepo) SumActivePrincipalUnder(ctx context.Context, rootID int64) (string, error) {
	ids, err := r.listUserIDsUnder(ctx, rootID)
	if err != nil {
		return "0", err
	}
	allIDs := make([]int64, 0, len(ids)+1)
	allIDs = append(allIDs, rootID)
	allIDs = append(allIDs, ids...)
	stakeMap, err := r.SumPrincipalByUserIDs(ctx, allIDs)
	if err != nil {
		return "0", err
	}
	total := decimal.Zero
	for _, id := range allIDs {
		v, _ := decimal.NewFromString(stakeMap[id])
		total = total.Add(v)
	}
	return total.String(), nil
}

func (r *userRepo) SumPrincipalByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(userIDs))
	for _, id := range userIDs {
		out[id] = "0"
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID int64
		Total  decimal.Decimal
	}
	var rows []row
	err := r.data.db.WithContext(ctx).Raw(`
		SELECT user_id, COALESCE(SUM(principal), 0) AS total
		FROM orders
		WHERE user_id IN ? AND status = ?
		GROUP BY user_id
	`, userIDs, biz.OrderStatusActive).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.UserID] = item.Total.String()
	}
	return out, nil
}

func (r *userRepo) SumCumulativePrincipalByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(userIDs))
	for _, id := range userIDs {
		out[id] = "0"
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID int64
		Total  decimal.Decimal
	}
	var rows []row
	err := r.data.db.WithContext(ctx).Raw(`
		SELECT user_id, COALESCE(SUM(principal), 0) AS total
		FROM orders
		WHERE user_id IN ? AND status IN ?
		GROUP BY user_id
	`, userIDs, []string{biz.OrderStatusActive, biz.OrderStatusExited}).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.UserID] = item.Total.String()
	}
	return out, nil
}

func (r *userRepo) SumExitAmountByUserIDs(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	return r.SumPrincipalByUserIDs(ctx, userIDs)
}

func (r *userRepo) UpdateMgmtStats(ctx context.Context, userID int64, level int32, smallArea, teamPerf string) error {
	sa, err := decimal.NewFromString(smallArea)
	if err != nil {
		return err
	}
	tp, err := decimal.NewFromString(teamPerf)
	if err != nil {
		return err
	}
	return r.data.db.WithContext(ctx).Model(&UserPO{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"mgmt_level":      level,
		"small_area_perf": sa,
		"team_perf":       tp,
	}).Error
}

func (r *userRepo) AdminUpdateUser(ctx context.Context, update *biz.AdminUserUpdate) error {
	updates := map[string]interface{}{}
	if update.UsdtRecharge != "" {
		v, err := decimal.NewFromString(update.UsdtRecharge)
		if err != nil {
			return err
		}
		updates["usdt_recharge"] = v
	}
	if update.UsdtReward != "" {
		v, err := decimal.NewFromString(update.UsdtReward)
		if err != nil {
			return err
		}
		updates["usdt_reward"] = v
	}
	if update.AixBalance != "" {
		v, err := decimal.NewFromString(update.AixBalance)
		if err != nil {
			return err
		}
		updates["aix_balance"] = v
	}
	if update.WinBalance != "" {
		v, err := decimal.NewFromString(update.WinBalance)
		if err != nil {
			return err
		}
		updates["win_balance"] = v
	}
	if update.WinRechargeBalance != "" {
		v, err := decimal.NewFromString(update.WinRechargeBalance)
		if err != nil {
			return err
		}
		updates["win_recharge_balance"] = v
	}
	if update.WinARechargeBalance != "" {
		v, err := decimal.NewFromString(update.WinARechargeBalance)
		if err != nil {
			return err
		}
		updates["win_a_recharge_balance"] = v
	}
	if update.PendingMgmtReward != "" {
		v, err := decimal.NewFromString(update.PendingMgmtReward)
		if err != nil {
			return err
		}
		updates["overflow_reward"] = v
		updates["pending_mgmt_reward"] = v
	}
	if update.OverflowReward != "" {
		v, err := decimal.NewFromString(update.OverflowReward)
		if err != nil {
			return err
		}
		updates["overflow_reward"] = v
		updates["pending_mgmt_reward"] = v
	}
	if update.StaticUsdtTotal != "" {
		v, err := decimal.NewFromString(update.StaticUsdtTotal)
		if err != nil {
			return err
		}
		updates["static_usdt_total"] = v
	}
	if update.Balance != "" {
		v, err := decimal.NewFromString(update.Balance)
		if err != nil {
			return err
		}
		updates["usdt_recharge"] = v
	}
	if update.ReleasedBalance != "" {
		v, err := decimal.NewFromString(update.ReleasedBalance)
		if err != nil {
			return err
		}
		updates["usdt_reward"] = v
	}
	if update.Role != "" {
		updates["role"] = update.Role
	}
	if update.SetCommunityLevel {
		// parse A0-A10 / W0-W10 / V0-V10 or bare 0-10
		level := int32(0)
		lv := strings.ToUpper(strings.TrimSpace(update.CommunityLevel))
		lv = strings.TrimPrefix(lv, "A")
		lv = strings.TrimPrefix(lv, "W")
		lv = strings.TrimPrefix(lv, "V")
		if n, err := strconv.Atoi(lv); err == nil && n >= 0 {
			if n > 10 {
				n = 10
			}
			level = int32(n)
		}
		updates["mgmt_level"] = level
		updates["mgmt_level_locked"] = true
	}
	if update.CommunityStake != "" {
		v, err := decimal.NewFromString(update.CommunityStake)
		if err != nil {
			return err
		}
		updates["small_area_perf"] = v
	}
	if update.TeamStake != "" {
		v, err := decimal.NewFromString(update.TeamStake)
		if err != nil {
			return err
		}
		updates["team_perf"] = v
	}
	if update.InviterID != nil {
		updates["inviter_id"] = update.InviterID
	}
	if update.SetIsZeroAccount {
		if update.IsZeroAccount {
			return fmt.Errorf("zero account deprecated: use community subsidy rate 5/10/15 instead")
		}
		var po UserPO
		if err := r.data.db.WithContext(ctx).Select(
			"id", "is_zero_account", "is_community_subsidy", "community_subsidy_rate",
			"zero_account_reward_total", "community_subsidy_total",
		).First(&po, update.UserID).Error; err != nil {
			return err
		}
		now := time.Now()
		updates["is_zero_account"] = false
		updates["zero_account_set_at"] = now
		if po.IsZeroAccount {
			newRate := biz.MergeZeroAccountIntoSubsidyRate(po.CommunitySubsidyRate, po.IsCommunitySubsidy)
			updates["is_community_subsidy"] = true
			updates["community_subsidy_rate"] = newRate
			updates["community_subsidy_set_at"] = now
			mergedTotal := po.CommunitySubsidyTotal.Add(po.ZeroAccountRewardTotal)
			updates["community_subsidy_total"] = mergedTotal
			updates["zero_account_reward_total"] = decimal.Zero
		}
	}
	if update.SetIsCommunitySubsidy {
		now := time.Now()
		updates["is_community_subsidy"] = update.IsCommunitySubsidy
		updates["community_subsidy_set_at"] = now
		if !update.IsCommunitySubsidy {
			updates["community_subsidy_rate"] = 0
		}
	}
	if update.SetCommunitySubsidyRate {
		now := time.Now()
		rate := update.CommunitySubsidyRate
		if rate < 0 {
			rate = 0
		}
		if rate > biz.SubsidyRateMax {
			rate = biz.SubsidyRateMax
		}
		if rate > 0 && rate != biz.SubsidyRateMin && rate != biz.SubsidyRateMid && rate != biz.SubsidyRateMax {
			return fmt.Errorf("invalid community subsidy rate %d", rate)
		}
		updates["community_subsidy_rate"] = rate
		updates["is_community_subsidy"] = rate > 0
		updates["community_subsidy_set_at"] = now
	}
	if update.SetIsFrozen {
		updates["is_frozen"] = update.IsFrozen
		if update.IsFrozen {
			now := time.Now()
			updates["frozen_at"] = now
		} else {
			updates["frozen_at"] = nil
		}
	}
	if update.SetExchangeEnabled {
		updates["exchange_enabled"] = update.ExchangeEnabled
	}
	if addr := strings.TrimSpace(update.Address); addr != "" {
		updates["address"] = addr
		updates["invite_code"] = addr // 注册时 invite_code=address，一并替换
	}
	if len(updates) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).Model(&UserPO{}).Where("id = ?", update.UserID).Updates(updates).Error
}

func (r *userRepo) SetFrozenForUsers(ctx context.Context, userIDs []int64, frozen bool) error {
	if len(userIDs) == 0 {
		return nil
	}
	updates := map[string]interface{}{"is_frozen": frozen}
	if frozen {
		updates["frozen_at"] = time.Now()
	} else {
		updates["frozen_at"] = nil
	}
	return r.data.db.WithContext(ctx).Model(&UserPO{}).Where("id IN ?", userIDs).Updates(updates).Error
}

func (r *userRepo) SetExchangeEnabledForUsers(ctx context.Context, userIDs []int64, enabled bool) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).Model(&UserPO{}).
		Where("id IN ?", userIDs).
		Update("exchange_enabled", enabled).Error
}

func (r *userRepo) SetRole(ctx context.Context, userID int64, role string) error {
	return r.data.db.WithContext(ctx).Model(&UserPO{}).Where("id = ?", userID).Update("role", role).Error
}

func (r *userRepo) UpdateUsername(ctx context.Context, userID int64, username string) error {
	return r.data.db.WithContext(ctx).Model(&UserPO{}).Where("id = ?", userID).Update("username", username).Error
}

// ResolveExchangeBindAddress 绑定或校验向交易所划转地址。
// 已绑定：返回原地址；若传入不同地址则报错。
// 未绑定：要求传入地址，校验全局未被占用后写入。
func (r *userRepo) ResolveExchangeBindAddress(ctx context.Context, userID int64, requestedAddress string) (string, error) {
	var bound string
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
			return err
		}
		existing := ""
		if u.ExchangeBindAddress != nil {
			existing = strings.TrimSpace(*u.ExchangeBindAddress)
		}
		requested := strings.TrimSpace(requestedAddress)
		if existing != "" {
			if requested != "" {
				reqNorm, err := eth.NormalizeAddress(requested)
				if err != nil {
					return fmt.Errorf("invalid exchange bind address")
				}
				existNorm, _ := eth.NormalizeAddress(existing)
				if !strings.EqualFold(reqNorm, existNorm) {
					return fmt.Errorf("exchange bind address already set")
				}
				bound = existNorm
				return nil
			}
			existNorm, err := eth.NormalizeAddress(existing)
			if err != nil {
				bound = strings.ToLower(existing)
				return nil
			}
			bound = existNorm
			return nil
		}
		if requested == "" {
			return fmt.Errorf("exchange bind address required")
		}
		norm, err := eth.NormalizeAddress(requested)
		if err != nil {
			return fmt.Errorf("invalid exchange bind address")
		}
		var other UserPO
		err = tx.Select("id").Where("exchange_bind_address = ? AND id <> ?", norm, userID).Limit(1).Take(&other).Error
		if err == nil {
			return fmt.Errorf("exchange bind address taken")
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := tx.Model(&u).Update("exchange_bind_address", norm).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return fmt.Errorf("exchange bind address taken")
			}
			return err
		}
		bound = norm
		return nil
	})
	return bound, err
}

func (r *userRepo) GetBalances(ctx context.Context, userID int64) (string, string, string, error) {
	var po UserPO
	if err := r.data.db.WithContext(ctx).Select("usdt_recharge", "usdt_reward", "aix_balance").First(&po, userID).Error; err != nil {
		return "", "", "", err
	}
	return po.UsdtRecharge.String(), po.UsdtReward.String(), po.AixBalance.String(), nil
}

func (r *userRepo) AddUsdtRecharge(ctx context.Context, userID int64, amount string) (string, error) {
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return "", err
	}
	var newBal decimal.Decimal
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, userID).Error; err != nil {
			return err
		}
		po.UsdtRecharge = po.UsdtRecharge.Add(amountDec)
		newBal = po.UsdtRecharge
		return tx.Model(&po).Update("usdt_recharge", po.UsdtRecharge).Error
	})
	return newBal.String(), err
}

func (r *userRepo) IsUplineOrDownline(ctx context.Context, a, b int64) (bool, error) {
	if a == b {
		return false, nil
	}
	parents, err := r.loadInviterParents(ctx)
	if err != nil {
		return false, err
	}
	return biz.IsLinealRelation(a, b, parents), nil
}

func (r *userRepo) IsUplineOf(ctx context.Context, uplineID, downlineID int64) (bool, error) {
	if uplineID == downlineID {
		return false, nil
	}
	parents, err := r.loadInviterParents(ctx)
	if err != nil {
		return false, err
	}
	return biz.IsAncestorOf(uplineID, downlineID, parents), nil
}

func (r *userRepo) loadInviterParents(ctx context.Context) (map[int64]int64, error) {
	var users []UserPO
	if err := r.data.db.WithContext(ctx).Select("id", "inviter_id").Find(&users).Error; err != nil {
		return nil, err
	}
	parents := make(map[int64]int64, len(users))
	for _, user := range users {
		if user.InviterID != nil {
			parents[user.ID] = *user.InviterID
		}
	}
	return parents, nil
}

func (r *userRepo) SetWithdrawReset(ctx context.Context, userID int64, reset bool) error {
	return nil
}
func (r *userRepo) ClearWithdrawReset(ctx context.Context, userID int64) error {
	return nil
}
func (r *userRepo) IsWithdrawReset(ctx context.Context, userID int64) (bool, error) {
	return false, nil
}
func (r *userRepo) UpdateCommunityStats(ctx context.Context, userID int64, level string, communityStake, teamStake string) error {
	lv := int32(0)
	if len(level) >= 2 {
		prefix := level[0]
		if prefix == 'A' || prefix == 'a' || prefix == 'W' || prefix == 'w' || prefix == 'V' || prefix == 'v' {
			n := 0
			for i := 1; i < len(level); i++ {
				if level[i] >= '0' && level[i] <= '9' {
					n = n*10 + int(level[i]-'0')
				}
			}
			lv = int32(n)
		}
	}
	return r.UpdateMgmtStats(ctx, userID, lv, communityStake, teamStake)
}
func (r *userRepo) GetBalance(ctx context.Context, userID int64) (string, error) {
	recharge, _, _, err := r.GetBalances(ctx, userID)
	return recharge, err
}
func (r *userRepo) GetReleasedBalance(ctx context.Context, userID int64) (string, error) {
	_, reward, _, err := r.GetBalances(ctx, userID)
	return reward, err
}
func (r *userRepo) AddBalance(ctx context.Context, userID int64, amount string) (string, error) {
	return r.AddUsdtRecharge(ctx, userID, amount)
}
func (r *userRepo) AddReleasedBalance(ctx context.Context, userID int64, amount string) (string, error) {
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return "", err
	}
	var newBal decimal.Decimal
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, userID).Error; err != nil {
			return err
		}
		po.UsdtReward = po.UsdtReward.Add(amountDec)
		newBal = po.UsdtReward
		return tx.Model(&po).Update("usdt_reward", po.UsdtReward).Error
	})
	return newBal.String(), err
}
func (r *userRepo) ClaimReleasedToAccount(ctx context.Context, userID int64, amount string) (string, string, error) {
	return "", "", fmt.Errorf("not supported in AIX")
}
func (r *userRepo) DeductBalance(ctx context.Context, userID int64, amount string) (string, error) {
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return "", err
	}
	var newBal decimal.Decimal
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, userID).Error; err != nil {
			return err
		}
		if po.UsdtRecharge.LessThan(amountDec) {
			return fmt.Errorf("insufficient balance")
		}
		po.UsdtRecharge = po.UsdtRecharge.Sub(amountDec)
		newBal = po.UsdtRecharge
		return tx.Model(&po).Update("usdt_recharge", po.UsdtRecharge).Error
	})
	return newBal.String(), err
}

func (r *userRepo) toBiz(ctx context.Context, po *UserPO) *biz.User {
	inviterAddress := ""
	if po.InviterID != nil {
		var addr string
		if err := r.data.db.WithContext(ctx).Model(&UserPO{}).
			Select("address").Where("id = ?", *po.InviterID).Scan(&addr).Error; err == nil {
			inviterAddress = addr
		}
	}
	return r.toBizWithInviter(po, inviterAddress)
}

func (r *userRepo) toBizWithInviter(po *UserPO, inviterAddress string) *biz.User {
	user := &biz.User{
		ID:                   po.ID,
		Address:              po.Address,
		InviteCode:           po.InviteCode,
		Username:             po.Username,
		UsdtRecharge:         po.UsdtRecharge.String(),
		UsdtReward:           po.UsdtReward.String(),
		AixBalance:           po.AixBalance.String(),
		WinBalance:          po.WinBalance.String(),
		WinRechargeBalance:  po.WinRechargeBalance.String(),
		WinARechargeBalance: po.WinARechargeBalance.String(),
		UsdtWithdrawable:    po.UsdtWithdrawable.String(),
		PendingMgmtReward:    po.OverflowReward.Add(po.OverflowDirect).String(), // 兼容：溢出奖励合计
		OverflowReward:       po.OverflowReward.String(),
		OverflowDirect:       po.OverflowDirect.String(),
		Points:               po.Points.String(),
		PointsAll:            po.PointsAll.String(),
		StaticUsdtTotal:      po.StaticUsdtTotal.String(),
		MgmtLevel:            po.MgmtLevel,
		CommunityLevelLocked: po.MgmtLevelLocked,
		LargeAreaPerf:        po.LargeAreaPerf.String(),
		SmallAreaPerf:        po.SmallAreaPerf.String(),
		TeamPerf:             po.TeamPerf.String(),
		IsZeroAccount:          po.IsZeroAccount,
		IsCommunitySubsidy:     po.IsCommunitySubsidy,
		CommunitySubsidyRate:   po.CommunitySubsidyRate,
		ZeroAccountSetAt:       po.ZeroAccountSetAt,
		CommunitySubsidySetAt:  po.CommunitySubsidySetAt,
		ZeroAccountRewardTotal: po.ZeroAccountRewardTotal.String(),
		CommunitySubsidyTotal:  po.CommunitySubsidyTotal.String(),
		Status:               po.Status,
		IsFrozen:             po.IsFrozen,
		FrozenAt:             po.FrozenAt,
		ExchangeEnabled:      po.ExchangeEnabled,
		InviterID:            po.InviterID,
		Role:                 po.Role,
		CreatedTime:          po.CreatedTime,
		UpdatedTime:          po.UpdatedTime,
		InviterAddress:       inviterAddress,
	}
	if po.ExchangeBindAddress != nil {
		user.ExchangeBindAddress = strings.TrimSpace(*po.ExchangeBindAddress)
	}
	user.SyncCompatFields()
	return user
}
