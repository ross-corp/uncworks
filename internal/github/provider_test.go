package github

import (
	"context"
	"strings"
	"testing"
)

func TestPATProvider_ReturnsToken(t *testing.T) {
	p := NewPATProvider("ghp_test123")
	token, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "ghp_test123" {
		t.Errorf("got %q, want %q", token, "ghp_test123")
	}
}

func TestPATProvider_EmptyToken(t *testing.T) {
	p := NewPATProvider("")
	_, err := p.Token(context.Background())
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error = %q, want to contain 'not configured'", err.Error())
	}
}

func TestAppProvider_NotImplemented(t *testing.T) {
	a := &AppProvider{AppID: 12345, InstallationID: 67890}
	_, err := a.Token(context.Background())
	if err == nil {
		t.Fatal("expected error from stub AppProvider")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("error = %q, want to contain 'not yet implemented'", err.Error())
	}
	if !strings.Contains(err.Error(), "12345") {
		t.Errorf("error = %q, want to contain appID", err.Error())
	}
}

func TestInjectTokenInURL_GitHubHTTPS(t *testing.T) {
	got := InjectTokenInURL("https://github.com/org/repo.git", "ghp_abc123")
	want := "https://x-access-token:ghp_abc123@github.com/org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInjectTokenInURL_GenericHTTPS(t *testing.T) {
	got := InjectTokenInURL("https://gitlab.com/org/repo.git", "tok_xyz")
	want := "https://x-access-token:tok_xyz@gitlab.com/org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInjectTokenInURL_NonHTTPS(t *testing.T) {
	// SSH URLs are returned unchanged
	got := InjectTokenInURL("git@github.com:org/repo.git", "ghp_abc123")
	want := "git@github.com:org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInjectTokenInURL_NoSuffix(t *testing.T) {
	got := InjectTokenInURL("https://github.com/org/repo", "ghp_abc123")
	want := "https://x-access-token:ghp_abc123@github.com/org/repo"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
