package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
)

func TestNewIPExtractor(t *testing.T) {
	request := func(remoteAddr string, xff string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		return req
	}

	direct, err := newIPExtractor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := direct(request("203.0.113.7:4000", "198.51.100.1")); got != "203.0.113.7" {
		t.Errorf("without trusted proxies X-Forwarded-For must be ignored, got %s", got)
	}
	// Even from a private address (echo's default trusts those).
	if got := direct(request("10.0.0.2:4000", "198.51.100.1")); got != "10.0.0.2" {
		t.Errorf("without trusted proxies X-Forwarded-For must be ignored, got %s", got)
	}

	proxied, err := newIPExtractor([]string{"10.0.0.0/8", "192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		remote, xff, want string
	}{
		{"10.1.2.3:4000", "198.51.100.1", "198.51.100.1"},               // via a trusted proxy
		{"192.0.2.10:4000", "198.51.100.1", "198.51.100.1"},             // single trusted IP
		{"10.1.2.3:4000", "6.6.6.6, 198.51.100.1", "198.51.100.1"},      // spoofed entry before the real client
		{"203.0.113.7:4000", "198.51.100.1", "203.0.113.7"},             // not from a proxy
		{"172.16.0.5:4000", "198.51.100.1", "172.16.0.5"},               // private, but not configured
		{"10.1.2.3:4000", "", "10.1.2.3"},                               // no header
		{"10.1.2.3:4000", "192.0.2.10, 198.51.100.1", "198.51.100.1"},   // two hops
		{"192.0.2.11:4000", "198.51.100.1", "192.0.2.11"},               // neighbour of the trusted IP
		{"10.1.2.3:4000", "not-an-ip, 198.51.100.1", "198.51.100.1"},    // garbage further out is never reached
		{"10.1.2.3:4000", "198.51.100.1, 10.9.9.9", "198.51.100.1"},     // trusted inner proxy
		{"[2001:db8::1]:4000", "198.51.100.1", "2001:db8::1"},           // IPv6 client, not trusted
		{"10.1.2.3:4000", "2001:db8::2", "2001:db8::2"},                 // IPv6 behind the proxy
		{"192.0.2.10:4000", "198.51.100.1, 203.0.113.9", "203.0.113.9"}, // nearest untrusted hop wins
	}
	for _, tt := range tests {
		if got := proxied(request(tt.remote, tt.xff)); got != tt.want {
			t.Errorf("remote %s, XFF %q: got %s; want %s", tt.remote, tt.xff, got, tt.want)
		}
	}

	if _, err := newIPExtractor([]string{"nonsense"}); err == nil {
		t.Error("expected an error for an invalid entry")
	}
	if _, err := newIPExtractor([]string{"10.0.0.0/33"}); err == nil {
		t.Error("expected an error for an invalid CIDR")
	}
}

// The rate limiter must run before basic auth, so password guessing is
// limited; /health stays reachable.
func TestRateLimiterRunsBeforeBasicAuth(t *testing.T) {
	original := *config.CONFIG
	t.Cleanup(func() { *config.CONFIG = original })
	config.CONFIG.BasicAuthConfig = config.BasicAuthConfig{Username: "user", Password: "secret"}
	config.CONFIG.RateLimiterConfig = config.RateLimiterConfig{Rate: 0.001, Burst: 2, ExpiresInMinutes: 1}
	config.CONFIG.TrustedProxies = nil

	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	do := func(path string, remote string, xff string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = remote
		req.SetBasicAuth("user", "wrong")
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := 0; i < 2; i++ {
		if code := do("/api/v1/jobs/x", "203.0.113.7:4000", ""); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d; want 401", i+1, code)
		}
	}
	if code := do("/api/v1/jobs/x", "203.0.113.7:4000", ""); code != http.StatusTooManyRequests {
		t.Errorf("third attempt: got %d; want 429", code)
	}
	// A forged X-Forwarded-For doesn't buy a fresh budget.
	if code := do("/api/v1/jobs/x", "203.0.113.7:4000", "198.51.100.99"); code != http.StatusTooManyRequests {
		t.Errorf("forged X-Forwarded-For: got %d; want 429", code)
	}
	if code := do("/health", "203.0.113.7:4000", ""); code != http.StatusOK {
		t.Errorf("/health: got %d; want 200", code)
	}
	if code := do("/api/v1/jobs/x", "203.0.113.8:4000", ""); code != http.StatusUnauthorized {
		t.Errorf("another client: got %d; want 401", code)
	}
}
