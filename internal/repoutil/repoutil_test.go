package repoutil

import "testing"

func TestNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/repo.git", "repo"},
		{"https://github.com/org/repo", "repo"},
		{"git@github.com:org/repo.git", "repo"},
		{"repo-name", "repo-name"},
		{"https://github.com/example/repo/", "repo"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := NameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("NameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
