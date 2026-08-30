package biz

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend/internal/conf"
	"backend/internal/pkg/partnersign"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
)

// 合作方转账加款接口的业务码（对接文档 §6）。
// 这些码直接决定对方是否退款，语义不能随意改：
//   - 2xxx/3xxx 表示明确未加款，对方会退款给用户
//   - 5000/5002 表示结果不确定，对方不退款、转人工核查
//   - 5001 表示请求完全未受理，对方可安全退款
const (
	TransferCodeOK             = "0000"
	TransferCodeAddressUnknown = "2001"
	TransferCodeAccountFrozen  = "2002"
	TransferCodeBelowMin       = "2003"
	TransferCodeAboveMax       = "2004"
	TransferCodeAboveDaily     = "2005"
	TransferCodeBadSign        = "1001"
	TransferCodePartnerUnknown = "1002"
	TransferCodeStaleTimestamp = "1003"
	TransferCodeReplay         = "1004"
	TransferCodePartnerOff     = "1006"
	TransferCodeBadRequest     = "3002"
	TransferCodeBadFormat      = "3003"
	TransferCodeInternal       = "5000"
	TransferCodeUnavailable    = "5001"
)

var (
	// 文档 §3.1：0x + 40 位十六进制，大小写原样保留
	partnerAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	// 文档 §3.2：无符号十进制，最多 18 位整数 + 8 位小数，禁止科学计数法与千分位
	partnerAmountRe = regexp.MustCompile(`^\d{1,18}(\.\d{1,8})?$`)
)

// PartnerNonceRepo 合作方 nonce 去重。
type PartnerNonceRepo interface {
	// Occupy 原子占用 (partnerID, nonce)，true=首次出现，false=重放。
	// retain 为保留窗口，不得小于时间戳允许偏差。
	Occupy(ctx context.Context, partnerID, nonce string, retain time.Duration) (bool, error)
}

// PartnerCreditOutcome 加款事务的结果分类。
type PartnerCreditOutcome int

const (
	// PartnerCreditDone 本次调用真实完成了加款。
	PartnerCreditDone PartnerCreditOutcome = iota
	// PartnerCreditDuplicate 同一幂等键此前已加过款。仍按成功返回：
	// 余额确实已经到账，返回 false 会让对方错误退款。
	PartnerCreditDuplicate
	// PartnerCreditUserNotFound 地址未注册。
	PartnerCreditUserNotFound
	// PartnerCreditUserFrozen 账户被冻结。
	PartnerCreditUserFrozen
)

// PartnerCreditInput 加款入参。
type PartnerCreditInput struct {
	// IdempotencyKey 写入 recharges.tx_hash，靠唯一索引保证不重复加款。
	IdempotencyKey string
	// Address 原始地址字符串（已验签），仓储层按小写匹配用户。
	Address   string
	Amount    string
	PartnerID string
	Nonce     string
}

// PartnerCreditResult 加款结果。
type PartnerCreditResult struct {
	Outcome    PartnerCreditOutcome
	RechargeID int64
	NewBalance string
	CreditedAt time.Time
}

// TransferCreditError 携带对接文档业务码的错误。
type TransferCreditError struct {
	Code    string
	Message string
}

func (e *TransferCreditError) Error() string { return e.Code + ": " + e.Message }

func transferErr(code, message string) *TransferCreditError {
	return &TransferCreditError{Code: code, Message: message}
}

// TransferCodeOf 提取错误上的业务码与提示。
// 未携带业务码的错误一律归为 5000（结果不确定），绝不能降级成
// 「明确未加款」的码——那会让对方错误退款。
func TransferCodeOf(err error) (string, string) {
	if err == nil {
		return TransferCodeOK, "success"
	}
	var te *TransferCreditError
	if errors.As(err, &te) {
		return te.Code, te.Message
	}
	return TransferCodeInternal, "internal error"
}

// TransferCreditRequest 已解析的请求体。
type TransferCreditRequest struct {
	Address   string
	Amount    string
	PartnerID string
	Timestamp int64
	Nonce     string
	Sign      string
}

// SignedFields 返回参与签名的字段（不含 sign）。
// 地址与金额一律用原始字符串：任何大小写或格式归一化都会让签名对不上（文档 §3.1）。
func (r *TransferCreditRequest) SignedFields() map[string]string {
	return map[string]string{
		"address":    r.Address,
		"amount":     r.Amount,
		"partner_id": r.PartnerID,
		"timestamp":  fmt.Sprintf("%d", r.Timestamp),
		"nonce":      r.Nonce,
	}
}

