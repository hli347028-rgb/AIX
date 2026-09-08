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

// CreateExchangeTransfer 扣 points 并落库；nonce 字段存 WinBit 幂等键 request_no。
func (r *walletRepo) CreateExchangeTransfer(ctx context.Context, userID int64, address, amount, nonce string) (*biz.ExchangeTransfer, string, error) {
	amt, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil || !amt.GreaterThan(decimal.Zero) {
		return nil, "", fmt.Errorf("invalid amount")
	}
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return nil, "", fmt.Errorf("request_no required")
	}
	var created *biz.ExchangeTransfer
	var left string
	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
			return err
		}
		if u.Points.LessThan(amt) {
			return fmt.Errorf("insufficient points")
		}
		u.Points = u.Points.Sub(amt)
		if err := tx.Model(&u).Update("points", u.Points).Error; err != nil {
			return err
		}
		addr := strings.TrimSpace(address)
		if addr == "" {
			addr = u.Address
		}
		po := &ExchangeTransferPO{
			UserID:  userID,
			Address: addr,
			Asset:   biz.TokenSDT,
			Amount:  amt,
			Status:  "pending",
			Nonce:   nonce,
			Remark:  "AIX-USDT transfer to exchange",
		}
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		left = u.Points.String()
		created = exchangeTransferToBiz(po)
		return nil
	})
	return created, left, err
}

func (r *walletRepo) CompleteExchangeTransfer(ctx context.Context, id int64, partnerTxnID, partnerCode string) error {
	partnerTxnID = strings.TrimSpace(partnerTxnID)
	partnerCode = strings.TrimSpace(partnerCode)
	updates := map[string]any{
		"status":         "completed",
		"partner_txn_id": partnerTxnID,
		"partner_code":   partnerCode,
		"remark":         "exchange transfer completed",
	}
	res := r.data.db.WithContext(ctx).Model(&ExchangeTransferPO{}).
		Where("id = ? AND status = ?", id, "pending").Updates(updates)
	if res.Error != nil {
		// 写状态失败时仍尽量留下对方单号，避免后续被误退款。
		_ = r.attachExchangeTransferPartner(ctx, id, partnerTxnID, partnerCode)
		return res.Error
	}
	if res.RowsAffected == 0 {
		_ = r.data.db.WithContext(ctx).Model(&ExchangeTransferPO{}).Where("id = ?", id).Updates(map[string]any{
			"partner_txn_id": partnerTxnID,
			"partner_code":   partnerCode,
		}).Error
	}
	return nil
}

// attachExchangeTransferPartner 在仍为 pending 时写入对方单号，避免「对方已成功但本地未完结」时被误退款。
func (r *walletRepo) attachExchangeTransferPartner(ctx context.Context, id int64, partnerTxnID, partnerCode string) error {
	return r.data.db.WithContext(ctx).Model(&ExchangeTransferPO{}).
		Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{
		"partner_txn_id": strings.TrimSpace(partnerTxnID),
		"partner_code":   strings.TrimSpace(partnerCode),
	}).Error
}

func (r *walletRepo) FailAndRefundExchangeTransfer(ctx context.Context, id int64, partnerCode, remark string) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po ExchangeTransferPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error; err != nil {
			return err
		}
		if po.Status != "pending" {
			return nil
		}
		var u UserPO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, po.UserID).Error; err != nil {
			return err
		}
		u.Points = u.Points.Add(po.Amount)
		if err := tx.Model(&u).Update("points", u.Points).Error; err != nil {
			return err
		}
		if strings.TrimSpace(remark) == "" {
			remark = "exchange transfer failed; points refunded"
		}
		return tx.Model(&po).Updates(map[string]any{
			"status":       "failed",
			"partner_code": strings.TrimSpace(partnerCode),
			"remark":       truncateRemark(remark, 512),
		}).Error
	})
}

// ResolveStuckExchangeTransfers 收尾卡住的 pending：
// - 已有对方单号 → 标为成功（对方已入账，不可退款）
// - 否则且超过 olderThan → 失败并退回 points
// userID>0 时仅处理该用户；olderThan<=0 时处理全部无对方单号的 pending。
func (r *walletRepo) ResolveStuckExchangeTransfers(ctx context.Context, userID int64, olderThan time.Duration) (completed, refunded int, err error) {
	db := r.data.db.WithContext(ctx).Model(&ExchangeTransferPO{}).Where("status = ?", "pending")
	if userID > 0 {
		db = db.Where("user_id = ?", userID)
	}
	if olderThan > 0 {
		db = db.Where("created_time <= ?", time.Now().Add(-olderThan))
	}
	var list []ExchangeTransferPO
	if err = db.Order("id asc").Find(&list).Error; err != nil {
		return 0, 0, err
	}
	for i := range list {
		po := &list[i]
		if strings.TrimSpace(po.PartnerTxnID) != "" {
			if e := r.CompleteExchangeTransfer(ctx, po.ID, po.PartnerTxnID, po.PartnerCode); e != nil {
				return completed, refunded, e
			}
			completed++
			continue
		}
		remark := "stuck pending auto-failed; points refunded"
		if e := r.FailAndRefundExchangeTransfer(ctx, po.ID, po.PartnerCode, remark); e != nil {
			return completed, refunded, e
		}
		refunded++
	}
	return completed, refunded, nil
}

func (r *walletRepo) ListExchangeTransfersByUser(ctx context.Context, userID int64) ([]*biz.ExchangeTransfer, error) {
	var list []ExchangeTransferPO
	// 用户侧只展示成功/失败，不展示处理中。
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{"completed", "failed"}).
		Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ExchangeTransfer, 0, len(list))
	for i := range list {
		out = append(out, exchangeTransferToBiz(&list[i]))
	}
	return out, nil
}

func exchangeTransferToBiz(po *ExchangeTransferPO) *biz.ExchangeTransfer {
	if po == nil {
		return nil
	}
	return &biz.ExchangeTransfer{
		ID:           po.ID,
		UserID:       po.UserID,
		Address:      po.Address,
		Asset:        po.Asset,
		Amount:       po.Amount.String(),
		Status:       po.Status,
		Nonce:        po.Nonce,
		PartnerTxnID: po.PartnerTxnID,
		PartnerCode:  po.PartnerCode,
		Remark:       po.Remark,
		CreatedTime:  po.CreatedTime,
		UpdatedTime:  po.UpdatedTime,
	}
}

func truncateRemark(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
