package data

import (
	"context"
	"fmt"
	"strings"

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
	return r.data.db.WithContext(ctx).Model(&ExchangeTransferPO{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{
		"status":         "completed",
		"partner_txn_id": strings.TrimSpace(partnerTxnID),
		"partner_code":   strings.TrimSpace(partnerCode),
		"remark":         "exchange transfer completed",
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

func (r *walletRepo) ListExchangeTransfersByUser(ctx context.Context, userID int64) ([]*biz.ExchangeTransfer, error) {
	var list []ExchangeTransferPO
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error; err != nil {
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
