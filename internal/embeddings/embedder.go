// Package embeddings provides text embedding via Ollama's embedding endpoint.
// For v1, this uses Ollama instead of local ONNX runtime for simplicity.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultOllamaURL is the default Ollama endpoint in the k0s cluster.
	DefaultOllamaURL = "http://ollama:11434"

	// DefaultModel is the embedding model to use.
	DefaultModel = "qwen2.5:0.5b"

	// EmbeddingDim is the expected embedding dimension.
	EmbeddingDim = 384
)

// Embedder generates text embeddings using Ollama's /api/embed endpoint.
type Embedder struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewEmbedder creates an Embedder that calls Ollama for embeddings.
func NewEmbedder(baseURL, model string, httpClient *http.Client) *Embedder {
	if baseURL == "" {
		baseURL = DefaultOllamaURL
	}
	if model == "" {
		model = DefaultModel
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Embedder{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: httpClient,
	}
}

// embedRequest is the request body for Ollama's /api/embed endpoint.
type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// embedResponse is the response body from Ollama's /api/embed endpoint.
type embedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// Embed generates a 384-dimensional embedding for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(embedRequest{
		Model: e.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("unmarshal embed response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert float64 to float32
	raw := embedResp.Embeddings[0]
	result := make([]float32, len(raw))
	for i, v := range raw {
		result[i] = float32(v)
	}

	return result, nil
}

// ChunkText splits text into chunks at paragraph boundaries with overlap.
// Each chunk is at most maxTokens estimated tokens (using word count / 0.75 as a rough estimate).
func ChunkText(content string, maxTokens, overlapTokens int) []string {
	if maxTokens <= 0 {
		maxTokens = 512
	}
	if overlapTokens <= 0 {
		overlapTokens = 64
	}

	paragraphs := strings.Split(content, "\n\n")
	var chunks []string
	var current strings.Builder
	currentTokens := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraTokens := estimateTokens(para)

		if currentTokens+paraTokens > maxTokens && currentTokens > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			// Start new chunk with overlap from the end of previous
			overlap := getOverlapText(current.String(), overlapTokens)
			current.Reset()
			current.WriteString(overlap)
			currentTokens = estimateTokens(overlap)
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
		currentTokens += paraTokens
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

// ChunkCode splits code content by function-like boundaries (double newlines as a heuristic).
// Returns chunks with their estimated node type.
type CodeChunk struct {
	Text     string
	NodeType string
}

// ChunkCodeSimple does a simple heuristic code chunking: split on double newlines,
// group small blocks together, and detect function/class boundaries by keyword patterns.
func ChunkCodeSimple(content string, language string) []CodeChunk {
	blocks := strings.Split(content, "\n\n")
	var chunks []CodeChunk
	var current strings.Builder
	currentTokens := 0

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		blockTokens := estimateTokens(block)

		if currentTokens+blockTokens > 512 && currentTokens > 0 {
			text := strings.TrimSpace(current.String())
			chunks = append(chunks, CodeChunk{
				Text:     text,
				NodeType: detectNodeType(text, language),
			})
			current.Reset()
			currentTokens = 0
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(block)
		currentTokens += blockTokens
	}

	if current.Len() > 0 {
		text := strings.TrimSpace(current.String())
		chunks = append(chunks, CodeChunk{
			Text:     text,
			NodeType: detectNodeType(text, language),
		})
	}

	return chunks
}

// BoostForNodeType returns the structural boost score for a given AST node type.
func BoostForNodeType(nodeType string) float32 {
	switch nodeType {
	case "function", "method":
		return 1.0
	case "class", "struct":
		return 0.9
	case "import":
		return 0.3
	case "whitespace":
		return 0.1
	default:
		return 0.7
	}
}

// detectNodeType uses simple heuristics to guess the AST node type of a code chunk.
func detectNodeType(text, language string) string {
	lower := strings.ToLower(text)

	// Check for whitespace-only
	if strings.TrimSpace(text) == "" {
		return "whitespace"
	}

	// Import detection
	if strings.HasPrefix(lower, "import ") || strings.HasPrefix(lower, "from ") ||
		strings.Contains(lower, "require(") || strings.HasPrefix(lower, "use ") {
		return "import"
	}

	// Function/method detection
	switch language {
	case "go":
		if strings.Contains(text, "func ") {
			return "function"
		}
		if strings.Contains(text, "type ") && strings.Contains(text, "struct") {
			return "struct"
		}
	case "python":
		if strings.HasPrefix(strings.TrimSpace(text), "def ") {
			return "function"
		}
		if strings.HasPrefix(strings.TrimSpace(text), "class ") {
			return "class"
		}
	case "javascript", "typescript", "tsx", "jsx":
		if strings.Contains(text, "function ") || strings.Contains(text, "=> {") ||
			strings.Contains(text, "=> (") {
			return "function"
		}
		if strings.Contains(text, "class ") {
			return "class"
		}
	default:
		if strings.Contains(lower, "func ") || strings.Contains(lower, "function ") ||
			strings.Contains(lower, "def ") {
			return "function"
		}
		if strings.Contains(lower, "class ") {
			return "class"
		}
	}

	return "block"
}

// estimateTokens gives a rough token count estimate (word count / 0.75).
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	tokens := int(float64(words) / 0.75)
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

// getOverlapText returns the last ~overlapTokens worth of text.
func getOverlapText(text string, overlapTokens int) string {
	words := strings.Fields(text)
	overlapWords := int(float64(overlapTokens) * 0.75)
	if overlapWords >= len(words) {
		return text
	}
	return strings.Join(words[len(words)-overlapWords:], " ")
}
