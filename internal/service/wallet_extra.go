package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/biz"
	authmw "backend/internal/middleware"
	"backend/internal/pkg/token"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
)

// RegisterWalletExtraRoutes mounts AIX-specific routes without regenerating protos.
func RegisterWalletExtraRoutes(srv *khttp.Server, wallet *WalletService) {
	r := srv.Route("/")
	r.POST("/v1/wallet/subscribe-aix", wallet.HandleSubscribeAIX)
	r.GET("/v1/wallet/subscribe-orders", wallet.HandleSubscribeOrders)
	r.POST("/v1/wallet/transfer", wallet.HandleTransfer)
	r.POST("/v1/wallet/recharge-to-reward", wallet.HandleRechargeToReward)
	r.GET("/v1/wallet/transfer-records/self", wallet.HandleSelfTransferRecords)
	r.GET("/v1/wallet/transfer-records/lineal", wallet.HandleLinealTransferRecords)
	r.POST("/v1/wallet/withdraw-aix", wallet.HandleWithdrawAIX)
	r.POST("/v1/wallet/exchange-aix-to-win", wallet.HandleExchangeAixToWin)
	r.POST("/v1/wallet/withdraw-sdt", wallet.HandleWithdrawSDT)
	r.POST("/v1/wallet/withdraw-usdt", wallet.HandleWithdrawUSDT)
	r.GET("/v1/wallet/withdraw-records", wallet.HandleWithdrawRecords)
	r.GET("/v1/wallet/exchange-records", wallet.HandleExchangeRecords)
	r.GET("/v1/wallet/aix-price", wallet.HandleAixPrice)
	r.GET("/v1/wallet/rewards", wallet.HandleRewards)
	r.GET("/v1/wallet/management-rewards", wallet.HandleManagementRewards)
	r.GET("/v1/wallet/aix-profile", wallet.HandleAixProfile)
	r.GET("/v1/wallet/downline-usdt-recharges", wallet.HandleDownlineUSDTRecharges)
	r.GET("/v1/wallet/points-records", wallet.HandlePointsRecords)
	r.POST("/v1/wallet/recharge-win", wallet.HandleCreateWinRecharge)
	r.POST("/v1/wallet/recharge-win/confirm", wallet.HandleConfirmWinRecharge)
	r.GET("/v1/wallet/recharges-win", wallet.HandleListWinRecharges)
}

func transferRecordPagination(ctx khttp.Context) (page, pageSize int, err error) {
	page = 1
	pageSize = 10
	query := ctx.Request().URL.Query()
	if raw := strings.TrimSpace(query.Get("page")); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page <= 0 {
			return 0, 0, fmt.Errorf("page 必须为大于0的整数")
		}
	}
	if raw := strings.TrimSpace(query.Get("page_size")); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize <= 0 || pageSize > 100 {
			return 0, 0, fmt.Errorf("page_size 必须为1到100的整数")
		}
	}
	return page, pageSize, nil
}

