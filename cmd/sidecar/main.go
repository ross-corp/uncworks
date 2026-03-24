package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/uncworks/aot/internal/litellm"
	"github.com/uncworks/aot/internal/sidecar"
)

type piCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type piModel struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Reasoning     bool     `json:"reasoning"`
	Input         []string `json:"input"`
	Cost          piCost   `json:"cost"`
	ContextWindow int      `json:"contextWindow"`
	MaxTokens     int      `json:"maxTokens"`
}

type piProvider struct {
	BaseURL string    `json:"baseUrl"`
	APIKey  string    `json:"apiKey"`
	API     string    `json:"api"`
	Models  []piModel `json:"models"`
}

type piConfig struct {
	Providers map[string]piProvider `json:"providers"`
}

func main() {
	port := 50052
	if p := os.Getenv("AOT_SIDECAR_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	// Generate pi-coding-agent models.json for LiteLLM integration
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		if err := writePiModelsConfig(baseURL); err != nil {
			log.Printf("WARNING: Failed to write pi models config: %v", err)
		}
	}

	gw := sidecar.NewGateway(port)

	go func() {
		if err := gw.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down RPC Gateway...")
	gw.Stop()
}

// writePiModelsConfig generates ~/.pi/agent/models.json so pi-coding-agent
// routes LLM calls through LiteLLM proxy instead of directly to OpenAI.
// It dynamically fetches the available models from the LiteLLM proxy.
func writePiModelsConfig(baseURL string) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "OPENAI_API_KEY"
	}

	models, err := fetchModelsFromProxy(baseURL, apiKey)
	if err != nil {
		log.Printf("WARNING: Failed to fetch models from LiteLLM proxy, using fallback: %v", err)
		models = fallbackModels()
	}

	cfg := piConfig{
		Providers: map[string]piProvider{
			"litellm": {
				BaseURL: baseURL,
				APIKey:  apiKey,
				API:     "openai-completions",
				Models:  models,
			},
		},
	}

	dir := os.ExpandEnv("$HOME/.pi/agent")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	path := dir + "/models.json"
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	log.Printf("Wrote pi models config to %s (%d models from %s)", path, len(models), baseURL)
	return nil
}

// fetchModelsFromProxy queries the LiteLLM proxy's /v1/models endpoint
// and converts the response into pi-compatible model entries.
func fetchModelsFromProxy(baseURL, apiKey string) ([]piModel, error) {
	client := litellm.NewClient(baseURL, apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	var models []piModel
	for _, m := range resp.Data {
		models = append(models, piModel{
			ID:            m.ID,
			Name:          humanName(m.ID),
			Input:         []string{"text"},
			ContextWindow: 128000,
			MaxTokens:     4096,
		})
	}
	return models, nil
}

// humanName converts a model ID like "deepseek-v3.1" to "Deepseek V3.1".
func humanName(id string) string {
	words := strings.Split(id, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// fallbackModels returns a static list used when the proxy is unreachable.
func fallbackModels() []piModel {
	return []piModel{
		{ID: "default", Name: "Default", Input: []string{"text"}, ContextWindow: 8192, MaxTokens: 4096},
		{ID: "default-cloud", Name: "Default Cloud", Input: []string{"text"}, ContextWindow: 128000, MaxTokens: 4096},
		{ID: "premium", Name: "Premium", Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 8192,
			Cost: piCost{Input: 0.003, Output: 0.015}},
		{ID: "ci", Name: "CI", Input: []string{"text"}, ContextWindow: 4096, MaxTokens: 2048},
	}
}
