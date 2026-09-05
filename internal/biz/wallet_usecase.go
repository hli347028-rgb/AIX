package biz

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"backend/internal/conf"
	"backend/internal/pkg/eth"
	"backend/internal/pkg/token"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
)

// WalletUsecase handles recharge, subscribe and transfer logic.
type WalletUsecase struct {
	userRepo    UserRepo
	walletRepo  WalletRepo
	stakingRepo StakingRepo
	authCfg     *conf.AuthConfig
	walletCfg   *conf.WalletConfig
	log         *log.Helper
}

func NewWalletUsecase(userRepo UserRepo, walletRepo WalletRepo, stakingRepo StakingRepo, authCfg *conf.AuthConfig, walletCfg *conf.WalletConfig, logger log.Logger) *WalletUsecase {
	return &WalletUsecase{
		userRepo:    userRepo,
		walletRepo:  walletRepo,
		stakingRepo: stakingRepo,
		authCfg:     authCfg,
		walletCfg:   walletCfg,
		log:         log.NewHelper(logger),
	}
}

func (uc *WalletUsecase) resolveUser(ctx context.Context, tokenString string) (*User, error) {
	address, err := token.Parse(tokenString, uc.authCfg.GetJwtSecret())
	if err != nil {
		return nil, errors.Unauthorized("UNAUTHORIZED", "token 无效或已过期")
	}
	user, err := uc.userRepo.FindByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.NotFound("USER_NOT_FOUND", "用户不存在")
	}
	// 冻结在此统一拦截：签发在冻结之前的 token 仍然有效，若只在资金操作处校验，
	// 被冻结的账户依旧能读取全部数据直到 token 过期。
	if user.IsFrozen {
		return nil, errors.Forbidden(AccountFrozenReason, AccountFrozenMessage)
	}
	return user, nil
}

// GetBalance returns user + AIX balances mapped into legacy string slots.
// balance=usdt_recharge, released=usdt_reward, claimed=aix_balance, pending=daily static estimate, unexited=remaining exit cap
func (uc *WalletUsecase) GetBalance(ctx context.Context, tokenString string) (*User, *OrderReleaseSummary, string, string, string, string, string, int64, int64, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, nil, "", "", "", "", "", 0, 0, err
	}
	orders, err := uc.walletRepo.ListOrdersByUser(ctx, user.ID)
	if err != nil {
		return nil, nil, "", "", "", "", "", 0, 0, err
	}
	summary := SummarizeOrders(orders)
	pending := summary.PendingTotal
	unexited := summary.UnexitedTotal
	now := token.NowChina()
	nextRelease := token.NextChinaMidnight(now)
	return user, &summary, user.UsdtRecharge, user.UsdtReward, pending, user.AixBalance, unexited, nextRelease.Unix(), now.Unix(), nil
}

func SummarizeOrders(orders []*Order) OrderReleaseSummary {
	exitTotal := decimal.Zero
	earnedTotal := decimal.Zero
	pending := decimal.Zero
	unexited := decimal.Zero
	rate := decimal.NewFromFloat(StaticRate).Div(decimal.NewFromInt(100))
	nodes := int32(0)
	for _, o := range orders {
		nodes++
		p, _ := decimal.NewFromString(o.Principal)
		cap, _ := decimal.NewFromString(o.ExitCap)
		earned, _ := decimal.NewFromString(o.EarnedTotal)
		exitTotal = exitTotal.Add(cap)
		earnedTotal = earnedTotal.Add(earned)
		remain := cap.Sub(earned)
		// 追缴作废的订单结算时已被排除，剩余额度永远不会释放，不计入出局剩余
		if remain.IsPositive() && o.Status != OrderStatusCancelled {
			unexited = unexited.Add(remain)
		}
		if o.Status == OrderStatusActive {
			day := p.Mul(rate)
			if day.GreaterThan(remain) && remain.IsPositive() {
				day = remain
			}
			if remain.IsPositive() {
				pending = pending.Add(day)
			}
		}
	}
	return OrderReleaseSummary{
		ExitTotal:     exitTotal.String(),
		ReleasedTotal: earnedTotal.String(),
		PendingTotal:  pending.String(),
		UnexitedTotal: unexited.String(),
		TotalNodes:    nodes,
	}
}

func (uc *WalletUsecase) IsDevMode() bool {
	return uc.walletCfg.GetRPCURL() == ""
}

