package biz

import (
	"context"
	"encoding/json"
	"time"

	"backend/internal/conf"
	"backend/internal/pkg/token"

	"github.com/shopspring/decimal"
)

// bumpAixPrice applies the daily growth rate (default +2%) on top of the previous day's price.
func bumpAixPrice(base decimal.Decimal) decimal.Decimal {
	if !base.IsPositive() {
		base = decimal.NewFromFloat(AixPriceInitial)
	}
	if !base.IsPositive() {
		base = decimal.NewFromInt(1)
	}
	growth := decimal.NewFromFloat(AixPriceDailyGrowthRate)
	multiplier := decimal.NewFromInt(1).Add(growth)
	return base.Mul(multiplier).Round(AixPriceDecimals)
}

func previousChinaDate(date string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", date, token.ChinaLocation())
	if err != nil {
		return "", err
	}
	return t.Add(-24 * time.Hour).Format("2006-01-02"), nil
}

func nextChinaDate(date string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", date, token.ChinaLocation())
	if err != nil {
		return "", err
	}
	return t.Add(24 * time.Hour).Format("2006-01-02"), nil
}

// resolvePriceOnDate 返回指定自然日的 AIX 价格（精确匹配，否则取该日之前最近一条）。
func (uc *SettlementUsecase) resolvePriceOnDate(ctx context.Context, date string) (decimal.Decimal, error) {
	if priceStr, err := uc.stakingRepo.GetAixPrice(ctx, date); err != nil {
		return decimal.Zero, err
	} else if priceStr != "" && priceStr != "0" {
		return decimal.NewFromString(priceStr)
	}
	nextDay, err := nextChinaDate(date)
	if err != nil {
		return decimal.Zero, err
	}
	if priceStr, err := uc.stakingRepo.GetLatestAixPriceBefore(ctx, nextDay); err != nil {
		return decimal.Zero, err
	} else if priceStr != "" && priceStr != "0" {
		return decimal.NewFromString(priceStr)
	}
	return decimal.NewFromFloat(AixPriceInitial), nil
}

// ensureAixPriceForSettlement 每日 0 点：写入「今日」自然日价格（昨日价 +2%），静态结算仍用结算日价格。
func (uc *SettlementUsecase) ensureAixPriceForSettlement(ctx context.Context, settlementDate string) (decimal.Decimal, error) {
	today := token.NowChina().Format("2006-01-02")
	yesterday, err := previousChinaDate(today)
	if err != nil {
		return decimal.Zero, err
	}

	base, err := uc.resolvePriceOnDate(ctx, yesterday)
	if err != nil {
		return decimal.Zero, err
	}
	todayPrice := bumpAixPrice(base)
	todayStr := FormatAixPriceDecimal(todayPrice)
	if err := uc.stakingRepo.UpsertAixPrice(ctx, today, todayStr, "daily +2%"); err != nil {
		return decimal.Zero, err
	}
	if err := uc.persistAixPriceInitial(ctx, todayPrice); err != nil {
		uc.log.Warnf("persist aix price after daily bump failed: %v", err)
	}
	uc.log.Infof("aix price for calendar day %s set to %s (base %s from %s, +%.2f%%)",
		today, todayStr, base.String(), yesterday, AixPriceDailyGrowthRate*100)

	settlePrice, err := uc.resolvePriceOnDate(ctx, settlementDate)
	if err != nil {
		return decimal.Zero, err
	}
	if !settlePrice.IsPositive() {
		settlePrice = todayPrice
	}
	if settlementDate != today {
		if existing, _ := uc.stakingRepo.GetAixPrice(ctx, settlementDate); existing == "" {
			settleStr := FormatAixPriceDecimal(settlePrice)
			if err := uc.stakingRepo.UpsertAixPrice(ctx, settlementDate, settleStr, "daily +2%"); err != nil {
				return decimal.Zero, err
			}
		}
	}
	return settlePrice, nil
}

func (uc *SettlementUsecase) persistAixPriceInitial(ctx context.Context, price decimal.Decimal) error {
	f, _ := price.Float64()
	if f <= 0 {
		return nil
	}
	AixPriceInitial = f
	if uc.settingsRepo == nil {
		return nil
	}
	raw, err := uc.settingsRepo.Get(ctx, conf.SettingsKeySystemConfig)
	if err != nil || raw == "" {
		return err
	}
	var snapshot conf.SystemConfigSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return err
	}
	snapshot.AixPriceInitial = f
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return uc.settingsRepo.Set(ctx, conf.SettingsKeySystemConfig, string(data))
}
