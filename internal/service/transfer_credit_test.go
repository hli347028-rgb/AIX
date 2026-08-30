package service

import (
	"net/http"
	"testing"

	"backend/internal/biz"
	"backend/internal/conf"
	"backend/internal/pkg/partnersign"

	"github.com/go-kratos/kratos/v2/log"
)

const testPartnerSecret = "aix_test_secret_key_0123456789ab"

func newTestService() *TransferCreditService {
	cfg := &conf.TransferPartnerConfig{
		Partners: []conf.TransferPartner{{
			PartnerID:  "AIX10001",
			Enabled:    true,
			SecretKeys: []string{testPartnerSecret},
		}},
	}
	return NewTransferCreditService(nil, cfg, log.DefaultLogger)
}

// 响应签名只覆盖顶层标量字段——data 是嵌套对象，文档 §4 未定义其拼接形态，
// 已与对方约定排除。此用例把该口径固定下来，避免无意改动导致对方验签失败。
func TestSignResponseCoversScalarTopLevelFieldsOnly(t *testing.T) {
	s := newTestService()
	payload := map[string]any{
		"success":   true,
		"code":      "0000",
		"msg":       "success",
		"timestamp": int64(1787922966740),
		"nonce":     "fa77264d9ad8",
		"data":      map[string]any{"amount": "100.50"},
	}
	got := s.signResponse(&biz.TransferCreditRequest{PartnerID: "AIX10001"}, payload)

	want := partnersign.Sign(testPartnerSecret, map[string]string{
		"success":   "true",
		"code":      "0000",
		"msg":       "success",
		"timestamp": "1787922966740",
		"nonce":     "fa77264d9ad8",
	})
	if got != want {
		t.Fatalf("response signature mismatch\n got: %s\nwant: %s", got, want)
	}

	// data 变化不应影响签名，正是因为它未被纳入
	payload["data"] = map[string]any{"amount": "999.99"}
	if s.signResponse(&biz.TransferCreditRequest{PartnerID: "AIX10001"}, payload) != want {
		t.Fatal("data is currently out of scope and must not affect the signature")
	}
}

// 发给对方的响应方向对齐向量（对接说明 §6.1）。用对方文档 §4.1 的公开测试密钥，
// 与请求方向的 vectorSign 用例作用相同：双方据此确认响应验签实现一致。
// 这些值一旦变动就意味着线上响应会被对方判为签名错误。
func TestResponseSignatureAlignmentVectors(t *testing.T) {
	const specSecret = "aix_test_secret_key_0123456789ab"

	cases := []struct {
		name            string
		payload         map[string]any
		wantSignPayload string
		wantSign        string
	}{
		{
			name: "success",
			payload: map[string]any{
				"success":   true,
				"code":      "0000",
				"msg":       "success",
				"timestamp": int64(1787875200455),
				"nonce":     "c7d21be40a9f",
				"data": map[string]any{
					"aix_txn_id":  "AIXTX2026082800009912",
					"address":     "0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a",
					"amount":      "100.50",
					"credited_at": int64(1787875200431),
				},
			},
			wantSignPayload: "code=0000&msg=success&nonce=c7d21be40a9f&success=true&timestamp=1787875200455",
			wantSign:        "3e5b79bc5409928fed49d545d828c71b91ac77878973d6fc9c9fe4fe452c4155",
		},
		{
			// msg 含空格，按文档 §4 第 3 条取原始值、不做 URL 编码
			name: "address not found",
			payload: map[string]any{
				"success":   false,
				"code":      "2001",
				"msg":       "address not found",
				"timestamp": int64(1787875200455),
				"nonce":     "c7d21be40a9f",
			},
			wantSignPayload: "code=2001&msg=address not found&nonce=c7d21be40a9f&success=false&timestamp=1787875200455",
			wantSign:        "63b1db5c7b1035d41809b4fd994d74bfbb69826a0bcb7ab7520404a68184ceb0",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fields := responseSignFields(c.payload)
			if got := partnersign.Payload(fields); got != c.wantSignPayload {
				t.Fatalf("payload mismatch\n got: %s\nwant: %s", got, c.wantSignPayload)
			}
			if got := partnersign.Sign(specSecret, fields); got != c.wantSign {
				t.Fatalf("sign mismatch\n got: %s\nwant: %s", got, c.wantSign)
			}
		})
	}
}

