package job

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"backend/internal/biz"
	"backend/internal/conf"
	"backend/internal/pkg/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
)

// WithdrawPayoutResult describes one payout attempt.
type WithdrawPayoutResult struct {
	Processed  bool   `json:"processed"`
	Skipped    bool   `json:"skipped,omitempty"`
	Reason     string `json:"reason,omitempty"`
	WithdrawID int64  `json:"withdraw_id,omitempty"`
	TxHash     string `json:"tx_hash,omitempty"`
	Error      string `json:"error,omitempty"`
}

// WithdrawPayoutCycleResult is returned when cron/HTTP starts a background cycle.
type WithdrawPayoutCycleResult struct {
	Accepted        bool   `json:"accepted"`
	Queries         int    `json:"queries"`
	IntervalSeconds int64  `json:"interval_seconds"`
	Reason          string `json:"reason,omitempty"`
}

// WithdrawPayoutJob processes pending WIN withdrawals with the hot wallet private key.
type WithdrawPayoutJob struct {
	walletRepo biz.WalletRepo
	cfg        *conf.WalletConfig
	log        *log.Helper
	cycling    atomic.Bool
}

func NewWithdrawPayoutJob(walletRepo biz.WalletRepo, cfg *conf.WalletConfig, logger log.Logger) *WithdrawPayoutJob {
	return &WithdrawPayoutJob{
		walletRepo: walletRepo,
		cfg:        cfg,
		log:        log.NewHelper(logger),
	}
}

func (j *WithdrawPayoutJob) cycleParams() (queries int, gap time.Duration) {
	queries = int(j.cfg.GetWithdrawPayoutQueriesPerCycle())
	if queries <= 0 {
		queries = 10
	}
	gap = time.Duration(j.cfg.GetWithdrawPayoutQueryIntervalSeconds()) * time.Second
	if gap <= 0 {
		gap = 5 * time.Second
	}
	return queries, gap
}

// TriggerCycle starts a background cycle: N payout attempts, gap seconds apart.
func (j *WithdrawPayoutJob) TriggerCycle() *WithdrawPayoutCycleResult {
	queries, gap := j.cycleParams()
	res := &WithdrawPayoutCycleResult{
		Queries: queries, IntervalSeconds: int64(gap / time.Second),
	}
	if !j.cfg.IsWithdrawPayoutEnabled() {
		res.Accepted = false
		res.Reason = "withdraw payout disabled"
		return res
	}
	if j.cfg.GetWithdrawPrivateKey() == "" && j.cfg.GetSdtPrivateKey() == "" {
		res.Accepted = false
		res.Reason = "withdraw private key not configured"
		return res
	}
	if !j.cycling.CompareAndSwap(false, true) {
		res.Accepted = false
		res.Reason = "cycle already running"
		return res
	}
	res.Accepted = true
	go func() {
		defer j.cycling.Store(false)
		j.log.Infof("withdraw payout cycle started: queries=%d interval=%s", queries, gap)
		for i := 0; i < queries; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			if result, err := j.ProcessOnce(ctx); err != nil {
				j.log.Errorf("withdraw payout cycle #%d/%d error: %v", i+1, queries, err)
			} else if result != nil {
				if result.Processed {
					j.log.Infof("withdraw payout cycle #%d/%d: id=%d tx=%s err=%s",
						i+1, queries, result.WithdrawID, result.TxHash, result.Error)
				} else {
					j.log.Infof("withdraw payout cycle #%d/%d: no pending withdrawal", i+1, queries)
				}
			}
			cancel()
			if i >= queries-1 {
				break
			}
			time.Sleep(gap)
		}
		j.log.Info("withdraw payout cycle finished")
	}()
	return res
}

