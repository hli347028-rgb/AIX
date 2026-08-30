package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/conf"
	"backend/internal/pkg/partnersign"

	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// TransferCreditService 实现合作方转账加款接口 POST /v1/transfer/credit。
//
// 该路由对公网开放（生产 nginx 已整段反代 /v1/），且不使用用户 JWT，
// secret_key 是唯一准入凭证，因此所有校验都必须在本 handler 内自守。
type TransferCreditService struct {
	uc  *biz.TransferCreditUsecase
	cfg *conf.TransferPartnerConfig
	log *log.Helper

	ipLimiter      *rateLimiter
	partnerLimiter *rateLimiter
}

func NewTransferCreditService(
	uc *biz.TransferCreditUsecase,
	cfg *conf.TransferPartnerConfig,
	logger log.Logger,
) *TransferCreditService {
	return &TransferCreditService{
		uc:             uc,
		cfg:            cfg,
		log:            log.NewHelper(logger),
		ipLimiter:      newRateLimiter(),
		partnerLimiter: newRateLimiter(),
	}
}

// RegisterTransferCreditRoutes mounts POST /v1/transfer/credit.
func RegisterTransferCreditRoutes(srv *khttp.Server, s *TransferCreditService) {
	r := srv.Route("/")
	r.POST("/v1/transfer/credit", s.HandleCredit)
}

// maxCreditBodyBytes 限制请求体大小，避免未认证流量用超大 body 占内存。
const maxCreditBodyBytes = 8 << 10

// transferHTTPStatus 按对接文档 §6 映射业务码到 HTTP 状态码。
// 业务结果（含地址无效）一律 200，非 200 仅用于认证失败与服务故障。
func transferHTTPStatus(code string) int {
	switch code {
	case biz.TransferCodeBadSign,
		biz.TransferCodePartnerUnknown,
		biz.TransferCodeStaleTimestamp,
		biz.TransferCodeReplay:
		return http.StatusUnauthorized
	case biz.TransferCodePartnerOff:
		return http.StatusForbidden
	case biz.TransferCodeBadRequest, biz.TransferCodeBadFormat:
		return http.StatusBadRequest
	case biz.TransferCodeInternal:
		return http.StatusInternalServerError
	case biz.TransferCodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusOK
	}
}

