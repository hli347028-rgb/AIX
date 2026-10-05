package biz

import "testing"

func TestParseDownlinePointsSource(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "", want: "", ok: true},
		{raw: "USDT", want: PointsSourceRecharge, ok: true},
		{raw: "recharge", want: PointsSourceRecharge, ok: true},
		{raw: "win", want: PointsSourceWin, ok: true},
		{raw: "reward", want: PointsSourceReinvest, ok: true},
		{raw: "transfer_reinvest", want: PointsSourceReinvest, ok: true},
		{raw: "reward_legacy", want: PointsSourceReinvest, ok: true},
		{raw: "other", ok: false},
	}
	for _, tc := range cases {
		got, ok := ParseDownlinePointsSource(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("ParseDownlinePointsSource(%q) = %q, %v; want %q, %v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDownlinePointsCategory(t *testing.T) {
	if got := DownlinePointsCategory(PointsSourceWin); got != PointsSourceWin {
		t.Fatalf("win category = %s", got)
	}
	if got := DownlinePointsCategory(PointsSourceRewardLegacy); got != PointsSourceReinvest {
		t.Fatalf("legacy category = %s", got)
	}
	if got := DownlinePointsCategory(PointsSourceTransferReinvest); got != PointsSourceReinvest {
		t.Fatalf("reinvest category = %s", got)
	}
	if got := DownlinePointsCategory(""); got != PointsSourceRecharge {
		t.Fatalf("empty category = %s", got)
	}
}
