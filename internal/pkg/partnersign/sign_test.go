package partnersign

import "testing"

// 对接文档 §4.1 给出的对齐向量。这组数据是双方实现一致性的唯一判据，
// 任何改动导致此用例失败都意味着线上会全量签名错误。
//
// 注意这里的 partner_id 是对方文档里的示例值 PT10001，不是我方分配的 AIX10001。
// 它是向量的一部分，改成 AIX10001 会让签名对不上，不要"顺手统一"。
const (
	vectorSecret  = "aix_test_secret_key_0123456789ab"
	vectorPayload = "address=0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a&amount=100.50&nonce=9f1c4a2e7b83&partner_id=PT10001&timestamp=1787875200000"
	vectorSign    = "9cca49754ec1132a38c1eb66dde83f753b05a8812eaf547ec22893dbf44dfc66"
)

func vectorFields() map[string]string {
	return map[string]string{
		"address":    "0x7A9c8B4f2E1d6C3a5F8b0E4D2c9A1B3e6D8F5C7a",
		"amount":     "100.50",
		"partner_id": "PT10001",
		"timestamp":  "1787875200000",
		"nonce":      "9f1c4a2e7b83",
	}
}

func TestPayloadMatchesSpecVector(t *testing.T) {
	if got := Payload(vectorFields()); got != vectorPayload {
		t.Fatalf("payload mismatch\n got: %s\nwant: %s", got, vectorPayload)
	}
}

func TestSignMatchesSpecVector(t *testing.T) {
	if got := Sign(vectorSecret, vectorFields()); got != vectorSign {
		t.Fatalf("sign mismatch\n got: %s\nwant: %s", got, vectorSign)
	}
}

func TestPayloadExcludesSignAndEmptyValues(t *testing.T) {
	fields := vectorFields()
	fields["sign"] = "should-be-ignored"
	fields["extra"] = ""
	if got := Payload(fields); got != vectorPayload {
		t.Fatalf("payload should drop sign and empty values\n got: %s\nwant: %s", got, vectorPayload)
	}
}

// 地址大小写是文档 §3.1 反复强调的坑：验签必须用原始字符串。
func TestSignIsCaseSensitiveOnAddress(t *testing.T) {
	fields := vectorFields()
	fields["address"] = "0x7a9c8b4f2e1d6c3a5f8b0e4d2c9a1b3e6d8f5c7a"
	if Sign(vectorSecret, fields) == vectorSign {
		t.Fatal("lowercasing the address must change the signature")
	}
}

func TestVerify(t *testing.T) {
	if !Verify(vectorSecret, vectorFields(), vectorSign) {
		t.Fatal("valid signature rejected")
	}
	if Verify(vectorSecret, vectorFields(), "") {
		t.Fatal("empty signature accepted")
	}
	if Verify("wrong_secret", vectorFields(), vectorSign) {
		t.Fatal("signature accepted under wrong secret")
	}
	if Verify(vectorSecret, vectorFields(), vectorSign[:len(vectorSign)-1]+"0") {
		t.Fatal("tampered signature accepted")
	}
}

func TestVerifyAnySupportsKeyRotation(t *testing.T) {
	secrets := []string{"old_key_that_no_longer_matches_xx", vectorSecret}
	if !VerifyAny(secrets, vectorFields(), vectorSign) {
		t.Fatal("signature under the new key rejected during rotation window")
	}
	if VerifyAny([]string{"a", "b"}, vectorFields(), vectorSign) {
		t.Fatal("signature accepted under unrelated keys")
	}
	if VerifyAny(nil, vectorFields(), vectorSign) {
		t.Fatal("signature accepted with no configured keys")
	}
	if VerifyAny([]string{"", "  "}, vectorFields(), vectorSign) {
		t.Fatal("blank configured keys must not authenticate anything")
	}
}
