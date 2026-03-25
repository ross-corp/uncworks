package temporal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/uncworks/aot/internal/brain"
	"github.com/uncworks/aot/internal/embeddings"
)

// KnowledgeActivities holds dependencies for knowledge system Temporal activities.
type KnowledgeActivities struct {
	BrainStore *brain.Store
	Embedder   *embeddings.Embedder
}

// PersistRunDataInput contains the parameters for persisting run data.
type PersistRunDataInput struct {
	AgentRunID    string
	WorkspacePath string
	RepoURL       string
}

// PersistRunData reads logs and spans from the agent workspace and saves them to PostgreSQL.
// This activity runs after agent completion, before scale-down.
func (ka *KnowledgeActivities) PersistRunData(ctx context.Context, input PersistRunDataInput) error {
	logger := activity.GetLogger(ctx)

	if ka.BrainStore == nil {
		logger.Warn("Brain store not configured, skipping run data persistence")
		return nil
	}

	// Read and persist logs
	logPath := input.WorkspacePath + "/.aot/logs/agent.log"
	if content, err := os.ReadFile(logPath); err == nil && len(content) > 0 {
		if err := ka.BrainStore.SaveRunLog(ctx, input.AgentRunID, string(content)); err != nil {
			logger.Warn("Failed to save run logs", "error", err)
		} else {
			logger.Info("Persisted run logs", "agentRunID", input.AgentRunID, "bytes", len(content))
		}
	} else if err != nil && !os.IsNotExist(err) {
		logger.Warn("Failed to read agent log", "path", logPath, "error", err)
	}

	// Read and persist spans from JSONL file
	spansPath := input.WorkspacePath + "/.aot/traces/spans.jsonl"
	if f, err := os.Open(spansPath); err == nil {
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		spanCount := 0
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var raw struct {
				Name      string                 `json:"name"`
				Type      string                 `json:"type"`
				ParentID  string                 `json:"parent_id"`
				StartTime string                 `json:"start_time"`
				EndTime   string                 `json:"end_time"`
				Metadata  map[string]interface{} `json:"metadata"`
			}
			if err := json.Unmarshal(line, &raw); err != nil {
				logger.Warn("Failed to parse span line", "error", err)
				continue
			}

			span := brain.TraceSpan{
				Name:     raw.Name,
				Type:     raw.Type,
				ParentID: raw.ParentID,
				Metadata: raw.Metadata,
			}
			if t, err := time.Parse(time.RFC3339Nano, raw.StartTime); err == nil {
				span.StartTime = t
			} else {
				span.StartTime = time.Now()
			}
			if raw.EndTime != "" {
				if t, err := time.Parse(time.RFC3339Nano, raw.EndTime); err == nil {
					span.EndTime = &t
				}
			}

			if err := ka.BrainStore.SaveRunSpan(ctx, input.AgentRunID, span); err != nil {
				logger.Warn("Failed to save run span", "error", err)
			} else {
				spanCount++
			}
		}
		if spanCount > 0 {
			logger.Info("Persisted run spans", "agentRunID", input.AgentRunID, "count", spanCount)
		}
	} else if !os.IsNotExist(err) {
		logger.Warn("Failed to read spans file", "path", spansPath, "error", err)
	}

	// Read diffs from workspace (git diff output or .aot/diffs/)
	diffsDir := input.WorkspacePath + "/.aot/diffs"
	if entries, err := os.ReadDir(diffsDir); err == nil {
		diffCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(diffsDir + "/" + entry.Name())
			if err != nil {
				continue
			}
			if err := ka.BrainStore.SaveRunDiff(ctx, input.AgentRunID, "", entry.Name(), string(content)); err != nil {
				logger.Warn("Failed to save run diff", "file", entry.Name(), "error", err)
			} else {
				diffCount++
			}
		}
		if diffCount > 0 {
			logger.Info("Persisted run diffs", "agentRunID", input.AgentRunID, "count", diffCount)
		}
	}

	return nil
}

// EmbedRunDataInput contains the parameters for embedding run data.
type EmbedRunDataInput struct {
	AgentRunID string
	RepoURL    string
}

