package biz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"backend/internal/conf"
	"backend/internal/pkg/partnersign"

	"github.com/go-kratos/kratos/v2/log"
)

const testSecret = "aix_test_secret_key_0123456789ab"

// stubWalletRepo 只实现本用例用到的方法。嵌入 WalletRepo 接口后，
// 若被调用到未覆盖的方法会直接 panic，从而暴露非预期的依赖。
type stubWalletRepo struct {
	WalletRepo
	creditResult *PartnerCreditResult
	creditErr    error
	creditCalls  int
	lastInput    PartnerCreditInput
	dailySum     string
}

func (s *stubWalletRepo) CreditPartnerWin(_ context.Context, in PartnerCreditInput) (*PartnerCreditResult, error) {
	s.creditCalls++
	s.lastInput = in
	if s.creditErr != nil {
		return nil, s.creditErr
	}
	if s.creditResult != nil {
		return s.creditResult, nil
	}
	return &PartnerCreditResult{Outcome: PartnerCreditDone, RechargeID: 42, CreditedAt: time.Now()}, nil
}

func (s *stubWalletRepo) SumPartnerCreditedSince(_ context.Context, _ string, _ time.Time) (string, error) {
	if s.dailySum == "" {
		return "0", nil
	}
	return s.dailySum, nil
}

type stubNonceRepo struct {
	seen map[string]bool
	err  error
}

func (s *stubNonceRepo) Occupy(_ context.Context, partnerID, nonce string, _ time.Duration) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	if s.seen == nil {
		s.seen = map[string]bool{}
	}
	k := partnerID + ":" + nonce
	if s.seen[k] {
		return false, nil
	}
	s.seen[k] = true
	return true, nil
}

func newTestUsecase(t *testing.T, wallet *stubWalletRepo, nonce *stubNonceRepo) (*TransferCreditUsecase, *conf.TransferPartnerConfig) {
	t.Helper()
	cfg := &conf.TransferPartnerConfig{
		TimestampSkew: "300s",
		Partners: []conf.TransferPartner{{
			PartnerID:  "AIX10001",
			Enabled:    true,
			SecretKeys: []string{testSecret},
			MaxAmount:  "100000",
			DailyLimit: "1000000",
		}},
	}
	return NewTransferCreditUsecase(wallet, nonce, cfg, log.DefaultLogger), cfg
}

func signedRequest(amount string) *TransferCreditRequest {
	req := &TransferCreditRequest{
		Address:   "0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a",
		Amount:    amount,
		PartnerID: "AIX10001",
		Timestamp: time.Now().UnixMilli(),
		Nonce:     fmt.Sprintf("n%011d", time.Now().UnixNano()%1e11),
	}
	req.Sign = partnersign.Sign(testSecret, req.SignedFields())
	return req
}

func assertCode(t *testing.T, err error, want string) {
	t.Helper()
	got, _ := TransferCodeOf(err)
	if got != want {
		t.Fatalf("expected code %s, got %s (err=%v)", want, got, err)
	}
}

func TestCreditHappyPath(t *testing.T) {
	wallet := &stubWalletRepo{}
	uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
	req := signedRequest("100.50")

	partner, err := uc.LookupPartner(req.PartnerID)
	if err != nil {
		t.Fatalf("lookup partner: %v", err)
	}
	if err := uc.VerifyRequest(req, partner); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := uc.OccupyNonce(context.Background(), req); err != nil {
		t.Fatalf("nonce: %v", err)
	}
	receipt, err := uc.Credit(context.Background(), req, partner)
	if err != nil {
		t.Fatalf("credit: %v", err)
	}
	if receipt.Amount != "100.50" {
		t.Fatalf("amount must echo the request exactly, got %s", receipt.Amount)
	}
	if receipt.AixTxnID == "" {
		t.Fatal("missing aix_txn_id")
	}
	// 幂等键必须由 partner_id + nonce 构成，否则重放无法被 tx_hash 唯一索引拦住
	if want := PartnerIdempotencyKey(req.PartnerID, req.Nonce); wallet.lastInput.IdempotencyKey != want {
		t.Fatalf("idempotency key = %s, want %s", wallet.lastInput.IdempotencyKey, want)
	}
}

func TestLookupPartnerErrors(t *testing.T) {
	wallet := &stubWalletRepo{}
	uc, cfg := newTestUsecase(t, wallet, &stubNonceRepo{})

	_, err := uc.LookupPartner("AIX99999")
	assertCode(t, err, TransferCodePartnerUnknown)

	cfg.Partners[0].Enabled = false
	_, err = uc.LookupPartner("AIX10001")
	assertCode(t, err, TransferCodePartnerOff)

	// 配置了合作方但没有可用密钥属于部署问题，不能当成签名错误
	cfg.Partners[0].Enabled = true
	cfg.Partners[0].SecretKeys = nil
	_, err = uc.LookupPartner("AIX10001")
	assertCode(t, err, TransferCodePartnerOff)
}

func TestVerifyRequestRejectsBadSignature(t *testing.T) {
	uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{})
	partner, _ := uc.LookupPartner("AIX10001")

	req := signedRequest("100.50")
	req.Sign = "deadbeef"
	assertCode(t, uc.VerifyRequest(req, partner), TransferCodeBadSign)

	// 篡改金额后原签名必须失效
	req = signedRequest("100.50")
	req.Amount = "999.00"
	assertCode(t, uc.VerifyRequest(req, partner), TransferCodeBadSign)

	// 地址大小写被改动同样应失效（文档 §3.1）
	req = signedRequest("100.50")
	req.Address = "0x7a9c8b4f2e1d6c3a5f8b0e4d2c9a1b3e6d8f5c7a"
	assertCode(t, uc.VerifyRequest(req, partner), TransferCodeBadSign)
}