// ProcessOnce recovers stuck rows and pays at most one pending withdrawal.
func (j *WithdrawPayoutJob) ProcessOnce(ctx context.Context) (*WithdrawPayoutResult, error) {
	if !j.cfg.IsWithdrawPayoutEnabled() {
		return &WithdrawPayoutResult{Skipped: true, Reason: "disabled"}, nil
	}

	if err := j.recoverInProgress(ctx); err != nil {
		j.log.Warnf("recover in-progress withdrawals: %v", err)
	}
	if err := j.reconcileDoingWithoutTxHash(ctx); err != nil {
		j.log.Warnf("reconcile doing withdrawals without tx_hash: %v", err)
	}
	if err := j.releaseStaleDoingWithdrawals(ctx); err != nil {
		j.log.Warnf("release stale doing withdrawals: %v", err)
	}

	w, err := j.walletRepo.ClaimNextPendingWinWithdrawal(ctx)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return &WithdrawPayoutResult{Processed: false}, nil
	}

	privKey := j.payoutPrivateKey(w)
	if privKey == "" {
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, "payout private key not configured")
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID,
			Error: fmt.Sprintf("no private key for asset %s", w.Asset),
		}, nil
	}

	if done, txHash, err := j.reconcileWithdrawal(ctx, w, privKey); err != nil {
		return nil, err
	} else if done {
		return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, TxHash: txHash}, nil
	}

	if hasConfirmed, err := j.walletRepo.HasConfirmedWithdrawalPayout(ctx, w.ID); err != nil {
		return nil, err
	} else if hasConfirmed {
		if w.TxHash != "" {
			_ = j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, w.TxHash)
		}
		return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, TxHash: w.TxHash}, nil
	}

	amount, err := decimal.NewFromString(w.NetAmount)
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, "invalid payout amount")
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID,
			Error: fmt.Sprintf("invalid amount: %s", w.NetAmount),
		}, nil
	}

	fromAddress, nonce, err := eth.PendingNativeTransferNonce(ctx, j.cfg.GetRPCURL(), privKey)
	if err != nil {
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, err.Error())
		return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, Error: err.Error()}, nil
	}
	if err := j.walletRepo.SetWithdrawalPayoutNonce(ctx, w.ID, nonce); err != nil {
		j.log.Warnf("withdraw %d save payout nonce failed: %v", w.ID, err)
	}

	sendResult, err := j.sendWithdrawTransfer(ctx, w, privKey, nonce)
	if err != nil {
		if recovered, txHash, recErr := j.recoverByStoredNonce(ctx, w, fromAddress, nonce); recErr == nil && recovered {
			return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, TxHash: txHash}, nil
		}
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, err.Error())
		return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, Error: err.Error()}, nil
	}

	if err := j.walletRepo.CreateWithdrawalPayoutAttempt(
		ctx, w.ID, sendResult.TxHash, sendResult.FromAddress, w.ToAddress, amount.String(), sendResult.Nonce,
	); err != nil {
		j.log.Errorf("withdraw %d record payout attempt failed: %v", w.ID, err)
	}

	if err := j.walletRepo.SetWithdrawalTxHash(ctx, w.ID, sendResult.TxHash); err != nil {
		j.log.Errorf("withdraw %d tx sent (%s) but save tx_hash failed: %v", w.ID, sendResult.TxHash, err)
		if done, txHash, recErr := j.reconcileByTxHash(ctx, w, sendResult.TxHash); recErr == nil && done {
			return &WithdrawPayoutResult{Processed: true, WithdrawID: w.ID, TxHash: txHash}, nil
		}
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID, TxHash: sendResult.TxHash,
			Error: "tx sent but db update failed; reconciled on next cycle",
		}, nil
	}

	return j.finalizeSubmittedTx(ctx, w.ID, sendResult.TxHash)
}

func (j *WithdrawPayoutJob) payoutPrivateKey(w *biz.Withdrawal) string {
	if w == nil {
		return j.cfg.GetWithdrawPrivateKey()
	}
	if strings.EqualFold(strings.TrimSpace(w.Asset), biz.TokenSDT) {
		return j.cfg.GetSdtPrivateKey()
	}
	return j.cfg.GetWithdrawPrivateKey()
}

func (j *WithdrawPayoutJob) sendWithdrawTransfer(
	ctx context.Context,
	w *biz.Withdrawal,
	privKey string,
	nonce uint64,
) (*eth.NativeTransferSendResult, error) {
	amount, err := decimal.NewFromString(w.NetAmount)
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		return nil, fmt.Errorf("invalid payout amount")
	}
	asset := strings.ToUpper(strings.TrimSpace(w.Asset))
	switch asset {
	case biz.TokenWIN:
		// EOEO 链上 WIN 即主币，提现直接 native 转账到用户地址，无需 ERC20 合约。
		return eth.SendNativeTransferWithNonce(
			ctx, j.cfg.GetRPCURL(), privKey, w.ToAddress, amount, j.cfg.GetWinDecimals(), &nonce,
		)
	case biz.TokenSDT:
		contract := j.cfg.GetSdtContract()
		if contract == "" {
			return nil, fmt.Errorf("sdt contract not configured")
		}
		result, err := eth.SendERC20TransferWithNonce(
			ctx, j.cfg.GetRPCURL(), privKey, contract, w.ToAddress, amount, j.cfg.GetSdtDecimals(), &nonce,
		)
		if err != nil {
			return nil, err
		}
		return &eth.NativeTransferSendResult{
			TxHash:      result.TxHash,
			Nonce:       result.Nonce,
			FromAddress: result.FromAddress,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported withdraw asset %s", w.Asset)
	}
}