func (uc *WalletUsecase) CreateRecharge(ctx context.Context, tokenString, amount string) (*Recharge, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	amountDec, err := ParseAmount(amount)
	minRecharge, _ := ParseAmount(GetMinUsdtRecharge())
	if err != nil || amountDec.LessThan(minRecharge) {
		return nil, errors.BadRequest("INVALID_AMOUNT", fmt.Sprintf("USDT 充值金额不能小于%s", GetMinUsdtRecharge()))
	}
	depositAddress := uc.walletCfg.GetDepositAddress()
	if !uc.IsDevMode() {
		if depositAddress == "" || depositAddress == ZeroAddress {
			return nil, errors.BadRequest("DEPOSIT_NOT_CONFIGURED", "平台 USDT 收款地址未配置，请联系管理员")
		}
	}
	if uc.walletCfg.GetUsdtContract() == "" {
		return nil, errors.BadRequest("USDT_NOT_CONFIGURED", "USDT 合约地址未配置")
	}
	now := time.Now()
	message := fmt.Sprintf(
		"Recharge USDT to AIX account\nAddress: %s\nAmount: %s USDT\nToken: %s\nRechargeAt: %d",
		user.Address, amountDec.String(), uc.walletCfg.GetUsdtContract(), now.Unix(),
	)
	recharge := &Recharge{
		UserID:      user.ID,
		Address:     user.Address,
		FromAddress: user.Address,
		ToAddress:   depositAddress,
		Amount:      amountDec.String(),
		Message:     message,
		Status:      RechargeStatusPending,
		ExpireAt:    now.Add(30 * time.Minute),
	}
	return uc.walletRepo.CreateRecharge(ctx, recharge)
}

func (uc *WalletUsecase) ConfirmRecharge(ctx context.Context, tokenString string, rechargeID int64, txHash string, txHashes []string, signature string) (string, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return "", "", err
	}
	recharge, err := uc.walletRepo.FindRecharge(ctx, rechargeID)
	if err != nil {
		return "", "", err
	}
	if recharge == nil {
		return "", "", errors.NotFound("RECHARGE_NOT_FOUND", "充值记录不存在")
	}
	if recharge.UserID != user.ID {
		return "", "", errors.Forbidden("RECHARGE_FORBIDDEN", "无权确认该充值记录")
	}
	if recharge.Status == RechargeStatusConfirmed {
		return "", "", errors.BadRequest("RECHARGE_CONFIRMED", "充值记录已确认")
	}
	if !recharge.ExpireAt.IsZero() && time.Now().After(recharge.ExpireAt) {
		return "", "", errors.BadRequest("RECHARGE_EXPIRED", "充值单已过期，请重新创建")
	}
	hashes := normalizeTxHashes(txHash, txHashes)
	if len(hashes) == 0 {
		return "", "", errors.BadRequest("INVALID_TX_HASH", "交易哈希不能为空")
	}
	for _, h := range hashes {
		exists, err := uc.walletRepo.FindRechargeByTxHash(ctx, h)
		if err != nil {
			return "", "", err
		}
		if exists != nil && exists.ID != rechargeID {
			// The background scanner may have credited this user's transaction
			// before the browser submits its confirmation request.
			if exists.UserID != user.ID || exists.Status != RechargeStatusConfirmed {
				return "", "", errors.BadRequest("TX_HASH_USED", "交易哈希已被使用")
			}
		}
	}
	if err := eth.VerifyPersonalSign(recharge.Message, signature, user.Address); err != nil {
		return "", "", errors.Unauthorized("INVALID_SIGNATURE", "签名校验失败")
	}
	amountDec, _ := ParseAmount(recharge.Amount)
	depositAddrs := uc.walletCfg.GetDepositAddresses()
	splits := SplitEqualAmounts(amountDec, len(depositAddrs), uc.walletCfg.GetUsdtDecimals())
	if uc.walletCfg.GetRPCURL() != "" {
		if len(depositAddrs) == 0 {
			return "", "", errors.BadRequest("DEPOSIT_NOT_CONFIGURED", "平台 USDT 收款地址未配置，请联系管理员")
		}
		if len(depositAddrs) == 1 {
			if len(hashes) != 1 {
				return "", "", errors.BadRequest("INVALID_TX_HASH", "请提交 1 笔充值交易哈希")
			}
			if err := eth.VerifyUSDTTransfer(ctx, uc.walletCfg.GetRPCURL(), hashes[0], uc.walletCfg.GetUsdtContract(), depositAddrs, user.Address, amountDec, uc.walletCfg.GetUsdtDecimals()); err != nil {
				return "", "", errors.BadRequest("TX_VERIFY_FAILED", err.Error())
			}
		} else {
			if len(hashes) != len(depositAddrs) {
				return "", "", errors.BadRequest("INVALID_TX_HASH", fmt.Sprintf("请分别向 %d 个收款地址各转账一笔，并提交全部交易哈希", len(depositAddrs)))
			}
			used := make([]bool, len(hashes))
			for i, addr := range depositAddrs {
				matched := false
				var lastErr error
				for j, h := range hashes {
					if used[j] {
						continue
					}
					err := eth.VerifyUSDTTransfer(ctx, uc.walletCfg.GetRPCURL(), h, uc.walletCfg.GetUsdtContract(), []string{addr}, user.Address, splits[i], uc.walletCfg.GetUsdtDecimals())
					if err == nil {
						used[j] = true
						matched = true
						break
					}
					lastErr = err
				}
				if !matched {
					msg := "未找到对应收款地址的分账转账"
					if lastErr != nil {
						msg = lastErr.Error()
					}
					return "", "", errors.BadRequest("TX_VERIFY_FAILED", fmt.Sprintf("收款地址 %s 校验失败: %s", addr, msg))
				}
			}
		}
	}

	// Balance changes are exclusively performed by ChainRechargeJob after the
	// configured confirmation depth. The browser confirmation only validates
	// the submitted transaction and removes the temporary order.
	if err := uc.walletRepo.DeletePendingRecharge(ctx, rechargeID); err != nil {
		return "", "", err
	}
	balance, _, _, err := uc.userRepo.GetBalances(ctx, user.ID)
	if err != nil {
		return "", "", err
	}
	return balance, recharge.Amount, nil
}