// TransferCreditReceipt 加款成功的回执。
type TransferCreditReceipt struct {
	AixTxnID   string
	Address    string
	Amount     string
	CreditedAt int64
}

// TransferCreditUsecase 实现合作方转账加款的业务规则。
type TransferCreditUsecase struct {
	walletRepo WalletRepo
	nonceRepo  PartnerNonceRepo
	cfg        *conf.TransferPartnerConfig
	log        *log.Helper
}

func NewTransferCreditUsecase(
	walletRepo WalletRepo,
	nonceRepo PartnerNonceRepo,
	cfg *conf.TransferPartnerConfig,
	logger log.Logger,
) *TransferCreditUsecase {
	return &TransferCreditUsecase{
		walletRepo: walletRepo,
		nonceRepo:  nonceRepo,
		cfg:        cfg,
		log:        log.NewHelper(logger),
	}
}

// LookupPartner 按 partner_id 取合作方配置，区分「不存在」与「已禁用」。
// 供 handler 在验签前做限速决策。
func (uc *TransferCreditUsecase) LookupPartner(partnerID string) (*conf.TransferPartner, error) {
	p := uc.cfg.FindPartner(partnerID)
	if p == nil {
		return nil, transferErr(TransferCodePartnerUnknown, "partner not found")
	}
	if !p.Enabled {
		return nil, transferErr(TransferCodePartnerOff, "partner disabled")
	}
	if len(p.ActiveSecrets()) == 0 {
		// 配置了合作方却没有可用密钥，属于部署问题，不能当成签名错误放行排查。
		uc.log.Errorf("partner %s has no usable secret key configured", partnerID)
		return nil, transferErr(TransferCodePartnerOff, "partner disabled")
	}
	return p, nil
}

// VerifyRequest 执行文档 §5 第 4–5 步：时间戳窗口 → 验签。
//
// 必须在 OccupyNonce 之前调用：nonce 占用是一次数据库写入，若放在验签之前，
// 未认证的流量就能靠随机 nonce 撑爆去重表。
func (uc *TransferCreditUsecase) VerifyRequest(
	req *TransferCreditRequest,
	partner *conf.TransferPartner,
) error {
	drift := time.Since(time.UnixMilli(req.Timestamp))
	if drift < 0 {
		drift = -drift
	}
	if drift > uc.cfg.SkewDuration() {
		return transferErr(TransferCodeStaleTimestamp, "timestamp out of range")
	}
	if !partnersign.VerifyAny(partner.ActiveSecrets(), req.SignedFields(), req.Sign) {
		return transferErr(TransferCodeBadSign, "invalid signature")
	}
	return nil
}

// OccupyNonce 执行文档 §5 第 6 步：原子占用 nonce 防重放。
// 保留窗口取时间戳允许偏差，避免「nonce 已过期但时间戳仍有效」的空档。
func (uc *TransferCreditUsecase) OccupyNonce(ctx context.Context, req *TransferCreditRequest) error {
	fresh, err := uc.nonceRepo.Occupy(ctx, req.PartnerID, req.Nonce, uc.cfg.SkewDuration())
	if err != nil {
		// 无法判定是否重放时宁可拒绝，但要用「未受理」语义让对方安全退款。
		uc.log.Errorf("nonce occupy failed partner=%s: %v", req.PartnerID, err)
		return transferErr(TransferCodeUnavailable, "nonce store unavailable")
	}
	if !fresh {
		return transferErr(TransferCodeReplay, "duplicate nonce")
	}
	return nil
}