func TestVerifyRequestRejectsStaleTimestamp(t *testing.T) {
	uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{})
	partner, _ := uc.LookupPartner("AIX10001")

	for _, offset := range []time.Duration{-10 * time.Minute, 10 * time.Minute} {
		req := signedRequest("100.50")
		req.Timestamp = time.Now().Add(offset).UnixMilli()
		req.Sign = partnersign.Sign(testSecret, req.SignedFields())
		assertCode(t, uc.VerifyRequest(req, partner), TransferCodeStaleTimestamp)
	}
}

func TestOccupyNonceRejectsReplay(t *testing.T) {
	nonces := &stubNonceRepo{}
	uc, _ := newTestUsecase(t, &stubWalletRepo{}, nonces)
	req := signedRequest("100.50")

	if err := uc.OccupyNonce(context.Background(), req); err != nil {
		t.Fatalf("first use should pass: %v", err)
	}
	assertCode(t, uc.OccupyNonce(context.Background(), req), TransferCodeReplay)
}

// nonce 存储不可用时必须返回 5001「未受理」，让对方能安全退款，
// 而不是 5000「结果不确定」——此时确实一分钱都没动。
func TestOccupyNonceStoreFailureIsUnavailable(t *testing.T) {
	uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{err: fmt.Errorf("db down")})
	assertCode(t, uc.OccupyNonce(context.Background(), signedRequest("100.50")), TransferCodeUnavailable)
}

func TestCreditRejectsBadFormats(t *testing.T) {
	uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{})
	partner, _ := uc.LookupPartner("AIX10001")

	cases := map[string]struct{ address, amount string }{
		"short address":     {"0x123", "100"},
		"no 0x prefix":      {"7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "100"},
		"non hex address":   {"0xZZ9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "100"},
		"negative amount":   {"0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "-100"},
		"scientific amount": {"0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "1e5"},
		"too many decimals": {"0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "1.123456789"},
		"thousands sep":     {"0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a", "1,000"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			req := signedRequest(c.amount)
			req.Address = c.address
			assertCode(t, mustCreditErr(t, uc, req, partner), TransferCodeBadFormat)
		})
	}
}

func TestCreditEnforcesLimits(t *testing.T) {
	t.Run("below system minimum", func(t *testing.T) {
		uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		// 未单独配置 min_amount 时应回落到系统 WIN 最小充值额（10）
		assertCode(t, mustCreditErr(t, uc, signedRequest("5"), partner), TransferCodeBelowMin)
	})

	t.Run("above per transaction limit", func(t *testing.T) {
		uc, _ := newTestUsecase(t, &stubWalletRepo{}, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		assertCode(t, mustCreditErr(t, uc, signedRequest("100001"), partner), TransferCodeAboveMax)
	})

	t.Run("above daily limit", func(t *testing.T) {
		wallet := &stubWalletRepo{dailySum: "999950"}
		uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		assertCode(t, mustCreditErr(t, uc, signedRequest("100"), partner), TransferCodeAboveDaily)
		if wallet.creditCalls != 0 {
			t.Fatal("must not touch balances after a limit rejection")
		}
	})
}

func TestCreditMapsAccountOutcomes(t *testing.T) {
	t.Run("address not found", func(t *testing.T) {
		wallet := &stubWalletRepo{creditResult: &PartnerCreditResult{Outcome: PartnerCreditUserNotFound}}
		uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		assertCode(t, mustCreditErr(t, uc, signedRequest("100"), partner), TransferCodeAddressUnknown)
	})

	t.Run("frozen account", func(t *testing.T) {
		wallet := &stubWalletRepo{creditResult: &PartnerCreditResult{Outcome: PartnerCreditUserFrozen}}
		uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		assertCode(t, mustCreditErr(t, uc, signedRequest("100"), partner), TransferCodeAccountFrozen)
	})

	// 事务失败时是否加款不确定，必须返回 5000 让对方转人工，绝不能返回
	// 「明确未加款」的码——那会导致对方错误退款、资金凭空多出。
	t.Run("transaction failure is indeterminate", func(t *testing.T) {
		wallet := &stubWalletRepo{creditErr: fmt.Errorf("deadlock")}
		uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		assertCode(t, mustCreditErr(t, uc, signedRequest("100"), partner), TransferCodeInternal)
	})

	// 幂等重放：钱早已到账，必须按成功返回，否则对方会把已到账的钱退回去。
	t.Run("duplicate credit still succeeds", func(t *testing.T) {
		wallet := &stubWalletRepo{creditResult: &PartnerCreditResult{
			Outcome: PartnerCreditDuplicate, RechargeID: 7, CreditedAt: time.Now(),
		}}
		uc, _ := newTestUsecase(t, wallet, &stubNonceRepo{})
		partner, _ := uc.LookupPartner("AIX10001")
		receipt, err := uc.Credit(context.Background(), signedRequest("100"), partner)
		if err != nil {
			t.Fatalf("duplicate credit must not fail: %v", err)
		}
		if receipt.AixTxnID == "" {
			t.Fatal("duplicate credit should still return a txn id")
		}
	})
}

func TestFormatAixTxnID(t *testing.T) {
	at := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	if got, want := FormatAixTxnID(9912, at), "AIXTX2026082800009912"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func mustCreditErr(t *testing.T, uc *TransferCreditUsecase, req *TransferCreditRequest, partner *conf.TransferPartner) error {
	t.Helper()
	_, err := uc.Credit(context.Background(), req, partner)
	if err == nil {
		t.Fatal("expected an error")
	}
	return err
}