func normalizeTxHashes(txHash string, txHashes []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(txHashes)+1)
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == ' ' || r == '\n' || r == '\t'
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			key := strings.ToLower(p)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, p)
		}
	}
	for _, h := range txHashes {
		add(h)
	}
	add(txHash)
	return out
}

func SplitEqualAmounts(total decimal.Decimal, n int, decimals int32) []decimal.Decimal {
	if n <= 0 {
		return nil
	}
	if decimals < 0 {
		decimals = 0
	}
	if n == 1 {
		return []decimal.Decimal{total}
	}
	out := make([]decimal.Decimal, n)
	base := total.Div(decimal.NewFromInt(int64(n))).Truncate(decimals)
	assigned := decimal.Zero
	for i := 0; i < n-1; i++ {
		out[i] = base
		assigned = assigned.Add(base)
	}
	out[n-1] = total.Sub(assigned)
	return out
}

func (uc *WalletUsecase) ListRecharges(ctx context.Context, tokenString string) ([]*Recharge, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	// /v1/wallet/recharges 仅返回 USDT；WIN 走 /v1/wallet/recharges-win
	return uc.walletRepo.ListRechargesByUserAsset(ctx, user.ID, TokenUSDT)
}

// ListDownlineUSDTRecharges 当前用户所有下级 USDT 充值记录（已确认；不含 WIN）。
func (uc *WalletUsecase) ListDownlineUSDTRecharges(
	ctx context.Context, tokenString string, page, pageSize int,
) ([]*Recharge, int64, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, 0, err
	}
	ids, err := uc.userRepo.ListUserIDsUnder(ctx, user.ID)
	if err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return uc.walletRepo.ListConfirmedUSDTRechargesByUserIDs(ctx, ids, offset, pageSize)
}

// ListDownlineWINRecharges 当前用户所有下级 WIN 充值记录（已确认）。
// 含链上 WIN 充值，以及交易所划转入账（recharges.tx_hash 以 partner: 开头）。
func (uc *WalletUsecase) ListDownlineWINRecharges(
	ctx context.Context, tokenString string, page, pageSize int,
) ([]*Recharge, int64, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, 0, err
	}
	ids, err := uc.userRepo.ListUserIDsUnder(ctx, user.ID)
	if err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return uc.walletRepo.ListConfirmedWINRechargesByUserIDs(ctx, ids, offset, pageSize)
}

// ListDownlineSubscribeOrders 当前用户所有下级的认购订单（含复投 / WIN 支付）。
func (uc *WalletUsecase) ListDownlineSubscribeOrders(
	ctx context.Context, tokenString string, page, pageSize int,
) ([]*AdminOrderDetail, int64, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, 0, err
	}
	ids, err := uc.userRepo.ListUserIDsUnder(ctx, user.ID)
	if err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return uc.walletRepo.ListOrdersByUserIDs(ctx, ids, offset, pageSize)
}

func (uc *WalletUsecase) CreateWithdraw(ctx context.Context, tokenString, amount, toAddress, signature string, withdrawAt int64) (*Withdrawal, string, error) {
	return nil, "", errors.BadRequest("USDT_WITHDRAW_FORBIDDEN", "仅支持提现 WIN 代币，不支持提现 USDT")
}

// CreateAixWithdraw AIX 代币当前禁止提现，需先兑换为 WIN 后再提现
func (uc *WalletUsecase) CreateAixWithdraw(ctx context.Context, tokenString, amount, toAddress string) (*Withdrawal, string, error) {
	return nil, "", errors.BadRequest("AIX_WITHDRAW_FORBIDDEN", "AIX 不可直接提现，请先兑换为可提 U 余额")
}

