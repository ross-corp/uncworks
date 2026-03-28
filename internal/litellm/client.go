// Package litellm provides a client for the LiteLLM Admin API.
// It handles virtual key provisioning, revocation, and spend tracking.
package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultHTTPTimeout is the per-request timeout for all LiteLLM Admin API calls.
// Without a timeout an unresponsive proxy stalls the activity worker indefinitely.
const defaultHTTPTimeout = 30 * time.Second

// Client communicates with the LiteLLM Admin API.
type Client struct {
	baseURL    string
	masterKey  string
	httpClient *http.Client
}

// NewClient creates a LiteLLM Admin API client.
func NewClient(baseURL, masterKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		masterKey: masterKey,
		// Explicit timeout prevents stalled requests from blocking activity workers
		// indefinitely when the LiteLLM proxy is unresponsive.
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// MasterKey returns the master API key.
func (c *Client) MasterKey() string {
	return c.masterKey
}

// safeBody reads up to 512 bytes from a response body and redacts any value
// that resembles a bearer key (starts with "sk-") to prevent secret leakage
// in error messages that are propagated through Temporal and may be logged.
func safeBody(b []byte) string {
	const maxLen = 512
	if len(b) > maxLen {
		b = b[:maxLen]
	}
	s := string(b)
	// Blank out sk-* tokens so they do not appear in logs or error payloads.
	var out strings.Builder
	for {
		idx := strings.Index(s, "sk-")
		if idx < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:idx])
		out.WriteString("sk-[REDACTED]")
		s = s[idx+3:]
		// skip to the first delimiter after the token value
		for len(s) > 0 && s[0] != '"' && s[0] != ' ' && s[0] != ',' && s[0] != '}' {
			s = s[1:]
		}
	}
	return out.String()
}

// GenerateKeyRequest is the request body for POST /key/generate.
type GenerateKeyRequest struct {
	// KeyAlias is a human-readable alias for the key.
	KeyAlias string `json:"key_alias,omitempty"`
	// MaxBudget is the maximum spend in USD for this key.
	MaxBudget *float64 `json:"max_budget,omitempty"`
	// Models restricts which models this key can access.
	Models []string `json:"models,omitempty"`
	// Metadata is arbitrary key-value metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GenerateKeyResponse is the response from POST /key/generate.
type GenerateKeyResponse struct {
	Key     string  `json:"key"`
	KeyName string  `json:"key_name,omitempty"`
	Expires string  `json:"expires,omitempty"`
	Spend   float64 `json:"spend"`
}

// DeleteKeyRequest is the request body for POST /key/delete.
type DeleteKeyRequest struct {
	Keys []string `json:"keys"`
}

// DeleteKeyResponse is the response from POST /key/delete.
type DeleteKeyResponse struct {
	DeletedKeys []string `json:"deleted_keys"`
}

// GenerateKey provisions a new virtual key via the LiteLLM Admin API.
func (c *Client) GenerateKey(ctx context.Context, req GenerateKeyRequest) (*GenerateKeyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/key/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.masterKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		// safeBody redacts any sk-* tokens to prevent key material in error logs.
		return nil, fmt.Errorf("key/generate returned %d: %s", resp.StatusCode, safeBody(respBody))
	}

	var result GenerateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// DeleteKey revokes virtual keys via the LiteLLM Admin API.
func (c *Client) DeleteKey(ctx context.Context, keys []string) (*DeleteKeyResponse, error) {
	body, err := json.Marshal(DeleteKeyRequest{Keys: keys})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/key/delete", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.masterKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("key/delete returned %d: %s", resp.StatusCode, safeBody(respBody))
	}

	var result DeleteKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ModelInfo represents a model returned by the LiteLLM /v1/models endpoint.
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by,omitempty"`
}

// ListModelsResponse is the response from GET /v1/models.
type ListModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ListModels returns all models available on the LiteLLM proxy.
func (c *Client) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.masterKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("/v1/models returned %d: %s", resp.StatusCode, safeBody(respBody))
	}

	var result ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// ModelIDs returns just the model ID strings from all available models.
func (c *Client) ModelIDs(ctx context.Context) ([]string, error) {
	resp, err := c.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(resp.Data))
	for i, m := range resp.Data {
		ids[i] = m.ID
	}
	return ids, nil
}
