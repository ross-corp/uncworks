package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/uncworks/aot/internal/sidecar"
)

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
func writePiModelsConfig(baseURL string) error {
	type cost struct {
		Input      float64 `json:"input"`
		Output     float64 `json:"output"`
		CacheRead  float64 `json:"cacheRead"`
		CacheWrite float64 `json:"cacheWrite"`
	}
	type model struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Reasoning     bool     `json:"reasoning"`
		Input         []string `json:"input"`
		Cost          cost     `json:"cost"`
		ContextWindow int      `json:"contextWindow"`
		MaxTokens     int      `json:"maxTokens"`
	}
	type provider struct {
		BaseURL string  `json:"baseUrl"`
		APIKey  string  `json:"apiKey"`
		API     string  `json:"api"`
		Models  []model `json:"models"`
	}
	type config struct {
		Providers map[string]provider `json:"providers"`
	}

	cfg := config{
		Providers: map[string]provider{
			"litellm": {
				BaseURL: baseURL,
				APIKey:  "OPENAI_API_KEY",
				API:     "openai-completions",
				Models: []model{
					{ID: "default", Name: "Default", Input: []string{"text"}, ContextWindow: 8192, MaxTokens: 4096},
					{ID: "default-cloud", Name: "Default Cloud", Input: []string{"text"}, ContextWindow: 128000, MaxTokens: 4096},
					{ID: "premium", Name: "Premium", Input: []string{"text", "image"}, ContextWindow: 200000, MaxTokens: 8192,
						Cost: cost{Input: 0.003, Output: 0.015}},
					{ID: "ci", Name: "CI", Input: []string{"text"}, ContextWindow: 4096, MaxTokens: 2048},
				},
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

	log.Printf("Wrote pi models config to %s (baseURL: %s)", path, baseURL)
	return nil
}