// ExchangeAixToWin AIX → 可提 U（USDT）兑换。
// 当全网当日已兑换 AIX（含待审）加上本笔后，超过「今日AIX数量 × 审核阈值%」时进入待审核：
// 先扣 AIX，审核通过后再入可提 U；拒绝则退回 AIX。
func (uc *WalletUsecase) ExchangeAixToWin(ctx context.Context, tokenString, aixAmount string) (*ExchangeRecord, string, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, "", "", err
	}
	if !user.ExchangeEnabled {
		return nil, "", "", errors.BadRequest("EXCHANGE_DISABLED", "兑换功能已关闭，请联系客服")
	}
	amt, err := ParseAmount(aixAmount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", "", errors.BadRequest("INVALID_AMOUNT", "兑换金额必须大于0")
	}
	needReview, err := uc.exchangeNeedsReview(ctx, amt)
	if err != nil {
		return nil, "", "", err
	}
	rec, aixLeft, usdtWithdrawable, err := uc.walletRepo.ExchangeAixToWin(ctx, user.ID, amt.String(), needReview)
	if err != nil {
		if strings.Contains(err.Error(), "exchange disabled") {
			return nil, "", "", errors.BadRequest("EXCHANGE_DISABLED", "兑换功能已关闭，请联系客服")
		}
		if strings.Contains(err.Error(), "insufficient") {
			return nil, "", "", errors.BadRequest("INSUFFICIENT_AIX", "AIX 代币余额不足")
		}
		if strings.Contains(err.Error(), "aix price") {
			return nil, "", "", errors.BadRequest("AIX_PRICE_NOT_CONFIGURED", "AIX 价格未配置")
		}
		if strings.Contains(err.Error(), "net amount too small") {
			return nil, "", "", errors.BadRequest("USDT_NET_AMOUNT_TOO_SMALL", "兑换金额过小，扣除手续费后 USDT 净量为 0")
		}
		return nil, "", "", err
	}
	return rec, aixLeft, usdtWithdrawable, nil
}

// exchangeNeedsReview 判断本笔兑换是否需进审核。
// 今日AIX数量 = 今天 0 点结算任务对应结算日（昨日）发放的静态 AIX 总量。
// 已兑换量只计当日已完成（completed）；待审核不占配额，也不会计入次日阈值。
// 今日尚无结算产量时不触发审核（避免 0 基数下任意兑换都被拦）。
func (uc *WalletUsecase) exchangeNeedsReview(ctx context.Context, amt decimal.Decimal) (bool, error) {
	pct, err := decimal.NewFromString(strings.TrimSpace(GetExchangeReviewThresholdPercent()))
	if err != nil || pct.IsNegative() {
		pct, _ = decimal.NewFromString(conf.DefaultExchangeReviewThresholdPercent)
	}
	todayAixStr, err := uc.walletRepo.SumStaticAixBySettlementDate(ctx, TodaySettlementDate(time.Now()))
	if err != nil {
		return false, err
	}
	todayAix, _ := ParseAmount(todayAixStr)
	if !todayAix.IsPositive() {
		return false, nil
	}
	limit := todayAix.Mul(pct).Div(decimal.NewFromInt(100))
	now := time.Now().In(token.ChinaLocation())
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	exchangedStr, err := uc.walletRepo.SumExchangedAixSince(ctx, since)
	if err != nil {
		return false, err
	}
	exchanged, _ := ParseAmount(exchangedStr)
	return exchanged.Add(amt).GreaterThan(limit), nil
}

// CreateWinWithdraw WIN 提现已关闭
func (uc *WalletUsecase) CreateWinWithdraw(ctx context.Context, tokenString, amount, toAddress string) (*Withdrawal, string, error) {
	return nil, "", errors.BadRequest("WIN_WITHDRAW_DISABLED", "WIN 提现已关闭")
}

// CreateSdtWithdraw AIX-USDT 提现已关闭
func (uc *WalletUsecase) CreateSdtWithdraw(ctx context.Context, tokenString, amount, toAddress string) (*Withdrawal, string, error) {
	return nil, "", errors.BadRequest("SDT_WITHDRAW_DISABLED", "AIX-USDT 提现已关闭")
}

// CreateUsdtWithdraw 提现可提 U 余额（链上 USDT ERC20）
func (uc *WalletUsecase) CreateUsdtWithdraw(ctx context.Context, tokenString, amount, toAddress string) (*Withdrawal, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, "", err
	}
	amt, err := ParseAmount(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", errors.BadRequest("INVALID_AMOUNT", "提现金额必须大于0")
	}
	minWithdraw, err := decimal.NewFromString(uc.walletCfg.GetMinWithdraw())
	if err == nil && minWithdraw.GreaterThan(decimal.Zero) && amt.LessThan(minWithdraw) {
		return nil, "", errors.BadRequest("INVALID_AMOUNT", "提现金额低于最低限额")
	}
	toNorm := strings.TrimSpace(toAddress)
	if toNorm == "" {
		toNorm = user.Address
	} else {
		toNorm, err = eth.NormalizeAddress(toNorm)
		if err != nil {
			return nil, "", errors.BadRequest("INVALID_ADDRESS", "提现地址无效")
		}
	}
	w, left, err := uc.walletRepo.CreateUsdtWithdrawal(ctx, user.ID, amt.String(), toNorm, initialWithdrawStatus(TokenUSDT, amt))
	if err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			return nil, "", errors.BadRequest("INSUFFICIENT_USDT", "可提 U 余额不足")
		}
		return nil, "", err
	}
	return w, left, nil
}

