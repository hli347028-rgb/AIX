package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/conf"
	"backend/internal/data"
	authmw "backend/internal/middleware"
	jwtpkg "backend/internal/pkg/token"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
)

// AdminLegacyService serves /api/admin_dhb/* compatibility routes for the Vue admin UI.
type AdminLegacyService struct {
	admin      *biz.AdminUsecase
	userRepo   biz.UserRepo
	walletRepo biz.WalletRepo
	data       *data.Data
	authCfg    *conf.AuthConfig
	walletCfg  *conf.WalletConfig
	partnerCfg *conf.TransferPartnerConfig
}

func NewAdminLegacyService(
	admin *biz.AdminUsecase,
	userRepo biz.UserRepo,
	walletRepo biz.WalletRepo,
	data *data.Data,
	authCfg *conf.AuthConfig,
	walletCfg *conf.WalletConfig,
	partnerCfg *conf.TransferPartnerConfig,
) *AdminLegacyService {
	return &AdminLegacyService{
		admin: admin, userRepo: userRepo, walletRepo: walletRepo,
		data: data, authCfg: authCfg, walletCfg: walletCfg, partnerCfg: partnerCfg,
	}
}

var legacyMenuPaths = []string{
	"/home", "/member", "/recharge", "/withdrawList", "/subscription",
	"/ordersList", "/config", "/exchangeList", "/transferList",
	"/exchangeTransfer", "/settlement", "/news", "/newsEdit", "/lookChildren",
}

var legacyMainOnlyMenuPaths = []string{
	"/operationLog",
}

// subAccountConfigDefs 主账户在配置项维护的子账户密码与模块（固定 user1~user3）。
var subAccountConfigDefs = []struct {
	ID       int
	Slot     int
	Kind     string // password | modules
	NameTmpl string
}{
	{101, 0, "password", "子账户%s密码"},
	{102, 0, "modules", "子账户%s可访问模块"},
	{103, 1, "password", "子账户%s密码"},
	{104, 1, "modules", "子账户%s可访问模块"},
	{105, 2, "password", "子账户%s密码"},
	{106, 2, "modules", "子账户%s可访问模块"},
}

func normalizeSubAccountMenuPaths(modules []string) []string {
	allowed := make(map[string]bool)
	for _, p := range legacyMenuPaths {
		allowed[p] = true
	}
	out := make([]string, 0, len(modules)+2)
	seen := make(map[string]bool)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || !allowed[p] || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, m := range modules {
		add(m)
	}
	if seen["/news"] {
		add("/newsEdit")
	}
	if seen["/member"] {
		add("/lookChildren")
	}
	return out
}

func parseSubAccountModulesInput(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	allowed := make(map[string]bool)
	for _, p := range legacyMenuPaths {
		allowed[p] = true
	}
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if !allowed[p] {
			return nil, fmt.Errorf("无效模块路径: %s", p)
		}
		out = append(out, p)
	}
	return out, nil
}

func ensureSnapshotSubAccounts(snapshot *conf.SystemConfigSnapshot) {
	if snapshot == nil {
		return
	}
	defaults := conf.DefaultAdminSubAccounts()
	if len(snapshot.AdminSubAccounts) >= len(defaults) {
		return
	}
	base := make([]conf.AdminSubAccount, len(defaults))
	for i, d := range defaults {
		base[i] = d
	}
	for i := range base {
		if i < len(snapshot.AdminSubAccounts) {
			item := snapshot.AdminSubAccounts[i]
			if strings.TrimSpace(item.Account) != "" {
				base[i].Account = item.Account
			}
			if strings.TrimSpace(item.Password) != "" {
				base[i].Password = item.Password
			}
			if len(item.Modules) > 0 {
				base[i].Modules = append([]string(nil), item.Modules...)
			}
		}
	}
	snapshot.AdminSubAccounts = base
}

func subAccountConfigName(slot int, kind string) string {
	defaults := conf.DefaultAdminSubAccounts()
	acc := "user"
	if slot >= 0 && slot < len(defaults) {
		acc = defaults[slot].Account
	}
	for _, def := range subAccountConfigDefs {
		if def.Slot == slot && def.Kind == kind {
			return fmt.Sprintf(def.NameTmpl, acc)
		}
	}
	return ""
}

type legacyConfigItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

var legacyConfigDefs = []struct {
	ID   int
	Name string
}{
	{1, "最低认购(U)"},
	{2, "静态利率(%)"},
	{3, "出局倍数"},
	{4, "直推比例"},
	{5, "收款地址"},
	{6, "USDT合约"},
	{7, "AIX价格(USDT/枚)"},
	{8, "WIN价格(USDT/枚)"},
	{36, "WIN-A价格(USDT/枚)"},
	{9, "兑换手续费率(%)"},
	{31, "USDT充值最小值"},
	{32, "WIN充值最小值"},
	{37, "WIN-A充值最小值"},
	{33, "WIN提现审核阈值(超过需审核)"},
	{34, "AIX-USDT提现审核阈值(超过需审核)"},
	{35, "USDT提现审核阈值(超过需审核)"},
	{38, "交易所划转单笔下限(WIN)"},
	{39, "交易所划转单笔上限(WIN)"},
	{40, "交易所划转单日上限(WIN)"},
	{41, "AIX兑换审核阈值(%)"},
	{11, "W1 收益系数"},
	{12, "W2 收益系数"},
	{13, "W3 收益系数"},
	{14, "W4 收益系数"},
	{15, "W5 收益系数"},
	{16, "W6 收益系数"},
	{17, "W7 收益系数"},
	{18, "W8 收益系数"},
	{19, "W9 收益系数"},
	{20, "W10 收益系数"},
	{21, "成为 W1 的小区业绩金额(USDT)"},
	{22, "成为 W2 的小区业绩金额(USDT)"},
	{23, "成为 W3 的小区业绩金额(USDT)"},
	{24, "成为 W4 的小区业绩金额(USDT)"},
	{25, "成为 W5 的小区业绩金额(USDT)"},
	{26, "成为 W6 的小区业绩金额(USDT)"},
	{27, "成为 W7 的小区业绩金额(USDT)"},
	{28, "成为 W8 的小区业绩金额(USDT)"},
	{29, "成为 W9 的小区业绩金额(USDT)"},
	{30, "成为 W10 的小区业绩金额(USDT)"},
}

func (s *AdminLegacyService) HandleLogin(ctx khttp.Context) error {
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	account := firstNonEmpty(
		ctx.Request().Form.Get("username"),
		ctx.Request().Form.Get("account"),
		ctx.Request().Form.Get("email"),
	)
	password := ctx.Request().Form.Get("password")
	var session *AdminSession
	var jwtAddr string
	if account == s.authCfg.GetAdminAccount() && password == s.authCfg.GetAdminPassword() {
		session = &AdminSession{Operator: s.authCfg.GetAdminAccount(), IsMain: true}
		jwtAddr = biz.ZeroAddress
	} else if s.validateSubLogin(account, password) {
		sub := strings.TrimSpace(account)
		session = &AdminSession{Operator: sub, IsMain: false}
		jwtAddr = s.authCfg.SubAccountJWTAddress(sub)
	} else {
		return errors.Unauthorized("UNAUTHORIZED", "账号或密码错误")
	}
	token, _, err := jwtpkg.Generate(jwtAddr, s.authCfg.GetJwtSecret(), time.Now())
	if err != nil {
		return err
	}
	s.logAdminLogin(ctx, session, ctx.Request())
	return ctx.Result(200, map[string]string{"token": token})
}

func (s *AdminLegacyService) HandleMyAuthList(ctx khttp.Context) error {
	session, err := s.requireAnyAdmin(ctx)
	if err != nil {
		return err
	}
	var paths []string
	if session.IsMain {
		paths = append([]string{}, legacyMenuPaths...)
		paths = append(paths, legacyMainOnlyMenuPaths...)
	} else {
		modules := s.subAccountModules(session.Operator)
		paths = normalizeSubAccountMenuPaths(modules)
		if len(paths) == 0 {
			paths = append([]string{}, legacyMenuPaths...)
		}
	}
	auth := make([]map[string]string, 0, len(paths))
	for _, p := range paths {
		auth = append(auth, map[string]string{"path": p})
	}
	super := "0"
	if session.IsMain {
		super = "1"
	}
	return ctx.Result(200, map[string]interface{}{
		"super":   super,
		"auth":    auth,
		"account": session.Operator,
	})
}

func (s *AdminLegacyService) HandleAll(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	stats, err := s.buildDashboardStats(ctx)
	if err != nil {
		return err
	}
	return ctx.Result(200, stats)
}

