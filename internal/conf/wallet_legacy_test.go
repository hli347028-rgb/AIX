package conf

import "testing"

func TestLegacyUSDTDepositContractsDefault(t *testing.T) {
	got := (*WalletConfig)(nil).GetLegacyUSDTDepositContracts()
	if len(got) != 1 || got[0] != DefaultLegacyUSDTDepositContract {
		t.Fatalf("default = %#v", got)
	}
	got = (&WalletConfig{}).GetLegacyUSDTDepositContracts()
	if len(got) != 1 || got[0] != DefaultLegacyUSDTDepositContract {
		t.Fatalf("empty = %#v", got)
	}
	got = (&WalletConfig{LegacyUSDTDepositContracts: []string{" 0x1111111111111111111111111111111111111111 "}}).GetLegacyUSDTDepositContracts()
	if len(got) != 1 || got[0] != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("configured = %#v", got)
	}
}