func initialWithdrawStatus(asset string, amount decimal.Decimal) string {
	if NeedsWithdrawReview(asset, amount) {
		return WithdrawStatusReview
	}
	return WithdrawStatusPending
}

func (uc *WalletUsecase) ListExchangeRecords(ctx context.Context, tokenString string) ([]*ExchangeRecord, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListExchangeRecordsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) ListWithdrawals(ctx context.Context, tokenString string) ([]*Withdrawal, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListWithdrawalsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) DepositAddress() string {
	return uc.walletCfg.GetDepositAddress()
}

func (uc *WalletUsecase) DepositAddresses() []string {
	return uc.walletCfg.GetDepositAddresses()
}

func (uc *WalletUsecase) SplitDepositAmounts(amount string) []string {
	total, err := ParseAmount(amount)
	if err != nil || !total.GreaterThan(decimal.Zero) {
		return nil
	}
	addrs := uc.walletCfg.GetDepositAddresses()
	parts := SplitEqualAmounts(total, len(addrs), uc.walletCfg.GetUsdtDecimals())
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.String())
	}
	return out
}

func (uc *WalletUsecase) UsdtContract() string {
	return uc.walletCfg.GetUsdtContract()
}

// Subscribe AIX 报单：amount + pay_from(recharge|reward|win)
func (uc *WalletUsecase) Subscribe(ctx context.Context, tokenString string, productID int64, quantity int32, amountStr string) (*Order, string, error) {
	// Legacy proto path without pay_from — reject and ask for custom route
	return nil, "", errors.BadRequest("PAY_FROM_REQUIRED", "请使用 /v1/wallet/subscribe-aix 并传 pay_from=recharge|reward|win")
}

// SubscribeAIX 单源认购：recharge（USDT 充值钱包）/ reward（复投）/ win
func (uc *WalletUsecase) SubscribeAIX(ctx context.Context, tokenString, amountStr, payFrom string) (*Order, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, "", err
	}
	payFrom = strings.ToLower(strings.TrimSpace(payFrom))
	if payFrom == PayFromWinA {
		return nil, "", errors.BadRequest("WIN_A_SUBSCRIBE_DISABLED", "WIN-A 认购已关闭")
	}
	if payFrom != PayFromRecharge && payFrom != PayFromReward && payFrom != PayFromWin {
		return nil, "", errors.BadRequest("INVALID_PAY_FROM", "pay_from 必须为 recharge、reward 或 win")
	}

	minSubscribe, err := ParseAmount(uc.walletCfg.GetMinSubscribe())
	if err != nil {
		minSubscribe = decimal.NewFromInt(100)
	}
	total, err := ParseAmount(strings.TrimSpace(amountStr))
	if err != nil || !total.GreaterThan(decimal.Zero) {
		return nil, "", errors.BadRequest("INVALID_AMOUNT", "认购金额必须大于0")
	}
	if total.LessThan(minSubscribe) {
		return nil, "", errors.BadRequest("MIN_SUBSCRIBE_LIMIT", fmt.Sprintf("认购金额不能低于 %s USDT", minSubscribe.String()))
	}

	if payFrom == PayFromWin {
		winPrice := decimal.NewFromFloat(GetWinPrice())
		if !winPrice.IsPositive() {
			return nil, "", errors.BadRequest("WIN_PRICE_NOT_CONFIGURED", "WIN 价格未配置")
		}
	}

	order, bal, err := uc.walletRepo.Subscribe(ctx, user.ID, SubscribeInput{
		Amount:     total.String(),
		PayFrom:    payFrom,
		ExitMul:    ExitMultiplier,
		DirectRate: DirectRate,
	})
	if err != nil {
		if strings.Contains(err.Error(), "insufficient usdt_recharge") || strings.Contains(err.Error(), "insufficient usdt_reward") {
			return nil, "", errors.BadRequest("INSUFFICIENT_BALANCE", "账户余额不足")
		}
		if strings.Contains(err.Error(), "insufficient win_recharge_balance") {
			return nil, "", errors.BadRequest("INSUFFICIENT_WIN", "WIN 充值余额不足")
		}
		if strings.Contains(err.Error(), "win price not configured") {
			return nil, "", errors.BadRequest("WIN_PRICE_NOT_CONFIGURED", "WIN 价格未配置")
		}
		if strings.Contains(err.Error(), "win amount too small") {
			return nil, "", errors.BadRequest("INVALID_AMOUNT", "WIN 数量过小")
		}
		return nil, "", err
	}
	order.SyncCompatFields()
	return order, bal, nil
}