func (s *AdminLegacyService) HandleUserList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	addressFilter := strings.TrimSpace(q.Get("address"))

	users, err := s.userRepo.ListAllUsers(ctx)
	if err != nil {
		return err
	}
	activeStake, err := s.sumActivePrincipalByUser(ctx)
	if err != nil {
		return err
	}
	allStake, err := s.sumAllPrincipalByUser(ctx)
	if err != nil {
		return err
	}
	totalIncome, releasedAmt, pendingRelease, err := s.sumOrderReleaseByUser(ctx)
	if err != nil {
		return err
	}
	directCount := map[int64]int{}
	for _, u := range users {
		if u.InviterID != nil {
			directCount[*u.InviterID]++
		}
	}

	filtered := make([]*biz.User, 0, len(users))
	for _, u := range users {
		if addressFilter != "" && !strings.Contains(strings.ToLower(u.Address), strings.ToLower(addressFilter)) {
			continue
		}
		filtered = append(filtered, u)
	}
	total := len(filtered)
	pageUsers := paginateSlice(filtered, offset, pageSize)

	items := make([]map[string]interface{}, 0, len(pageUsers))
	for _, u := range pageUsers {
		u.SyncCompatFields()
		inviteeCount := directCount[u.ID]
		vip := formatMgmtVIP(u.MgmtLevel)
		za, _ := decimal.NewFromString(strings.TrimSpace(u.ZeroAccountRewardTotal))
		cs, _ := decimal.NewFromString(strings.TrimSpace(u.CommunitySubsidyTotal))
		usdtWithdrawableDisplay := za.Add(cs).String() // 可提 U = 零号账户累计 + 社区补贴累计
		items = append(items, map[string]interface{}{
			"userId":              u.ID,
			"id":                  u.ID,
			"address":             u.Address,
			"username":            u.Username,
			"usdt_recharge":       u.UsdtRecharge,
			"usdt_reward":         u.UsdtReward,
			"aix_balance":         u.AixBalance,        // AIX 代币数
			"win_balance":            u.WinBalance,            // WIN 提现钱包
			"win_recharge_balance":   u.WinRechargeBalance,    // WIN 充值钱包
			"win_a_recharge_balance": u.WinARechargeBalance,   // WIN-A 充值钱包
			"usdt_withdrawable":      usdtWithdrawableDisplay,
			"pending_mgmt_reward": u.OverflowTotal(), // 兼容旧字段
			"overflow_reward":     u.OverflowTotal(), // 溢出奖励合计（管理奖+直推）
			"points":              u.Points,         // 当前积分
			"points_all":          u.PointsAll,      // 累计总积分
			"static_usdt_total":   u.StaticUsdtTotal,   // 静态总收益 USDT
			"mgmt_level":          u.MgmtLevel,
			"large_area_perf":     u.LargeAreaPerf,
			"small_area_perf":     u.SmallAreaPerf,
			"team_perf":           u.TeamPerf,
			"invitee_count":       inviteeCount,
			"createdAt":           formatLegacyTime(u.CreatedAt),
			"myRecommendAddress":  u.InviterAddress,
			"bAmount":             u.UsdtRecharge,
			"amountUsdtCurrent":   zeroIfEmpty(allStake[u.ID]),       // 总订单 = 全部认购本金（与积分一致）
			"amountUsdtActive":    zeroIfEmpty(activeStake[u.ID]),    // 进行中本金
			"totalIncome":         zeroIfEmpty(totalIncome[u.ID]),     // 总收益（可释放总额）
			"releasedAmount":      zeroIfEmpty(releasedAmt[u.ID]),    // 已释放
			"pendingRelease":      zeroIfEmpty(pendingRelease[u.ID]),  // 待释放
			"vip":                 vip,
			"historyRecommend":    strconv.Itoa(inviteeCount),
			"is_frozen":           u.IsFrozen,
			"frozen_at":           formatLegacyTimePtr(u.FrozenAt),
			"is_zero_account":     u.IsZeroAccount,
			"is_community_subsidy": u.IsCommunitySubsidy,
			"zero_account_set_at": formatLegacyTimePtr(u.ZeroAccountSetAt),
			"community_subsidy_set_at": formatLegacyTimePtr(u.CommunitySubsidySetAt),
			"zero_account_reward_total": u.ZeroAccountRewardTotal,
			"community_subsidy_total":   u.CommunitySubsidyTotal,
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"users": items,
		"count": total,
		"page":  page,
	})
}

func (s *AdminLegacyService) HandleConfig(ctx khttp.Context) error {
	session, err := s.requireAnyAdmin(ctx)
	if err != nil {
		return err
	}
	cfg := s.admin.GetPersistedConfigSnapshot()
	liveAixPrice := ""
	today := jwtpkg.NowChina().Format("2006-01-02")
	if price, err := s.walletRepo.GetAixPrice(ctx, today); err == nil && strings.TrimSpace(price) != "" {
		liveAixPrice = biz.FormatAixPrice(price)
	} else if latest, err := s.walletRepo.GetLatestAixPriceBefore(ctx, today); err == nil && strings.TrimSpace(latest) != "" && latest != "0" {
		liveAixPrice = biz.FormatAixPrice(latest)
	}
	items := make([]legacyConfigItem, 0, len(legacyConfigDefs)+len(subAccountConfigDefs))
	for _, def := range legacyConfigDefs {
		value := legacyConfigValue(cfg, s.walletCfg, def.ID)
		if def.ID == 7 && liveAixPrice != "" {
			value = liveAixPrice
		}
		items = append(items, legacyConfigItem{
			ID: def.ID, Name: def.Name, Value: value,
		})
	}
	if session.IsMain {
		ensureSnapshotSubAccounts(cfg)
		for _, def := range subAccountConfigDefs {
			items = append(items, legacyConfigItem{
				ID:    def.ID,
				Name:  subAccountConfigName(def.Slot, def.Kind),
				Value: subAccountConfigValue(cfg, def.Slot, def.Kind),
			})
		}
	}
	return ctx.Result(200, map[string]interface{}{"config": items})
}

