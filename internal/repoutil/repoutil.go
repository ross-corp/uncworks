// Package repoutil provides shared utility functions for repository URL handling.
package repoutil

import (
	"net/url"
	"path/filepath"
	"strings"
)

// NameFromURL extracts the repository name from a URL.
// Handles HTTPS ("https://github.com/org/repo.git") and bare names ("repo-name").
// Returns empty string for empty input.
func NameFromURL(repoURL string) string {
	if repoURL == "" {
		return ""
	}
	if u, err := url.Parse(repoURL); err == nil && u.Path != "" && u.Path != "/" {
		base := filepath.Base(u.Path)
		if base != "." && base != "/" {
			return strings.TrimSuffix(base, ".git")
		}
	}
	base := filepath.Base(repoURL)
	return strings.TrimSuffix(base, ".git")
}
