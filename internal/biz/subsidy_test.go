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
	// chain[0]=recharger；充值人自身档位不占用起点（highest 从 0 起）。
	chain := []struct {
		pct  int32
		subs bool
	}{
		{pct: 10, subs: true}, // recharger
		{0, false},
		{10, true}, // first ancestor with tier: full 10%
		{0, false},
		{15, true}, // gap 5%
	}
	amount := decimal.NewFromInt(1000)
	highest := int32(0)
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
	if payouts[2].String() != "100" {
		t.Fatalf("first 10%% ancestor should get 100, got %s", payouts[2])
	}
	if payouts[4].String() != "50" {
		t.Fatalf("15%% ancestor should get 50 gap, got %s", payouts[4])
	}

	// A(15%) 直推 B(15%)：B 充值时 A 应拿满 15%。
	chain2 := []struct {
		pct  int32
		subs bool
	}{
		{pct: 15, subs: true}, // B recharger
		{15, true},            // A
	}
	highest = 0
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
	if total.String() != "150" {
		t.Fatalf("direct upline same 15%% should get 150, got %s", total)
	}

	// C 充值，中间 B(15%)，再上 A(15%)：A 仍被中间同档阻断。
	chain3 := []struct {
		pct  int32
		subs bool
	}{
		{pct: 0, subs: false}, // C
		{15, true},            // B
		{15, true},            // A
	}
	highest = 0
	payouts3 := map[int]decimal.Decimal{}
	for i := 1; i < len(chain3); i++ {
		node := chain3[i]
		pct := EffectiveSubsidyRatePercent(node.subs, node.pct)
		gap := SubsidyGapRate(pct, highest)
		if gap.IsPositive() {
			payouts3[i] = amount.Mul(gap)
		}
		if pct > highest {
			highest = pct
		}
	}
	if payouts3[1].String() != "150" {
		t.Fatalf("middle 15%% should get 150, got %s", payouts3[1])
	}
	if !payouts3[2].IsZero() {
		t.Fatalf("top same-tier after middle block should get 0, got %s", payouts3[2])
	}
}
