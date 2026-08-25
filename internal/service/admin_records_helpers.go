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

func (s *AdminLegacyService) sumChainRecharge(ctx context.Context, asset string, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("recharges r").
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		Where("r.tx_hash NOT LIKE ?", "admin-%")
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

func (s *AdminLegacyService) sumOrderPrincipal(ctx context.Context, fundSource string, since *time.Time) (decimal.Decimal, error) {
	db := s.data.DB().WithContext(ctx).Table("orders")
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
		Where("r.status = ?", biz.RechargeStatusConfirmed)
	if addressFilter != "" {
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
			COALESCE(SUM(CASE WHEN UPPER(r.asset) = ? AND r.tx_hash NOT LIKE 'admin-%' THEN r.amount ELSE 0 END),0) as win_total,
			COALESCE(SUM(CASE WHEN UPPER(r.asset) = ? AND r.tx_hash NOT LIKE 'admin-%' THEN r.amount ELSE 0 END),0) as win_a_total,
			COALESCE(SUM(CASE WHEN (UPPER(r.asset) = ? OR r.asset = '' OR r.asset IS NULL) AND r.tx_hash NOT LIKE 'admin-%' THEN r.amount ELSE 0 END),0) as usdt_total`,
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