func (j *WithdrawPayoutJob) recoverInProgress(ctx context.Context) error {
	list, err := j.walletRepo.ListDoingWithdrawalsWithTxHash(ctx)
	if err != nil {
		return err
	}
	for _, w := range list {
		if strings.TrimSpace(w.TxHash) == "" {
			continue
		}
		privKey := j.payoutPrivateKey(w)
		done, _, err := j.reconcileByTxHash(ctx, w, w.TxHash)
		if err != nil {
			j.log.Warnf("reconcile withdraw %d tx %s: %v", w.ID, w.TxHash, err)
			continue
		}
		if !done {
			j.log.Infof("withdraw %d tx %s still pending", w.ID, w.TxHash)
		}
		_ = privKey
	}
	return nil
}

func (j *WithdrawPayoutJob) reconcileDoingWithoutTxHash(ctx context.Context) error {
	list, err := j.walletRepo.ListDoingWithdrawalsWithoutTxHash(ctx)
	if err != nil {
		return err
	}
	for _, w := range list {
		privKey := j.payoutPrivateKey(w)
		if _, _, err := j.reconcileWithdrawal(ctx, w, privKey); err != nil {
			j.log.Warnf("reconcile withdraw %d without tx_hash: %v", w.ID, err)
		}
	}
	return nil
}

func (j *WithdrawPayoutJob) releaseStaleDoingWithdrawals(ctx context.Context) error {
	staleBefore := time.Now().Add(-10 * time.Minute)
	list, err := j.walletRepo.ListStaleDoingWinWithdrawals(ctx, staleBefore)
	if err != nil {
		return err
	}
	for _, w := range list {
		privKey := j.payoutPrivateKey(w)
		if done, _, err := j.reconcileWithdrawal(ctx, w, privKey); err != nil {
			j.log.Warnf("stale withdraw %d reconcile failed: %v", w.ID, err)
			continue
		} else if done {
			continue
		}
		if err := j.safeResetForRetry(ctx, w, privKey, "payout timed out; retry pending"); err != nil {
			j.log.Warnf("stale withdraw %d reset skipped: %v", w.ID, err)
		} else {
			j.log.Infof("released stale withdraw %d back to pending", w.ID)
		}
	}
	return nil
}

func (j *WithdrawPayoutJob) reconcileWithdrawal(ctx context.Context, w *biz.Withdrawal, privKey string) (done bool, txHash string, err error) {
	if hasConfirmed, err := j.walletRepo.HasConfirmedWithdrawalPayout(ctx, w.ID); err != nil {
		return false, "", err
	} else if hasConfirmed {
		hash := strings.TrimSpace(w.TxHash)
		if hash == "" {
			attempts, err := j.walletRepo.ListWithdrawalPayoutAttempts(ctx, w.ID)
			if err != nil {
				return false, "", err
			}
			for _, attempt := range attempts {
				if attempt.Status == biz.PayoutStatusConfirmed {
					hash = attempt.TxHash
					break
				}
			}
		}
		if hash != "" && w.Status != biz.WithdrawStatusCompleted {
			if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, hash); err != nil {
				return false, hash, err
			}
		}
		return true, hash, nil
	}

	if hash := strings.TrimSpace(w.TxHash); hash != "" {
		return j.reconcileByTxHash(ctx, w, hash)
	}

	attempts, err := j.walletRepo.ListWithdrawalPayoutAttempts(ctx, w.ID)
	if err != nil {
		return false, "", err
	}
	for _, attempt := range attempts {
		if attempt.Status == biz.PayoutStatusConfirmed {
			if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, attempt.TxHash); err != nil {
				return false, attempt.TxHash, err
			}
			return true, attempt.TxHash, nil
		}
		if attempt.Status == biz.PayoutStatusSubmitted {
			if done, txHash, err := j.reconcileByTxHash(ctx, w, attempt.TxHash); err != nil || done {
				return done, txHash, err
			}
		}
	}

	fromAddress, err := eth.SenderAddressFromPrivateKey(privKey)
	if err != nil {
		return false, "", err
	}
	if w.PayoutNonce != nil {
		if done, txHash, err := j.recoverByStoredNonce(ctx, w, fromAddress, *w.PayoutNonce); err != nil || done {
			return done, txHash, err
		}
	}
	return false, "", nil
}