func pageBounds(total, page, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

// tokenFromRequest 自定义 Route 可能不走 Middleware，需直接从 HTTP 头取 Bearer。
func tokenFromRequest(ctx khttp.Context, fallback string) string {
	if t := resolveToken(ctx, fallback); t != "" {
		return t
	}
	if t := authmw.TokenFromContext(ctx); t != "" {
		return t
	}
	req := ctx.Request()
	if req == nil {
		return strings.TrimSpace(fallback)
	}
	if t := authmw.ParseBearer(req.Header.Get("Authorization")); t != "" {
		return t
	}
	if t := strings.TrimSpace(req.Header.Get("Access-Token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(req.Header.Get("token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(req.URL.Query().Get("token")); t != "" {
		return t
	}
	return strings.TrimSpace(fallback)
}

type subscribeAIXReq struct {
	Token   string `json:"token"`
	Amount  string `json:"amount"`
	PayFrom string `json:"pay_from"`
	// 已废弃：混合报单
	PayFrom2 string `json:"pay_from_2"`
	Amount1  string `json:"amount_1"`
}

func (s *WalletService) HandleSubscribeAIX(ctx khttp.Context) error {
	var req subscribeAIXReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	if strings.TrimSpace(req.PayFrom2) != "" || strings.TrimSpace(req.Amount1) != "" {
		return ctx.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"reason":  "MIX_SUBSCRIBE_DISABLED",
			"message": "混合报单已关闭",
		})
	}
	token := tokenFromRequest(ctx, req.Token)
	order, bal, err := s.uc.SubscribeAIX(ctx, token, req.Amount, req.PayFrom)
	if err != nil {
		return err
	}
	resp := map[string]any{
		"order_id":     order.ID,
		"principal":    order.Principal,
		"exit_cap":     order.ExitCap,
		"fund_source":  order.FundSource,
		"direct_base":  order.DirectBase,
		"status":       order.Status,
		"balance":      bal,
		"total_amount": order.Principal,
		"points":       order.Points, // 本单获得积分（= 认购金额）
	}
	if user, _, _, _, _, _, _, _, _, uerr := s.uc.GetBalance(ctx, token); uerr == nil && user != nil {
		resp["points_balance"] = user.Points
		resp["points_all"] = user.PointsAll
	}
	if order.FromRecharge != "" && order.FromRecharge != "0" {
		resp["from_recharge"] = order.FromRecharge
	}
	if order.FromWin != "" && order.FromWin != "0" {
		resp["from_win"] = order.FromWin
		resp["win_price"] = order.WinPrice
		if order.WinPrice == "" {
			resp["win_price"] = fmt.Sprintf("%v", biz.GetWinPrice())
		}
	}
	if order.FromWinA != "" && order.FromWinA != "0" {
		resp["from_win_a"] = order.FromWinA
		resp["win_a_price"] = order.WinAPrice
		if order.WinAPrice == "" {
			resp["win_a_price"] = fmt.Sprintf("%v", biz.GetWinAPrice())
		}
	}
	return ctx.JSON(http.StatusOK, resp)
}

// HandleSubscribeOrders 用户端认购记录（含 from_recharge / from_win / from_win_a）
func (s *WalletService) HandleSubscribeOrders(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	orders, err := s.uc.ListOrders(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(orders))
	for _, o := range orders {
		items = append(items, map[string]any{
			"id":              o.ID,
			"principal":       o.Principal,
			"total_amount":    o.Principal,
			"amount":          o.Principal,
			"exit_cap":        o.ExitCap,
			"exit_target":     o.ExitCap,
			"exit_multiplier": o.ExitMultiplier,
			"released_amount": o.EarnedTotal,
			"earned_total":    o.EarnedTotal,
			"direct_base":     o.DirectBase,
			"from_recharge":   o.FromRecharge,
			"from_reward":     o.FromReward,
			"from_win":        o.FromWin,
			"from_win_a":      o.FromWinA,
			"points":          o.Points,
			"win_price":       o.WinPrice,
			"win_a_price":     o.WinAPrice,
			"fund_source":     o.FundSource,
			"product_name":    o.FundSource,
			"status":          o.Status,
			"created_at":      o.CreatedTime.Unix(),
			"createdAt":       o.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"orders": items, "count": len(items)})
}

type transferReq struct {
	Token     string `json:"token"`
	ToAddress string `json:"to_address"`
	Asset     string `json:"asset"`
	Amount    string `json:"amount"`
	PayFrom   string `json:"pay_from"`
}

func (s *WalletService) HandleTransfer(ctx khttp.Context) error {
	var req transferReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	t, err := s.uc.Transfer(ctx, token, req.ToAddress, req.Asset, req.Amount, req.PayFrom)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"id": t.ID, "from_user_id": t.FromUserID, "to_user_id": t.ToUserID,
		"asset": t.Asset, "amount": t.Amount, "pay_from": t.PayFrom,
		"to_credit_reward": t.ToCreditReward, "to_credit_aix": t.ToCreditAix,
	})
}

