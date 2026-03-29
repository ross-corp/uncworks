//go:build regression

// test/regression/rate_limiting_test.go — Tests that the rate limiter
// correctly returns 429 responses when a client exceeds the configured
// threshold. Uses the server.RateLimiter and RateLimitMiddleware directly.
package regression

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uncworks/aot/internal/server"
)

// okHandler is a trivial handler that always returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// sendRequests fires n sequential requests against handler from the given IP
// and returns the slice of HTTP status codes received.
func sendRequests(handler http.Handler, n int, ip string) []int {
	codes := make([]int, 0, n)
	for i := 0; i < n; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	return codes
}

// countCode returns how many entries in codes equal target.
func countCode(codes []int, target int) int {
	n := 0
	for _, c := range codes {
		if c == target {
			n++
		}
	}
	return n
}

// TestRateLimiting_429AfterThreshold verifies that after exhausting the burst
// the middleware returns 429 Too Many Requests.
func TestRateLimiting_429AfterThreshold(t *testing.T) {
	// Burst of 2 at near-zero RPS so tokens never replenish during the test.
	cfg := server.RateLimiterConfig{
		Enabled:    true,
		RPS:        0.0001,
		Burst:      2,
		TTLMinutes: 10,
	}
	rl := server.NewRateLimiter(cfg)
	handler := server.RateLimitMiddleware(rl)(okHandler)

	codes := sendRequests(handler, 10, "10.0.0.1")

	got200 := countCode(codes, http.StatusOK)
	got429 := countCode(codes, http.StatusTooManyRequests)

	require.Equal(t, 2, got200,
		"exactly burst=2 requests should succeed before rate limiting kicks in")
	require.Equal(t, 8, got429,
		"the remaining 8 requests should be rate-limited (429)")
}

// TestRateLimiting_HeadersOnThrottled verifies that 429 responses include the
// required Retry-After and X-RateLimit-* headers.
func TestRateLimiting_HeadersOnThrottled(t *testing.T) {
	cfg := server.RateLimiterConfig{
		Enabled:    true,
		RPS:        0.0001,
		Burst:      1,
		TTLMinutes: 10,
	}
	rl := server.NewRateLimiter(cfg)
	handler := server.RateLimitMiddleware(rl)(okHandler)

	// Exhaust the single burst token.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Next request should be throttled with proper headers.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:9999"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"),
		"throttled response must include Retry-After")
	require.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"),
		"throttled response must include X-RateLimit-Limit")
	require.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"),
		"X-RateLimit-Remaining should be 0 when throttled")
}

// TestRateLimiting_PerIPIsolation verifies that rate limiting is applied
// per-IP: exhausting one IP's tokens does not affect a different IP.
func TestRateLimiting_PerIPIsolation(t *testing.T) {
	cfg := server.RateLimiterConfig{
		Enabled:    true,
		RPS:        0.0001,
		Burst:      2,
		TTLMinutes: 10,
	}
	rl := server.NewRateLimiter(cfg)
	handler := server.RateLimitMiddleware(rl)(okHandler)

	// Exhaust IP-A's burst.
	codesA := sendRequests(handler, 5, "10.0.0.10")
	got429A := countCode(codesA, http.StatusTooManyRequests)
	require.Greater(t, got429A, 0, "IP-A should get throttled after burst exhaustion")

	// IP-B should still have a fresh bucket.
	codesB := sendRequests(handler, 2, "10.0.0.20")
	got200B := countCode(codesB, http.StatusOK)
	require.Equal(t, 2, got200B,
		"IP-B should have its own full burst; first 2 requests must succeed")
}

// TestRateLimiting_DisabledPassesThrough verifies that when the rate limiter is
// disabled all requests are passed through regardless of volume.
func TestRateLimiting_DisabledPassesThrough(t *testing.T) {
	cfg := server.RateLimiterConfig{
		Enabled:    false,
		RPS:        0.0001, // would deny immediately if enabled
		Burst:      1,
		TTLMinutes: 10,
	}
	rl := server.NewRateLimiter(cfg)
	handler := server.RateLimitMiddleware(rl)(okHandler)

	codes := sendRequests(handler, 20, "10.0.0.30")
	got429 := countCode(codes, http.StatusTooManyRequests)

	require.Zero(t, got429, "disabled rate limiter must not return any 429s")
}