// EmbedRunData chunks and embeds persisted run data into pgvector tables.
// This activity runs after PersistRunData and does NOT block workflow completion.
func (ka *KnowledgeActivities) EmbedRunData(ctx context.Context, input EmbedRunDataInput) error {
	logger := activity.GetLogger(ctx)

	if ka.BrainStore == nil || ka.Embedder == nil {
		logger.Warn("Brain store or embedder not configured, skipping embedding")
		return nil
	}

	if !ka.BrainStore.PgvectorReady() {
		logger.Warn("pgvector not available, skipping embedding")
		return nil
	}

	// Embed diffs as code chunks
	diffs, err := ka.BrainStore.GetRunDiffs(ctx, input.AgentRunID)
	if err != nil {
		logger.Warn("Failed to get run diffs for embedding", "error", err)
	} else {
		var codeChunks []brain.CodeChunkRecord
		for _, diff := range diffs {
			chunks := embeddings.ChunkCodeSimple(diff.Patch, detectLanguage(diff.FilePath))
			for _, chunk := range chunks {
				vec, err := ka.Embedder.Embed(ctx, chunk.Text)
				if err != nil {
					return fmt.Errorf("embed code chunk: %w", err)
				}
				codeChunks = append(codeChunks, brain.CodeChunkRecord{
					AgentRunID: input.AgentRunID,
					DiffID:     diff.ID,
					ChunkText:  chunk.Text,
					FilePath:   diff.FilePath,
					Language:   detectLanguage(diff.FilePath),
					NodeType:   chunk.NodeType,
					RepoURL:    input.RepoURL,
					Boost:      embeddings.BoostForNodeType(chunk.NodeType),
					Embedding:  vec,
				})
			}
		}
		if len(codeChunks) > 0 {
			if err := ka.BrainStore.SaveCodeChunks(ctx, codeChunks); err != nil {
				logger.Warn("Failed to save code chunks", "error", err)
			} else {
				logger.Info("Embedded code chunks", "agentRunID", input.AgentRunID, "count", len(codeChunks))
			}
		}
	}

	// Embed logs as trace chunks
	logContent, err := ka.BrainStore.GetRunLogs(ctx, input.AgentRunID)
	if err != nil {
		logger.Warn("Failed to get run logs for embedding", "error", err)
	} else if logContent != "" {
		textChunks := embeddings.ChunkText(logContent, 512, 64)
		var traceChunks []brain.TraceChunkRecord
		for _, text := range textChunks {
			vec, err := ka.Embedder.Embed(ctx, text)
			if err != nil {
				return fmt.Errorf("embed trace chunk: %w", err)
			}
			traceChunks = append(traceChunks, brain.TraceChunkRecord{
				AgentRunID: input.AgentRunID,
				ChunkText:  text,
				ChunkType:  "log",
				RepoURL:    input.RepoURL,
				Embedding:  vec,
			})
		}
		if len(traceChunks) > 0 {
			if err := ka.BrainStore.SaveTraceChunks(ctx, traceChunks); err != nil {
				logger.Warn("Failed to save trace chunks", "error", err)
			} else {
				logger.Info("Embedded trace chunks", "agentRunID", input.AgentRunID, "count", len(traceChunks))
			}
		}
	}

	return nil
}

// HydrateContextInput contains the parameters for context hydration.
type HydrateContextInput struct {
	AgentRunID    string
	WorkspacePath string
	Prompt        string
	RepoURL       string
	AgentType     string // "senior", "orchestrator", or empty for junior/single
}

// HydrateContextOutput contains the result of context hydration.
type HydrateContextOutput struct {
	ContextWritten bool
}