func (j *WithdrawPayoutJob) reconcileByTxHash(ctx context.Context, w *biz.Withdrawal, txHash string) (done bool, outHash string, err error) {
	outcome, err := eth.InspectTransactionReceipt(ctx, j.cfg.GetRPCURL(), txHash)
	if err != nil {
		return false, txHash, err
	}
	if outcome.Pending {
		return false, txHash, nil
	}
	if outcome.Success {
		if err := j.walletRepo.SetWithdrawalTxHash(ctx, w.ID, txHash); err != nil {
			j.log.Warnf("withdraw %d backfill tx_hash failed: %v", w.ID, err)
		}
		if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, txHash); err != nil {
			return false, txHash, err
		}
		j.log.Infof("recovered completed withdraw %d tx=%s", w.ID, txHash)
		return true, txHash, nil
	}

	_ = j.walletRepo.MarkWithdrawalPayoutFailed(ctx, w.ID, txHash)
	if err := j.safeResetForRetry(ctx, w, j.payoutPrivateKey(w), "chain tx failed; retry pending"); err != nil {
		j.log.Warnf("reset failed withdraw %d: %v", w.ID, err)
	} else {
		j.log.Infof("reset failed withdraw %d tx=%s for retry", w.ID, txHash)
	}
	return false, txHash, nil
}

func (j *WithdrawPayoutJob) recoverByStoredNonce(ctx context.Context, w *biz.Withdrawal, fromAddress string, nonce uint64) (done bool, txHash string, err error) {
	txHash, success, found, err := eth.FindTransactionBySenderNonce(ctx, j.cfg.GetRPCURL(), fromAddress, nonce, 500)
	if err != nil || !found {
		return false, "", err
	}
	if err := j.walletRepo.CreateWithdrawalPayoutAttempt(
		ctx, w.ID, txHash, fromAddress, w.ToAddress, w.NetAmount, nonce,
	); err != nil {
		j.log.Warnf("withdraw %d backfill payout attempt failed: %v", w.ID, err)
	}
	if err := j.walletRepo.SetWithdrawalTxHash(ctx, w.ID, txHash); err != nil {
		j.log.Warnf("withdraw %d backfill tx_hash from nonce failed: %v", w.ID, err)
	}
	if !success {
		outcome, err := eth.InspectTransactionReceipt(ctx, j.cfg.GetRPCURL(), txHash)
		if err != nil {
			return false, txHash, err
		}
		if outcome.Pending {
			return false, txHash, nil
		}
		if !outcome.Success {
			_ = j.walletRepo.MarkWithdrawalPayoutFailed(ctx, w.ID, txHash)
			if err := j.safeResetForRetry(ctx, w, j.payoutPrivateKey(w), "chain tx failed; retry pending"); err != nil {
				return false, txHash, err
			}
			return false, txHash, nil
		}
	}
	if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, txHash); err != nil {
		return false, txHash, err
	}
	j.log.Infof("recovered withdraw %d from nonce=%d tx=%s", w.ID, nonce, txHash)
	return true, txHash, nil
}

func (j *WithdrawPayoutJob) safeResetForRetry(ctx context.Context, w *biz.Withdrawal, privKey, remark string) error {
	if done, _, err := j.reconcileWithdrawal(ctx, w, privKey); err != nil {
		return err
	} else if done {
		return fmt.Errorf("withdrawal %d already reconciled on chain", w.ID)
	}
	fromAddress, err := eth.SenderAddressFromPrivateKey(privKey)
	if err != nil {
		return err
	}
	if w.PayoutNonce != nil {
		mined, err := eth.IsSenderNonceMined(ctx, j.cfg.GetRPCURL(), fromAddress, *w.PayoutNonce)
		if err != nil {
			return err
		}
		if mined {
			if _, txHash, err := j.recoverByStoredNonce(ctx, w, fromAddress, *w.PayoutNonce); err != nil {
				return err
			} else if txHash != "" {
				return fmt.Errorf("withdrawal %d nonce %d already mined as %s", w.ID, *w.PayoutNonce, txHash)
			}
		}
	}
	return j.walletRepo.ResetWithdrawalForRetry(ctx, w.ID, remark)
}

func (j *WithdrawPayoutJob) finalizeSubmittedTx(ctx context.Context, withdrawID int64, txHash string) (*WithdrawPayoutResult, error) {
	if err := j.waitReceipt(ctx, txHash); err != nil {
		j.log.Warnf("withdraw %d tx %s receipt pending: %v", withdrawID, txHash, err)
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: withdrawID, TxHash: txHash,
			Error: "tx submitted; awaiting confirmation",
		}, nil
	}
	if err := j.walletRepo.CompleteWithdrawalPayout(ctx, withdrawID, txHash); err != nil {
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: withdrawID, TxHash: txHash, Error: err.Error(),
		}, nil
	}
	return &WithdrawPayoutResult{Processed: true, WithdrawID: withdrawID, TxHash: txHash}, nil
}

func (j *WithdrawPayoutJob) waitReceipt(ctx context.Context, txHash string) error {
	client, err := ethclient.DialContext(ctx, j.cfg.GetRPCURL())
	if err != nil {
		return err
	}
	defer client.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(60 * time.Second)
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	hash := common.HexToHash(txHash)
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return fmt.Errorf("transaction failed on chain")
			}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("receipt timeout")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
