package client

import (
	"testing"
)

func TestParseIP4PRecord(t *testing.T) {
	cases := []struct {
		name     string
		record   string
		wantHost string
		wantPort int
		wantOK   bool
	}{
		{"base64 standard", "MjAzLjAuMTEzLjc6MTgyNDY=", "203.0.113.7", 18246, true},
		{"base64 raw (no padding)", "MjAzLjAuMTEzLjc6MTgyNDY", "203.0.113.7", 18246, true},
		{"plain text", "203.0.113.7:18246", "203.0.113.7", 18246, true},
		{"quoted and padded", "  \"MjAzLjAuMTEzLjc6MTgyNDY=\"  ", "203.0.113.7", 18246, true},
		{"empty", "", "", 0, false},
		{"garbage", "not-an-ip4p-record", "", 0, false},
		{"non-ip host", "MjAzLjAuMTEzLm5vdGlwOjgw", "", 0, false},
		{"port out of range", "203.0.113.7:99999", "", 0, false},
		{"missing port", "203.0.113.7", "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host, port, ok := parseIP4PRecord(c.record)
			if ok != c.wantOK || host != c.wantHost || port != c.wantPort {
				t.Fatalf("parseIP4PRecord(%q) = (%q, %d, %v), want (%q, %d, %v)",
					c.record, host, port, ok, c.wantHost, c.wantPort, c.wantOK)
			}
		})
	}
}

func TestLookupIP4P_Fallback(t *testing.T) {
	// 不存在的域名应回退到普通模式，保持原 addr/port。
	addr, port := lookupIP4P(
		t.Context(), "no-such-host-ip4p-test.invalid", 7000)
	if addr != "no-such-host-ip4p-test.invalid" || port != 7000 {
		t.Fatalf("回退模式异常: %s:%d", addr, port)
	}
}