// HydrateContext queries pgvector for relevant past work and writes a context file
// to the agent's workspace before the agent starts.
func (ka *KnowledgeActivities) HydrateContext(ctx context.Context, input HydrateContextInput) (*HydrateContextOutput, error) {
	logger := activity.GetLogger(ctx)

	if ka.BrainStore == nil || ka.Embedder == nil {
		logger.Warn("Brain store or embedder not configured, skipping context hydration")
		return &HydrateContextOutput{}, nil
	}

	if !ka.BrainStore.PgvectorReady() {
		logger.Warn("pgvector not available, skipping context hydration")
		return &HydrateContextOutput{}, nil
	}

	// Embed the prompt
	queryVec, err := ka.Embedder.Embed(ctx, input.Prompt)
	if err != nil {
		return nil, fmt.Errorf("embed prompt for context hydration: %w", err)
	}

	// Determine top-K based on agent type
	topK := 10
	if input.AgentType == "senior" || input.AgentType == "orchestrator" {
		topK = 25
	}

	// Search code chunks
	codeResults, err := ka.BrainStore.SearchCodeChunks(ctx, queryVec, input.RepoURL, topK, nil, nil)
	if err != nil {
		logger.Warn("Failed to search code chunks", "error", err)
	}

	// Search trace chunks
	traceResults, err := ka.BrainStore.SearchTraceChunks(ctx, queryVec, input.RepoURL, topK, nil, nil)
	if err != nil {
		logger.Warn("Failed to search trace chunks", "error", err)
	}

	if len(codeResults) == 0 && len(traceResults) == 0 {
		logger.Info("No relevant past work found for context hydration")
		return &HydrateContextOutput{}, nil
	}

	// Format context file
	contextContent := formatContextFile(codeResults, traceResults)

	// Write context file
	contextDir := input.WorkspacePath + "/.aot/context"
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		logger.Warn("Failed to create context directory", "error", err)
		return &HydrateContextOutput{}, nil
	}

	contextPath := contextDir + "/past-work.md"
	if err := os.WriteFile(contextPath, []byte(contextContent), 0o644); err != nil {
		logger.Warn("Failed to write context file", "error", err)
		return &HydrateContextOutput{}, nil
	}

	logger.Info("Context hydration complete",
		"agentRunID", input.AgentRunID,
		"codeResults", len(codeResults),
		"traceResults", len(traceResults))

	return &HydrateContextOutput{ContextWritten: true}, nil
}

// formatContextFile produces a markdown context file from search results.
func formatContextFile(codeResults []brain.CodeChunkResult, traceResults []brain.TraceChunkResult) string {
	var b []byte

	b = append(b, "# Past Work Context\n\n"...)
	b = append(b, "This file contains relevant past work from previous agent runs.\n"...)
	b = append(b, "Use this context to avoid repeating past mistakes and build on previous solutions.\n\n"...)

	if len(codeResults) > 0 {
		b = append(b, "## Relevant Code Changes\n\n"...)
		for _, r := range codeResults {
			b = append(b, fmt.Sprintf("### %s (run: %s, similarity: %.2f)\n\n", r.FilePath, r.AgentRunID, r.Similarity)...)
			if r.Language != "" {
				b = append(b, fmt.Sprintf("```%s\n%s\n```\n\n", r.Language, r.ChunkText)...)
			} else {
				b = append(b, fmt.Sprintf("```\n%s\n```\n\n", r.ChunkText)...)
			}
		}
	}

	if len(traceResults) > 0 {
		b = append(b, "## Relevant Logs & Traces\n\n"...)
		for _, r := range traceResults {
			b = append(b, fmt.Sprintf("### Run: %s (type: %s, similarity: %.2f)\n\n", r.AgentRunID, r.ChunkType, r.Similarity)...)
			b = append(b, r.ChunkText+"\n\n"...)
		}
	}

	// Enforce 8000 token cap (rough estimate: 4 chars per token)
	const maxChars = 8000 * 4
	if len(b) > maxChars {
		b = b[:maxChars]
		b = append(b, "\n\n... (truncated)\n"...)
	}

	return string(b)
}

// detectLanguage returns the programming language based on file extension.
func detectLanguage(filePath string) string {
	switch {
	case hasSuffix(filePath, ".go"):
		return "go"
	case hasSuffix(filePath, ".py"):
		return "python"
	case hasSuffix(filePath, ".ts"):
		return "typescript"
	case hasSuffix(filePath, ".tsx"):
		return "tsx"
	case hasSuffix(filePath, ".js"):
		return "javascript"
	case hasSuffix(filePath, ".jsx"):
		return "jsx"
	case hasSuffix(filePath, ".rs"):
		return "rust"
	case hasSuffix(filePath, ".java"):
		return "java"
	case hasSuffix(filePath, ".rb"):
		return "ruby"
	case hasSuffix(filePath, ".sh"), hasSuffix(filePath, ".bash"):
		return "bash"
	case hasSuffix(filePath, ".yaml"), hasSuffix(filePath, ".yml"):
		return "yaml"
	case hasSuffix(filePath, ".json"):
		return "json"
	case hasSuffix(filePath, ".sql"):
		return "sql"
	default:
		return ""
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
