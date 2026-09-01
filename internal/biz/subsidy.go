package biz

import (
	"github.com/shopspring/decimal"
)

// Community subsidy tiers (percent): 5 / 10 / 15.
const (
	SubsidyRateMin     int32 = 5
	SubsidyRateMid     int32 = 10
	SubsidyRateMax     int32 = 15
	ZeroAccountLegacyPct int32 = 10 // 历史 0 号账户比例，合并进补贴后逐步废弃
)

// MergeZeroAccountIntoSubsidyRate 将原 0 号账户 10% 并入社区补贴档位（上限 15%）。
// 仅 0 号 → 10%；0 号 + 社区补贴（含历史无档位列、等价 5%+10%）→ 15%；0 号 + 已设档位 → 原档位 +10。
func MergeZeroAccountIntoSubsidyRate(currentRate int32, hasSubsidy bool) int32 {
	if !hasSubsidy {
		return SubsidyRateMid
	}
	if currentRate <= 0 {
		// 历史双开：0 号 10% + 社区补贴 5%，合并为 15%
		return SubsidyRateMax
	}
	rate := currentRate + ZeroAccountLegacyPct
	if rate > SubsidyRateMax {
		rate = SubsidyRateMax
	}
	return rate
}

// EffectiveSubsidyRatePercent returns the user's subsidy tier percent (0 if none).
// 须后台明确设置 5/10/15，无档位时不默认 5%。
func EffectiveSubsidyRatePercent(isCommunitySubsidy bool, subsidyRate int32) int32 {
	if !isCommunitySubsidy || subsidyRate <= 0 {
		return 0
	}
	if subsidyRate != SubsidyRateMin && subsidyRate != SubsidyRateMid && subsidyRate != SubsidyRateMax {
		return 0
	}
	return subsidyRate
}

func EffectiveSubsidyRateDecimal(isCommunitySubsidy bool, subsidyRate int32) decimal.Decimal {
	pct := EffectiveSubsidyRatePercent(isCommunitySubsidy, subsidyRate)
	if pct <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt32(pct).Div(decimal.NewFromInt(100))
}

// SubsidyGapRate returns the differential rate an ancestor earns on a downline USDT recharge.
func SubsidyGapRate(ancestorPct, highestLowerPct int32) decimal.Decimal {
	if ancestorPct <= highestLowerPct {
		return decimal.Zero
	}
	return decimal.NewFromInt32(ancestorPct - highestLowerPct).Div(decimal.NewFromInt(100))
}