func (s *AdminLegacyService) HandleConfigUpdate(ctx khttp.Context) error {
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.Atoi(ctx.Request().Form.Get("id"))
	value := ctx.Request().Form.Get("value")
	if id >= 101 && id <= 106 {
		if _, err := s.requireMainAdmin(ctx); err != nil {
			return err
		}
	} else if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	snapshot := s.admin.GetPersistedConfigSnapshot()
	if id >= 101 && id <= 106 {
		if err := applySubAccountConfigUpdate(snapshot, id, value); err != nil {
			return err
		}
	} else if err := applyLegacyConfigUpdate(snapshot, s.walletCfg, id, value); err != nil {
		return err
	}
	if _, err := s.admin.UpdateSystemConfig(ctx, s.token(ctx), snapshot); err != nil {
		return err
	}
	if id >= 21 && id <= 30 {
		if err := s.userRepo.RefreshPerformance(ctx); err != nil {
			return err
		}
	}
	if id == 8 && strings.TrimSpace(value) != "" {
		_ = s.walletRepo.UpsertCurrentWinPrice(ctx, strings.TrimSpace(value), "admin")
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleBuyList(ctx khttp.Context) error {
	return s.handleOrderList(ctx)
}

func (s *AdminLegacyService) HandleWithdrawList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	addressFilter := strings.TrimSpace(q.Get("address"))
	assetFilter := strings.ToUpper(strings.TrimSpace(q.Get("asset")))
	statusFilter := strings.TrimSpace(q.Get("status"))
	start, end := parseLegacyTimeRange(q)

	list, err := s.admin.ListAllWithdrawals(ctx, s.token(ctx))
	if err != nil {
		return err
	}
	filtered := make([]*biz.Withdrawal, 0, len(list))
	for _, w := range list {
		if addressFilter != "" && !strings.Contains(strings.ToLower(w.Address), strings.ToLower(addressFilter)) {
			continue
		}
		if assetFilter != "" && strings.ToUpper(strings.TrimSpace(w.Asset)) != assetFilter {
			continue
		}
		if statusFilter != "" && strings.TrimSpace(w.Status) != statusFilter {
			continue
		}
		if !withdrawalWithinTime(w, start, end) {
			continue
		}
		filtered = append(filtered, w)
	}
	stats := sumWithdrawalStats(filtered)
	total := len(filtered)
	pageItems := paginateSlice(filtered, offset, pageSize)
	items := make([]map[string]interface{}, 0, len(pageItems))
	for _, w := range pageItems {
		items = append(items, map[string]interface{}{
			"id":        w.ID,
			"address":   w.Address,
			"toAddress": w.ToAddress,
			"amount":    w.Amount,
			"fee":       w.Fee,
			"netAmount": w.NetAmount,
			"asset":     w.Asset,
			"status":    w.Status,
			"txHash":    w.TxHash,
			"remark":    w.Remark,
			"createdAt": formatLegacyTime(w.CreatedAt),
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"withdraw": items,
		"list":     items,
		"count":    total,
		"page":     page,
		"stats":    stats,
	})
}

// HandleExchangeList 管理端：AIX→USDT 兑换记录列表
func (s *AdminLegacyService) HandleExchangeList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	addressFilter := strings.TrimSpace(q.Get("address"))
	statusFilter := strings.ToLower(strings.TrimSpace(q.Get("status")))
	if statusFilter == "undefined" || statusFilter == "null" {
		statusFilter = ""
	}

	list, err := s.admin.ListExchangeRecords(ctx, s.token(ctx))
	if err != nil {
		return err
	}
	filtered := make([]*biz.ExchangeRecord, 0, len(list))
	for _, r := range list {
		if addressFilter != "" && !strings.Contains(strings.ToLower(r.UserAddress), strings.ToLower(addressFilter)) {
			continue
		}
		if statusFilter != "" && strings.ToLower(r.Status) != statusFilter {
			continue
		}
		filtered = append(filtered, r)
	}
	total := len(filtered)
	pageItems := paginateSlice(filtered, offset, pageSize)
	items := make([]map[string]interface{}, 0, len(pageItems))
	reviewCount := 0
	for _, r := range filtered {
		if r.Status == biz.ExchangeStatusReview {
			reviewCount++
		}
	}
	for _, r := range pageItems {
		items = append(items, map[string]interface{}{
			"id":            r.ID,
			"address":       r.UserAddress,
			"fromAsset":     r.FromAsset,
			"fromAmount":    r.FromAmount,
			"toAsset":       r.ToAsset,
			"toAmount":      r.ToAmount,
			"feeAmount":     r.FeeAmount,
			"feeRate":       r.FeeRate,
			"exchangePrice": r.ExchangePrice,
			"status":        r.Status,
			"remark":        r.Remark,
			"createdAt":     formatLegacyTime(r.CreatedTime),
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":  items,
		"count": total,
		"page":  page,
		"stats": map[string]interface{}{
			"reviewCount": reviewCount,
		},
	})
}

// HandleTransferList 管理端：用户互转 + 充值钱包→奖励钱包
func (s *AdminLegacyService) HandleTransferList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	addressFilter := strings.ToLower(strings.TrimSpace(q.Get("address")))
	typeFilter := strings.TrimSpace(q.Get("type"))
	if typeFilter == "undefined" || typeFilter == "null" {
		typeFilter = ""
	}

	reqCtx := ctx.Request().Context()
	var pos []data.TransferPO
	if err := s.data.DB().WithContext(reqCtx).Order("id desc").Find(&pos).Error; err != nil {
		return err
	}

	userIDs := make([]int64, 0, len(pos)*2)
	seen := map[int64]bool{}
	addID := func(id int64) {
		if id <= 0 || seen[id] {
			return
		}
		seen[id] = true
		userIDs = append(userIDs, id)
	}
	for _, po := range pos {
		addID(po.FromUserID)
		addID(po.ToUserID)
	}
	addrByID := map[int64]string{}
	if len(userIDs) > 0 {
		var users []data.UserPO
		if err := s.data.DB().WithContext(reqCtx).Select("id", "address").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}
		for _, u := range users {
			addrByID[u.ID] = u.Address
		}
	}

	filtered := make([]map[string]interface{}, 0, len(pos))
	for _, po := range pos {
		kind := transferKind(po.FromUserID, po.ToUserID, po.PayFrom)
		if typeFilter != "" && typeFilter != kind {
			continue
		}
		fromAddr := addrByID[po.FromUserID]
		toAddr := addrByID[po.ToUserID]
		if addressFilter != "" {
			if !strings.Contains(strings.ToLower(fromAddr), addressFilter) &&
				!strings.Contains(strings.ToLower(toAddr), addressFilter) {
				continue
			}
		}
		fromWallet, toWallet, kindLabel := transferKindLabel(kind)
		filtered = append(filtered, map[string]interface{}{
			"id":           po.ID,
			"type":         kind,
			"typeLabel":    kindLabel,
			"fromAddress":  fromAddr,
			"toAddress":    toAddr,
			"from_address": fromAddr,
			"to_address":   toAddr,
			"fromWallet":   fromWallet,
			"toWallet":     toWallet,
			"asset":        po.Asset,
			"amount":       po.Amount.String(),
			"payFrom":      po.PayFrom,
			"remark":       po.Remark,
			"createdAt":    formatLegacyTime(po.CreatedTime),
		})
	}
	total := len(filtered)
	pageItems := paginateSlice(filtered, offset, pageSize)
	if pageItems == nil {
		pageItems = []map[string]interface{}{}
	}
	return ctx.Result(200, map[string]interface{}{
		"list":  pageItems,
		"count": total,
		"page":  page,
	})
}

func transferKind(fromUserID, toUserID int64, payFrom string) string {
	if fromUserID == toUserID || payFrom == biz.PayFromRecharge {
		return "self"
	}
	return "user"
}

func transferKindLabel(kind string) (fromWallet, toWallet, label string) {
	if kind == "self" {
		return "充值钱包", "奖励钱包", "充值钱包→奖励钱包"
	}
	return "奖励钱包", "奖励钱包", "用户互转"
}

func (s *AdminLegacyService) HandleWithdrawPass(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	if err := s.admin.ApproveWithdrawalReview(ctx, s.token(ctx), id); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "审核通过，已进入打款队列"})
}

func (s *AdminLegacyService) HandleWithdrawReject(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	remark := strings.TrimSpace(ctx.Request().Form.Get("remark"))
	if err := s.admin.RejectWithdrawalReview(ctx, s.token(ctx), id, remark); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "已拒绝并退回余额"})
}

func (s *AdminLegacyService) HandleExchangePass(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	if err := s.admin.ApproveExchangeReview(ctx, s.token(ctx), id); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "审核通过，已入账可提 U"})
}

func (s *AdminLegacyService) HandleExchangeReject(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	remark := strings.TrimSpace(ctx.Request().Form.Get("remark"))
	if err := s.admin.RejectExchangeReview(ctx, s.token(ctx), id, remark); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "已拒绝并退回 AIX"})
}

func (s *AdminLegacyService) HandleRewardList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)

	var total int64
	if err := s.rewardListDB(ctx, q).Count(&total).Error; err != nil {
		return err
	}
	stats, err := s.rewardStats(ctx, q)
	if err != nil {
		return err
	}

	var rows []rewardListRow
	if err := s.rewardListDB(ctx, q).
		Select(`rl.id, rl.type, rl.asset, rl.amount, u.address,
			COALESCE(fu.address,'') as from_address, rl.settlement_date, rl.created_time`).
		Order("rl.id desc").Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, rewardRowToItem(r))
	}
	return ctx.Result(200, map[string]interface{}{
		"rewards": items,
		"list":    items,
		"count":   total,
		"page":    page,
		"stats":   stats,
	})
}

func normalizeRewardType(t string) string {
	switch strings.TrimSpace(t) {
	case biz.RewardTypeMgmtPoolRelease, biz.RewardTypeMgmtOverflow:
		return biz.RewardTypeMgmt // 管理端统一展示为管理奖（含出局额度不足/全部出局的溢出）
	case biz.RewardTypeDirectPoolRelease:
		return biz.RewardTypeDynamicUsdt // 管理端统一展示为直推奖
	default:
		return t
	}
}

func (s *AdminLegacyService) HandleRecordList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)

	var total int64
	if err := s.rechargeListDB(ctx, q).Count(&total).Error; err != nil {
		return err
	}
	stats, err := s.rechargeStats(ctx, q)
	if err != nil {
		return err
	}

	var rows []rechargeListRow
	if err := s.rechargeListDB(ctx, q).Order("r.id desc").Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, rechargeRowToItem(r))
	}
	return ctx.Result(200, map[string]interface{}{
		"rewards":   items,
		"list":      items,
		"locations": items,
		"count":     total,
		"page":      page,
		"stats":     stats,
	})
}

func (s *AdminLegacyService) HandleRecordListExport(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	var rows []rechargeListRow
	if err := s.rechargeListDB(ctx, q).Order("r.id desc").Scan(&rows).Error; err != nil {
		return err
	}
	w := ctx.Response()
	return writeRechargeCSV(w, rows)
}

func classifyRechargeType(asset, txHash, message string) (remark, typeCode string) {
	txHash = strings.TrimSpace(txHash)
	asset = strings.ToUpper(strings.TrimSpace(asset))
	message = strings.ToLower(strings.TrimSpace(message))
	if strings.HasPrefix(txHash, "admin-") {
		return "后台充值", "admin"
	}
	if asset == biz.TokenWINA || strings.Contains(message, "win_a_deposit") || strings.Contains(message, "win_a_recharge") {
		return "WIN-A充值", "win_a"
	}
	if asset == biz.TokenWIN || strings.Contains(message, "win_deposit") || strings.Contains(message, "win_recharge") {
		return "WIN充值", "win"
	}
	return "USDT充值", "usdt"
}

func (s *AdminLegacyService) HandleAdminRecharge(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	address := strings.TrimSpace(ctx.Request().Form.Get("address"))
	amount := strings.TrimSpace(ctx.Request().Form.Get("amount"))
	balance, credited, err := s.admin.AdminCreditBalance(ctx, s.token(ctx), address, amount)
	if err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{
		"status":  "ok",
		"balance": balance,
		"amount":  credited,
		"message": "充值成功",
	})
}

