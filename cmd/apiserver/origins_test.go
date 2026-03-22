package main

import (
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