func (s *WalletService) HandleRechargeToReward(ctx khttp.Context) error {
	var req struct {
		Token  string `json:"token"`
		Amount string `json:"amount"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	rechargeBal, rewardBal, err := s.uc.MoveRechargeToReward(ctx, token, req.Amount)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"usdt_recharge": rechargeBal,
		"usdt_reward":   rewardBal,
	})
}

func (s *WalletService) HandleSelfTransferRecords(ctx khttp.Context) error {
	page, pageSize, err := transferRecordPagination(ctx)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{
			"code": 400, "reason": "INVALID_PAGE", "message": err.Error(),
		})
	}
	records, err := s.uc.ListSelfTransferRecords(ctx, tokenFromRequest(ctx, ""))
	if err != nil {
		return err
	}
	start, end := pageBounds(len(records), page, pageSize)
	list := make([]map[string]any, 0, end-start)
	for _, record := range records[start:end] {
		list = append(list, map[string]any{
			"id": record.ID, "asset": record.Asset, "amount": record.Amount,
			"from_wallet": record.FromWallet, "to_wallet": record.ToWallet,
			"created_at": record.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"page": page, "page_size": pageSize, "total": len(records), "list": list,
	})
}

func (s *WalletService) HandleLinealTransferRecords(ctx khttp.Context) error {
	page, pageSize, err := transferRecordPagination(ctx)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{
			"code": 400, "reason": "INVALID_PAGE", "message": err.Error(),
		})
	}
	direction := strings.ToLower(strings.TrimSpace(ctx.Request().URL.Query().Get("direction")))
	records, err := s.uc.ListLinealTransferRecords(ctx, tokenFromRequest(ctx, ""), direction)
	if err != nil {
		return err
	}
	start, end := pageBounds(len(records), page, pageSize)
	list := make([]map[string]any, 0, end-start)
	for _, record := range records[start:end] {
		list = append(list, map[string]any{
			"id": record.ID, "direction": record.Direction, "relationship": record.Relationship,
			"counterparty_user_id": record.CounterpartyUserID,
			"counterparty_address": record.CounterpartyAddress,
			"asset":                record.Asset, "amount": record.Amount,
			"from_wallet": record.FromWallet, "to_wallet": record.ToWallet,
			"created_at": record.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"page": page, "page_size": pageSize, "total": len(records), "list": list,
	})
}

func (s *WalletService) HandleWithdrawAIX(ctx khttp.Context) error {
	var req struct {
		Token     string `json:"token"`
		Amount    string `json:"amount"`
		ToAddress string `json:"to_address"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	w, left, err := s.uc.CreateAixWithdraw(ctx, token, req.Amount, req.ToAddress)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"withdraw_id":  w.ID,
		"asset":        "AIX",
		"amount":       w.Amount,
		"to_address":   w.ToAddress,
		"status":       w.Status,
		"tx_hash":      w.TxHash, // 合约未就绪前为空
		"aix_balance":  left,
		"aix_contract": "", // TODO: 待配置 AIX 代币合约
	})
}

func (s *WalletService) HandleAixPrice(ctx khttp.Context) error {
	date := strings.TrimSpace(ctx.Request().URL.Query().Get("date"))
	price, err := s.uc.GetAixPrice(ctx, date)
	if err != nil {
		return err
	}
	if date == "" {
		date = token.NowChina().Format("2006-01-02")
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"price": biz.FormatAixPrice(price), "date": date,
		"aix_contract": "", // TODO: 待配置 AIX 代币合约
	})
}

