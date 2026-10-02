package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ClientIP trusts only the entries the configured proxies appended, from the
// right of X-Forwarded-For; whatever a client puts to their left is ignored.
func TestClientIP(t *testing.T) {
	testCaseList := []struct {
		name              string
		remoteAddr        string
		headerList        []string
		trustedProxyCount int
		want              string
	}{
		{"no header, count 0", "10.0.0.5:5555", nil, 0, "10.0.0.5"},
		{"no header, count 1", "127.0.0.1:5555", nil, 1, "127.0.0.1"},
		{
			"count 0 ignores the header", "127.0.0.1:5555",
			[]string{"203.0.113.9"}, 0, "127.0.0.1",
		},
		{
			"one entry, count 1", "127.0.0.1:5555",
			[]string{"203.0.113.9"}, 1, "203.0.113.9",
		},
		{
			"forged leading entries, count 1", "127.0.0.1:5555",
			[]string{"1.1.1.1, 2.2.2.2, 203.0.113.9"}, 1, "203.0.113.9",
		},
		{
			"forged entry in an earlier header line, count 1",
			"127.0.0.1:5555",
			[]string{"1.1.1.1", "203.0.113.9"}, 1, "203.0.113.9",
		},
		{
			"count 2 takes the second from the right", "127.0.0.1:5555",
			[]string{"1.1.1.1, 203.0.113.9, 10.0.0.2"}, 2, "203.0.113.9",
		},
		{
			"header shorter than count falls back", "127.0.0.1:5555",
			[]string{"203.0.113.9"}, 2, "127.0.0.1",
		},
		{
			"hop that is not an address falls back", "127.0.0.1:5555",
			[]string{"203.0.113.9, unknown"}, 1, "127.0.0.1",
		},
		{
			"port stripped from v4 hop", "127.0.0.1:5555",
			[]string{"203.0.113.9:4711"}, 1, "203.0.113.9",
		},
		{
			"port stripped from v6 hop", "127.0.0.1:5555",
			[]string{"[2001:db8::1]:4711"}, 1, "2001:db8::1",
		},
		{
			"bare v6 hop", "127.0.0.1:5555",
			[]string{"2001:db8::1"}, 1, "2001:db8::1",
		},
		{"v6 RemoteAddr", "[::1]:5555", nil, 0, "::1"},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = testCase.remoteAddr
			for _, value := range testCase.headerList {
				r.Header.Add("X-Forwarded-For", value)
			}
			got := ClientIP(r, testCase.trustedProxyCount)
			if got != testCase.want {
				t.Errorf("ClientIP = %q, want %q", got, testCase.want)
			}
		})
	}
}
