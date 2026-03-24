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
)

// Client communicates with the LiteLLM Admin API.
type Client struct {
	baseURL    string
	masterKey  string
	httpClient *http.Client
}

// NewClient creates a LiteLLM Admin API client.
func NewClient(baseURL, masterKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		masterKey:  masterKey,
		httpClient: &http.Client{},
	}
}

// MasterKey returns the master API key.
func (c *Client) MasterKey() string {
	return c.masterKey
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
		return nil, fmt.Errorf("key/generate returned %d: %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("key/delete returned %d: %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("/v1/models returned %d: %s", resp.StatusCode, string(respBody))
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
