// Copyright 2022 hev, r@hev.cc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"encoding/base64"
	"net"
	"strconv"
	"strings"

	"github.com/fatedier/frp/pkg/util/xlog"
)

// parseIP4PRecord parses a single DNS TXT record value into an IP address and
// port. Accepted forms (in order of attempt):
//
//   - Base64 encoded "IPv4:port" (Std / RawStd / URL / RawURL), e.g.
//     MjAzLjAuMTEzLjc6MTgyNDY= -> 203.0.113.7:18246
//   - Plain-text "IPv4:port"
//
// Surrounding quotes and whitespace are tolerated because some resolvers keep
// them. Invalid values, non-IP hosts and out-of-range ports are rejected.
func parseIP4PRecord(record string) (string, int, bool) {
	v := strings.TrimSpace(record)
	v = strings.TrimSpace(strings.Trim(v, "\""))
	if v == "" {
		return "", 0, false
	}

	candidates := []string{v}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if decoded, err := enc.DecodeString(v); err == nil {
			candidates = append(candidates, strings.TrimSpace(string(decoded)))
		}
	}

	for _, candidate := range candidates {
		host, portStr, err := net.SplitHostPort(candidate)
		if err != nil {
			continue
		}
		if net.ParseIP(strings.Trim(host, "[]")) == nil {
			continue
		}
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			continue
		}
		return host, p, true
	}
	return "", 0, false
}

// lookupIP4P resolves server_addr into the real IPv4 address and port advertised
// by a NAT-traversal / P2P peer. Three strategies are tried in order:
//
//  1. TXT record (used by natmap / lucky / DDNS-style deployments, and by the
//     qaz741wsd856/frp fork): the domain's TXT record holds a Base64-encoded
//     "IPv4:port" string. Plain-text TXT values are also accepted.
//  2. AAAA record carrying an IP4P-encoded IPv6 address (2001:0000::/16):
//     port is in bytes 10-11 and the IPv4 address in bytes 12-15.
//  3. Neither found: return addr/port unchanged, i.e. normal frp behaviour.
//
// On success the resolved port overrides the configured serverPort, which is
// what makes IP4P work with dynamically allocated public ports.
func lookupIP4P(ctx context.Context, addr string, port int) (string, int) {
	xl := xlog.FromContextSafe(ctx)

	// 1) TXT record -> Base64("IPv4:port")
	if records, err := net.DefaultResolver.LookupTXT(ctx, addr); err == nil {
		for _, rec := range records {
			if host, p, ok := parseIP4PRecord(rec); ok {
				xl.Infof("ip4p: resolved server address from TXT record, %s -> %s:%d", addr, host, p)
				return host, p
			}
		}
	} else {
		xl.Debugf("ip4p: no usable TXT record for %s: %v", addr, err)
	}

	// 2) AAAA record -> IP4P-encoded IPv6 (2001:0000::/16)
	if ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", addr); err == nil {
		for _, ip := range ips {
			if len(ip) == 16 &&
				ip[0] == 0x20 && ip[1] == 0x01 &&
				ip[2] == 0x00 && ip[3] == 0x00 {
				resolvedAddr := net.IPv4(ip[12], ip[13], ip[14], ip[15]).String()
				resolvedPort := int(ip[10])<<8 | int(ip[11])
				xl.Infof("ip4p: resolved server address from IP4P AAAA record, %s -> %s:%d",
					addr, resolvedAddr, resolvedPort)
				return resolvedAddr, resolvedPort
			}
		}
	} else {
		xl.Debugf("ip4p: no usable IP4P AAAA record for %s: %v", addr, err)
	}

	// 3) fall back to normal mode
	xl.Debugf("ip4p: no IP4P record found for %s, using %s:%d (normal mode)", addr, addr, port)
	return addr, port
}