type creditRequestBody struct {
	Address   string `json:"address"`
	Amount    string `json:"amount"`
	PartnerID string `json:"partner_id"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Sign      string `json:"sign"`
}

// HandleCredit 按文档 §5 的顺序校验并加款。
func (s *TransferCreditService) HandleCredit(ctx khttp.Context) (err error) {
	req := ctx.Request()
	ip := clientIP(req)
	started := time.Now()

	// 裸路由不经过 kratos 的 recovery 中间件（http.Middleware 只对
	// ctx.Middleware 包裹的 handler 生效），panic 会直接断开连接、不返回任何
	// 响应体，对方将无法判定资金状态。这里必须自己兜住并返回 5000。
	defer func() {
		if rec := recover(); rec != nil {
			s.log.Errorf("transfer credit panic ip=%s: %v\n%s", ip, rec, debug.Stack())
			err = s.respond(ctx, nil, biz.TransferCodeInternal, "internal error", nil)
		}
	}()

	// 第 0 步：验签前按 IP 限速。partner_id 此时还是未经认证的自称值，
	// 只按它限速会让攻击者冒用正规合作方 ID 打满其配额造成拒绝服务。
	if !s.ipLimiter.allow(ip, s.cfg.IPRateLimit()) {
		s.log.Warnf("transfer credit ip rate limited ip=%s", ip)
		return s.respond(ctx, nil, biz.TransferCodeUnavailable, "rate limited", nil)
	}

	body, parseErr := readCreditBody(req)
	if parseErr != nil {
		s.log.Warnf("transfer credit bad body ip=%s: %v", ip, parseErr)
		return s.respond(ctx, nil, biz.TransferCodeBadRequest, "invalid request body", nil)
	}

	creditReq := &biz.TransferCreditRequest{
		Address:   body.Address,
		Amount:    body.Amount,
		PartnerID: body.PartnerID,
		Timestamp: body.Timestamp,
		Nonce:     body.Nonce,
		Sign:      body.Sign,
	}
	echo := map[string]any{"address": body.Address, "amount": body.Amount}

	if missing := missingCreditFields(body); missing != "" {
		s.log.Warnf("transfer credit missing field=%s ip=%s", missing, ip)
		return s.respond(ctx, creditReq, biz.TransferCodeBadRequest, "missing field: "+missing, echo)
	}

	partner, err := s.uc.LookupPartner(body.PartnerID)
	if err != nil {
		return s.finish(ctx, creditReq, err, echo, ip, started)
	}

	// 第 4–5 步：时间戳窗口 → 验签。
	if err := s.uc.VerifyRequest(creditReq, partner); err != nil {
		code, _ := biz.TransferCodeOf(err)
		s.log.Warnf("transfer credit auth failed code=%s partner=%s ip=%s sign=%s",
			code, body.PartnerID, ip, apiKeyHint(body.Sign))
		return s.finish(ctx, creditReq, err, echo, ip, started)
	}

	// 验签通过后才按 partner_id 限速，此时 partner_id 已可信。
	if !s.partnerLimiter.allow(body.PartnerID, partner.RateLimit()) {
		s.log.Warnf("transfer credit partner rate limited partner=%s ip=%s", body.PartnerID, ip)
		return s.respond(ctx, creditReq, biz.TransferCodeUnavailable, "rate limited", echo)
	}

	// 第 6 步：nonce 原子占用，必须在验签之后。
	if err := s.uc.OccupyNonce(ctx, creditReq); err != nil {
		return s.finish(ctx, creditReq, err, echo, ip, started)
	}

	// 第 7–8 步：格式与限额校验，然后单事务加款。
	receipt, err := s.uc.Credit(ctx, creditReq, partner)
	if err != nil {
		return s.finish(ctx, creditReq, err, echo, ip, started)
	}

	s.log.Infof("transfer credit ok partner=%s nonce=%s amount=%s txn=%s ip=%s cost=%s",
		body.PartnerID, body.Nonce, body.Amount, receipt.AixTxnID, ip, time.Since(started))

	return s.respond(ctx, creditReq, biz.TransferCodeOK, "success", map[string]any{
		"aix_txn_id":  receipt.AixTxnID,
		"address":     receipt.Address,
		"amount":      receipt.Amount,
		"credited_at": receipt.CreditedAt,
	})
}

func (s *TransferCreditService) finish(
	ctx khttp.Context,
	req *biz.TransferCreditRequest,
	err error,
	echo map[string]any,
	ip string,
	started time.Time,
) error {
	code, msg := biz.TransferCodeOf(err)
	s.log.Warnf("transfer credit rejected code=%s partner=%s ip=%s cost=%s msg=%s",
		code, req.PartnerID, ip, time.Since(started), msg)
	return s.respond(ctx, req, code, msg, echo)
}

// respond 自行写出响应体。不能走 kratos 的统一 error 编码器，
// 否则业务码形态会被改写成 kratos 的 {code,reason,message} 结构。
func (s *TransferCreditService) respond(
	ctx khttp.Context,
	req *biz.TransferCreditRequest,
	code, msg string,
	data map[string]any,
) error {
	now := time.Now()
	payload := map[string]any{
		"success":   code == biz.TransferCodeOK,
		"code":      code,
		"msg":       msg,
		"timestamp": now.UnixMilli(),
		"nonce":     responseNonce(now),
	}
	if data != nil {
		payload["data"] = data
	}
	if sign := s.signResponse(req, payload); sign != "" {
		payload["sign"] = sign
	}
	return ctx.JSON(transferHTTPStatus(code), payload)
}

// signResponse 用该合作方的当前密钥对响应签名。
func (s *TransferCreditService) signResponse(req *biz.TransferCreditRequest, payload map[string]any) string {
	if req == nil {
		return ""
	}
	partner := s.cfg.FindPartner(req.PartnerID)
	secrets := partner.ActiveSecrets()
	if len(secrets) == 0 {
		return ""
	}
	// 轮换期用最新一把密钥签响应，对方切换到新钥后即可验证。
	return partnersign.Sign(secrets[len(secrets)-1], responseSignFields(payload))
}

// responseSignFields 选出参与响应签名的字段。
//
// 已与对方约定：只签顶层标量字段（success/code/msg/timestamp/nonce），data 不参与。
// 文档 §4 的拼接规则建立在「值为原始标量」之上，而 data 是嵌套对象、没有无歧义的
// 原始值形态，把它排除掉才能让两边算出同一个串。
// 布尔渲染为 true/false 小写字面量，整数渲染为纯十进制。
//
// 改动此函数等于改变对外契约，会让对方全量验签失败；
// TestResponseSignatureAlignmentVectors 固定了双方对齐用的向量。
func responseSignFields(payload map[string]any) map[string]string {
	fields := make(map[string]string, len(payload))
	for k, v := range payload {
		switch val := v.(type) {
		case string:
			fields[k] = val
		case bool:
			fields[k] = fmt.Sprintf("%t", val)
		case int64:
			fields[k] = fmt.Sprintf("%d", val)
		}
	}
	return fields
}

func responseNonce(now time.Time) string {
	return fmt.Sprintf("%012x", now.UnixNano()&0xffffffffffff)
}

// readCreditBody 严格解析请求体。
// DisallowUnknownFields 不开启（允许对方将来加字段），但金额必须是 JSON 字符串：
// 浮点数有精度丢失风险，文档 §3.2 要求直接拒绝，这里靠字段类型天然拒绝。
func readCreditBody(req *http.Request) (*creditRequestBody, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("empty body")
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, maxCreditBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxCreditBodyBytes {
		return nil, fmt.Errorf("body too large")
	}
	var body creditRequestBody
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

func missingCreditFields(b *creditRequestBody) string {
	switch {
	case strings.TrimSpace(b.Address) == "":
		return "address"
	case strings.TrimSpace(b.Amount) == "":
		return "amount"
	case strings.TrimSpace(b.PartnerID) == "":
		return "partner_id"
	case b.Timestamp <= 0:
		return "timestamp"
	case strings.TrimSpace(b.Nonce) == "":
		return "nonce"
	case strings.TrimSpace(b.Sign) == "":
		return "sign"
	}
	// 文档 §3：nonce 长度 8–32 位
	if n := len(strings.TrimSpace(b.Nonce)); n < 8 || n > 32 {
		return "nonce"
	}
	return ""
}
