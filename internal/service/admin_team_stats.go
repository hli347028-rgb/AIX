package service

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/biz"
	"backend/internal/data"
	"backend/internal/pkg/eth"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const teamStatsNameMaxRunes = 32

func (s *AdminLegacyService) HandleTeamStatsList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	items, err := s.teamStatsItems(ctx)
	if err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{
		"data":  items,
		"count": len(items),
	})
}

func (s *AdminLegacyService) HandleTeamStatsExport(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	items, err := s.teamStatsItems(ctx)
	if err != nil {
		return err
	}
	return writeTeamStatsCSV(ctx.Response(), items)
}

func (s *AdminLegacyService) teamStatsItems(ctx khttp.Context) ([]map[string]interface{}, error) {
	start, end := parseLegacyTimeRange(ctx.Request().URL.Query())
	var rows []data.AdminTeamWatchPO
	if err := s.data.DB().WithContext(ctx).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item, err := s.teamStatsItem(ctx, row, start, end)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *AdminLegacyService) HandleTeamStatsSave(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	address, err := eth.NormalizeAddress(ctx.Request().Form.Get("address"))
	if err != nil {
		return errors.BadRequest("INVALID_ADDRESS", "地址格式不正确")
	}
	_, updateName := ctx.Request().Form["name"]
	name := strings.TrimSpace(ctx.Request().Form.Get("name"))
	if utf8.RuneCountInString(name) > teamStatsNameMaxRunes {
		return errors.BadRequest("INVALID_NAME", "名字最多 32 个字")
	}
	user, err := s.userRepo.FindByAddress(ctx, address)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.BadRequest("USER_NOT_FOUND", "该地址未注册")
	}
	var row data.AdminTeamWatchPO
	err = s.data.DB().WithContext(ctx).Where("address = ?", address).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = data.AdminTeamWatchPO{Address: address, DisplayName: name}
		if err := s.data.DB().WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if updateName {
		if err := s.data.DB().WithContext(ctx).Model(&row).Update("display_name", name).Error; err != nil {
			return err
		}
		row.DisplayName = name
	}
	item, err := s.teamStatsItem(ctx, row, nil, nil)
	if err != nil {
		return err
	}
	return ctx.Result(200, item)
}

func (s *AdminLegacyService) HandleTeamStatsDelete(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id := strings.TrimSpace(ctx.Request().Form.Get("id"))
	if id == "" {
		return errors.BadRequest("INVALID_ID", "缺少记录")
	}
	if err := s.data.DB().WithContext(ctx).Delete(&data.AdminTeamWatchPO{}, "id = ?", id).Error; err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) teamStatsItem(ctx khttp.Context, row data.AdminTeamWatchPO, start, end *time.Time) (map[string]interface{}, error) {
	item := map[string]interface{}{
		"id":              row.ID,
		"address":         row.Address,
		"display_name":    row.DisplayName,
		"usdt_recharge":   "0",
		"win_recharge":    "0",
		"downline_count":  0,
		"team_perf":       "0",
		"large_area_perf": "0",
		"small_area_perf": "0",
		"registered":      false,
	}
	user, err := s.userRepo.FindByAddress(ctx, row.Address)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return item, nil
	}
	item["registered"] = true
	ids, err := s.userRepo.ListUserIDsUnder(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	db := s.data.DB().WithContext(ctx)
	if start == nil && end == nil {
		item["team_perf"] = blankPerf(user.TeamPerf)
		item["large_area_perf"] = blankPerf(user.LargeAreaPerf)
		item["small_area_perf"] = blankPerf(user.SmallAreaPerf)
		item["downline_count"] = len(ids)
	} else {
		count, err := countDownlineInRange(db, ids, start, end)
		if err != nil {
			return nil, err
		}
		item["downline_count"] = count
		large, small, team, err := areaPerfInRange(db, user.ID, ids, start, end)
		if err != nil {
			return nil, err
		}
		item["team_perf"] = formatStatAmount(team)
		item["large_area_perf"] = formatStatAmount(large)
		item["small_area_perf"] = formatStatAmount(small)
	}
	usdtWhere, usdtArgs := withCreatedTime(`
		status = ? AND (UPPER(asset) = ? OR asset = '' OR asset IS NULL) AND tx_hash NOT LIKE 'partner:%'
	`, []interface{}{biz.RechargeStatusConfirmed, biz.TokenUSDT}, start, end)
	usdt, err := sumDownlineRecharge(db, ids, usdtWhere, usdtArgs...)
	if err != nil {
		return nil, err
	}
	winWhere, winArgs := withCreatedTime(`
		status = ? AND UPPER(asset) = ?
	`, []interface{}{biz.RechargeStatusConfirmed, biz.TokenWIN}, start, end)
	win, err := sumDownlineRecharge(db, ids, winWhere, winArgs...)
	if err != nil {
		return nil, err
	}
	item["usdt_recharge"] = formatStatAmount(usdt)
	item["win_recharge"] = formatStatAmount(win)
	return item, nil
}

func withCreatedTime(where string, args []interface{}, start, end *time.Time) (string, []interface{}) {
	if start != nil {
		where += " AND created_time >= ?"
		args = append(args, *start)
	}
	if end != nil {
		where += " AND created_time <= ?"
		args = append(args, *end)
	}
	return where, args
}