func (s *AdminLegacyService) HandleAdminRechargeWin(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	address := strings.TrimSpace(ctx.Request().Form.Get("address"))
	amount := strings.TrimSpace(ctx.Request().Form.Get("amount"))
	balance, credited, err := s.admin.AdminCreditWinBalance(ctx, s.token(ctx), address, amount)
	if err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{
		"status":               "ok",
		"asset":                "WIN",
		"win_balance":          balance,
		"win_recharge_balance": balance,
		"amount":               credited,
		"message":              "WIN 充值成功",
	})
}

func (s *AdminLegacyService) HandleRechargeToReward(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(ctx.Request().Form.Get("user_id"), 10, 64)
	amount := strings.TrimSpace(ctx.Request().Form.Get("amount"))
	rechargeBal, rewardBal, err := s.admin.AdminMoveRechargeToReward(ctx, s.token(ctx), userID, amount)
	if err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{
		"status":        "ok",
		"usdt_recharge": rechargeBal,
		"usdt_reward":   rewardBal,
		"message":       "已转入奖励钱包",
	})
}

func (s *AdminLegacyService) HandleGoodList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{
		"goods": []interface{}{},
		"count": 0,
		"page":  1,
	})
}

func (s *AdminLegacyService) HandleStubGoods(ctx khttp.Context) error {
	return s.HandleGoodList(ctx)
}

func (s *AdminLegacyService) HandleLockUserReward(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	form := ctx.Request().Form
	if form.Get("settlement_date") != "" || form.Get("trigger_settlement") == "1" {
		return s.triggerSettlement(ctx, form.Get("settlement_date"))
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleLocationInsert(ctx khttp.Context) error {
	return s.HandleLockUserReward(ctx)
}

func (s *AdminLegacyService) HandleSettlementList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	batches, total, err := s.admin.ListSettlementBatches(ctx, s.token(ctx), offset, pageSize)
	if err != nil {
		return err
	}
	list := make([]map[string]interface{}, 0, len(batches))
	for _, b := range batches {
		status := b.Status
		if status == biz.SettlementStatusSuccess {
			status = "completed"
		}
		item := map[string]interface{}{
			"id":             b.ID,
			"settlementDate": b.SettlementDate,
			"status":         status,
			"releaseTotal":   b.ReleaseTotal,
			"aixPrice":       biz.FormatAixPrice(b.AixPrice),
			"staticAmount":   b.StaticAmount,
			"mgmtAmount":     b.MgmtAmount,
			"startedAt":      "",
			"finishedAt":     "",
		}
		if !b.StartedAt.IsZero() {
			item["startedAt"] = b.StartedAt.In(jwtpkg.ChinaLocation()).Format("2006-01-02 15:04:05")
		}
		if b.FinishedAt != nil {
			item["finishedAt"] = b.FinishedAt.In(jwtpkg.ChinaLocation()).Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}
	return ctx.Result(200, map[string]interface{}{
		"list":              list,
		"total":             total,
		"count":             total,
		"page":              page,
		"defaultSettleDate": biz.TodaySettlementDate(jwtpkg.NowChina()),
	})
}

func (s *AdminLegacyService) HandleSettlementTrigger(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	date := ""
	_ = ctx.Request().ParseForm()
	date = firstNonEmpty(
		ctx.Request().Form.Get("settlement_date"),
		ctx.Request().Form.Get("date"),
		ctx.Request().URL.Query().Get("date"),
	)
	if date == "" {
		body, _ := io.ReadAll(ctx.Request().Body)
		var m map[string]string
		if json.Unmarshal(body, &m) == nil {
			date = firstNonEmpty(m["settlement_date"], m["date"])
		}
	}
	return s.triggerSettlement(ctx, date)
}

func (s *AdminLegacyService) HandleVipUpdate(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	vipRaw := firstNonEmpty(
		ctx.Request().Form.Get("vip"),
		ctx.Request().Form.Get("level"),
	)
	level := parseMgmtLevel(vipRaw)
	if level < 0 || level > 10 {
		return errors.BadRequest("INVALID_VIP", "级别须为 W0–W10")
	}
	if _, err := s.admin.UpdateUser(ctx, s.token(ctx), &biz.AdminUserUpdate{
		UserID: userID, CommunityLevel: formatMgmtVIP(level), SetCommunityLevel: true,
	}); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleSetZeroAccount(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	enabled := strings.TrimSpace(ctx.Request().Form.Get("enabled")) == "1" ||
		strings.EqualFold(strings.TrimSpace(ctx.Request().Form.Get("enabled")), "true")
	if userID <= 0 {
		return errors.BadRequest("INVALID_USER", "用户无效")
	}
	if _, err := s.admin.UpdateUser(ctx, s.token(ctx), &biz.AdminUserUpdate{
		UserID: userID, SetIsZeroAccount: true, IsZeroAccount: enabled,
	}); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleSetFrozen(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	enabled := strings.TrimSpace(ctx.Request().Form.Get("enabled")) == "1" ||
		strings.EqualFold(strings.TrimSpace(ctx.Request().Form.Get("enabled")), "true")
	if userID <= 0 {
		return errors.BadRequest("INVALID_USER", "用户无效")
	}
	if _, err := s.admin.UpdateUser(ctx, s.token(ctx), &biz.AdminUserUpdate{
		UserID: userID, SetIsFrozen: true, IsFrozen: enabled,
	}); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleSetCommunitySubsidy(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	enabled := strings.TrimSpace(ctx.Request().Form.Get("enabled")) == "1" ||
		strings.EqualFold(strings.TrimSpace(ctx.Request().Form.Get("enabled")), "true")
	if userID <= 0 {
		return errors.BadRequest("INVALID_USER", "用户无效")
	}
	if _, err := s.admin.UpdateUser(ctx, s.token(ctx), &biz.AdminUserUpdate{
		UserID: userID, SetIsCommunitySubsidy: true, IsCommunitySubsidy: enabled,
	}); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleSetInviter(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	inviterAddress := firstNonEmpty(
		ctx.Request().Form.Get("inviter_address"),
		ctx.Request().Form.Get("address"),
		ctx.Request().Form.Get("recommend_address"),
	)
	if err := s.admin.SetUserInviter(ctx, s.token(ctx), userID, inviterAddress); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleChangeAddress(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	userID, _ := strconv.ParseInt(firstNonEmpty(
		ctx.Request().Form.Get("user_id"),
		ctx.Request().Form.Get("userId"),
	), 10, 64)
	newAddress := firstNonEmpty(
		ctx.Request().Form.Get("new_address"),
		ctx.Request().Form.Get("address"),
		ctx.Request().Form.Get("wallet_address"),
	)
	if err := s.admin.ChangeUserAddress(ctx, s.token(ctx), userID, newAddress); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleUpdateGoods(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "not supported in AIX"})
}

func (s *AdminLegacyService) HandleUploadGoods(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "not supported in AIX"})
}

func (s *AdminLegacyService) HandleStubOK(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandleStubRewards(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{"rewards": []interface{}{}, "count": 0})
}

func (s *AdminLegacyService) HandleStubLocations(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{"locations": []interface{}{}, "count": 0})
}

func (s *AdminLegacyService) HandleUserRecommend(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	userID, _ := strconv.ParseInt(firstNonEmpty(q.Get("userId"), q.Get("user_id")), 10, 64)
	address := strings.TrimSpace(firstNonEmpty(q.Get("address"), q.Get("Address")))
	nodes, err := s.admin.ListUserDownline(ctx, s.token(ctx), userID, address)
	if err != nil {
		return err
	}
	users := make([]map[string]interface{}, 0, len(nodes))
	for _, n := range nodes {
		if n == nil || n.User == nil {
			continue
		}
		n.User.SyncCompatFields()
		users = append(users, map[string]interface{}{
			"userId":             n.User.ID,
			"address":            n.User.Address,
			"username":           n.User.Username,
			"createdAt":          formatLegacyTime(n.User.CreatedAt),
			"amount":             n.User.UsdtRecharge,
			"recommendAllAmount": n.RecommendAmount,
			"recommendNum":       n.DirectCount,
			"vip":                formatMgmtVIP(n.User.MgmtLevel),
			"mgmt_level":         n.User.MgmtLevel,
			"team_perf":          n.User.TeamPerf,
			"teamStake":          n.User.TeamPerf,
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func (s *AdminLegacyService) handleOrderList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	addressFilter := strings.TrimSpace(q.Get("address"))
	statusFilter := strings.ToLower(strings.TrimSpace(q.Get("status")))
	if statusFilter == "undefined" || statusFilter == "null" {
		statusFilter = ""
	}
	if statusFilter == "completed" {
		statusFilter = biz.OrderStatusExited
	}
	if strings.EqualFold(addressFilter, "undefined") || strings.EqualFold(addressFilter, "null") {
		addressFilter = ""
	}
	fundSourceFilter := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("fund_source"), q.Get("fundSource"))))
	if fundSourceFilter == "undefined" || fundSourceFilter == "null" {
		fundSourceFilter = ""
	}
	start, end := parseLegacyTimeRange(q)

	orders, err := s.walletRepo.ListAllOrders(ctx)
	if err != nil {
		return err
	}
	filtered := make([]*biz.AdminOrderDetail, 0, len(orders))
	for _, o := range orders {
		if addressFilter != "" && !strings.Contains(strings.ToLower(o.UserAddress), strings.ToLower(addressFilter)) {
			continue
		}
		if statusFilter != "" && strings.ToLower(strings.TrimSpace(o.Order.Status)) != statusFilter {
			continue
		}
		if fundSourceFilter != "" && strings.ToLower(strings.TrimSpace(o.Order.FundSource)) != fundSourceFilter {
			continue
		}
		if !orderWithinTime(o, start, end) {
			continue
		}
		filtered = append(filtered, o)
	}
	stats := sumBuyOrderStats(filtered)
	total := len(filtered)
	pageItems := paginateSlice(filtered, offset, pageSize)
	rewards := make([]map[string]interface{}, 0, len(pageItems))
	for _, o := range pageItems {
		rewards = append(rewards, mapLegacyBuyOrder(o))
	}
	return ctx.Result(200, map[string]interface{}{
		"rewards":  rewards,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
		"stats":    stats,
	})
}

func (s *AdminLegacyService) triggerSettlement(ctx khttp.Context, settlementDate string) error {
	if err := s.admin.TriggerSettlement(ctx, s.token(ctx), settlementDate); err != nil {
		return err
	}
	return ctx.Result(200, map[string]string{"status": "ok", "message": "结算任务已触发"})
}

func mapAnnouncementItem(po data.AnnouncementPO) map[string]interface{} {
	return map[string]interface{}{
		"id":         po.ID,
		"title":      po.Title,
		"content":    po.Content,
		"status":     po.Status,
		"add_time":   po.CreatedTime.Unix(),
		"created_at": formatLegacyTime(po.CreatedTime),
		"updated_at": formatLegacyTime(po.UpdatedTime),
	}
}

func (s *AdminLegacyService) HandleAnnouncementList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	if num := strings.TrimSpace(q.Get("num")); num != "" {
		if v, err := strconv.Atoi(num); err == nil && v > 0 && v <= 1000 {
			pageSize = v
			offset = (page - 1) * pageSize
		}
	}
	db := s.data.DB().WithContext(ctx).Model(&data.AnnouncementPO{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}
	var rows []data.AnnouncementPO
	if err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAnnouncementItem(row))
	}
	return ctx.Result(200, map[string]interface{}{
		"data":  items,
		"count": total,
		"page":  page,
	})
}

func (s *AdminLegacyService) HandleAnnouncementDetail(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	_ = ctx.Request().ParseForm()
	idStr := strings.TrimSpace(ctx.Request().URL.Query().Get("id"))
	if idStr == "" {
		idStr = strings.TrimSpace(ctx.Request().Form.Get("id"))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return errors.BadRequest("INVALID_ID", "公告ID无效")
	}
	var po data.AnnouncementPO
	if err := s.data.DB().WithContext(ctx).First(&po, id).Error; err != nil {
		return errors.NotFound("NOT_FOUND", "公告不存在")
	}
	return ctx.Result(200, map[string]interface{}{
		"data": mapAnnouncementItem(po),
	})
}

func (s *AdminLegacyService) HandleAnnouncementSave(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	title := strings.TrimSpace(ctx.Request().Form.Get("title"))
	content := ctx.Request().Form.Get("content")
	if title == "" {
		return errors.BadRequest("INVALID_TITLE", "标题不能为空")
	}
	if strings.TrimSpace(content) == "" {
		return errors.BadRequest("INVALID_CONTENT", "内容不能为空")
	}
	db := s.data.DB().WithContext(ctx)
	if id > 0 {
		var po data.AnnouncementPO
		if err := db.First(&po, id).Error; err != nil {
			return errors.NotFound("NOT_FOUND", "公告不存在")
		}
		po.Title = title
		po.Content = content
		if err := db.Save(&po).Error; err != nil {
			return err
		}
		return ctx.Result(200, map[string]interface{}{"status": "ok", "data": mapAnnouncementItem(po)})
	}
	po := data.AnnouncementPO{Title: title, Content: content, Status: 1}
	if err := db.Create(&po).Error; err != nil {
		return err
	}
	return ctx.Result(200, map[string]interface{}{"status": "ok", "data": mapAnnouncementItem(po)})
}

func (s *AdminLegacyService) HandleAnnouncementDelete(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	if err := ctx.Request().ParseForm(); err != nil {
		return errors.BadRequest("INVALID_FORM", "请求格式错误")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(ctx.Request().Form.Get("id")), 10, 64)
	if err != nil || id <= 0 {
		return errors.BadRequest("INVALID_ID", "公告ID无效")
	}
	res := s.data.DB().WithContext(ctx).Delete(&data.AnnouncementPO{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.NotFound("NOT_FOUND", "公告不存在")
	}
	return ctx.Result(200, map[string]string{"status": "ok"})
}

func (s *AdminLegacyService) HandlePublicAnnouncementList(ctx khttp.Context) error {
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	db := s.data.DB().WithContext(ctx).Model(&data.AnnouncementPO{}).Where("status = ?", 1)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}
	var rows []data.AnnouncementPO
	if err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]interface{}{
			"id":         row.ID,
			"title":      row.Title,
			"content":    row.Content,
			"created_at": formatLegacyTime(row.CreatedTime),
			"add_time":   row.CreatedTime.Unix(),
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":  items,
		"count": total,
		"page":  page,
	})
}

