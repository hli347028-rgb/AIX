// Package partnersign implements the HMAC-SHA256 request/response signing
// scheme agreed with the transfer-credit partner.
//
// Rules (see 对接文档 §4):
//  1. take all top-level fields except "sign"
//  2. drop fields whose value is null or the empty string
//  3. sort field names in ascending ASCII order
//  4. join as key1=value1&key2=value2, using raw values without URL encoding,
//     quoting, or case conversion
//  5. sign = lowercase hex of HMAC_SHA256(secret_key, joined)
package partnersign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Payload builds the canonical string that gets signed. Exported so that
// signature mismatches can be diagnosed without reimplementing the rules.
func Payload(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k, v := range fields {
		// 空值必须剔除而不是签成 k=，否则与对方实现不一致。
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(fields[k])
	}
	return b.String()
}

// Sign returns the lowercase hex HMAC-SHA256 of the canonical payload.
func Sign(secret string, fields map[string]string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(Payload(fields)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether got matches the expected signature for fields.
// Comparison is constant time: a plain == leaks how many leading bytes
// matched, which is enough to brute-force a signature byte by byte.
func Verify(secret string, fields map[string]string, got string) bool {
	if got == "" {
		return false
	}
	want := Sign(secret, fields)
	return hmac.Equal([]byte(want), []byte(strings.ToLower(strings.TrimSpace(got))))
}

// VerifyAny reports whether got matches under any of the supplied secrets.
// Multiple secrets support zero-downtime key rotation: during the overlap
// window both the old and the new key are accepted.
//
// Every secret is checked even after a match so that the total work does not
// depend on which key matched.
func VerifyAny(secrets []string, fields map[string]string, got string) bool {
	matched := false
	for _, s := range secrets {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if Verify(s, fields, got) {
			matched = true
		}
	}
	return matched
}