func (s *WalletService) HandleRewards(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	logs, err := s.uc.ListRewardLogs(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		items = append(items, map[string]any{
			"id": l.ID, "type": l.Type, "asset": l.Asset, "amount": l.Amount,
			"base_amount": l.BaseAmount, "rate": l.Rate, "exit_applied": l.ExitApplied,
			"settlement_date": l.SettlementDate, "created_time": l.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"rewards": items})
}

func (s *WalletService) HandleManagementRewards(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	rewards, err := s.uc.ListMgmtRewards(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(rewards))
	for _, reward := range rewards {
		sourceAddress := ""
		if address, err := s.uc.FindUserAddress(ctx, reward.FromUserID); err == nil {
			sourceAddress = address
		}
		items = append(items, map[string]any{
			"id": reward.ID, "source_user_id": reward.FromUserID,
			"source_address": sourceAddress, "source_order_id": reward.SourceOrderID,
			"base_amount": reward.BaseAmount, "rate": reward.Rate,
			"total_amount": reward.TotalAmount, "released_amount": reward.ReleasedAmount,
			"pending_amount": reward.PendingAmount, "created_time": reward.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"rewards": items})
}

// HandleAixProfile 用户端资产总览（含静态总收益）
func (s *WalletService) HandleAixProfile(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	user, summary, recharge, reward, pending, aix, unexited, nextReleaseAt, serverTime, err := s.uc.GetBalance(ctx, token)
	if err != nil {
		return err
	}
	totalNodes := int32(0)
	if summary != nil {
		totalNodes = summary.TotalNodes
	}
	staticTotal := "0"
	if user != nil && user.StaticUsdtTotal != "" {
		staticTotal = user.StaticUsdtTotal
	}
	mgmtLevel := int32(0)
	largeArea := "0"
	smallArea := "0"
	teamPerf := "0"
	if user != nil {
		mgmtLevel = user.MgmtLevel
		if user.LargeAreaPerf != "" {
			largeArea = user.LargeAreaPerf
		}
		if user.SmallAreaPerf != "" {
			smallArea = user.SmallAreaPerf
		}
		if user.TeamPerf != "" {
			teamPerf = user.TeamPerf
		}
	}
	mgmtSummary, err := s.uc.GetMgmtRewardSummary(ctx, token)
	if err != nil {
		return err
	}
	directRewardTotal := "0"
	if directTotal, directErr := s.uc.GetDirectRewardTotal(ctx, token); directErr == nil && strings.TrimSpace(directTotal) != "" {
		directRewardTotal = directTotal
	}
	teamActiveSubscribe := "0"
	if user != nil {
		if v, sumErr := s.uc.GetTeamActiveSubscribePrincipal(ctx, token); sumErr == nil && strings.TrimSpace(v) != "" {
			teamActiveSubscribe = v
		}
	}

	// AIX 现价优先取当日 aix_prices，统一格式化为 15 位小数
	aixPriceStr := ""
	if priceStr, priceErr := s.uc.GetAixPrice(ctx, ""); priceErr == nil && strings.TrimSpace(priceStr) != "" {
		aixPriceStr = priceStr
	} else {
		aixPriceStr = biz.FormatAixPriceDecimal(decimal.NewFromFloat(biz.AixPriceInitial))
	}
	aixPriceDec, _ := decimal.NewFromString(aixPriceStr)
	winPrice := biz.GetWinPrice()
	aixToWinRate := "0"
	if winPrice > 0 && aixPriceDec.IsPositive() {
		// 1 AIX 可兑多少 WIN（毛量，未扣手续费）= aix_price / win_price
		aixToWinRate = aixPriceDec.Div(decimal.NewFromFloat(winPrice)).StringFixed(biz.AixPriceDecimals)
	}

	overflow := "0"
	isZeroAccount := false
	isCommunitySubsidy := false
	zeroAccountReward := "0"
	communitySubsidyReward := "0"
	if user != nil {
		overflow = user.OverflowTotal()
		if overflow == "" {
			overflow = "0"
		}
		isZeroAccount = user.IsZeroAccount
		isCommunitySubsidy = user.IsCommunitySubsidy
		if user.ZeroAccountRewardTotal != "" {
			zeroAccountReward = user.ZeroAccountRewardTotal
		}
		if user.CommunitySubsidyTotal != "" {
			communitySubsidyReward = user.CommunitySubsidyTotal
		}
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"address":              user.Address,
		"username":             user.Username,
		"usdt_recharge":        recharge,
		"usdt_reward":          reward,
		"aix_balance":          aix,
		"win_balance":            user.WinBalance,
		"win_recharge_balance":   user.WinRechargeBalance,
		"win_a_recharge_balance": user.WinARechargeBalance,
		"usdt_withdrawable":      zeroIfEmpty(user.UsdtWithdrawable),
		"pending_mgmt_reward":  overflow, // 兼容旧字段，值为溢出奖励
		"overflow_reward":      overflow,
		"is_zero_account":          isZeroAccount,
		"is_community_subsidy":     isCommunitySubsidy,
		"community_subsidy_rate":   user.CommunitySubsidyRate,
		"zero_account_reward_total": zeroAccountReward,
		"community_subsidy_total":   communitySubsidyReward,
		"points":               zeroIfEmpty(user.Points),
		"points_all":           zeroIfEmpty(user.PointsAll),
		"static_usdt_total":    staticTotal,
		"pending_amount":       pending,
		"unexited_amount":      unexited,
		"total_nodes":          totalNodes,
		"mgmt_level":           mgmtLevel,
		"mgmt_reward_released": mgmtSummary.Released,
		"mgmt_reward_pending":  mgmtSummary.Pending,
		"mgmt_reward_total":    mgmtSummary.Total,
		"direct_reward_total":  directRewardTotal,
		"large_area_perf":      largeArea,
		"small_area_perf":      smallArea,
		"team_perf":            teamPerf,
		"team_active_subscribe_principal": teamActiveSubscribe,
		"server_time":          serverTime,
		"next_release_at":      nextReleaseAt,
		"aix_price":            aixPriceStr,
		"win_price":            winPrice,
		"aix_to_win_rate":      aixToWinRate,
		"exchange_fee_rate":    biz.GetExchangeFeeRate(),
		"aix_contract":         "", // TODO
		"win_contract":           s.uc.WinContract(),
		"win_a_recharge_enabled": false,
		"min_usdt_recharge":    s.uc.MinUsdtRecharge(),
		"min_win_recharge":     s.uc.MinWinRecharge(),
	})
}

// HandleDownlineUSDTRecharges 当前用户所有下级 USDT 充值记录。
func (s *WalletService) HandleDownlineUSDTRecharges(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	page, pageSize, err := transferRecordPagination(ctx)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": err.Error()})
	}
	records, total, err := s.uc.ListDownlineUSDTRecharges(ctx, token, page, pageSize)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		if rec == nil {
			continue
		}
		createdAt := rec.CreatedTime.Unix()
		if rec.ConfirmedTime != nil {
			createdAt = rec.ConfirmedTime.Unix()
		}
		items = append(items, map[string]any{
			"id":         rec.ID,
			"address":    rec.Address,
			"amount":     rec.Amount,
			"asset":      rec.Asset,
			"status":     rec.Status,
			"created_at": createdAt,
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"records": items,
		"count":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

// HandlePointsRecords 用户端：积分获取记录（认购产生，时间=订单创建时间）
func (s *WalletService) HandlePointsRecords(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	user, _, _, _, _, _, _, _, _, err := s.uc.GetBalance(ctx, token)
	if err != nil {
		return err
	}
	orders, err := s.uc.ListOrders(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(orders))
	for _, o := range orders {
		if o.FundSource == biz.PayFromWinA {
			continue
		}
		pts := o.Points
		if pts == "" || pts == "0" {
			pts = o.Principal
		}
		items = append(items, map[string]any{
			"id":           o.ID,
			"order_id":     o.ID,
			"points":       pts,
			"principal":    o.Principal,
			"total_amount": o.Principal,
			"amount":       o.Principal,
			"from_win":     o.FromWin,
			"from_win_a":   o.FromWinA,
			"fund_source":  o.FundSource,
			"status":       o.Status,
			"created_at":   o.CreatedTime.Unix(),
			"created_time": o.CreatedTime.Unix(),
		})
	}
	points := "0"
	pointsAll := "0"
	if user != nil {
		if user.Points != "" {
			points = user.Points
		}
		if user.PointsAll != "" {
			pointsAll = user.PointsAll
		}
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"points":     points,
		"points_all": pointsAll,
		"records":    items,
		"count":      len(items),
	})
}

// HandleExchangeAixToWin 用户端：AIX → 可提 U（USDT）兑换
func (s *WalletService) HandleExchangeAixToWin(ctx khttp.Context) error {
	var req struct {
		Token     string `json:"token"`
		AixAmount string `json:"aix_amount"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	rec, aixLeft, usdtWithdrawable, err := s.uc.ExchangeAixToWin(ctx, token, req.AixAmount)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"record_id":         rec.ID,
		"from_asset":        rec.FromAsset,
		"from_amount":       rec.FromAmount,
		"to_asset":          rec.ToAsset,
		"to_amount":         rec.ToAmount,
		"exchange_price":    rec.ExchangePrice,
		"exchange_fee_rate": biz.GetExchangeFeeRate(),
		"status":            rec.Status,
		"aix_balance":       aixLeft,
		"usdt_withdrawable": usdtWithdrawable,
		"created_at":        rec.CreatedTime.Unix(),
	})
}

// HandleWithdrawSDT 用户端：AIX-USDT 提现
func (s *WalletService) HandleWithdrawSDT(ctx khttp.Context) error {
	var req struct {
		Token     string `json:"token"`
		Amount    string `json:"amount"`
		ToAddress string `json:"to_address"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	w, left, err := s.uc.CreateSdtWithdraw(ctx, token, req.Amount, req.ToAddress)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"withdraw_id":  w.ID,
		"asset":        w.Asset,
		"amount":       w.Amount,
		"to_address":   w.ToAddress,
		"status":       w.Status,
		"tx_hash":      w.TxHash,
		"points":       left,
		"sdt_contract": s.uc.SdtContract(),
	})
}

// HandleWithdrawUSDT 用户端：可提 U 余额提现
func (s *WalletService) HandleWithdrawUSDT(ctx khttp.Context) error {
	var req struct {
		Token     string `json:"token"`
		Amount    string `json:"amount"`
		ToAddress string `json:"to_address"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	w, left, err := s.uc.CreateUsdtWithdraw(ctx, token, req.Amount, req.ToAddress)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"withdraw_id":     w.ID,
		"asset":           w.Asset,
		"amount":          w.Amount,
		"to_address":      w.ToAddress,
		"status":          w.Status,
		"tx_hash":         w.TxHash,
		"usdt_withdrawable": left,
		"usdt_contract":   s.uc.UsdtContract(),
	})
}

// HandleCreateWinRecharge 创建 WIN 充值单（链上转账到平台收款地址后确认入账）
func (s *WalletService) HandleCreateWinRecharge(ctx khttp.Context) error {
	var req struct {
		Token  string `json:"token"`
		Amount string `json:"amount"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	recharge, err := s.uc.CreateWinRecharge(ctx, token, req.Amount)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"recharge_id":        recharge.ID,
		"asset":              biz.TokenWIN,
		"amount":             recharge.Amount,
		"deposit_address":    s.uc.DepositAddress(),
		"deposit_addresses":  s.uc.DepositAddresses(),
		"win_contract":       s.uc.WinContract(),
		"win_decimals":       s.uc.WinDecimals(),
		"token_symbol":       biz.TokenWIN,
		"message":            recharge.Message,
		"expire_at":          recharge.ExpireAt.Unix(),
		"dev_mode":           s.uc.IsDevMode(),
		"win_price":          biz.GetWinPrice(),
	})
}

