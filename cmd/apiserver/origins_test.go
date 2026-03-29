package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseAllowedOrigins_EmptyDefaultsToWildcard(t *testing.T) {
	origins := parseAllowedOrigins("")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("empty input should default to [\"*\"], got %v", origins)
	}
}

func TestParseAllowedOrigins_ExplicitWildcard(t *testing.T) {
	origins := parseAllowedOrigins("*")
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("\"*\" input should return [\"*\"], got %v", origins)
	}
}

func TestParseAllowedOrigins_CommaSeparated(t *testing.T) {
	origins := parseAllowedOrigins("https://app.example.com, https://staging.example.com")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d: %v", len(origins), origins)
	}
	if origins[0] != "https://app.example.com" {
		t.Errorf("origins[0] = %q, want %q", origins[0], "https://app.example.com")
	}
	if origins[1] != "https://staging.example.com" {
		t.Errorf("origins[1] = %q, want %q", origins[1], "https://staging.example.com")
	}
}

func TestParseAllowedOrigins_SingleOrigin(t *testing.T) {
	origins := parseAllowedOrigins("https://app.example.com")
	if len(origins) != 1 || origins[0] != "https://app.example.com" {
		t.Errorf("single origin should return [\"https://app.example.com\"], got %v", origins)
	}
}

func TestParseAllowedOrigins_SkipsEmptyEntries(t *testing.T) {
	origins := parseAllowedOrigins("https://a.com,,, https://b.com,")
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins (empty entries skipped), got %d: %v", len(origins), origins)
	}
}

func TestParseAllowedOrigins_NoHardcodedLocalhost(t *testing.T) {
	// Verify that when the env var is empty, no hardcoded localhost origins appear.
	origins := parseAllowedOrigins("")
	for _, o := range origins {
		if o != "*" {
			t.Errorf("empty input should only contain \"*\", found %q", o)
		}
	}
}

func TestIsOriginAllowed_WildcardAllowsAll(t *testing.T) {
	allowed := []string{"*"}
	tests := []string{
		"http://localhost:3000",
		"https://app.example.com",
		"http://192.168.1.1:8080",
	}
	for _, origin := range tests {
		if !isOriginAllowed(origin, allowed) {
			t.Errorf("wildcard should allow %q", origin)
		}
	}
}

func TestIsOriginAllowed_ExactMatch(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://staging.example.com"}
	if !isOriginAllowed("https://app.example.com", allowed) {
		t.Error("exact match should be allowed")
	}
	if isOriginAllowed("https://evil.com", allowed) {
		t.Error("non-matching origin should be rejected")
	}
}

func TestIsOriginAllowed_CaseInsensitive(t *testing.T) {
	allowed := []string{"https://App.Example.COM"}
	if !isOriginAllowed("https://app.example.com", allowed) {
		t.Error("origin matching should be case-insensitive")
	}
}

// --- withAuth tests ---

func okHandler(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func TestWithAuth_NoKey_PassesThrough(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("empty apiKey should pass all requests through; got %d", rec.Code)
	}
}

func TestWithAuth_NoToken_Returns401(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "secret")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing token should return 401; got %d", rec.Code)
	}
}

func TestWithAuth_WrongToken_Returns401(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "correct")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token should return 401; got %d", rec.Code)
	}
}

func TestWithAuth_ValidBearerToken_Passes(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "correct")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer correct")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid Bearer token should pass; got %d", rec.Code)
	}
}

func TestWithAuth_QueryParamToken_Passes(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "mykey")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token=mykey", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("query-param token should pass; got %d", rec.Code)
	}
}

func TestWithAuth_HealthzExempt(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "secret")
	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("path %s should bypass auth; got %d", path, rec.Code)
		}
	}
}

func TestWithAuth_GRPCPathsExempt(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "secret")
	for _, path := range []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.reflection.v1.ServerReflection/Info",
		"/grpc.reflection.v1alpha.ServerReflection/Info",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("gRPC path %s should bypass auth; got %d", path, rec.Code)
		}
	}
}

func TestWithAuth_WebhookExempt(t *testing.T) {
	h := withAuth(http.HandlerFunc(okHandler), "secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("webhook path should bypass Bearer auth; got %d", rec.Code)
	}
}

// --- env helper tests ---

func TestEnvIntOrDefault_UsesDefault(t *testing.T) {
	os.Unsetenv("TEST_INT_KEY_UNUSED")
	if got := envIntOrDefault("TEST_INT_KEY_UNUSED", 42); got != 42 {
		t.Errorf("envIntOrDefault with unset key = %d, want 42", got)
	}
}

func TestEnvIntOrDefault_ReadsEnv(t *testing.T) {
	t.Setenv("TEST_INT_KEY", "7")
	if got := envIntOrDefault("TEST_INT_KEY", 42); got != 7 {
		t.Errorf("envIntOrDefault with set key = %d, want 7", got)
	}
}

func TestEnvIntOrDefault_InvalidFallsBack(t *testing.T) {
	t.Setenv("TEST_INT_INVALID", "notanumber")
	if got := envIntOrDefault("TEST_INT_INVALID", 99); got != 99 {
		t.Errorf("envIntOrDefault with invalid value = %d, want 99", got)
	}
}

func TestEnvFloatOrDefault_UsesDefault(t *testing.T) {
	os.Unsetenv("TEST_FLOAT_UNUSED")
	if got := envFloatOrDefault("TEST_FLOAT_UNUSED", 3.14); got != 3.14 {
		t.Errorf("envFloatOrDefault with unset key = %f, want 3.14", got)
	}
}

func TestEnvFloatOrDefault_ReadsEnv(t *testing.T) {
	t.Setenv("TEST_FLOAT_KEY", "2.71")
	if got := envFloatOrDefault("TEST_FLOAT_KEY", 0); got != 2.71 {
		t.Errorf("envFloatOrDefault with set key = %f, want 2.71", got)
	}
}

func TestEnvOrDefault_UsesDefault(t *testing.T) {
	os.Unsetenv("TEST_STR_UNUSED")
	if got := envOrDefault("TEST_STR_UNUSED", "fallback"); got != "fallback" {
		t.Errorf("envOrDefault with unset key = %q, want %q", got, "fallback")
	}
}

func TestEnvOrDefault_ReadsEnv(t *testing.T) {
	t.Setenv("TEST_STR_KEY", "hello")
	if got := envOrDefault("TEST_STR_KEY", "fallback"); got != "hello" {
		t.Errorf("envOrDefault with set key = %q, want %q", got, "hello")
	}
}
