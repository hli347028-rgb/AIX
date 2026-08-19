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
	Processed bool   `json:"processed"`
	Skipped   bool   `json:"skipped,omitempty"`
	Reason    string `json:"reason,omitempty"`
	WithdrawID int64  `json:"withdraw_id,omitempty"`
	TxHash    string `json:"tx_hash,omitempty"`
	Error     string `json:"error,omitempty"`
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
	if j.cfg.GetWithdrawPrivateKey() == "" {
		res.Accepted = false
		res.Reason = "withdraw private key not configured"
		return res
	}
	if j.cfg.GetWinContract() == "" {
		res.Accepted = false
		res.Reason = "win_contract not configured"
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
	privKey := j.cfg.GetWithdrawPrivateKey()
	if privKey == "" {
		return &WithdrawPayoutResult{Skipped: true, Reason: "no private key"}, nil
	}
	winContract := j.cfg.GetWinContract()
	if winContract == "" {
		return &WithdrawPayoutResult{Skipped: true, Reason: "no win contract"}, nil
	}

	if err := j.recoverInProgress(ctx); err != nil {
		j.log.Warnf("recover in-progress withdrawals: %v", err)
	}
	staleBefore := time.Now().Add(-10 * time.Minute)
	if n, err := j.walletRepo.ReleaseStaleDoingWithdrawals(ctx, staleBefore); err != nil {
		j.log.Warnf("release stale doing withdrawals: %v", err)
	} else if n > 0 {
		j.log.Infof("released %d stale doing withdrawals back to pending", n)
	}

	w, err := j.walletRepo.ClaimNextPendingWinWithdrawal(ctx)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return &WithdrawPayoutResult{Processed: false}, nil
	}

	amount, err := decimal.NewFromString(w.NetAmount)
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, "invalid payout amount")
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID,
			Error: fmt.Sprintf("invalid amount: %s", w.NetAmount),
		}, nil
	}

	txHash, err := eth.SendERC20Transfer(
		ctx,
		j.cfg.GetRPCURL(),
		privKey,
		winContract,
		w.ToAddress,
		amount,
		j.cfg.GetWinDecimals(),
	)
	if err != nil {
		_ = j.walletRepo.ReleaseWithdrawalPayout(ctx, w.ID, err.Error())
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID, Error: err.Error(),
		}, nil
	}

	if err := j.walletRepo.SetWithdrawalTxHash(ctx, w.ID, txHash); err != nil {
		j.log.Errorf("withdraw %d tx sent (%s) but save tx_hash failed: %v", w.ID, txHash, err)
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID, TxHash: txHash,
			Error: "tx sent but db update failed; will recover on next cycle",
		}, nil
	}

	if err := j.waitReceipt(ctx, txHash); err != nil {
		j.log.Warnf("withdraw %d tx %s receipt pending: %v", w.ID, txHash, err)
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID, TxHash: txHash,
			Error: "tx submitted; awaiting confirmation",
		}, nil
	}

	if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, txHash); err != nil {
		return &WithdrawPayoutResult{
			Processed: true, WithdrawID: w.ID, TxHash: txHash, Error: err.Error(),
		}, nil
	}
	return &WithdrawPayoutResult{
		Processed: true, WithdrawID: w.ID, TxHash: txHash,
	}, nil
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
		if err := j.waitReceipt(ctx, w.TxHash); err != nil {
			j.log.Infof("withdraw %d tx %s still pending: %v", w.ID, w.TxHash, err)
			continue
		}
		if err := j.walletRepo.CompleteWithdrawalPayout(ctx, w.ID, w.TxHash); err != nil {
			j.log.Warnf("complete recovered withdraw %d: %v", w.ID, err)
		} else {
			j.log.Infof("recovered completed withdraw %d tx=%s", w.ID, w.TxHash)
		}
	}
	return nil
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
