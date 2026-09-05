package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 用户端「向交易所划转」AIX-USDT 记录（表 exchange_transfers）。

type exchangeTransferAdminRow struct {
	ID           int64
	UserID       int64
	Address      string
	Asset        string
	Amount       decimal.Decimal
	Status       string
	Nonce        string
	PartnerTxnID string
	PartnerCode  string
	Remark       string
	CreatedTime  time.Time
}

func (s *AdminLegacyService) exchangeTransferListDB(ctx context.Context, q url.Values) *gorm.DB {
	db := s.data.DB().WithContext(ctx).Table("exchange_transfers et")
	if address := strings.TrimSpace(q.Get("address")); address != "" {
		db = db.Where("et.address LIKE ?", "%"+address+"%")
	}
	status := strings.TrimSpace(q.Get("status"))
	if lower := strings.ToLower(status); lower == "undefined" || lower == "null" {
		status = ""
	}
	if status != "" {
		db = db.Where("et.status = ?", status)
	}
	start, end := parseLegacyTimeRange(q)
	if start != nil {
		db = db.Where("et.created_time >= ?", *start)
	}
	if end != nil {
		db = db.Where("et.created_time <= ?", *end)
	}
	return db
}

func (s *AdminLegacyService) exchangeTransferStats(ctx context.Context, q url.Values) (map[string]interface{}, error) {
	var row struct {
		TotalCount     int64
		AmountTotal    decimal.Decimal
		CompletedTotal decimal.Decimal
		CompletedCount int64
		FailedCount    int64
	}
	err := s.exchangeTransferListDB(ctx, q).
		Select(`COUNT(*) as total_count,
			COALESCE(SUM(et.amount),0) as amount_total,
			COALESCE(SUM(CASE WHEN et.status = 'completed' THEN et.amount ELSE 0 END),0) as completed_total,
			COALESCE(SUM(CASE WHEN et.status = 'completed' THEN 1 ELSE 0 END),0) as completed_count,
			COALESCE(SUM(CASE WHEN et.status = 'failed' THEN 1 ELSE 0 END),0) as failed_count`).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"totalCount":     row.TotalCount,
		"amountTotal":    row.AmountTotal.String(),
		"completedTotal": row.CompletedTotal.String(),
		"completedCount": row.CompletedCount,
		"failedCount":    row.FailedCount,
	}, nil
}

// HandleExchangeTransferList GET /api/admin_dhb/exchange_transfer_list
func (s *AdminLegacyService) HandleExchangeTransferList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)

	stats, err := s.exchangeTransferStats(ctx, q)
	if err != nil {
		return errors.InternalServer("DB_ERROR", "查询失败")
	}

	var total int64
	if err := s.exchangeTransferListDB(ctx, q).Count(&total).Error; err != nil {
		return errors.InternalServer("DB_ERROR", "查询失败")
	}

	var rows []exchangeTransferAdminRow
	if err := s.exchangeTransferListDB(ctx, q).
		Select(`et.id, et.user_id, et.address, et.asset, et.amount, et.status, et.nonce,
			et.partner_txn_id, et.partner_code, et.remark, et.created_time`).
		Order("et.id desc").Limit(pageSize).Offset(offset).
		Scan(&rows).Error; err != nil {
		return errors.InternalServer("DB_ERROR", "查询失败")
	}

	list := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		asset := strings.TrimSpace(r.Asset)
		if asset == "" || strings.EqualFold(asset, "SDT") {
			asset = "AIX-USDT"
		}
		list = append(list, map[string]interface{}{
			"id":           r.ID,
			"userId":       r.UserID,
			"address":      r.Address,
			"asset":        asset,
			"amount":       r.Amount.String(),
			"status":       r.Status,
			"requestNo":    r.Nonce,
			"partnerTxnId": r.PartnerTxnID,
			"partnerCode":  r.PartnerCode,
			"remark":       r.Remark,
			"createdAt":    formatLegacyTime(r.CreatedTime),
		})
	}

	return ctx.Result(200, map[string]interface{}{
		"list":     list,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
		"stats":    stats,
	})
}
