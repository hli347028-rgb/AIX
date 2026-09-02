package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/data"
	jwtpkg "backend/internal/pkg/token"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type rechargeListRow struct {
	ID          int64
	Address     string
	Asset       string
	Amount      decimal.Decimal
	TxHash      string
	Message     string
	CreatedTime time.Time
}

func parseLegacyTimeRange(q url.Values) (start, end *time.Time) {
	loc := jwtpkg.ChinaLocation()
	parseOne := func(raw string, endOfDay bool) *time.Time {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
				if endOfDay && layout == "2006-01-02" {
					t = t.Add(24*time.Hour - time.Nanosecond)
				}
				return &t
			}
		}
		return nil
	}
	return parseOne(q.Get("startTime"), false), parseOne(q.Get("endTime"), true)
}

func parseTeamQuery(q url.Values) bool {
	v := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("teamQuery"), q.Get("team_query"))))
	return v == "1" || v == "true" || v == "yes"
}

func (s *AdminLegacyService) teamUserIDsForQuery(ctx context.Context, q url.Values) ([]int64, bool, error) {
	if !parseTeamQuery(q) {
		return nil, false, nil
	}
	address := strings.TrimSpace(q.Get("address"))
	if address == "" {
		return []int64{-1}, true, nil
	}
	user, err := s.userRepo.FindByAddress(ctx, address)
	if err != nil {
		return nil, true, err
	}
	if user == nil {
		return []int64{-1}, true, nil
	}
	ids, err := s.userRepo.ListUserIDsUnder(ctx, user.ID)
	if err != nil {
		return nil, true, err
	}
	ids = append(ids, user.ID)
	return ids, true, nil
}

func (s *AdminLegacyService) buildTeamSummary(ctx context.Context, q url.Values) (map[string]interface{}, error) {
	ids, teamMode, err := s.teamUserIDsForQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	if !teamMode {
		return nil, nil
	}
	address := strings.TrimSpace(q.Get("address"))
	summary := map[string]interface{}{
		"memberCount": len(ids),
		"rootAddress": address,
	}
	user, err := s.userRepo.FindByAddress(ctx, address)
	if err != nil {
		return summary, err
	}
	if user != nil {
		summary["teamPerformance"] = user.TeamPerf
		summary["smallAreaPerformance"] = user.SmallAreaPerf
		summary["largeAreaPerformance"] = user.LargeAreaPerf
		summary["communitySubsidyTotal"] = user.CommunitySubsidyTotal
		summary["communitySubsidyRate"] = user.CommunitySubsidyRate
	}
	return summary, nil
}

