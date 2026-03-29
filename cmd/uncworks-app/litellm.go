//go:build darwin

// litellm.go — LiteLLM proxy health check for the UNCWORKS desktop app.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// LiteLLMCheckResult is returned by CheckLiteLLM.
type LiteLLMCheckResult struct {
	OK     bool     `json:"ok"`
	Models []string `json:"models"`
	Error  string   `json:"error,omitempty"`
}

// CheckLiteLLM tests connectivity to the LiteLLM proxy at url and returns
// the list of available model IDs. Exposed as a Wails binding.
func (a *App) CheckLiteLLM(url string) LiteLLMCheckResult {
	if url == "" {
		s, _ := loadAppSettings()
		url = s.LiteLLMURL
	}
	if url == "" {
		url = "http://litellm:4000"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url+"/v1/models", nil)
	if err != nil {
		return LiteLLMCheckResult{Error: fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer sk-uncworks-local")

	resp, err := client.Do(req)
	if err != nil {
		// Distinguish connection-refused (port-forward not running) from timeout
		// (proxy reachable but slow) so the frontend can show a useful message.
		var netErr *net.OpError
		if errors.As(err, &netErr) && netErr.Op == "dial" {
			return LiteLLMCheckResult{Error: "connection refused — is the port-forward running?"}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return LiteLLMCheckResult{Error: "timed out — LiteLLM proxy is not responding"}
		}
		return LiteLLMCheckResult{Error: fmt.Sprintf("unreachable: %v", err)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return LiteLLMCheckResult{Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 120))}
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return LiteLLMCheckResult{Error: fmt.Sprintf("parse response: %v", err)}
	}

	ids := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		ids = append(ids, m.ID)
	}
	return LiteLLMCheckResult{OK: true, Models: ids}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