func countDownlineInRange(db *gorm.DB, ids []int64, start, end *time.Time) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var total int64
	const chunk = 500
	for i := 0; i < len(ids); i += chunk {
		j := i + chunk
		if j > len(ids) {
			j = len(ids)
		}
		q := db.Model(&data.UserPO{}).Where("id IN ?", ids[i:j])
		if start != nil {
			q = q.Where("created_time >= ?", *start)
		}
		if end != nil {
			q = q.Where("created_time <= ?", *end)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return 0, err
		}
		total += n
	}
	return int(total), nil
}

func areaPerfInRange(db *gorm.DB, rootID int64, ids []int64, start, end *time.Time) (large, small, team decimal.Decimal, err error) {
	if rootID <= 0 || len(ids) == 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero, nil
	}
	type parentRow struct {
		ID        int64
		InviterID *int64
	}
	type principalRow struct {
		UserID int64
		Total  decimal.Decimal
	}
	children := map[int64][]int64{}
	stake := map[int64]decimal.Decimal{}
	const chunk = 500
	for i := 0; i < len(ids); i += chunk {
		j := i + chunk
		if j > len(ids) {
			j = len(ids)
		}
		part := ids[i:j]
		var parents []parentRow
		if err := db.Model(&data.UserPO{}).Select("id", "inviter_id").Where("id IN ?", part).Find(&parents).Error; err != nil {
			return decimal.Zero, decimal.Zero, decimal.Zero, err
		}
		for _, row := range parents {
			stake[row.ID] = decimal.Zero
			if row.InviterID != nil {
				children[*row.InviterID] = append(children[*row.InviterID], row.ID)
			}
		}
		q := db.Model(&data.OrderPO{}).
			Select("user_id, COALESCE(SUM(principal), 0) AS total").
			Where("user_id IN ?", part).
			Where("status IN ?", []string{biz.OrderStatusActive, biz.OrderStatusExited})
		if start != nil {
			q = q.Where("created_time >= ?", *start)
		}
		if end != nil {
			q = q.Where("created_time <= ?", *end)
		}
		var principals []principalRow
		if err := q.Group("user_id").Scan(&principals).Error; err != nil {
			return decimal.Zero, decimal.Zero, decimal.Zero, err
		}
		for _, row := range principals {
			stake[row.UserID] = row.Total
		}
	}
	memo := map[int64]decimal.Decimal{}
	visiting := map[int64]bool{}
	var subtree func(int64) (decimal.Decimal, error)
	subtree = func(id int64) (decimal.Decimal, error) {
		if value, ok := memo[id]; ok {
			return value, nil
		}
		if visiting[id] {
			return decimal.Zero, fmt.Errorf("invite relationship contains a cycle at user %d", id)
		}
		visiting[id] = true
		total := stake[id]
		for _, childID := range children[id] {
			childTotal, err := subtree(childID)
			if err != nil {
				return decimal.Zero, err
			}
			total = total.Add(childTotal)
		}
		visiting[id] = false
		memo[id] = total
		return total, nil
	}
	branches := make([]decimal.Decimal, 0, len(children[rootID]))
	for _, childID := range children[rootID] {
		value, err := subtree(childID)
		if err != nil {
			return decimal.Zero, decimal.Zero, decimal.Zero, err
		}
		branches = append(branches, value)
	}
	large, small, team = biz.CalcAreaPerformance(branches)
	return large, small, team, nil
}

func writeTeamStatsCSV(w http.ResponseWriter, items []map[string]interface{}) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="team_stats.csv"`)
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"名字", "地址", "下级USDT充值", "下级WIN充值", "下级人数", "团队总业绩", "大区业绩", "小区业绩"}); err != nil {
		return err
	}
	for _, item := range items {
		if err := cw.Write([]string{
			fmt.Sprint(item["display_name"]),
			fmt.Sprint(item["address"]),
			fmt.Sprint(item["usdt_recharge"]),
			fmt.Sprint(item["win_recharge"]),
			fmt.Sprint(item["downline_count"]),
			fmt.Sprint(item["team_perf"]),
			fmt.Sprint(item["large_area_perf"]),
			fmt.Sprint(item["small_area_perf"]),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func sumDownlineRecharge(db *gorm.DB, ids []int64, where string, args ...interface{}) (decimal.Decimal, error) {
	total := decimal.Zero
	if len(ids) == 0 {
		return total, nil
	}
	const chunk = 500
	for start := 0; start < len(ids); start += chunk {
		end := start + chunk
		if end > len(ids) {
			end = len(ids)
		}
		var part decimal.Decimal
		q := append([]interface{}{}, args...)
		q = append(q, ids[start:end])
		err := db.Model(&data.RechargePO{}).
			Where(where+" AND user_id IN ?", q...).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&part).Error
		if err != nil {
			return decimal.Zero, err
		}
		total = total.Add(part)
	}
	return total, nil
}

func formatStatAmount(d decimal.Decimal) string {
	s := d.StringFixed(4)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func blankPerf(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0"
	}
	return v
}
