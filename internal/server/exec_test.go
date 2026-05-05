package server

import (
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	allowed := []string{"https://app.example.com", "http://localhost:3000"}
	h := NewExecHandler(nil, nil, "test-ns", allowed)

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"empty origin passes (non-browser client)", "", true},
		{"exact allowed origin", "https://app.example.com", true},
		{"case-insensitive match", "HTTPS://APP.EXAMPLE.COM", true},
		{"second allowed origin", "http://localhost:3000", true},
		{"unlisted origin blocked", "https://evil.com", false},
		{"subdomain blocked", "https://sub.app.example.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := h.checkOrigin(r); got != tc.want {
				t.Errorf("checkOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}

func TestCheckOriginWildcard(t *testing.T) {
	h := NewExecHandler(nil, nil, "test-ns", []string{"*"})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://any.domain.com")
	if !h.checkOrigin(r) {
		t.Error("wildcard should allow any origin")
	}
}

func TestCheckOriginNoAllowed(t *testing.T) {
	h := NewExecHandler(nil, nil, "test-ns", nil)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	if h.checkOrigin(r) {
		t.Error("no allowed origins should block all origins")
	}
}
