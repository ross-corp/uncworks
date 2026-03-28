// Package github provides GitHub authentication token management.
package github

import (
	"context"
	"encoding/base64"
	"fmt"
)

// TokenProvider returns a valid GitHub API token.
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

// PATProvider returns a static personal access token.
type PATProvider struct {
	token string
}

// NewPATProvider creates a PATProvider with the given token.
func NewPATProvider(token string) *PATProvider {
	return &PATProvider{token: token}
}

// Token returns the configured personal access token.
func (p *PATProvider) Token(_ context.Context) (string, error) {
	if p.token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN not configured")
	}
	return p.token, nil
}

// BasicAuthHeader returns the value for an HTTP Authorization header that
// authenticates git over HTTPS using a GitHub token without embedding it in
// the remote URL. The returned string has the form "Basic <base64>".
func BasicAuthHeader(token string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return "Basic " + encoded
}

// InjectTokenInURL embeds a token into a GitHub HTTPS URL for authenticated git operations.
// Example: https://github.com/org/repo.git -> https://x-access-token:TOKEN@github.com/org/repo.git
func InjectTokenInURL(repoURL, token string) string {
	const prefix = "https://github.com/"
	if len(repoURL) > len(prefix) && repoURL[:len(prefix)] == prefix {
		return "https://x-access-token:" + token + "@github.com/" + repoURL[len(prefix):]
	}
	// Fallback: generic https:// URL
	const httpsPrefix = "https://"
	if len(repoURL) > len(httpsPrefix) && repoURL[:len(httpsPrefix)] == httpsPrefix {
		return httpsPrefix + "x-access-token:" + token + "@" + repoURL[len(httpsPrefix):]
	}
	return repoURL
}
