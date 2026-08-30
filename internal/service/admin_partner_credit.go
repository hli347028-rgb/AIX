package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"backend/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 交易所划转记录（合作方转账加款接口 POST /v1/transfer/credit）。
//
// 这些加款没有独立的表，而是复用 recharges，靠 tx_hash 的
// "partner:{partner_id}:{nonce}" 前缀与普通充值区分（见 biz.PartnerIdempotencyKey）。
// 列表因此全部建立在该前缀上，切勿改成按 asset 或 message 匹配。

type partnerCreditRow struct {
	ID          int64
	Address     string
	Amount      decimal.Decimal
	TxHash      string
	CreatedTime time.Time
}

func (s *AdminLegacyService) partnerCreditListDB(ctx context.Context, q url.Values) *gorm.DB {
	db := s.data.DB().WithContext(ctx).
		Table("recharges r").
		Select(`r.id, COALESCE(NULLIF(r.from_address,''), u.address) as address,
			r.amount, r.tx_hash, r.created_time`).
		Joins("JOIN users u ON u.id = r.user_id").
		Where("r.status = ?", biz.RechargeStatusConfirmed).
		Where("r.tx_hash LIKE ?", "partner:%")

	if address := strings.TrimSpace(q.Get("address")); address != "" {
		db = db.Where("(r.from_address LIKE ? OR u.address LIKE ?)", "%"+address+"%", "%"+address+"%")
	}
	// 空下拉时 antd 会把字面量 "undefined"/"null" 发上来，必须清洗掉，
	// 否则会被当成真实的 partner_id 而过滤出空列表。
	partnerID := strings.TrimSpace(firstNonEmpty(q.Get("partner_id"), q.Get("partnerId")))
	if lower := strings.ToLower(partnerID); lower == "undefined" || lower == "null" {
		partnerID = ""
	}
	if partnerID != "" {
		db = db.Where("r.tx_hash LIKE ?", biz.PartnerIdempotencyPrefix(partnerID)+"%")
	}
	start, end := parseLegacyTimeRange(q)
	if start != nil {
		db = db.Where("r.created_time >= ?", *start)
	}
	if end != nil {
		db = db.Where("r.created_time <= ?", *end)
	}
	return db
}

func (s *AdminLegacyService) partnerCreditStats(ctx context.Context, q url.Values) (map[string]interface{}, error) {
	var row struct {
		TotalCount  int64
		AmountTotal decimal.Decimal
	}
	err := s.partnerCreditListDB(ctx, q).
		Select("COUNT(*) as total_count, COALESCE(SUM(r.amount),0) as amount_total").
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"totalCount":  row.TotalCount,
		"amountTotal": row.AmountTotal.String(),
	}, nil
}

// sumPartnerCreditWin 统计所有合作方（交易所）划转进来的 WIN 总量。
// 加款记录只增不删，因此该值单调递增。
func (s *AdminLegacyService) sumPartnerCreditWin(ctx context.Context) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := s.data.DB().WithContext(ctx).Table("recharges").
		Where("status = ?", biz.RechargeStatusConfirmed).
		Where("tx_hash LIKE ?", "partner:%").
		Select("COALESCE(SUM(amount),0)").Scan(&total).Error
	if err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

// HandlePartnerCreditList GET /api/admin_dhb/partner_credit_list
func (s *AdminLegacyService) HandlePartnerCreditList(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)

	stats, err := s.partnerCreditStats(ctx, q)
	if err != nil {
		return errors.InternalServer("DB_ERROR", "查询失败")
	}

	var rows []partnerCreditRow
	if err := s.partnerCreditListDB(ctx, q).
		Order("r.id desc").Limit(pageSize).Offset(offset).
		Scan(&rows).Error; err != nil {
		return errors.InternalServer("DB_ERROR", "查询失败")
	}

	list := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		partnerID, nonce := splitPartnerTxHash(r.TxHash)
		list = append(list, map[string]interface{}{
			"id":        r.ID,
			"partnerId": partnerID,
			"nonce":     nonce,
			"address":   r.Address,
			"asset":     biz.TokenWIN,
			"amount":    r.Amount.String(),
			"aixTxnId":  biz.FormatAixTxnID(r.ID, r.CreatedTime),
			"createdAt": formatLegacyTime(r.CreatedTime),
		})
	}

	total := int64(0)
	if v, ok := stats["totalCount"].(int64); ok {
		total = v
	}
	return ctx.Result(200, map[string]interface{}{
		"list":     list,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
		"stats":    stats,
	})
}

// HandlePartnerCreditPartners GET /api/admin_dhb/partner_credit_partners
// 返回已配置的合作方列表，供前端筛选下拉使用。
func (s *AdminLegacyService) HandlePartnerCreditPartners(ctx khttp.Context) error {
	if err := s.requireAdmin(ctx); err != nil {
		return err
	}
	list := make([]map[string]interface{}, 0)
	if s.partnerCfg != nil {
		for _, p := range s.partnerCfg.Partners {
			list = append(list, map[string]interface{}{
				"partnerId": p.PartnerID,
				"enabled":   p.Enabled,
			})
		}
	}
	return ctx.Result(200, map[string]interface{}{"partners": list})
}

// splitPartnerTxHash 从 "partner:{partner_id}:{nonce}" 中拆出两段。
// nonce 本身不含冒号，但用 SplitN 保证即使含也归到 nonce 一侧。
func splitPartnerTxHash(txHash string) (partnerID, nonce string) {
	rest := strings.TrimPrefix(strings.TrimSpace(txHash), "partner:")
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return rest, ""
	}
	return parts[0], parts[1]
}
