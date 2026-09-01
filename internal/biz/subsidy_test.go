package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMergeZeroAccountIntoSubsidyRate(t *testing.T) {
	tests := []struct {
		name       string
		current    int32
		hasSubsidy bool
		want       int32
	}{
		{name: "zero only", hasSubsidy: false, want: 10},
		{name: "zero + subsidy 5", current: 5, hasSubsidy: true, want: 15},
		{name: "zero + subsidy 10", current: 10, hasSubsidy: true, want: 15},
		{name: "zero + subsidy 15", current: 15, hasSubsidy: true, want: 15},
		{name: "zero + subsidy no tier", current: 0, hasSubsidy: true, want: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeZeroAccountIntoSubsidyRate(tt.current, tt.hasSubsidy)
			if got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}

func TestEffectiveSubsidyRatePercent(t *testing.T) {
	tests := []struct {
		name    string
		subsidy bool
		rate    int32
		want    int32
	}{
		{name: "none", want: 0},
		{name: "subsidy no tier", subsidy: true, want: 0},
		{name: "subsidy 5", subsidy: true, rate: 5, want: 5},
		{name: "subsidy 10", subsidy: true, rate: 10, want: 10},
		{name: "subsidy 15", subsidy: true, rate: 15, want: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveSubsidyRatePercent(tt.subsidy, tt.rate)
			if got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}

func TestSubsidyDifferentialWalk(t *testing.T) {
	chain := []struct {
		pct  int32
		subs bool
	}{
		{pct: 10, subs: true},
		{0, false},
		{10, true},
		{0, false},
		{15, true},
	}
	amount := decimal.NewFromInt(1000)
	highest := EffectiveSubsidyRatePercent(chain[0].subs, chain[0].pct)
	payouts := map[int]decimal.Decimal{}
	for i := 1; i < len(chain); i++ {
		node := chain[i]
		pct := EffectiveSubsidyRatePercent(node.subs, node.pct)
		gap := SubsidyGapRate(pct, highest)
		if gap.IsPositive() {
			payouts[i] = amount.Mul(gap)
		}
		if pct > highest {
			highest = pct
		}
	}
	if !payouts[2].IsZero() {
		t.Fatalf("c should get 0 on same level as e")
	}
	if payouts[4].String() != "50" {
		t.Fatalf("a should get 50, got %s", payouts[4])
	}

	chain2 := []struct {
		pct  int32
		subs bool
	}{
		{pct: 15, subs: true},
		{0, false},
		{10, true},
		{0, false},
		{15, true},
	}
	highest = EffectiveSubsidyRatePercent(chain2[0].subs, chain2[0].pct)
	total := decimal.Zero
	for i := 1; i < len(chain2); i++ {
		node := chain2[i]
		pct := EffectiveSubsidyRatePercent(node.subs, node.pct)
		gap := SubsidyGapRate(pct, highest)
		if gap.IsPositive() {
			total = total.Add(amount.Mul(gap))
		}
		if pct > highest {
			highest = pct
		}
	}
	if !total.IsZero() {
		t.Fatalf("expected 0 payout when recharger is 15%%, got %s", total)
	}
}
