package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"backend/internal/biz"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *walletRepo) HasConfirmedWithdrawalPayout(ctx context.Context, withdrawID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&WithdrawalPayoutPO{}).
		Where("withdraw_id = ? AND status = ?", withdrawID, biz.PayoutStatusConfirmed).
		Count(&count).Error
	return count > 0, err
}

func (r *walletRepo) CreateWithdrawalPayoutAttempt(
	ctx context.Context, withdrawID int64, txHash, fromAddress, toAddress, amount string, nonce uint64,
) error {
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return fmt.Errorf("tx hash is empty")
	}
	amt, err := decimal.NewFromString(amount)
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return fmt.Errorf("invalid payout amount")
	}
	fromAddress = strings.ToLower(strings.TrimSpace(fromAddress))
	toAddress = strings.ToLower(strings.TrimSpace(toAddress))

	var existing WithdrawalPayoutPO
	err = r.data.db.WithContext(ctx).Where("tx_hash = ?", txHash).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	po := &WithdrawalPayoutPO{
		WithdrawID:  withdrawID,
		TxHash:      txHash,
		Nonce:       nonce,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Amount:      amt,
		Status:      biz.PayoutStatusSubmitted,
	}
	return r.data.db.WithContext(ctx).Create(po).Error
}

func (r *walletRepo) ListWithdrawalPayoutAttempts(ctx context.Context, withdrawID int64) ([]*biz.WithdrawalPayoutAttempt, error) {
	var list []WithdrawalPayoutPO
	if err := r.data.db.WithContext(ctx).
		Where("withdraw_id = ?", withdrawID).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.WithdrawalPayoutAttempt, 0, len(list))
	for i := range list {
		out = append(out, payoutAttemptToBiz(&list[i]))
	}
	return out, nil
}

func (r *walletRepo) MarkWithdrawalPayoutFailed(ctx context.Context, withdrawID int64, txHash string) error {
	txHash = strings.TrimSpace(txHash)
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPayoutPO{}).
		Where("withdraw_id = ? AND tx_hash = ?", withdrawID, txHash).
		Update("status", biz.PayoutStatusFailed)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return nil
}

func (r *walletRepo) SetWithdrawalPayoutNonce(ctx context.Context, id int64, nonce uint64) error {
	res := r.data.db.WithContext(ctx).Model(&WithdrawalPO{}).
		Where("id = ? AND status = ?", id, biz.WithdrawStatusDoing).
		Update("payout_nonce", nonce)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("withdrawal %d not in doing state", id)
	}
	return nil
}

func (r *walletRepo) ListDoingWithdrawalsWithoutTxHash(ctx context.Context) ([]*biz.Withdrawal, error) {
	var list []WithdrawalPO
	if err := r.data.db.WithContext(ctx).
		Where("asset IN ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '')",
			[]string{biz.TokenWIN, biz.TokenSDT, biz.TokenUSDT}, biz.WithdrawStatusDoing).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return r.withdrawalsWithUserAddress(ctx, list)
}

func (r *walletRepo) ListStaleDoingWinWithdrawals(ctx context.Context, staleBefore time.Time) ([]*biz.Withdrawal, error) {
	var list []WithdrawalPO
	if err := r.data.db.WithContext(ctx).
		Where("asset IN ? AND status = ? AND (tx_hash IS NULL OR tx_hash = '') AND updated_time < ?",
			[]string{biz.TokenWIN, biz.TokenSDT, biz.TokenUSDT}, biz.WithdrawStatusDoing, staleBefore).
		Order("id asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return r.withdrawalsWithUserAddress(ctx, list)
}

func payoutAttemptToBiz(po *WithdrawalPayoutPO) *biz.WithdrawalPayoutAttempt {
	return &biz.WithdrawalPayoutAttempt{
		ID:          po.ID,
		WithdrawID:  po.WithdrawID,
		TxHash:      po.TxHash,
		Nonce:       po.Nonce,
		FromAddress: po.FromAddress,
		ToAddress:   po.ToAddress,
		Amount:      po.Amount.String(),
		Status:      po.Status,
		CreatedAt:   po.CreatedTime,
	}
}

// safeResetWithdrawalForRetry clears a stuck payout only when no confirmed payout exists.
func (r *walletRepo) safeResetWithdrawalForRetry(ctx context.Context, id int64, remark string) error {
	if len(remark) > 255 {
		remark = remark[:255]
	}
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var confirmedCount int64
		if err := tx.Model(&WithdrawalPayoutPO{}).
			Where("withdraw_id = ? AND status = ?", id, biz.PayoutStatusConfirmed).
			Count(&confirmedCount).Error; err != nil {
			return err
		}
		if confirmedCount > 0 {
			return fmt.Errorf("withdrawal %d already has confirmed payout", id)
		}
		res := tx.Model(&WithdrawalPO{}).
			Where("id = ? AND status = ?", id, biz.WithdrawStatusDoing).
			Updates(map[string]interface{}{
				"status":       biz.WithdrawStatusPending,
				"tx_hash":      "",
				"payout_nonce": nil,
				"remark":       remark,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("withdrawal %d not in doing state", id)
		}
		return nil
	})
}

// backfillConfirmedPayoutFromHistory inserts a confirmed payout row for legacy completed withdrawals.
func backfillConfirmedPayoutFromHistory(db *gorm.DB) error {
	var rows []WithdrawalPO
	if err := db.Where("asset = ? AND status = ? AND tx_hash IS NOT NULL AND tx_hash != ''",
		biz.TokenWIN, biz.WithdrawStatusCompleted).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var count int64
		if err := db.Model(&WithdrawalPayoutPO{}).Where("withdraw_id = ? AND status = ?",
			row.ID, biz.PayoutStatusConfirmed).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		var dup int64
		if err := db.Model(&WithdrawalPayoutPO{}).Where("tx_hash = ?", row.TxHash).Count(&dup).Error; err != nil {
			return err
		}
		if dup > 0 {
			if err := db.Model(&WithdrawalPayoutPO{}).
				Where("tx_hash = ?", row.TxHash).
				Update("status", biz.PayoutStatusConfirmed).Error; err != nil {
				return err
			}
			continue
		}
		po := &WithdrawalPayoutPO{
			WithdrawID:  row.ID,
			TxHash:      row.TxHash,
			ToAddress:   strings.ToLower(row.ToAddress),
			Amount:      row.PayAmount,
			Status:      biz.PayoutStatusConfirmed,
		}
		if row.PayoutNonce != nil {
			po.Nonce = *row.PayoutNonce
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(po).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureWithdrawalPayoutGuards(db *gorm.DB) error {
	if err := backfillConfirmedPayoutFromHistory(db); err != nil {
		return err
	}
	var colCount int64
	if err := db.Raw(`
		SELECT COUNT(1)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'withdrawal_payouts'
		  AND column_name = 'confirmed_withdraw_id'
	`).Scan(&colCount).Error; err != nil {
		return err
	}
	if colCount > 0 {
		return nil
	}
	return db.Exec(`
		ALTER TABLE withdrawal_payouts
		ADD COLUMN confirmed_withdraw_id BIGINT
			GENERATED ALWAYS AS (CASE WHEN status = 'confirmed' THEN withdraw_id ELSE NULL END) STORED,
		ADD UNIQUE INDEX uk_withdrawal_payout_confirmed (confirmed_withdraw_id)
	`).Error
}