func (uc *WalletUsecase) CreateWinRecharge(ctx context.Context, tokenString, amount string) (*Recharge, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	amountDec, err := ParseAmount(amount)
	minRecharge, _ := ParseAmount(GetMinWinRecharge())
	if err != nil || amountDec.LessThan(minRecharge) {
		return nil, errors.BadRequest("INVALID_AMOUNT", fmt.Sprintf("WIN 充值数量不能小于%s", GetMinWinRecharge()))
	}
	depositAddress := uc.walletCfg.GetDepositAddress()
	if !uc.IsDevMode() {
		if depositAddress == "" || depositAddress == ZeroAddress {
			return nil, errors.BadRequest("DEPOSIT_NOT_CONFIGURED", "平台收款地址未配置，请联系管理员")
		}
	}
	winContract := uc.walletCfg.GetWinContract()
	if !uc.IsDevMode() && winContract == "" {
		return nil, errors.BadRequest("WIN_NOT_CONFIGURED", "WIN 合约地址未配置")
	}
	now := time.Now()
	message := fmt.Sprintf(
		"Recharge WIN to AIX account\nAddress: %s\nAmount: %s WIN\nToken: %s\nRechargeAt: %d",
		user.Address, amountDec.String(), winContract, now.Unix(),
	)
	recharge := &Recharge{
		UserID:      user.ID,
		Address:     user.Address,
		Asset:       TokenWIN,
		FromAddress: user.Address,
		ToAddress:   depositAddress,
		Amount:      amountDec.String(),
		Message:     message,
		Status:      RechargeStatusPending,
		ExpireAt:    now.Add(30 * time.Minute),
	}
	return uc.walletRepo.CreateRecharge(ctx, recharge)
}

func (uc *WalletUsecase) ConfirmWinRecharge(ctx context.Context, tokenString string, rechargeID int64, txHash string, signature string) (string, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return "", "", err
	}
	recharge, err := uc.walletRepo.FindRecharge(ctx, rechargeID)
	if err != nil {
		return "", "", err
	}
	if recharge == nil {
		return "", "", errors.NotFound("RECHARGE_NOT_FOUND", "充值记录不存在")
	}
	if recharge.UserID != user.ID {
		return "", "", errors.Forbidden("RECHARGE_FORBIDDEN", "无权确认该充值记录")
	}
	if !strings.EqualFold(recharge.Asset, TokenWIN) {
		return "", "", errors.BadRequest("INVALID_ASSET", "非 WIN 充值单，请使用 USDT 确认接口")
	}
	if recharge.Status == RechargeStatusConfirmed {
		return "", "", errors.BadRequest("RECHARGE_CONFIRMED", "充值记录已确认")
	}
	if !recharge.ExpireAt.IsZero() && time.Now().After(recharge.ExpireAt) {
		return "", "", errors.BadRequest("RECHARGE_EXPIRED", "充值单已过期，请重新创建")
	}
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return "", "", errors.BadRequest("INVALID_TX_HASH", "交易哈希不能为空")
	}
	exists, err := uc.walletRepo.FindRechargeByTxHash(ctx, txHash)
	if err != nil {
		return "", "", err
	}
	if exists != nil && exists.ID != rechargeID {
		if exists.UserID == user.ID && exists.Status == RechargeStatusConfirmed && strings.EqualFold(exists.Asset, TokenWIN) {
			userFresh, err2 := uc.userRepo.FindByID(ctx, user.ID)
			if err2 != nil || userFresh == nil {
				return "", "", err2
			}
			return userFresh.WinRechargeBalance, exists.Amount, nil
		}
		return "", "", errors.BadRequest("TX_HASH_USED", "交易哈希已被使用")
	}
	if err := eth.VerifyPersonalSign(recharge.Message, signature, user.Address); err != nil {
		return "", "", errors.Unauthorized("INVALID_SIGNATURE", "签名校验失败")
	}
	amountDec, _ := ParseAmount(recharge.Amount)
	depositAddrs := uc.walletCfg.GetDepositAddresses()
	if uc.walletCfg.GetRPCURL() != "" {
		if len(depositAddrs) == 0 {
			return "", "", errors.BadRequest("DEPOSIT_NOT_CONFIGURED", "平台收款地址未配置，请联系管理员")
		}
		winContract := uc.walletCfg.GetWinContract()
		if winContract == "" {
			return "", "", errors.BadRequest("WIN_NOT_CONFIGURED", "WIN 合约地址未配置")
		}
		if err := eth.VerifyERC20Transfer(ctx, uc.walletCfg.GetRPCURL(), txHash, winContract, depositAddrs, user.Address, amountDec, uc.walletCfg.GetWinDecimals(), TokenWIN); err != nil {
			return "", "", errors.BadRequest("TX_VERIFY_FAILED", err.Error())
		}
	}

	// 校验通过后直接入账 win_recharge_balance（无 WIN 链上扫描任务时由确认接口负责入账）
	balance, err := uc.walletRepo.ConfirmRechargeCredit(ctx, rechargeID, txHash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already confirmed") {
			credited, bal, creditErr := uc.walletRepo.AutoCreditWinRecharge(ctx, txHash, user.Address, recharge.ToAddress, recharge.Amount)
			if creditErr == nil && (credited || bal != "") {
				return bal, recharge.Amount, nil
			}
		}
		return "", "", err
	}
	return balance, recharge.Amount, nil
}