// 未知合作方无法签名：不能用别人的密钥、也不能退化成不签名却照常返回成功。
func TestSignResponseWithoutUsableSecret(t *testing.T) {
	s := newTestService()
	payload := map[string]any{"code": "1002", "msg": "partner not found"}
	if sign := s.signResponse(&biz.TransferCreditRequest{PartnerID: "AIX99999"}, payload); sign != "" {
		t.Fatalf("expected no signature for unknown partner, got %s", sign)
	}
	if sign := s.signResponse(nil, payload); sign != "" {
		t.Fatalf("expected no signature without a request, got %s", sign)
	}
}

// 业务结果一律 200，认证与服务故障才用非 200（文档 §6）。
func TestTransferHTTPStatusMapping(t *testing.T) {
	cases := map[string]int{
		biz.TransferCodeOK:             http.StatusOK,
		biz.TransferCodeAddressUnknown: http.StatusOK,
		biz.TransferCodeAccountFrozen:  http.StatusOK,
		biz.TransferCodeBelowMin:       http.StatusOK,
		biz.TransferCodeAboveMax:       http.StatusOK,
		biz.TransferCodeAboveDaily:     http.StatusOK,
		biz.TransferCodeBadSign:        http.StatusUnauthorized,
		biz.TransferCodePartnerUnknown: http.StatusUnauthorized,
		biz.TransferCodeStaleTimestamp: http.StatusUnauthorized,
		biz.TransferCodeReplay:         http.StatusUnauthorized,
		biz.TransferCodePartnerOff:     http.StatusForbidden,
		biz.TransferCodeBadRequest:     http.StatusBadRequest,
		biz.TransferCodeBadFormat:      http.StatusBadRequest,
		biz.TransferCodeInternal:       http.StatusInternalServerError,
		biz.TransferCodeUnavailable:    http.StatusServiceUnavailable,
	}
	for code, want := range cases {
		if got := transferHTTPStatus(code); got != want {
			t.Errorf("code %s: got HTTP %d, want %d", code, got, want)
		}
	}
}

func TestMissingCreditFields(t *testing.T) {
	valid := func() *creditRequestBody {
		return &creditRequestBody{
			Address:   "0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a",
			Amount:    "100.50",
			PartnerID: "AIX10001",
			Timestamp: 1787875200000,
			Nonce:     "9f1c4a2e7b83",
			Sign:      "abc",
		}
	}
	if got := missingCreditFields(valid()); got != "" {
		t.Fatalf("valid body reported missing field %q", got)
	}

	b := valid()
	b.Timestamp = 0
	if got := missingCreditFields(b); got != "timestamp" {
		t.Fatalf("got %q, want timestamp", got)
	}

	// 文档 §3：nonce 长度 8–32 位
	b = valid()
	b.Nonce = "short"
	if got := missingCreditFields(b); got != "nonce" {
		t.Fatalf("got %q, want nonce for an under-length value", got)
	}
	b.Nonce = "aaaaaaaaaabbbbbbbbbbccccccccccddddd"
	if got := missingCreditFields(b); got != "nonce" {
		t.Fatalf("got %q, want nonce for an over-length value", got)
	}
}

// 令牌桶按 key 隔离，且耗尽后拒绝——否则限速形同虚设。
func TestRateLimiter(t *testing.T) {
	l := newRateLimiter()
	for i := 0; i < 3; i++ {
		if !l.allow("a", 3) {
			t.Fatalf("request %d should be allowed within the burst", i+1)
		}
	}
	if l.allow("a", 3) {
		t.Fatal("burst exhausted, further requests must be rejected")
	}
	if !l.allow("b", 3) {
		t.Fatal("a different key must have its own bucket")
	}
}