// HandleConfirmWinRecharge 确认 WIN 链上充值并入账 win_recharge_balance
func (s *WalletService) HandleConfirmWinRecharge(ctx khttp.Context) error {
	var req struct {
		Token      string `json:"token"`
		RechargeID int64  `json:"recharge_id"`
		TxHash     string `json:"tx_hash"`
		Signature  string `json:"signature"`
	}
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil && err != io.EOF {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"code": 400, "message": "invalid json"})
	}
	token := tokenFromRequest(ctx, req.Token)
	balance, amount, err := s.uc.ConfirmWinRecharge(ctx, token, req.RechargeID, req.TxHash, req.Signature)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"asset":                biz.TokenWIN,
		"amount":               amount,
		"win_balance":          balance,
		"win_recharge_balance": balance,
	})
}

// HandleListWinRecharges 查询本人 WIN 充值记录
func (s *WalletService) HandleListWinRecharges(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	records, err := s.uc.ListWinRecharges(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		item := map[string]any{
			"id": r.ID, "asset": r.Asset, "amount": r.Amount,
			"tx_hash": r.TxHash, "status": r.Status,
			"created_at": r.CreatedAt.Unix(),
		}
		if r.ConfirmedAt != nil {
			item["confirmed_at"] = r.ConfirmedAt.Unix()
		}
		items = append(items, item)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"recharges": items})
}

