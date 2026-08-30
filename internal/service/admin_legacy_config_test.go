package service

import (
	"testing"

	"backend/internal/conf"
)

func TestLegacyMgmtThresholdConfigValue(t *testing.T) {
	snapshot := &conf.SystemConfigSnapshot{
		MgmtThresholds: conf.DefaultMgmtThresholds(),
		MgmtRates:      conf.DefaultMgmtRates(),
	}

	if got := legacyConfigValue(snapshot, nil, 21); got != "5000" {
		t.Fatalf("W1 threshold value=%s, want 5000", got)
	}
	if got := legacyConfigValue(snapshot, nil, 30); got != "30000000" {
		t.Fatalf("W10 threshold value=%s, want 30000000", got)
	}
}

func TestApplyLegacyMgmtThresholdConfig(t *testing.T) {
	snapshot := &conf.SystemConfigSnapshot{
		MgmtThresholds: conf.DefaultMgmtThresholds(),
		MgmtRates:      conf.DefaultMgmtRates(),
	}

	if err := applyLegacyConfigUpdate(snapshot, nil, 21, "6000"); err != nil {
		t.Fatalf("update W1 threshold: %v", err)
	}
	if snapshot.MgmtThresholds[0] != 6000 {
		t.Fatalf("W1 threshold=%v, want 6000", snapshot.MgmtThresholds[0])
	}
}

func TestLegacyPartnerLimitConfigValue(t *testing.T) {
	snapshot := &conf.SystemConfigSnapshot{
		PartnerMinAmount:  "10",
		PartnerMaxAmount:  "100000",
		PartnerDailyLimit: "1000000",
	}
	if got := legacyConfigValue(snapshot, nil, 38); got != "10" {
		t.Fatalf("min=%s, want 10", got)
	}
	if got := legacyConfigValue(snapshot, nil, 39); got != "100000" {
		t.Fatalf("max=%s, want 100000", got)
	}
	if got := legacyConfigValue(snapshot, nil, 40); got != "1000000" {
		t.Fatalf("daily=%s, want 1000000", got)
	}
	// 空快照回落到默认
	if got := legacyConfigValue(nil, nil, 38); got != conf.DefaultPartnerMinAmount {
		t.Fatalf("default min=%s, want %s", got, conf.DefaultPartnerMinAmount)
	}
}

func TestApplyLegacyPartnerLimitConfig(t *testing.T) {
	snapshot := &conf.SystemConfigSnapshot{
		PartnerMinAmount:  conf.DefaultPartnerMinAmount,
		PartnerMaxAmount:  conf.DefaultPartnerMaxAmount,
		PartnerDailyLimit: conf.DefaultPartnerDailyLimit,
	}
	if err := applyLegacyConfigUpdate(snapshot, nil, 38, "20"); err != nil {
		t.Fatalf("update min: %v", err)
	}
	if snapshot.PartnerMinAmount != "20" {
		t.Fatalf("min=%s, want 20", snapshot.PartnerMinAmount)
	}
	if err := applyLegacyConfigUpdate(snapshot, nil, 39, "50000"); err != nil {
		t.Fatalf("update max: %v", err)
	}
	if snapshot.PartnerMaxAmount != "50000" {
		t.Fatalf("max=%s, want 50000", snapshot.PartnerMaxAmount)
	}
}

func TestApplyLegacyPartnerLimitRequiresOrdering(t *testing.T) {
	tests := []struct {
		name  string
		id    int
		value string
	}{
		{name: "not positive", id: 38, value: "0"},
		{name: "not numeric", id: 39, value: "abc"},
		{name: "min above max", id: 38, value: "200000"},
		{name: "max above daily", id: 39, value: "2000000"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := &conf.SystemConfigSnapshot{
				PartnerMinAmount:  conf.DefaultPartnerMinAmount,
				PartnerMaxAmount:  conf.DefaultPartnerMaxAmount,
				PartnerDailyLimit: conf.DefaultPartnerDailyLimit,
			}
			if err := applyLegacyConfigUpdate(snapshot, nil, test.id, test.value); err == nil {
				t.Fatalf("expected validation error for id=%d value=%s", test.id, test.value)
			}
		})
	}
}

func TestSplitPartnerTxHash(t *testing.T) {
	partnerID, nonce := splitPartnerTxHash("partner:AIX10001:9f1c4a2e7b83")
	if partnerID != "AIX10001" || nonce != "9f1c4a2e7b83" {
		t.Fatalf("got partner=%s nonce=%s", partnerID, nonce)
	}
	partnerID, nonce = splitPartnerTxHash("partner:AIX10001:a:b:c")
	if partnerID != "AIX10001" || nonce != "a:b:c" {
		t.Fatalf("SplitN keep rest: partner=%s nonce=%s", partnerID, nonce)
	}
}

