package biz

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestBumpAixPrice(t *testing.T) {
	base := decimal.NewFromFloat(1)
	got := bumpAixPrice(base)
	want := decimal.NewFromFloat(1.02)
	if !got.Equal(want) {
		t.Fatalf("bumpAixPrice(1) = %s, want %s", got, want)
	}

	got2 := bumpAixPrice(decimal.NewFromFloat(2))
	want2 := decimal.NewFromFloat(2.04)
	if !got2.Equal(want2) {
		t.Fatalf("bumpAixPrice(2) = %s, want %s", got2, want2)
	}
}

func TestAixPriceTenDays(t *testing.T) {
	price := decimal.NewFromFloat(1)
	t.Logf("第0天（初始）: %s", price.StringFixed(AixPriceDecimals))
	for day := 1; day <= 10; day++ {
		price = bumpAixPrice(price)
		t.Logf("第%d天: %s", day, price.StringFixed(AixPriceDecimals))
	}
}