// HandleListWinARecharges 查询本人 WIN-A 充值记录
func (s *WalletService) HandleListWinARecharges(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	records, err := s.uc.ListWinARecharges(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		item := map[string]any{
			"id": r.ID, "asset": r.Asset, "amount": r.Amount,
			"tx_hash": r.TxHash, "status": r.Status,
			"created_at": r.CreatedAt.Unix(),
		}
		if r.ConfirmedAt != nil {
			item["confirmed_at"] = r.ConfirmedAt.Unix()
		}
		items = append(items, item)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"recharges": items})
}

// HandleExchangeRecords 用户端：查询本人的 AIX→WIN 兑换记录
func (s *WalletService) HandleExchangeRecords(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	records, err := s.uc.ListExchangeRecords(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		items = append(items, map[string]any{
			"id":             r.ID,
			"from_asset":     r.FromAsset,
			"from_amount":    r.FromAmount,
			"to_asset":       r.ToAsset,
			"to_amount":      r.ToAmount,
			"fee_amount":     r.FeeAmount,
			"fee_rate":       r.FeeRate,
			"exchange_price": r.ExchangePrice,
			"status":         r.Status,
			"remark":         r.Remark,
			"created_at":     r.CreatedTime.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"records": items})
}

// HandleWithdrawRecords 用户端：查询本人的 WIN 提现记录
func (s *WalletService) HandleWithdrawRecords(ctx khttp.Context) error {
	token := tokenFromRequest(ctx, "")
	records, err := s.uc.ListWithdrawals(ctx, token)
	if err != nil {
		return err
	}
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		asset := strings.TrimSpace(r.Asset)
		if asset == "" {
			asset = biz.TokenWIN
		}
		items = append(items, map[string]any{
			"id":         r.ID,
			"asset":      asset,
			"amount":     r.Amount,
			"fee":        r.Fee,
			"net_amount": r.NetAmount,
			"to_address": r.ToAddress,
			"status":     r.Status,
			"tx_hash":    r.TxHash,
			"remark":     r.Remark,
			"created_at": r.CreatedAt.Unix(),
			"updated_at": r.UpdatedAt.Unix(),
		})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"records": items})
}
