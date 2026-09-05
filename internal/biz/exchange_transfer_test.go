package biz

import (
	"testing"

	"backend/internal/pkg/partnersign"
)

func TestSignAixInboundMatchesPartnersign(t *testing.T) {
	fields := map[string]string{
		"partner_id": "AIX10001",
		"address":    "0xa795166bd44f970571199cfa9e8711a15b589dfa",
		"amount":     "10.03",
		"request_no": "REQ-20260904-0001",
		"timestamp":  "1788462203000",
		"sign":       "ignored",
	}
	secret := "doc-example-secret"
	want := partnersign.Sign(secret, fields)
	if got := signAixInbound(secret, fields); got != want {
		t.Fatalf("sign mismatch\n got: %s\nwant: %s", got, want)
	}
}
