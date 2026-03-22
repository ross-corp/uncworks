package github

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AppProvider mints installation tokens from a GitHub App private key.
// Stub -- full implementation deferred to multi-user milestone.
type AppProvider struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
	mu             sync.Mutex
	cached         string
	expiresAt      time.Time
}

// Token returns a GitHub App installation token.
// Currently returns an error as this provider is not yet implemented.
func (a *AppProvider) Token(_ context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Now().Before(a.expiresAt) {
		return a.cached, nil
	}
	// TODO: JWT -> installation token exchange
	return "", fmt.Errorf("GitHub App token provider not yet implemented (appID=%d)", a.AppID)
}