func (uc *WalletUsecase) ListWinRecharges(ctx context.Context, tokenString string) ([]*Recharge, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListRechargesByUserAsset(ctx, user.ID, TokenWIN)
}

func (uc *WalletUsecase) ListWinARecharges(ctx context.Context, tokenString string) ([]*Recharge, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListRechargesByUserAsset(ctx, user.ID, TokenWINA)
}

func (uc *WalletUsecase) WinContract() string {
	return uc.walletCfg.GetWinContract()
}

func (uc *WalletUsecase) WinADepositContract() string {
	return uc.walletCfg.GetWinADepositContract()
}

func (uc *WalletUsecase) WinARechargeEnabled() bool {
	return false
}

func (uc *WalletUsecase) WinAContract() string {
	return uc.walletCfg.GetWinAContract()
}

func (uc *WalletUsecase) WinADecimals() int32 {
	return uc.walletCfg.GetWinADecimals()
}

func (uc *WalletUsecase) WinDecimals() int32 {
	return uc.walletCfg.GetWinDecimals()
}

func (uc *WalletUsecase) SdtContract() string {
	return uc.walletCfg.GetSdtContract()
}

func (uc *WalletUsecase) SdtDecimals() int32 {
	return uc.walletCfg.GetSdtDecimals()
}

func (uc *WalletUsecase) MinUsdtRecharge() string {
	return GetMinUsdtRecharge()
}

func (uc *WalletUsecase) MinWinRecharge() string {
	return GetMinWinRecharge()
}

func (uc *WalletUsecase) MinWinARecharge() string {
	return GetMinWinARecharge()
}

func (uc *WalletUsecase) ListOrders(ctx context.Context, tokenString string) ([]*Order, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	orders, err := uc.walletRepo.ListOrdersByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		o.SyncCompatFields()
	}
	return orders, nil
}

// Transfer 上下级转账
func (uc *WalletUsecase) Transfer(ctx context.Context, tokenString, toAddress, asset, amount, payFrom string) (*Transfer, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	toNorm, err := eth.NormalizeAddress(toAddress)
	if err != nil {
		return nil, errors.BadRequest("INVALID_ADDRESS", "收款地址无效")
	}
	toUser, err := uc.userRepo.FindByAddress(ctx, toNorm)
	if err != nil {
		return nil, err
	}
	if toUser == nil {
		return nil, errors.NotFound("USER_NOT_FOUND", "收款用户不存在")
	}
	if toUser.ID == user.ID {
		return nil, errors.BadRequest("INVALID_TRANSFER", "不能转给自己")
	}
	ok, err := uc.userRepo.IsUplineOf(ctx, user.ID, toUser.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.BadRequest("NOT_DOWNLINE", "仅允许上级向邀请链下级划转")
	}
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if asset == "" {
		asset = TokenUSDT
	}
	if asset != TokenUSDT {
		return nil, errors.BadRequest("INVALID_ASSET", "用户划转仅支持奖励钱包 USDT")
	}
	amt, err := ParseAmount(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, errors.BadRequest("INVALID_AMOUNT", "转账金额必须大于0")
	}
	payFrom = strings.ToLower(strings.TrimSpace(payFrom))
	if payFrom != PayFromReward {
		return nil, errors.BadRequest("INVALID_PAY_FROM", "用户划转只能从奖励钱包扣款")
	}
	t := &Transfer{
		FromUserID: user.ID,
		ToUserID:   toUser.ID,
		Asset:      asset,
		Amount:     amt.String(),
		PayFrom:    payFrom,
	}
	created, err := uc.walletRepo.CreateTransfer(ctx, t)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			return nil, errors.BadRequest("INSUFFICIENT_BALANCE", "余额不足")
		}
		return nil, err
	}
	return created, nil
}

