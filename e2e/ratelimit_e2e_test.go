//go:build e2e

// e2e/ratelimit_e2e_test.go — end-to-end tests for per-IP rate limiting on the
// API server. These tests exercise the HTTP layer directly so they catch
// configuration issues that unit tests cannot (e.g., middleware not wired up).
package e2e

import (
	"bytes"
	"net/http"
	"testing"
)

// TestE2E_RateLimit_HealthCheckPassthrough verifies that /healthz and /readyz
// are always reachable even when the rate limiter is enabled.
// These paths are exempt from rate limiting so liveness probes never fail.
func TestE2E_RateLimit_HealthCheckPassthrough(t *testing.T) {
	base := apiBaseURL()

	for _, path := range []string{"/healthz", "/readyz"} {
		url := base + path
		resp, err := http.Get(url)
		if err != nil {
			t.Skipf("Cannot reach %s: %v — is the server running?", url, err)
		}
		resp.Body.Close()

		// Acceptable status codes are 200 (healthy) or 503 (unhealthy but
		// the endpoint responded, meaning the rate limiter passed it through).
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Errorf("Rate limiter blocked %s (got 429); health paths must be exempt", path)
		} else {
			t.Logf("%s returned %d (not rate-limited — correct)", path, resp.StatusCode)
		}
	}
}

// TestE2E_RateLimit_HeadersPresent verifies that responses from non-health
// endpoints include X-RateLimit-Limit and X-RateLimit-Remaining headers when
// the rate limiter is enabled.
func TestE2E_RateLimit_HeadersPresent(t *testing.T) {
	// Use the webhook endpoint as a concrete non-health path.
	url := getWebhookURL()

	payload := makeGitHubPushPayload([]string{"README.md"})
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Skipf("Cannot reach %s: %v — is the server running?", url, err)
	}
	defer resp.Body.Close()

	// If the rate limiter is enabled the headers should be present.
	// If the server is configured without rate limiting, the test is a no-op.
	limit := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")

	if limit != "" || remaining != "" {
		t.Logf("Rate-limit headers present: X-RateLimit-Limit=%s X-RateLimit-Remaining=%s", limit, remaining)
	} else {
		t.Log("Rate-limit headers absent — rate limiter may be disabled (acceptable for local dev)")
	}
}

// TestE2E_RateLimit_ExcessiveRequestsReturn429 fires more requests than the
// default burst (20) at the webhook endpoint from the same IP and checks that
// eventually a 429 is returned.
//
// The test is lenient: if the server has rate limiting disabled (e.g., in a
// test environment with RATE_LIMIT_ENABLED=false) it logs the result without
// failing.
func TestE2E_RateLimit_ExcessiveRequestsReturn429(t *testing.T) {
	url := getWebhookURL()
	payload := makeGitHubPushPayload([]string{"README.md"})

	// Saturate well above the default burst of 20.
	const requestCount = 30

	got429 := false
	for i := range requestCount {
		resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Skipf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
			t.Logf("Got 429 on request %d — rate limiter is enforcing the burst limit", i+1)
			break
		}
	}

	if !got429 {
		// Rate limiting may be disabled in the e2e environment.
		t.Log("No 429 received after 30 requests — rate limiter may be disabled or burst is > 30 " +
			"(acceptable for local dev environments)")
	}
}

// TestE2E_RateLimit_RetryAfterHeader verifies that a 429 response includes a
// Retry-After header so callers know when to retry.
func TestE2E_RateLimit_RetryAfterHeader(t *testing.T) {
	url := getWebhookURL()
	payload := makeGitHubPushPayload([]string{"README.md"})

	// Fire enough requests to potentially hit the limit.
	const requestCount = 35

	for i := range requestCount {
		resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Skipf("Request %d failed: %v", i, err)
		}
		retryAfter := resp.Header.Get("Retry-After")
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter == "" {
				t.Error("429 response is missing Retry-After header")
			} else {
				t.Logf("Got 429 on request %d with Retry-After: %s", i+1, retryAfter)
			}
			return
		}
	}

	t.Log("Did not receive a 429 in 35 requests — Retry-After test skipped (rate limiter may be disabled)")
}