func (s *AdminLegacyService) sumChainRecharge(ctx context.Context, asset string, since *time.Time) (decimal.Decimal, error) {
	// 仅统计用户真实链上充值：排除后台补录(admin-*)与交易所划转(partner:*)
	db := s.data.DB().WithContext(ctx).Table("recharges r").
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		Where("r.tx_hash NOT LIKE ?", "admin-%").
		Where("r.tx_hash NOT LIKE ?", "partner:%")
	switch strings.ToUpper(strings.TrimSpace(asset)) {
	case biz.TokenWIN:
		db = db.Where("UPPER(r.asset) = ?", biz.TokenWIN)
	case biz.TokenWINA:
		db = db.Where("UPPER(r.asset) = ?", biz.TokenWINA)
	default:
		db = db.Where("(UPPER(r.asset) = ? OR r.asset = '' OR r.asset IS NULL)", biz.TokenUSDT)
	}
	if since != nil {
		db = db.Where("r.created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(r.amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumWithdrawalAmount(ctx context.Context, asset string, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("withdrawals").
		Where("UPPER(asset) = ?", strings.ToUpper(strings.TrimSpace(asset))).
		Where("status NOT IN ?", []string{biz.WithdrawStatusRejected, biz.WithdrawStatusFailed})
	if since != nil {
		db = db.Where("created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumDynamicReward(ctx context.Context, since *time.Time) (decimal.Decimal, error) {
	types := []string{
		biz.RewardTypeDynamicUsdt,
		biz.RewardTypeDirectPoolRelease,
		biz.RewardTypeMgmt,
		biz.RewardTypeMgmtPoolRelease,
	}
	db := s.data.DB().WithContext(ctx).Table("reward_logs").Where("type IN ?", types)
	if since != nil {
		db = db.Where("created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumRewardByType(ctx context.Context, rewardType string, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("reward_logs").Where("type = ?", rewardType)
	if since != nil {
		db = db.Where("created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumStaticRelease(ctx context.Context, settlementDate string) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("reward_logs").
		Where("type = ?", biz.RewardTypeStaticAix)
	if settlementDate != "" {
		db = db.Where("settlement_date = ?", settlementDate)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(exit_applied),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

// sumStaticAixAmount 某结算日静态发放的 AIX 枚数（reward_logs.amount，不是 exit_applied 的 USDT）。
func (s *AdminLegacyService) sumStaticAixAmount(ctx context.Context, settlementDate string) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("reward_logs").
		Where("type = ?", biz.RewardTypeStaticAix)
	if settlementDate != "" {
		db = db.Where("settlement_date = ?", settlementDate)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumOrderPrincipal(ctx context.Context, fundSource string, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("orders").
		Where("status IN ?", []string{biz.OrderStatusActive, biz.OrderStatusExited})
	if fundSource != "" {
		db = db.Where("fund_source = ?", fundSource)
	}
	if since != nil {
		db = db.Where("created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(principal),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumUserAsset(ctx context.Context, column string) (decimal.Decimal, error) {
	var total decimal.Decimal
	if err := s.data.DB().WithContext(ctx).Model(&data.UserPO{}).
		Select("COALESCE(SUM("+column+"),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) sumAdminManualRecharge(ctx context.Context, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("recharges r").
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		Where("r.tx_hash LIKE ?", "admin-%")
	if since != nil {
		db = db.Where("r.created_time >= ?", *since)
	}
	var total decimal.Decimal
	if err := db.Select("COALESCE(SUM(r.amount),0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

func (s *AdminLegacyService) rechargeListDB(ctx context.Context, q url.Values) *gorm.DB {
	addressFilter := strings.TrimSpace(q.Get("address"))
	typeFilter := strings.ToLower(strings.TrimSpace(q.Get("type")))
	db := s.data.DB().WithContext(ctx).
		Table("recharges r").
		Select(`r.id, COALESCE(NULLIF(r.from_address,''), u.address) as address,
			r.asset, r.amount, r.tx_hash, COALESCE(r.message,'') as message, r.created_time`).
		Joins("JOIN users u ON u.id = r.user_id").
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		// 交易所划转只出现在 partner_credit_list，不进充值列表
		Where("r.tx_hash NOT LIKE ?", "partner:%")
	teamIDs, teamMode, err := s.teamUserIDsForQuery(ctx, q)
	if err == nil && teamMode {
		db = db.Where("r.user_id IN ?", teamIDs)
	} else if addressFilter != "" {
		db = db.Where("(r.from_address LIKE ? OR u.address LIKE ?)", "%"+addressFilter+"%", "%"+addressFilter+"%")
	}
	switch typeFilter {
	case "admin", "后台充值":
		db = db.Where("r.tx_hash LIKE ?", "admin-%")
	case "win", "win充值", "win_recharge":
		db = db.Where("UPPER(r.asset) = ? AND r.tx_hash NOT LIKE ?", biz.TokenWIN, "admin-%")
	case "win_a", "win-a", "win-a充值", "wina", "win_a_recharge":
		db = db.Where("UPPER(r.asset) = ? AND r.tx_hash NOT LIKE ?", biz.TokenWINA, "admin-%")
	case "usdt", "usdt充值", "usdt_recharge":
		db = db.Where("(UPPER(r.asset) = ? OR r.asset = '' OR r.asset IS NULL) AND r.tx_hash NOT LIKE ?", biz.TokenUSDT, "admin-%")
	}
	start, end := parseLegacyTimeRange(q)
	if start != nil {
		db = db.Where("r.created_time >= ?", *start)
	}
	if end != nil {
		db = db.Where("r.created_time <= ?", *end)
	}
	return db
}

func (s *AdminLegacyService) rechargeStats(ctx context.Context, q url.Values) (map[string]interface{}, error) {
	type statRow struct {
		TotalCount  int64
		UsdtTotal   decimal.Decimal
		WinTotal    decimal.Decimal
		WinATotal   decimal.Decimal
		AdminTotal  decimal.Decimal
	}
	var row statRow
	err := s.rechargeListDB(ctx, q).
		Select(`COUNT(*) as total_count,
			COALESCE(SUM(CASE WHEN r.tx_hash LIKE 'admin-%' THEN r.amount ELSE 0 END),0) as admin_total,
			COALESCE(SUM(CASE WHEN UPPER(r.asset) = ? AND r.tx_hash NOT LIKE 'admin-%' AND r.tx_hash NOT LIKE 'partner:%' THEN r.amount ELSE 0 END),0) as win_total,
			COALESCE(SUM(CASE WHEN UPPER(r.asset) = ? AND r.tx_hash NOT LIKE 'admin-%' AND r.tx_hash NOT LIKE 'partner:%' THEN r.amount ELSE 0 END),0) as win_a_total,
			COALESCE(SUM(CASE WHEN (UPPER(r.asset) = ? OR r.asset = '' OR r.asset IS NULL) AND r.tx_hash NOT LIKE 'admin-%' AND r.tx_hash NOT LIKE 'partner:%' THEN r.amount ELSE 0 END),0) as usdt_total`,
			biz.TokenWIN, biz.TokenWINA, biz.TokenUSDT).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"totalCount": row.TotalCount,
		"usdtTotal":  row.UsdtTotal.String(),
		"winTotal":   row.WinTotal.String(),
		"winATotal":  row.WinATotal.String(),
		"adminTotal": row.AdminTotal.String(),
	}, nil
}

// 管理端把释放/溢出归并到主类型展示，筛选与统计需按同样口径聚合原始 type
var (
	mgmtRewardTypes    = []string{biz.RewardTypeMgmt, biz.RewardTypeMgmtPoolRelease, biz.RewardTypeMgmtOverflow}
	dynamicRewardTypes = []string{biz.RewardTypeDynamicUsdt, biz.RewardTypeDirectPoolRelease}
)

type rewardListRow struct {
	ID             int64
	Type           string
	Asset          string
	Amount         decimal.Decimal
	Address        string
	FromAddress    string
	SettlementDate *string
	CreatedTime    time.Time
}

func (s *AdminLegacyService) rewardListDB(ctx context.Context, q url.Values) *gorm.DB {
	addressFilter := strings.TrimSpace(q.Get("address"))
	typeFilter := strings.TrimSpace(firstNonEmpty(q.Get("type"), q.Get("reason")))
	db := s.data.DB().WithContext(ctx).
		Table("reward_logs rl").
		Joins("JOIN users u ON u.id = rl.user_id").
		Joins("LEFT JOIN users fu ON fu.id = rl.from_user_id")
	teamIDs, teamMode, err := s.teamUserIDsForQuery(ctx, q)
	if err == nil && teamMode {
		db = db.Where("rl.user_id IN ?", teamIDs)
	} else if addressFilter != "" {
		db = db.Where("u.address LIKE ?", "%"+addressFilter+"%")
	}
	if typeFilter != "" && typeFilter != "undefined" && typeFilter != "null" {
		switch typeFilter {
		case "mgmt", "mgmt_pool_release", "管理奖":
			db = db.Where("rl.type IN ?", mgmtRewardTypes)
		case "dynamic_usdt", "direct_pool_release", "直推奖":
			db = db.Where("rl.type IN ?", dynamicRewardTypes)
		case "community_subsidy_5":
			db = db.Where("rl.type = ? AND u.community_subsidy_rate = ?", biz.RewardTypeCommunitySubsidy, biz.SubsidyRateMin)
		case "community_subsidy_10":
			db = db.Where("rl.type = ? AND u.community_subsidy_rate = ?", biz.RewardTypeCommunitySubsidy, biz.SubsidyRateMid)
		case "community_subsidy_15":
			db = db.Where("rl.type = ? AND u.community_subsidy_rate = ?", biz.RewardTypeCommunitySubsidy, biz.SubsidyRateMax)
		default:
			db = db.Where("rl.type = ?", typeFilter)
		}
	}
	start, end := parseLegacyTimeRange(q)
	if start != nil {
		db = db.Where("rl.created_time >= ?", *start)
	}
	if end != nil {
		db = db.Where("rl.created_time <= ?", *end)
	}
	return db
}

func (s *AdminLegacyService) rewardStats(ctx context.Context, q url.Values) (map[string]interface{}, error) {
	type statRow struct {
		TotalCount            int64
		AixTotal              decimal.Decimal
		UsdtTotal             decimal.Decimal
		StaticAixTotal        decimal.Decimal
		DynamicTotal          decimal.Decimal
		MgmtTotal             decimal.Decimal
		ZeroAccountTotal      decimal.Decimal
		CommunitySubsidyTotal decimal.Decimal
	}
	var row statRow
	err := s.rewardListDB(ctx, q).
		Select(`COUNT(*) as total_count,
			COALESCE(SUM(CASE WHEN UPPER(rl.asset) = ? THEN rl.amount ELSE 0 END),0) as aix_total,
			COALESCE(SUM(CASE WHEN UPPER(rl.asset) = ? THEN rl.amount ELSE 0 END),0) as usdt_total,
			COALESCE(SUM(CASE WHEN rl.type = ? THEN rl.amount ELSE 0 END),0) as static_aix_total,
			COALESCE(SUM(CASE WHEN rl.type IN (?,?) THEN rl.amount ELSE 0 END),0) as dynamic_total,
			COALESCE(SUM(CASE WHEN rl.type IN (?,?,?) THEN rl.amount ELSE 0 END),0) as mgmt_total,
			COALESCE(SUM(CASE WHEN rl.type = ? THEN rl.amount ELSE 0 END),0) as zero_account_total,
			COALESCE(SUM(CASE WHEN rl.type = ? THEN rl.amount ELSE 0 END),0) as community_subsidy_total`,
			biz.TokenAIX, biz.TokenUSDT,
			biz.RewardTypeStaticAix,
			biz.RewardTypeDynamicUsdt, biz.RewardTypeDirectPoolRelease,
			biz.RewardTypeMgmt, biz.RewardTypeMgmtPoolRelease, biz.RewardTypeMgmtOverflow,
			biz.RewardTypeZeroAccount, biz.RewardTypeCommunitySubsidy).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}

	// AIX 与 USDT 混在同一张表，合计只能按资产分组，不能相加成一个数
	type assetRow struct {
		Asset string
		Total decimal.Decimal
		Cnt   int64
	}
	var assetRows []assetRow
	if err := s.rewardListDB(ctx, q).
		Select("UPPER(rl.asset) as asset, COALESCE(SUM(rl.amount),0) as total, COUNT(*) as cnt").
		Group("UPPER(rl.asset)").
		Order("total desc").
		Scan(&assetRows).Error; err != nil {
		return nil, err
	}
	assetTotals := make([]map[string]interface{}, 0, len(assetRows))
	for _, a := range assetRows {
		if strings.TrimSpace(a.Asset) == "" {
			continue
		}
		assetTotals = append(assetTotals, map[string]interface{}{
			"asset": a.Asset,
			"total": a.Total.String(),
			"count": a.Cnt,
		})
	}

	return map[string]interface{}{
		"assetTotals":           assetTotals,
		"totalCount":            row.TotalCount,
		"aixTotal":              row.AixTotal.String(),
		"usdtTotal":             row.UsdtTotal.String(),
		"staticAixTotal":        row.StaticAixTotal.String(),
		"dynamicTotal":          row.DynamicTotal.String(),
		"mgmtTotal":             row.MgmtTotal.String(),
		"zeroAccountTotal":      row.ZeroAccountTotal.String(),
		"communitySubsidyTotal": row.CommunitySubsidyTotal.String(),
	}, nil
}

func rewardRowToItem(r rewardListRow) map[string]interface{} {
	settle := ""
	if r.SettlementDate != nil {
		settle = *r.SettlementDate
	}
	return map[string]interface{}{
		"id":             r.ID,
		"type":           normalizeRewardType(r.Type),
		"rawType":        r.Type,
		"asset":          r.Asset,
		"amount":         r.Amount.String(),
		"address":        r.Address,
		"addressTwo":     r.FromAddress,
		"reason":         normalizeRewardType(r.Type),
		"settlementDate": settle,
		"createdAt":      formatLegacyTime(r.CreatedTime),
		"date":           formatLegacyTime(r.CreatedTime),
	}
}

func rechargeRowToItem(r rechargeListRow) map[string]interface{} {
	remark, typeCode := classifyRechargeType(r.Asset, r.TxHash, r.Message)
	return map[string]interface{}{
		"id":        r.ID,
		"address":   r.Address,
		"asset":     strings.ToUpper(strings.TrimSpace(r.Asset)),
		"amount":    r.Amount.String(),
		"txHash":    r.TxHash,
		"remark":    remark,
		"type":      typeCode,
		"createdAt": formatLegacyTime(r.CreatedTime),
	}
}

func writeRechargeCSV(w http.ResponseWriter, rows []rechargeListRow) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="recharge_export.csv"`)
	// UTF-8 BOM for Excel
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"ID", "账户", "类型", "资产", "金额", "交易哈希", "创建时间"}); err != nil {
		return err
	}
	for _, r := range rows {
		remark, _ := classifyRechargeType(r.Asset, r.TxHash, r.Message)
		if err := cw.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.Address,
			remark,
			strings.ToUpper(strings.TrimSpace(r.Asset)),
			r.Amount.String(),
			r.TxHash,
			formatLegacyTime(r.CreatedTime),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func withdrawalWithinTime(w *biz.Withdrawal, start, end *time.Time) bool {
	if w == nil {
		return false
	}
	if start != nil && w.CreatedAt.Before(*start) {
		return false
	}
	if end != nil && w.CreatedAt.After(*end) {
		return false
	}
	return true
}

func orderWithinTime(o *biz.AdminOrderDetail, start, end *time.Time) bool {
	if o == nil || o.Order == nil {
		return false
	}
	t := o.Order.CreatedAt
	if t.IsZero() {
		t = o.Order.CreatedTime
	}
	if start != nil && t.Before(*start) {
		return false
	}
	if end != nil && t.After(*end) {
		return false
	}
	return true
}

func sumBuyOrderStats(list []*biz.AdminOrderDetail) map[string]interface{} {
	total := decimal.Zero
	for _, o := range list {
		if o == nil || o.Order == nil {
			continue
		}
		principal, err := decimal.NewFromString(strings.TrimSpace(o.Order.Principal))
		if err != nil {
			continue
		}
		total = total.Add(principal)
	}
	return map[string]interface{}{
		"totalCount":     len(list),
		"principalTotal": total.String(),
	}
}

func sumWithdrawalStats(list []*biz.Withdrawal) map[string]interface{} {
	winTotal := decimal.Zero
	sdtTotal := decimal.Zero
	usdtTotal := decimal.Zero
	winCount := int64(0)
	sdtCount := int64(0)
	usdtCount := int64(0)
	reviewCount := int64(0)
	for _, w := range list {
		if w == nil {
			continue
		}
		amt, err := decimal.NewFromString(strings.TrimSpace(w.Amount))
		if err != nil {
			continue
		}
		if w.Status == biz.WithdrawStatusRejected || w.Status == biz.WithdrawStatusFailed {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(w.Asset)) {
		case biz.TokenWIN:
			winTotal = winTotal.Add(amt)
			winCount++
		case biz.TokenSDT:
			sdtTotal = sdtTotal.Add(amt)
			sdtCount++
		case biz.TokenUSDT:
			usdtTotal = usdtTotal.Add(amt)
			usdtCount++
		}
		if w.Status == biz.WithdrawStatusReview {
			reviewCount++
		}
	}
	return map[string]interface{}{
		"winTotal":    winTotal.String(),
		"sdtTotal":    sdtTotal.String(),
		"usdtTotal":   usdtTotal.String(),
		"winCount":    winCount,
		"sdtCount":    sdtCount,
		"usdtCount":   usdtCount,
		"reviewCount": reviewCount,
	}
}