func (s *AdminLegacyService) HandlePublicAnnouncementDetail(ctx khttp.Context) error {
	id, err := strconv.ParseInt(strings.TrimSpace(ctx.Request().URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		return errors.BadRequest("INVALID_ID", "公告ID无效")
	}
	var po data.AnnouncementPO
	if err := s.data.DB().WithContext(ctx).Where("id = ? AND status = ?", id, 1).First(&po).Error; err != nil {
		return errors.NotFound("NOT_FOUND", "公告不存在")
	}
	return ctx.Result(200, map[string]interface{}{
		"id":         po.ID,
		"title":      po.Title,
		"content":    po.Content,
		"created_at": formatLegacyTime(po.CreatedTime),
		"add_time":   po.CreatedTime.Unix(),
	})
}

func (s *AdminLegacyService) token(ctx khttp.Context) string {
	if t := authmw.ResolveToken(ctx, ""); t != "" {
		return t
	}
	if t := authmw.ParseBearer(ctx.Request().Header.Get("Authorization")); t != "" {
		return t
	}
	if t := strings.TrimSpace(ctx.Request().Header.Get("Access-Token")); t != "" {
		return t
	}
	return strings.TrimSpace(ctx.Request().Header.Get("token"))
}

func (s *AdminLegacyService) buildDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	now := time.Now().In(jwtpkg.ChinaLocation())
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jwtpkg.ChinaLocation())
	yesterdayDate := todayStart.Add(-24 * time.Hour).Format("2006-01-02")

	var totalUserR int64
	var todayUserR int64
	if err := s.data.DB().WithContext(ctx).Model(&data.UserPO{}).Count(&totalUserR).Error; err != nil {
		return nil, err
	}
	if err := s.data.DB().WithContext(ctx).Model(&data.UserPO{}).
		Where("created_time >= ?", todayStart).Count(&todayUserR).Error; err != nil {
		return nil, err
	}

	var totalUser int64
	if err := s.data.DB().WithContext(ctx).Table("orders").
		Select("COUNT(DISTINCT user_id)").Scan(&totalUser).Error; err != nil {
		return nil, err
	}
	var todayUser int64
	if err := s.data.DB().WithContext(ctx).Table("orders").
		Where("created_time >= ?", todayStart).
		Select("COUNT(DISTINCT user_id)").Scan(&todayUser).Error; err != nil {
		return nil, err
	}

	buyTotal, err := s.sumOrderPrincipal(ctx, "", nil)
	if err != nil {
		return nil, err
	}
	todayBuy, err := s.sumOrderPrincipal(ctx, "", &todayStart)
	if err != nil {
		return nil, err
	}

	totalUsdtChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenUSDT, nil)
	if err != nil {
		return nil, err
	}
	todayUsdtChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenUSDT, &todayStart)
	if err != nil {
		return nil, err
	}
	totalWinChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenWIN, nil)
	if err != nil {
		return nil, err
	}
	todayWinChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenWIN, &todayStart)
	if err != nil {
		return nil, err
	}
	totalWinAChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenWINA, nil)
	if err != nil {
		return nil, err
	}
	todayWinAChainRecharge, err := s.sumChainRecharge(ctx, biz.TokenWINA, &todayStart)
	if err != nil {
		return nil, err
	}

	totalRewardReinvest, err := s.sumOrderPrincipal(ctx, biz.PayFromReward, nil)
	if err != nil {
		return nil, err
	}
	todayRewardReinvest, err := s.sumOrderPrincipal(ctx, biz.PayFromReward, &todayStart)
	if err != nil {
		return nil, err
	}

	totalDynamic, err := s.sumDynamicReward(ctx, nil)
	if err != nil {
		return nil, err
	}
	todayDynamic, err := s.sumDynamicReward(ctx, &todayStart)
	if err != nil {
		return nil, err
	}

	totalStaticRelease, err := s.sumStaticRelease(ctx, "")
	if err != nil {
		return nil, err
	}
	yesterdayStaticRelease, err := s.sumStaticRelease(ctx, yesterdayDate)
	if err != nil {
		return nil, err
	}

	totalWinWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenWIN, nil)
	if err != nil {
		return nil, err
	}
	todayWinWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenWIN, &todayStart)
	if err != nil {
		return nil, err
	}
	totalSdtWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenSDT, nil)
	if err != nil {
		return nil, err
	}
	todaySdtWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenSDT, &todayStart)
	if err != nil {
		return nil, err
	}

	totalSdtAsset, err := s.sumUserAsset(ctx, "points")
	if err != nil {
		return nil, err
	}
	totalRewardWallet, err := s.sumUserAsset(ctx, "usdt_reward")
	if err != nil {
		return nil, err
	}
	overflowMgmt, err := s.sumUserAsset(ctx, "overflow_reward")
	if err != nil {
		return nil, err
	}
	overflowDirect, err := s.sumUserAsset(ctx, "overflow_direct")
	if err != nil {
		return nil, err
	}
	totalOverflowWallet := overflowMgmt.Add(overflowDirect)
	winBalance, err := s.sumUserAsset(ctx, "win_balance")
	if err != nil {
		return nil, err
	}
	winRechargeBalance, err := s.sumUserAsset(ctx, "win_recharge_balance")
	if err != nil {
		return nil, err
	}
	totalWinAsset := winBalance.Add(winRechargeBalance)

	// 全网 AIX 持仓总量。注意与 totalSdtAsset 区分：后者统计的是 points 列，不是 AIX 数量。
	totalAixAsset, err := s.sumUserAsset(ctx, "aix_balance")
	if err != nil {
		return nil, err
	}
	// 今日AIX数量：今天 0 点结算任务发放的静态 AIX（结算日=昨日）。
	todayAixAmount, err := s.sumStaticAixAmount(ctx, biz.TodaySettlementDate(time.Now()))
	if err != nil {
		return nil, err
	}
	// 划转WIN数量：交易所（合作方）加款接口累计转入的 WIN，只增不减。
	totalPartnerCreditWin, err := s.sumPartnerCreditWin(ctx)
	if err != nil {
		return nil, err
	}

	totalAdminRecharge, err := s.sumAdminManualRecharge(ctx, nil)
	if err != nil {
		return nil, err
	}
	todayAdminRecharge, err := s.sumAdminManualRecharge(ctx, &todayStart)
	if err != nil {
		return nil, err
	}

	totalZeroAccountReward, err := s.sumUserAsset(ctx, "zero_account_reward_total")
	if err != nil {
		return nil, err
	}
	todayZeroAccountReward, err := s.sumRewardByType(ctx, biz.RewardTypeZeroAccount, &todayStart)
	if err != nil {
		return nil, err
	}
	totalCommunitySubsidyReward, err := s.sumUserAsset(ctx, "community_subsidy_total")
	if err != nil {
		return nil, err
	}
	todayCommunitySubsidyReward, err := s.sumRewardByType(ctx, biz.RewardTypeCommunitySubsidy, &todayStart)
	if err != nil {
		return nil, err
	}

	// 全网可提U / 总提现U / 今日提现U 中的「U」均指：零号账户 + 社区补贴
	totalUsdtWithdrawable := totalZeroAccountReward.Add(totalCommunitySubsidyReward)
	totalUsdtWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenUSDT, nil)
	if err != nil {
		return nil, err
	}
	todayUsdtWithdraw, err := s.sumWithdrawalAmount(ctx, biz.TokenUSDT, &todayStart)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalUserR":             totalUserR,
		"totalUser":              totalUser,
		"todayUserR":             todayUserR,
		"todayUser":              todayUser,
		"buyTotal":               buyTotal.String(),
		"todayBuy":               todayBuy.String(),
		"totalUsdtChainRecharge": totalUsdtChainRecharge.String(),
		"todayUsdtChainRecharge": todayUsdtChainRecharge.String(),
		"totalWinChainRecharge":  totalWinChainRecharge.String(),
		"todayWinChainRecharge":  todayWinChainRecharge.String(),
		"totalWinAChainRecharge": totalWinAChainRecharge.String(),
		"todayWinAChainRecharge": todayWinAChainRecharge.String(),
		"totalRewardReinvest":    totalRewardReinvest.String(),
		"todayRewardReinvest":    todayRewardReinvest.String(),
		"totalDynamic":           totalDynamic.String(),
		"todayDynamic":           todayDynamic.String(),
		"totalStaticRelease":     totalStaticRelease.String(),
		"yesterdayStaticRelease": yesterdayStaticRelease.String(),
		"totalWinWithdraw":       totalWinWithdraw.String(),
		"todayWinWithdraw":       todayWinWithdraw.String(),
		"totalSdtWithdraw":       totalSdtWithdraw.String(),
		"todaySdtWithdraw":       todaySdtWithdraw.String(),
		"totalSdtAsset":          totalSdtAsset.String(),
		"totalWinAsset":          totalWinAsset.String(),
		"totalAixAsset":          totalAixAsset.String(),
		"todayAixAmount":         todayAixAmount.String(),
		"totalPartnerCreditWin":  totalPartnerCreditWin.String(),
		"totalRewardWallet":      totalRewardWallet.String(),
		"totalOverflowWallet":    totalOverflowWallet.String(),
		"totalAdminRecharge":     totalAdminRecharge.String(),
		"todayAdminRecharge":     todayAdminRecharge.String(),
		"totalZeroAccountReward": totalZeroAccountReward.String(),
		"todayZeroAccountReward": todayZeroAccountReward.String(),
		"totalCommunitySubsidyReward": totalCommunitySubsidyReward.String(),
		"todayCommunitySubsidyReward": todayCommunitySubsidyReward.String(),
		"totalUsdtWithdrawable":  totalUsdtWithdrawable.String(),
		"totalUsdtWithdraw":      totalUsdtWithdraw.String(),
		"todayUsdtWithdraw":      todayUsdtWithdraw.String(),
	}, nil
}

