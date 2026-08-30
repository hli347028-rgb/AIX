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

// CreditPartnerWin 在单个事务内完成「查用户 → 加 win_recharge_balance → 落流水」。
//
// 对接文档 §2.1/§2.2 的两条硬约束在这里落地：
//   - 三步同事务，不会出现「加了款没记录」或「有记录没加款」
//   - 只要事务提交成功就必须让上层返回 success=true，否则对方会错误退款
func (r *walletRepo) CreditPartnerWin(ctx context.Context, in biz.PartnerCreditInput) (*biz.PartnerCreditResult, error) {
	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	address := strings.TrimSpace(in.Address)
	amount, err := decimal.NewFromString(strings.TrimSpace(in.Amount))
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		return nil, fmt.Errorf("invalid partner credit amount")
	}
	if idempotencyKey == "" || address == "" {
		return nil, fmt.Errorf("invalid partner credit input")
	}

	result := &biz.PartnerCreditResult{}
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 幂等：同一 partner_id+nonce 此前已加过款则直接复用原结果，
		// 不再重复入账，但仍按成功返回——钱确实在用户账上。
		var existing RechargePO
		findErr := tx.Where("tx_hash = ?", idempotencyKey).First(&existing).Error
		if findErr == nil {
			result.Outcome = biz.PartnerCreditDuplicate
			result.RechargeID = existing.ID
			if existing.ConfirmedTime != nil {
				result.CreditedAt = *existing.ConfirmedTime
			} else {
				result.CreditedAt = existing.CreatedTime
			}
			var user UserPO
			if err := tx.First(&user, existing.UserID).Error; err == nil {
				result.NewBalance = user.WinRechargeBalance.String()
			}
			return nil
		}
		if findErr != gorm.ErrRecordNotFound {
			return findErr
		}

		// 验签用原始大小写，查库统一按小写（文档 §3.1）。
		var user UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("LOWER(address) = ?", strings.ToLower(address)).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				result.Outcome = biz.PartnerCreditUserNotFound
				return nil
			}
			return err
		}
		if user.IsFrozen {
			result.Outcome = biz.PartnerCreditUserFrozen
			return nil
		}

		now := time.Now()
		recharge := &RechargePO{
			UserID:        user.ID,
			Asset:         biz.TokenWIN,
			Amount:        amount,
			TxHash:        idempotencyKey,
			FromAddress:   address,
			ToAddress:     user.Address,
			Status:        biz.RechargeStatusConfirmed,
			Message:       fmt.Sprintf("partner_credit:%s:%s", in.PartnerID, in.Nonce),
			ConfirmedTime: &now,
		}
		if err := tx.Create(recharge).Error; err != nil {
			// 并发的同键请求已插入：交给唯一索引裁决，本次不重复加款。
			if isDuplicateKeyErr(err) {
				result.Outcome = biz.PartnerCreditDuplicate
				result.CreditedAt = now
				return nil
			}
			return err
		}

		user.WinRechargeBalance = user.WinRechargeBalance.Add(amount)
		if err := tx.Model(&user).Update("win_recharge_balance", user.WinRechargeBalance).Error; err != nil {
			return err
		}

		result.Outcome = biz.PartnerCreditDone
		result.RechargeID = recharge.ID
		result.NewBalance = user.WinRechargeBalance.String()
		result.CreditedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SumPartnerCreditedSince 统计某合作方自 since 起已成功加款的总额。
// 按 tx_hash 前缀匹配，可以走 tx_hash 唯一索引。
func (r *walletRepo) SumPartnerCreditedSince(ctx context.Context, partnerID string, since time.Time) (string, error) {
	partnerID = strings.TrimSpace(partnerID)
	if partnerID == "" {
		return "0", nil
	}
	var total decimal.NullDecimal
	err := r.data.db.WithContext(ctx).
		Model(&RechargePO{}).
		Where("tx_hash LIKE ?", biz.PartnerIdempotencyPrefix(partnerID)+"%").
		Where("status = ?", biz.RechargeStatusConfirmed).
		Where("created_time >= ?", since).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	if err != nil {
		return "0", err
	}
	if !total.Valid {
		return "0", nil
	}
	return total.Decimal.String(), nil
}
