package biz

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/pkg/partnersign"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/shopspring/decimal"
)

// TransferToExchange 向交易所划转 AIX-USDT（扣 points，调用 WinBit aixInbound）。
func (uc *WalletUsecase) TransferToExchange(ctx context.Context, tokenString, amount string) (*ExchangeTransfer, string, error) {
	return uc.transferToExchangeImpl(ctx, tokenString, amount)
}

func (uc *WalletUsecase) transferToExchangeImpl(ctx context.Context, tokenString, amount string) (*ExchangeTransfer, string, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, "", err
	}
	if uc.walletCfg == nil || !uc.walletCfg.ExchangeTransferConfigured() {
		return nil, "", errors.BadRequest("EXCHANGE_TRANSFER_DISABLED", "向交易所划转暂未开通")
	}

	amt, err := ParseAmount(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", errors.BadRequest("INVALID_AMOUNT", "划转金额必须大于0")
	}
	minAmt, err := ParseAmount(GetExchangeTransferMinAmount())
	if err != nil || !minAmt.GreaterThan(decimal.Zero) {
		minAmt = decimal.NewFromInt(500)
	}
	if amt.LessThan(minAmt) {
		return nil, "", errors.BadRequest("AMOUNT_TOO_LOW", fmt.Sprintf("划转金额不能低于 %s", minAmt.String()))
	}

	requestNo, err := newExchangeTransferRequestNo()
	if err != nil {
		return nil, "", err
	}
	rec, left, err := uc.walletRepo.CreateExchangeTransfer(ctx, user.ID, user.Address, amt.String(), requestNo)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			return nil, "", errors.BadRequest("INSUFFICIENT_BALANCE", "AIX-USDT 余额不足")
		}
		return nil, "", err
	}

	partnerTxnID, partnerCode, callErr := uc.callExchangeTransferAPI(ctx, user.Address, amt.String(), requestNo)
	if callErr != nil {
		_ = uc.walletRepo.FailAndRefundExchangeTransfer(ctx, rec.ID, partnerCode, callErr.Error())
		userMsg := "向交易所划转失败，余额已退回"
		if partnerCode == "10015" || strings.Contains(callErr.Error(), "系统升级") || strings.Contains(strings.ToLower(callErr.Error()), "maintenance") {
			userMsg = "交易所系统升级中，请稍后再试（余额已退回）"
		}
		return nil, "", errors.BadRequest("EXCHANGE_TRANSFER_FAILED", userMsg)
	}
	if err := uc.walletRepo.CompleteExchangeTransfer(ctx, rec.ID, partnerTxnID, partnerCode); err != nil {
		uc.log.Errorf("complete exchange transfer %d: %v", rec.ID, err)
	}
	rec.Status = "completed"
	rec.PartnerTxnID = partnerTxnID
	rec.PartnerCode = partnerCode
	return rec, left, nil
}