func (uc *WalletUsecase) ListLinealTransferRecords(ctx context.Context, tokenString, direction string) ([]*LinealTransferRecord, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		direction = "all"
	}
	if direction != "all" && direction != "in" && direction != "out" {
		return nil, errors.BadRequest("INVALID_DIRECTION", "direction 必须为 all、in 或 out")
	}
	transfers, err := uc.walletRepo.ListTransfersByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	users, err := uc.userRepo.ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	userByID := make(map[int64]*User, len(users))
	parents := make(map[int64]int64, len(users))
	for _, item := range users {
		userByID[item.ID] = item
		if item.InviterID != nil {
			parents[item.ID] = *item.InviterID
		}
	}

	result := make([]*LinealTransferRecord, 0)
	for _, transfer := range transfers {
		if transfer.FromUserID == transfer.ToUserID || transfer.PayFrom != PayFromReward || transfer.Asset != TokenUSDT {
			continue
		}
		recordDirection := "in"
		counterpartyID := transfer.FromUserID
		if transfer.FromUserID == user.ID {
			recordDirection = "out"
			counterpartyID = transfer.ToUserID
		}
		if direction != "all" && direction != recordDirection {
			continue
		}
		counterparty := userByID[counterpartyID]
		if counterparty == nil || !IsLinealRelation(user.ID, counterpartyID, parents) {
			continue
		}
		relationship := "downline"
		for parentID, seen := parents[user.ID], map[int64]struct{}{}; parentID > 0; parentID = parents[parentID] {
			if parentID == counterpartyID {
				relationship = "upline"
				break
			}
			if _, ok := seen[parentID]; ok {
				break
			}
			seen[parentID] = struct{}{}
		}
		result = append(result, &LinealTransferRecord{
			ID:                  transfer.ID,
			Direction:           recordDirection,
			Relationship:        relationship,
			CounterpartyUserID:  counterpartyID,
			CounterpartyAddress: counterparty.Address,
			Asset:               transfer.Asset,
			Amount:              transfer.Amount,
			FromWallet:          PayFromReward,
			ToWallet:            PayFromReward,
			CreatedTime:         transfer.CreatedTime,
		})
	}
	return result, nil
}

func (uc *WalletUsecase) ListRewardLogs(ctx context.Context, tokenString string) ([]*RewardLog, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListRewardLogsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) GetTeamActiveSubscribePrincipal(ctx context.Context, tokenString string) (string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return "0", err
	}
	return uc.userRepo.SumActivePrincipalUnder(ctx, user.ID)
}

func (uc *WalletUsecase) GetMgmtRewardSummary(ctx context.Context, tokenString string) (*MgmtRewardSummary, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.GetMgmtRewardSummary(ctx, user.ID)
}

func (uc *WalletUsecase) GetDirectRewardTotal(ctx context.Context, tokenString string) (string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return "0", err
	}
	return uc.walletRepo.GetDirectRewardTotal(ctx, user.ID)
}

func (uc *WalletUsecase) ListMgmtRewards(ctx context.Context, tokenString string) ([]*MgmtReward, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListMgmtRewardsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) GetAixPrice(ctx context.Context, date string) (string, error) {
	if date == "" {
		date = token.NowChina().Format("2006-01-02")
	}
	price, err := uc.walletRepo.GetAixPrice(ctx, date)
	if err != nil {
		return "", err
	}
	if price == "" {
		if latest, err := uc.walletRepo.GetLatestAixPriceBefore(ctx, date); err == nil && strings.TrimSpace(latest) != "" && latest != "0" {
			return FormatAixPrice(latest), nil
		}
		return FormatAixPriceDecimal(decimal.NewFromFloat(AixPriceInitial)), nil
	}
	return FormatAixPrice(price), nil
}

func (uc *WalletUsecase) ListReleaseRecords(ctx context.Context, tokenString string) ([]*ReleaseRecord, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.stakingRepo.ListReleaseRecordsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) ListReferralRewards(ctx context.Context, tokenString string) ([]*ReferralReward, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.stakingRepo.ListReferralRewardsByUser(ctx, user.ID)
}

func (uc *WalletUsecase) SumReferralByOrderDate(ctx context.Context, orderID int64, settlementDate string) (string, error) {
	return uc.stakingRepo.SumReferralByOrderDate(ctx, orderID, settlementDate)
}

func (uc *WalletUsecase) FindOrder(ctx context.Context, orderID int64) (*Order, error) {
	return uc.walletRepo.FindOrder(ctx, orderID)
}

func (uc *WalletUsecase) FindUserAddress(ctx context.Context, userID int64) (string, error) {
	if userID <= 0 {
		return "", nil
	}
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return "", err
	}
	return user.Address, nil
}

func (uc *WalletUsecase) BuildOrderIndexMap(ctx context.Context, userID int64) (map[int64]int32, error) {
	orders, err := uc.walletRepo.ListOrdersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sorted := append([]*Order(nil), orders...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	result := make(map[int64]int32, len(sorted))
	for i, o := range sorted {
		result[o.ID] = int32(i + 1)
	}
	return result, nil
}
