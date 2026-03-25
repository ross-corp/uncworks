package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testConfig(rps float64, burst int) RateLimiterConfig {
	return RateLimiterConfig{
		Enabled:    true,
		RPS:        rps,
		Burst:      burst,
		TTLMinutes: 10,
	}
}

// TestRateLimiterAllow_BurstThenDeny verifies that Allow returns true up to burst,
// then returns false once tokens are exhausted, then recovers after a window.
func TestRateLimiterAllow_BurstThenDeny(t *testing.T) {
	cfg := testConfig(1, 3) // 1 RPS, burst of 3
	rl := NewRateLimiter(cfg)

	ip := "1.2.3.4"

	// Should allow burst times.
	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Fatalf("expected Allow to return true on request %d", i+1)
		}
	}

	// Should now deny.
	if rl.Allow(ip) {
		t.Fatal("expected Allow to return false after burst exhausted")
	}

	// Wait for token replenishment (1 RPS → 1 token after ~1s).
	time.Sleep(1100 * time.Millisecond)

	// Should allow again.
	if !rl.Allow(ip) {
		t.Fatal("expected Allow to return true after recovery")
	}
}

// TestRateLimiterAllow_Remaining verifies Remaining returns a sensible value.
func TestRateLimiterAllow_Remaining(t *testing.T) {
	cfg := testConfig(100, 5)
	rl := NewRateLimiter(cfg)
	ip := "2.2.2.2"

	// Before any request, Remaining should equal burst.
	if got := rl.Remaining(ip); got != 5 {
		t.Fatalf("expected Remaining=5 for unseen IP, got %d", got)
	}

	// After one allow, tokens should be < burst.
	rl.Allow(ip)
	if got := rl.Remaining(ip); got >= 5 {
		t.Fatalf("expected Remaining < 5 after one Allow, got %d", got)
	}
}

// TestRateLimiterTTLEviction verifies that entries are evicted after TTL.
func TestRateLimiterTTLEviction(t *testing.T) {
	cfg := RateLimiterConfig{
		Enabled:    true,
		RPS:        100,
		Burst:      10,
		TTLMinutes: 0, // TTL of 0 minutes → immediate eviction
	}
	rl := NewRateLimiter(cfg)
	ip := "3.3.3.3"

	// Populate an entry.
	rl.Allow(ip)

	// Run a manual sweep (TTL=0 means all entries are stale immediately).
	rl.sweep()

	rl.mu.Lock()
	_, exists := rl.entries[ip]
	rl.mu.Unlock()

	if exists {
		t.Fatal("expected IP entry to be evicted after TTL sweep")
	}
}

// TestRateLimiterTTLEviction_ActiveRetained verifies active IPs survive a sweep.
func TestRateLimiterTTLEviction_ActiveRetained(t *testing.T) {
	cfg := testConfig(100, 10) // TTLMinutes=10
	rl := NewRateLimiter(cfg)
	ip := "4.4.4.4"

	rl.Allow(ip)
	rl.sweep() // TTL is 10 minutes; entry was just seen, so should survive.

	rl.mu.Lock()
	_, exists := rl.entries[ip]
	rl.mu.Unlock()

	if !exists {
		t.Fatal("expected active IP entry to be retained after sweep")
	}
}

// TestRateLimitMiddleware_429WhenDenied verifies that the middleware returns 429
// with appropriate headers when the limiter denies a request.
func TestRateLimitMiddleware_429WhenDenied(t *testing.T) {
	// Config with burst=1: first request allowed, second denied.
	cfg := testConfig(0.001, 1) // very low RPS so second request fails quickly
	rl := NewRateLimiter(cfg)
	mid := RateLimitMiddleware(rl)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := mid(okHandler)

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "5.5.5.5:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// First request: should pass.
	rec := makeReq()
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second request: burst exhausted → 429.
	rec = makeReq()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("expected X-RateLimit-Limit header to be set")
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("expected X-RateLimit-Remaining=0, got %q", rec.Header().Get("X-RateLimit-Remaining"))
	}
}

// TestRateLimitMiddleware_Disabled verifies that a disabled limiter passes all requests.
func TestRateLimitMiddleware_Disabled(t *testing.T) {
	cfg := RateLimiterConfig{
		Enabled:    false,
		RPS:        0.001, // would deny immediately if enabled
		Burst:      1,
		TTLMinutes: 10,
	}
	rl := NewRateLimiter(cfg)
	mid := RateLimitMiddleware(rl)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mid(okHandler)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "6.6.6.6:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 when disabled, got %d", i+1, rec.Code)
		}
	}
}

// TestStripPort verifies port stripping from RemoteAddr.
func TestStripPort(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1.2.3.4:5678", "1.2.3.4"},
		{"[::1]:80", "[::1]"},
		{"192.168.1.1:443", "192.168.1.1"},
		{"noport", "noport"},
	}
	for _, c := range cases {
		if got := stripPort(c.in); got != c.want {
			t.Errorf("stripPort(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestClientIP_TrustProxy verifies X-Forwarded-For extraction.
func TestClientIP_TrustProxy(t *testing.T) {
	cfg := RateLimiterConfig{TrustProxy: true}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	req.RemoteAddr = "172.16.0.1:1234"

	got := clientIP(req, cfg)
	if got != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %q", got)
	}
}

// TestClientIP_NoTrustProxy verifies RemoteAddr is used when trustProxy is false.
func TestClientIP_NoTrustProxy(t *testing.T) {
	cfg := RateLimiterConfig{TrustProxy: false}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "172.16.0.1:1234"

	got := clientIP(req, cfg)
	if got != "172.16.0.1" {
		t.Errorf("expected 172.16.0.1, got %q", got)
	}
}