func (uc *WalletUsecase) ListExchangeTransfers(ctx context.Context, tokenString string) ([]*ExchangeTransfer, error) {
	user, err := uc.resolveUser(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	return uc.walletRepo.ListExchangeTransfersByUser(ctx, user.ID)
}

func (uc *WalletUsecase) ExchangeTransferMinAmount() string {
	return GetExchangeTransferMinAmount()
}

// callExchangeTransferAPI 调用 WinBit 入金接口 POST /v2/winA/aixInbound。
// 文档：aix-inbound-integration-zh-cn.md（2026-09-04）。
func (uc *WalletUsecase) callExchangeTransferAPI(ctx context.Context, address, amount, requestNo string) (partnerTxnID, partnerCode string, err error) {
	cfg := uc.walletCfg
	secrets := cfg.ExchangeTransferActiveSecrets()
	if len(secrets) == 0 {
		return "", "", fmt.Errorf("exchange transfer secret not configured")
	}
	secret := secrets[0]
	apiURL := strings.TrimSpace(cfg.ExchangeTransferAPIURL)
	partnerID := strings.TrimSpace(cfg.ExchangeTransferPartnerID)
	address = strings.TrimSpace(address)
	amount = strings.TrimSpace(amount)
	requestNo = strings.TrimSpace(requestNo)

	const maxAttempts = 3
	var lastCode string
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ts := time.Now().UnixMilli()
		tsStr := strconv.FormatInt(ts, 10)
		signFields := map[string]string{
			"partner_id": partnerID,
			"address":    address,
			"amount":     amount,
			"request_no": requestNo,
			"timestamp":  tsStr,
		}
		sign := partnersign.Sign(secret, signFields)

		body, err := json.Marshal(map[string]any{
			"partner_id": partnerID,
			"address":    address,
			"amount":     amount,
			"request_no": requestNo,
			"timestamp":  ts,
			"sign":       sign,
		})
		if err != nil {
			return "", "", err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			lastCode = ""
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return "", lastCode, lastErr
		}

		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		uc.log.Infof("aixInbound attempt=%d http=%d body=%s", attempt, resp.StatusCode, truncateErr(string(raw), 500))

		var parsed struct {
			Success     bool            `json:"success"`
			Code        json.RawMessage `json:"code"`
			Msg         string          `json:"msg"`
			Message     string          `json:"message"`
			Timestamp   any             `json:"timestamp"`
			Nonce       string          `json:"nonce"`
			Sign        string          `json:"sign"`
			Maintenance *struct {
				Active  bool   `json:"active"`
				Title   string `json:"title"`
				Content string `json:"content"`
			} `json:"maintenance"`
			Data struct {
				RequestNo     string `json:"request_no"`
				TransactionNo string `json:"transaction_no"`
				TxnID         string `json:"txn_id"`
				PartnerTxnID  string `json:"partner_txn_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			lastErr = fmt.Errorf("invalid partner response (http %d): %s", resp.StatusCode, truncateErr(string(raw), 200))
			lastCode = ""
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return "", lastCode, lastErr
		}

		partnerCode = strings.Trim(string(parsed.Code), "\"")
		lastCode = partnerCode

		// 对方维护窗口：不重试，直接失败退款。
		if (parsed.Maintenance != nil && parsed.Maintenance.Active) || partnerCode == "10015" || resp.StatusCode == http.StatusServiceUnavailable {
			msg := strings.TrimSpace(parsed.Msg)
			if msg == "" {
				msg = strings.TrimSpace(parsed.Message)
			}
			if msg == "" && parsed.Maintenance != nil {
				msg = firstNonEmpty(parsed.Maintenance.Content, parsed.Maintenance.Title)
			}
			if msg == "" {
				msg = "系统升级中"
			}
			return "", partnerCode, fmt.Errorf("%s (code=%s)", msg, firstNonEmpty(partnerCode, "10015"))
		}

		// 200 成功；409001 幂等命中（同 request_no 已处理）视为成功，切勿退款重做。
		if parsed.Success || partnerCode == "200" || partnerCode == "409001" {
			partnerTxnID = firstNonEmpty(
				parsed.Data.TransactionNo,
				parsed.Data.TxnID,
				parsed.Data.PartnerTxnID,
				parsed.Data.RequestNo,
				requestNo,
			)
			return partnerTxnID, partnerCode, nil
		}

		msg := strings.TrimSpace(parsed.Msg)
		if msg == "" {
			msg = strings.TrimSpace(parsed.Message)
		}
		if msg == "" {
			msg = "partner rejected"
		}
		lastErr = fmt.Errorf("%s (code=%s)", msg, partnerCode)

		// 500000 服务端异常：同一 request_no 重试（幂等不会重复加款）。
		if partnerCode == "500000" && attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return "", partnerCode, lastErr
	}
	return "", lastCode, lastErr
}

// signAixInbound 保留供单测；线上入金与出金共用 partnersign（末尾不带 &）。
func signAixInbound(secret string, fields map[string]string) string {
	return partnersign.Sign(secret, fields)
}

func newExchangeTransferRequestNo() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("AIX-ET-%d-%s", time.Now().UnixMilli(), hex.EncodeToString(b)), nil
}

func truncateErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