// Credit 执行文档 §5 第 7–8 步：格式与限额校验，然后在单事务内加款。
func (uc *TransferCreditUsecase) Credit(
	ctx context.Context,
	req *TransferCreditRequest,
	partner *conf.TransferPartner,
) (*TransferCreditReceipt, error) {
	if !partnerAddressRe.MatchString(req.Address) {
		return nil, transferErr(TransferCodeBadFormat, "invalid address format")
	}
	if !partnerAmountRe.MatchString(req.Amount) {
		return nil, transferErr(TransferCodeBadFormat, "invalid amount format")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		return nil, transferErr(TransferCodeBadFormat, "invalid amount")
	}

	if err := uc.checkLimits(ctx, req.PartnerID, partner, amount); err != nil {
		return nil, err
	}

	res, err := uc.walletRepo.CreditPartnerWin(ctx, PartnerCreditInput{
		IdempotencyKey: PartnerIdempotencyKey(req.PartnerID, req.Nonce),
		Address:        req.Address,
		Amount:         amount.String(),
		PartnerID:      req.PartnerID,
		Nonce:          req.Nonce,
	})
	if err != nil {
		// 事务失败：是否加款不确定，必须用 5000 让对方转人工而不是自动退款。
		uc.log.Errorf("partner credit tx failed partner=%s nonce=%s: %v", req.PartnerID, req.Nonce, err)
		return nil, transferErr(TransferCodeInternal, "credit failed")
	}

	switch res.Outcome {
	case PartnerCreditUserNotFound:
		return nil, transferErr(TransferCodeAddressUnknown, "address not found")
	case PartnerCreditUserFrozen:
		return nil, transferErr(TransferCodeAccountFrozen, "account frozen")
	}

	uc.log.Infof("partner credit ok partner=%s nonce=%s amount=%s balance=%s duplicate=%t",
		req.PartnerID, req.Nonce, amount.String(), res.NewBalance, res.Outcome == PartnerCreditDuplicate)

	return &TransferCreditReceipt{
		AixTxnID:   FormatAixTxnID(res.RechargeID, res.CreditedAt),
		Address:    req.Address,
		Amount:     req.Amount,
		CreditedAt: res.CreditedAt.UnixMilli(),
	}, nil
}

// checkLimits 校验单笔下限、单笔上限与单日累计上限。
func (uc *TransferCreditUsecase) checkLimits(
	ctx context.Context,
	partnerID string,
	partner *conf.TransferPartner,
	amount decimal.Decimal,
) error {
	// 三个限额的取值优先级：合作方在 config.yaml 里的单独配置 > 管理端全局配置 > 硬编码默认。
	// 管理端改动经 ApplyAixConfig 热生效，无需重启；yaml 里留空即表示交给管理端管。
	minAmount := partner.MinAmountDecimal()
	if minAmount.IsZero() {
		minAmount, _ = ParseAmount(GetPartnerMinAmount())
	}
	if minAmount.GreaterThan(decimal.Zero) && amount.LessThan(minAmount) {
		return transferErr(TransferCodeBelowMin, "amount below minimum")
	}
	maxAmount := partner.MaxAmountDecimal()
	if maxAmount.IsZero() {
		maxAmount, _ = ParseAmount(GetPartnerMaxAmount())
	}
	if maxAmount.GreaterThan(decimal.Zero) && amount.GreaterThan(maxAmount) {
		return transferErr(TransferCodeAboveMax, "amount above per-transaction limit")
	}

	daily := partner.DailyLimitDecimal()
	if daily.IsZero() {
		daily, _ = ParseAmount(GetPartnerDailyLimit())
	}
	// 按服务器本地自然日统计。time.Truncate 是相对 UTC 零点对齐的，
	// 直接用会把「单日」切在当地时间的非午夜时刻。
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sum, err := uc.walletRepo.SumPartnerCreditedSince(ctx, partnerID, since)
	if err != nil {
		// 查不到已用额度就无法保证不超限，按「未受理」拒绝比冒险放行安全。
		uc.log.Errorf("daily limit lookup failed partner=%s: %v", partnerID, err)
		return transferErr(TransferCodeUnavailable, "limit check unavailable")
	}
	used, _ := decimal.NewFromString(sum)
	if used.Add(amount).GreaterThan(daily) {
		return transferErr(TransferCodeAboveDaily, "daily limit exceeded")
	}
	return nil
}

// PartnerIdempotencyKey 构造写入 recharges.tx_hash 的幂等键。
// 合作方加款没有链上 hash，用 partner_id + nonce 复用该列的唯一索引。
func PartnerIdempotencyKey(partnerID, nonce string) string {
	return fmt.Sprintf("partner:%s:%s", strings.TrimSpace(partnerID), strings.TrimSpace(nonce))
}

// PartnerIdempotencyPrefix 供按合作方聚合流水时做前缀匹配。
func PartnerIdempotencyPrefix(partnerID string) string {
	return fmt.Sprintf("partner:%s:", strings.TrimSpace(partnerID))
}

// FormatAixTxnID 生成返回给对方的流水号：AIXTX + 日期 + 充值记录 ID。
// 带上 recharges.id 保证全局唯一，且能反查到具体流水。
func FormatAixTxnID(rechargeID int64, at time.Time) string {
	if at.IsZero() {
		at = time.Now()
	}
	return fmt.Sprintf("AIXTX%s%08d", at.Format("20060102"), rechargeID)
}
