//go:build darwin

// github_auth.go — GitHub OAuth device flow for the UNCWORKS desktop app.
// Uses the gh CLI's public client_id to avoid registering a separate OAuth App.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	// ghClientID is the gh CLI's public OAuth client_id (device flow only).
	ghClientID   = "178c6fc778ccc68e1d6a"
	ghScope      = "repo,read:org"
	keychainSvc  = "uncworks"
	keychainAcct = "github-token"
)

// DeviceFlowStart holds the response from GitHub's device authorization endpoint.
type DeviceFlowStart struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// StartGitHubDeviceFlow initiates the device flow and returns the user-facing code and URL.
func (a *App) StartGitHubDeviceFlow() (*DeviceFlowStart, error) {
	resp, err := http.PostForm("https://github.com/login/device/code", url.Values{
		"client_id": {ghClientID},
		"scope":     {ghScope},
	})
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// GitHub returns form-encoded OR JSON depending on Accept header; try JSON first.
	var result DeviceFlowStart
	if err := json.Unmarshal(body, &result); err != nil {
		// Fallback: form-encoded response.
		vals, _ := url.ParseQuery(string(body))
		result.DeviceCode = vals.Get("device_code")
		result.UserCode = vals.Get("user_code")
		result.VerificationURI = vals.Get("verification_uri")
	}
	if result.DeviceCode == "" {
		return nil, fmt.Errorf("empty device_code from GitHub")
	}
	if result.Interval == 0 {
		result.Interval = 5
	}
	return &result, nil
}

// DeviceFlowPollResult is returned by PollGitHubDeviceFlow.
type DeviceFlowPollResult struct {
	Done  bool   `json:"done"`
	Token string `json:"token,omitempty"`
}

// PollGitHubDeviceFlow polls for the OAuth token once.
// Returns {done:true, token:...} when authorised, {done:false} when still pending.
func (a *App) PollGitHubDeviceFlow(deviceCode string) (*DeviceFlowPollResult, error) {
	resp, err := http.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":   {ghClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Try JSON first, then form-encoded.
	var raw struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		vals, _ := url.ParseQuery(string(body))
		raw.AccessToken = vals.Get("access_token")
		raw.Error = vals.Get("error")
	}

	switch raw.Error {
	case "authorization_pending", "slow_down":
		return &DeviceFlowPollResult{Done: false}, nil
	case "expired_token":
		return nil, fmt.Errorf("device code expired — please restart authorization")
	case "access_denied":
		return nil, fmt.Errorf("access denied by user")
	case "":
		// ok — fall through
	default:
		return nil, fmt.Errorf("github error: %s", raw.Error)
	}

	if raw.AccessToken == "" {
		return &DeviceFlowPollResult{Done: false}, nil
	}

	// Store token in Keychain immediately so GetSettings reflects authed state.
	_ = keyring.Set(keychainSvc, keychainAcct, raw.AccessToken)
	return &DeviceFlowPollResult{Done: true, Token: raw.AccessToken}, nil
}

// SaveGitHubToken stores the OAuth token in the macOS Keychain.
func (a *App) SaveGitHubToken(token string) error {
	return keyring.Set(keychainSvc, keychainAcct, token)
}

// GetGitHubUser returns the authenticated GitHub username, reading the token from Keychain.
func (a *App) GetGitHubUser() (string, error) {
	token, err := keyring.Get(keychainSvc, keychainAcct)
	if err != nil {
		return "", fmt.Errorf("no token in keychain: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var u struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	return u.Login, nil
}

// DisconnectGitHub removes the GitHub token from Keychain.
func (a *App) DisconnectGitHub() error {
	return keyring.Delete(keychainSvc, keychainAcct)
}

// isGitHubAuthed returns true if a GitHub token exists in the Keychain.
func isGitHubAuthed() bool {
	_, err := keyring.Get(keychainSvc, keychainAcct)
	return err == nil
}