func (s *AdminLegacyService) sumActivePrincipalByUser(ctx context.Context) (map[int64]string, error) {
	return s.sumPrincipalByUser(ctx, true)
}

func (s *AdminLegacyService) sumAllPrincipalByUser(ctx context.Context) (map[int64]string, error) {
	return s.sumPrincipalByUser(ctx, false)
}

func (s *AdminLegacyService) sumPrincipalByUser(ctx context.Context, onlyActive bool) (map[int64]string, error) {
	type row struct {
		UserID int64
		Total  decimal.Decimal
	}
	var rows []row
	db := s.data.DB().WithContext(ctx).Table("orders").
		Select("user_id, COALESCE(SUM(principal),0) as total")
	if onlyActive {
		db = db.Where("status = ?", biz.OrderStatusActive)
	}
	if err := db.Group("user_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(rows))
	for _, r := range rows {
		result[r.UserID] = r.Total.String()
	}
	return result, nil
}

// sumOrderReleaseByUser 汇总用户全部订单的可释放总额 / 已释放 / 待释放。
func (s *AdminLegacyService) sumOrderReleaseByUser(ctx context.Context) (totalIncome, released, pending map[int64]string, err error) {
	type row struct {
		UserID      int64
		TotalIncome decimal.Decimal
		Released    decimal.Decimal
	}
	var rows []row
	err = s.data.DB().WithContext(ctx).Table("orders").
		Select("user_id, COALESCE(SUM(exit_cap),0) as total_income, COALESCE(SUM(earned_total),0) as released").
		Group("user_id").Scan(&rows).Error
	if err != nil {
		return nil, nil, nil, err
	}
	totalIncome = make(map[int64]string, len(rows))
	released = make(map[int64]string, len(rows))
	pending = make(map[int64]string, len(rows))
	for _, r := range rows {
		pend := r.TotalIncome.Sub(r.Released)
		if pend.IsNegative() {
			pend = decimal.Zero
		}
		totalIncome[r.UserID] = r.TotalIncome.String()
		released[r.UserID] = r.Released.String()
		pending[r.UserID] = pend.String()
	}
	return totalIncome, released, pending, nil
}

func mapLegacyBuyOrder(o *biz.AdminOrderDetail) map[string]interface{} {
	principal, err := decimal.NewFromString(o.Order.Principal)
	if err != nil {
		principal = decimal.Zero
	}
	exitCap, err := decimal.NewFromString(o.Order.ExitCap)
	if err != nil {
		exitCap = biz.CalcExitCap(principal)
	}
	earned, err := decimal.NewFromString(o.Order.EarnedTotal)
	if err != nil {
		earned = decimal.Zero
	}
	remain := exitCap.Sub(earned)
	if remain.IsNegative() {
		remain = decimal.Zero
	}
	exitMul := decimal.NewFromFloat(biz.ExitMultiplier)
	if !exitMul.IsPositive() {
		exitMul = decimal.NewFromInt(4)
	}
	status := o.Order.Status
	if status == "completed" {
		status = biz.OrderStatusExited
	}
	return map[string]interface{}{
		"id":          o.Order.ID,
		"address":     o.UserAddress,
		"amount":      principal.String(),
		"exitAmount":  exitMul.String(),
		"money":       exitCap.String(),
		"amountGet":   earned.String(),
		"amountLast":  remain.String(),
		"points":      o.Order.Points,
		"fund_source":   o.Order.FundSource,
		"from_recharge": o.Order.FromRecharge,
		"from_win":      o.Order.FromWin,
		"from_win_a":    o.Order.FromWinA,
		"status":        status,
		"createdAt":   formatLegacyTime(o.Order.CreatedAt),
		"one":         o.Order.FundSource,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func zeroIfEmpty(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}
	return v
}

func formatMgmtVIP(level int32) string {
	if level < 0 {
		level = 0
	}
	if level > 10 {
		level = 10
	}
	return "A" + strconv.Itoa(int(level))
}

func parseMgmtLevel(raw string) int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.ToUpper(raw)
	raw = strings.TrimPrefix(raw, "A")
	raw = strings.TrimPrefix(raw, "W")
	raw = strings.TrimPrefix(raw, "V")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	if n > 10 {
		return 10
	}
	return int32(n)
}

func formatLegacyTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(jwtpkg.ChinaLocation()).Format("2006-01-02 15:04:05")
}

func formatLegacyTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return formatLegacyTime(*t)
}

func parsePage(q url.Values) (page, pageSize, offset int) {
	page = 1
	pageSize = 20
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := q.Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 1000 {
			pageSize = v
		}
	} else if ps := q.Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 1000 {
			pageSize = v
		}
	}
	offset = (page - 1) * pageSize
	return page, pageSize, offset
}

func paginateSlice[T any](items []T, offset, limit int) []T {
	if offset >= len(items) {
		return nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func legacyConfigValue(cfg *conf.SystemConfigSnapshot, walletCfg *conf.WalletConfig, id int) string {
	if cfg != nil {
		conf.NormalizeBusinessDefaults(cfg)
	}
	switch id {
	case 1:
		if cfg != nil && cfg.MinSubscribe != "" {
			return cfg.MinSubscribe
		}
		if walletCfg != nil {
			return walletCfg.MinSubscribe
		}
		return conf.DefaultMinSubscribe
	case 2:
		if cfg != nil {
			return strconv.FormatFloat(cfg.StaticRate, 'f', -1, 64)
		}
		return strconv.FormatFloat(conf.DefaultStaticRate, 'f', -1, 64)
	case 3:
		if cfg != nil {
			return strconv.FormatFloat(cfg.ExitMultiplier, 'f', -1, 64)
		}
		return strconv.FormatFloat(conf.DefaultExitMultiplier, 'f', -1, 64)
	case 4:
		if cfg != nil {
			return strconv.FormatFloat(cfg.DirectRate, 'f', -1, 64)
		}
		return strconv.FormatFloat(conf.DefaultDirectRate, 'f', -1, 64)
	case 5:
		if cfg != nil {
			if len(cfg.DepositAddresses) > 0 {
				return strings.Join(cfg.DepositAddresses, ",")
			}
			if cfg.DepositAddress != "" {
				return cfg.DepositAddress
			}
		}
		if walletCfg != nil {
			addrs := walletCfg.GetDepositAddresses()
			if len(addrs) > 0 {
				return strings.Join(addrs, ",")
			}
		}
		return ""
	case 6:
		if cfg != nil && cfg.UsdtContract != "" {
			return cfg.UsdtContract
		}
		if walletCfg != nil {
			return walletCfg.UsdtContract
		}
		return ""
	case 7:
		if cfg != nil {
			return biz.FormatAixPriceDecimal(decimal.NewFromFloat(cfg.AixPriceInitial))
		}
		return biz.FormatAixPriceDecimal(decimal.NewFromFloat(conf.DefaultAixPrice))
	case 8:
		if cfg != nil {
			return strconv.FormatFloat(cfg.WinPrice, 'f', -1, 64)
		}
		return strconv.FormatFloat(conf.DefaultWinPrice, 'f', -1, 64)
	case 36:
		if cfg != nil {
			return strconv.FormatFloat(cfg.WinPrice, 'f', -1, 64) // WIN-A = WIN
		}
		return strconv.FormatFloat(conf.DefaultWinPrice, 'f', -1, 64)
	case 9:
		if cfg != nil {
			return strconv.FormatFloat(cfg.ExchangeFeeRate*100, 'f', -1, 64)
		}
		return strconv.FormatFloat(conf.DefaultExchangeFeeRate*100, 'f', -1, 64)
	case 31:
		if cfg != nil && strings.TrimSpace(cfg.MinUsdtRecharge) != "" {
			return cfg.MinUsdtRecharge
		}
		return conf.DefaultMinUsdtRecharge
	case 32:
		if cfg != nil && strings.TrimSpace(cfg.MinWinRecharge) != "" {
			return cfg.MinWinRecharge
		}
		return conf.DefaultMinWinRecharge
	case 37:
		if cfg != nil && strings.TrimSpace(cfg.MinWinARecharge) != "" {
			return cfg.MinWinARecharge
		}
		return conf.DefaultMinWinARecharge
	case 33:
		if cfg != nil && strings.TrimSpace(cfg.WinWithdrawReviewThreshold) != "" {
			return cfg.WinWithdrawReviewThreshold
		}
		return "0"
	case 34:
		if cfg != nil && strings.TrimSpace(cfg.SdtWithdrawReviewThreshold) != "" {
			return cfg.SdtWithdrawReviewThreshold
		}
		return "0"
	case 35:
		if cfg != nil && strings.TrimSpace(cfg.UsdtWithdrawReviewThreshold) != "" {
			return cfg.UsdtWithdrawReviewThreshold
		}
		return "0"
	case 38:
		if cfg != nil && strings.TrimSpace(cfg.PartnerMinAmount) != "" {
			return cfg.PartnerMinAmount
		}
		return conf.DefaultPartnerMinAmount
	case 39:
		if cfg != nil && strings.TrimSpace(cfg.PartnerMaxAmount) != "" {
			return cfg.PartnerMaxAmount
		}
		return conf.DefaultPartnerMaxAmount
	case 40:
		if cfg != nil && strings.TrimSpace(cfg.PartnerDailyLimit) != "" {
			return cfg.PartnerDailyLimit
		}
		return conf.DefaultPartnerDailyLimit
	case 41:
		if cfg != nil && strings.TrimSpace(cfg.ExchangeReviewThresholdPercent) != "" {
			return cfg.ExchangeReviewThresholdPercent
		}
		return conf.DefaultExchangeReviewThresholdPercent
	case 11, 12, 13, 14, 15, 16, 17, 18, 19, 20:
		idx := id - 11
		rates := conf.DefaultMgmtRates()
		if cfg != nil && len(cfg.MgmtRates) == 10 {
			rates = cfg.MgmtRates
		}
		return strconv.FormatFloat(rates[idx], 'f', -1, 64)
	case 21, 22, 23, 24, 25, 26, 27, 28, 29, 30:
		idx := id - 21
		thresholds := conf.DefaultMgmtThresholds()
		if cfg != nil && len(cfg.MgmtThresholds) == 10 {
			thresholds = cfg.MgmtThresholds
		}
		return strconv.FormatFloat(thresholds[idx], 'f', -1, 64)
	default:
		return ""
	}
}

func parseFloatOr(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}

func applyLegacyConfigUpdate(snapshot *conf.SystemConfigSnapshot, walletCfg *conf.WalletConfig, id int, value string) error {
	if snapshot == nil {
		return errors.BadRequest("INVALID_CONFIG", "配置不可用")
	}
	conf.NormalizeBusinessDefaults(snapshot)
	value = strings.TrimSpace(value)
	switch id {
	case 1:
		snapshot.MinSubscribe = value
	case 2:
		rate, err := strconv.ParseFloat(value, 64)
		if err != nil || rate <= 0 {
			return errors.BadRequest("INVALID_VALUE", "静态利率格式错误")
		}
		snapshot.StaticRate = rate
	case 3:
		mul, err := strconv.ParseFloat(value, 64)
		if err != nil || mul <= 0 {
			return errors.BadRequest("INVALID_VALUE", "出局倍数格式错误")
		}
		snapshot.ExitMultiplier = mul
	case 4:
		rate, err := strconv.ParseFloat(value, 64)
		if err != nil || rate < 0 || rate > 1 {
			return errors.BadRequest("INVALID_VALUE", "直推比例须为 0~1（如 0.5 表示 50%）")
		}
		snapshot.DirectRate = rate
	case 5:
		tmp := &conf.WalletConfig{DepositAddress: value}
		addrs := tmp.GetDepositAddresses()
		snapshot.DepositAddresses = addrs
		if len(addrs) > 0 {
			snapshot.DepositAddress = addrs[0]
		} else {
			snapshot.DepositAddress = value
		}
		_ = walletCfg
	case 6:
		snapshot.UsdtContract = value
	case 7:
		price, err := strconv.ParseFloat(value, 64)
		if err != nil || price <= 0 {
			return errors.BadRequest("INVALID_VALUE", "AIX价格格式错误")
		}
		snapshot.AixPriceInitial = price
	case 8:
		price, err := strconv.ParseFloat(value, 64)
		if err != nil || price <= 0 {
			return errors.BadRequest("INVALID_VALUE", "WIN价格格式错误")
		}
		snapshot.WinPrice = price
		snapshot.WinAPrice = price // WIN-A 与 WIN 同价
	case 36:
		// WIN-A 价格不允许单独配置，始终跟随 WIN
		snapshot.WinAPrice = snapshot.WinPrice
		if snapshot.WinAPrice <= 0 {
			snapshot.WinAPrice = conf.DefaultWinPrice
		}
	case 9:
		feePercent, err := strconv.ParseFloat(value, 64)
		if err != nil || feePercent < 0 || feePercent >= 100 {
			return errors.BadRequest("INVALID_VALUE", "手续费率须为 0~100 的百分数（如 5 表示 5%）")
		}
		snapshot.ExchangeFeeRate = feePercent / 100.0
	case 31:
		minAmt, err := strconv.ParseFloat(value, 64)
		if err != nil || minAmt < conf.FloorMinRechargeAmount {
			return errors.BadRequest("INVALID_VALUE", fmt.Sprintf("USDT充值最小值必须 ≥ %g", conf.FloorMinRechargeAmount))
		}
		snapshot.MinUsdtRecharge = strconv.FormatFloat(minAmt, 'f', -1, 64)
	case 32:
		minAmt, err := strconv.ParseFloat(value, 64)
		if err != nil || minAmt < conf.FloorMinRechargeAmount {
			return errors.BadRequest("INVALID_VALUE", fmt.Sprintf("WIN充值最小值必须 ≥ %g", conf.FloorMinRechargeAmount))
		}
		snapshot.MinWinRecharge = strconv.FormatFloat(minAmt, 'f', -1, 64)
	case 37:
		minAmt, err := strconv.ParseFloat(value, 64)
		if err != nil || minAmt < conf.FloorMinRechargeAmount {
			return errors.BadRequest("INVALID_VALUE", fmt.Sprintf("WIN-A充值最小值必须 ≥ %g", conf.FloorMinRechargeAmount))
		}
		snapshot.MinWinARecharge = strconv.FormatFloat(minAmt, 'f', -1, 64)
	case 33, 34, 35:
		threshold, err := strconv.ParseFloat(value, 64)
		if err != nil || threshold < 0 {
			return errors.BadRequest("INVALID_VALUE", "提现审核阈值须为 ≥0 的数字（0 表示不审核；超过该金额需人工审核）")
		}
		v := strconv.FormatFloat(threshold, 'f', -1, 64)
		switch id {
		case 33:
			snapshot.WinWithdrawReviewThreshold = v
		case 34:
			snapshot.SdtWithdrawReviewThreshold = v
		case 35:
			snapshot.UsdtWithdrawReviewThreshold = v
		}
	case 38, 39, 40:
		amt, err := strconv.ParseFloat(value, 64)
		if err != nil || amt <= 0 {
			return errors.BadRequest("INVALID_VALUE", "交易所划转限额必须为大于 0 的数字")
		}
		// 三者写入同一份快照，任一项改动都要重新校验 下限 ≤ 上限 ≤ 单日上限，
		// 否则会配出「任何金额都被拒」的死区。
		v := strconv.FormatFloat(amt, 'f', -1, 64)
		switch id {
		case 38:
			snapshot.PartnerMinAmount = v
		case 39:
			snapshot.PartnerMaxAmount = v
		case 40:
			snapshot.PartnerDailyLimit = v
		}
		min := parseFloatOr(snapshot.PartnerMinAmount, 0)
		max := parseFloatOr(snapshot.PartnerMaxAmount, 0)
		daily := parseFloatOr(snapshot.PartnerDailyLimit, 0)
		if min > max {
			return errors.BadRequest("INVALID_VALUE", "交易所划转单笔下限不能大于单笔上限")
		}
		if max > daily {
			return errors.BadRequest("INVALID_VALUE", "交易所划转单笔上限不能大于单日上限")
		}
	case 41:
		pct, err := strconv.ParseFloat(value, 64)
		if err != nil || pct < 0 {
			return errors.BadRequest("INVALID_VALUE", "AIX兑换审核阈值须为 ≥0 的数字（百分数，如 40 表示 40%；100 表示兑完今日AIX后才审）")
		}
		snapshot.ExchangeReviewThresholdPercent = strconv.FormatFloat(pct, 'f', -1, 64)
	case 11, 12, 13, 14, 15, 16, 17, 18, 19, 20:
		rate, err := strconv.ParseFloat(value, 64)
		if err != nil || rate < 0 {
			return errors.BadRequest("INVALID_VALUE", "收益系数格式错误（如 0.2 表示 20%）")
		}
		if len(snapshot.MgmtRates) != 10 {
			snapshot.MgmtRates = conf.DefaultMgmtRates()
		}
		snapshot.MgmtRates[id-11] = rate
	case 21, 22, 23, 24, 25, 26, 27, 28, 29, 30:
		threshold, err := strconv.ParseFloat(value, 64)
		if err != nil || threshold <= 0 {
			return errors.BadRequest("INVALID_VALUE", "晋级业绩金额必须为大于 0 的数字")
		}
		if len(snapshot.MgmtThresholds) != 10 {
			snapshot.MgmtThresholds = conf.DefaultMgmtThresholds()
		}
		idx := id - 21
		if idx > 0 && threshold <= snapshot.MgmtThresholds[idx-1] {
			return errors.BadRequest("INVALID_VALUE", "晋级业绩金额必须高于前一级")
		}
		if idx < len(snapshot.MgmtThresholds)-1 && threshold >= snapshot.MgmtThresholds[idx+1] {
			return errors.BadRequest("INVALID_VALUE", "晋级业绩金额必须低于后一级")
		}
		snapshot.MgmtThresholds[idx] = threshold
	default:
		return errors.BadRequest("INVALID_ID", "未知配置项")
	}
	return nil
}

func subAccountConfigValue(cfg *conf.SystemConfigSnapshot, slot int, kind string) string {
	ensureSnapshotSubAccounts(cfg)
	if cfg == nil || slot < 0 || slot >= len(cfg.AdminSubAccounts) {
		return ""
	}
	item := cfg.AdminSubAccounts[slot]
	switch kind {
	case "password":
		return item.Password
	case "modules":
		return strings.Join(item.Modules, ",")
	default:
		return ""
	}
}

func applySubAccountConfigUpdate(snapshot *conf.SystemConfigSnapshot, id int, value string) error {
	if snapshot == nil {
		return errors.BadRequest("INVALID_CONFIG", "配置不可用")
	}
	var def *struct {
		ID       int
		Slot     int
		Kind     string
		NameTmpl string
	}
	for i := range subAccountConfigDefs {
		if subAccountConfigDefs[i].ID == id {
			def = &subAccountConfigDefs[i]
			break
		}
	}
	if def == nil {
		return errors.BadRequest("INVALID_ID", "未知子账户配置项")
	}
	ensureSnapshotSubAccounts(snapshot)
	if def.Slot < 0 || def.Slot >= len(snapshot.AdminSubAccounts) {
		return errors.BadRequest("INVALID_CONFIG", "子账户槽位无效")
	}
	value = strings.TrimSpace(value)
	switch def.Kind {
	case "password":
		if value == "" {
			return errors.BadRequest("INVALID_VALUE", "密码不能为空")
		}
		snapshot.AdminSubAccounts[def.Slot].Password = value
	case "modules":
		modules, err := parseSubAccountModulesInput(value)
		if err != nil {
			return errors.BadRequest("INVALID_VALUE", err.Error())
		}
		snapshot.AdminSubAccounts[def.Slot].Modules = modules
	default:
		return errors.BadRequest("INVALID_ID", "未知子账户配置项")
	}
	return nil
}
