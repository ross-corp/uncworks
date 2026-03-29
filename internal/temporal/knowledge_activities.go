package temporal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/activity"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
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
	PodIP         string
}

// PersistRunData reads logs and spans from the agent workspace and saves them to PostgreSQL.
// This activity runs after agent completion, before scale-down.
// File access is routed through the sidecar ExecCommand RPC so reads target the agent
// pod's PVC rather than the Temporal worker's local filesystem.
func (ka *KnowledgeActivities) PersistRunData(ctx context.Context, input PersistRunDataInput) error {
	logger := activity.GetLogger(ctx)

	if ka.BrainStore == nil {
		logger.Warn("Brain store not configured, skipping run data persistence")
		return nil
	}

	if input.PodIP == "" {
		logger.Warn("No pod IP provided, skipping run data persistence")
		return nil
	}

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// Read and persist logs via sidecar
	logPath := input.WorkspacePath + "/.aot/logs/agent.log"
	logResp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "cat " + logPath,
		TimeoutSeconds: 30,
	}))
	if err == nil && logResp.Msg.ExitCode == 0 && len(logResp.Msg.Stdout) > 0 {
		if err := ka.BrainStore.SaveRunLog(ctx, input.AgentRunID, logResp.Msg.Stdout); err != nil {
			logger.Warn("Failed to save run logs", "error", err)
		} else {
			logger.Info("Persisted run logs", "agentRunID", input.AgentRunID, "bytes", len(logResp.Msg.Stdout))
		}
	} else if err != nil {
		logger.Warn("Failed to read agent log via sidecar", "path", logPath, "error", err)
	}

	// Read and persist spans from JSONL file via sidecar
	spansPath := input.WorkspacePath + "/.aot/traces/spans.jsonl"
	spansResp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "cat " + spansPath,
		TimeoutSeconds: 30,
	}))
	if err == nil && spansResp.Msg.ExitCode == 0 && len(spansResp.Msg.Stdout) > 0 {
		scanner := bufio.NewScanner(strings.NewReader(spansResp.Msg.Stdout))
		var spans []brain.TraceSpan
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
			spans = append(spans, span)
		}
		if len(spans) > 0 {
			if err := ka.BrainStore.SaveRunSpans(ctx, input.AgentRunID, spans); err != nil {
				logger.Warn("Failed to save run spans", "error", err)
			} else {
				logger.Info("Persisted run spans", "agentRunID", input.AgentRunID, "count", len(spans))
			}
		}
	} else if err != nil {
		logger.Warn("Failed to read spans file via sidecar", "path", spansPath, "error", err)
	}

	// List and read diffs from workspace via sidecar
	diffsDir := input.WorkspacePath + "/.aot/diffs"
	lsResp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "ls -1 " + diffsDir + " 2>/dev/null",
		TimeoutSeconds: 15,
	}))
	if err == nil && lsResp.Msg.ExitCode == 0 && len(lsResp.Msg.Stdout) > 0 {
		diffCount := 0
		for _, name := range strings.Split(strings.TrimSpace(lsResp.Msg.Stdout), "\n") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			catResp, catErr := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
				Command:        "cat " + diffsDir + "/" + name,
				TimeoutSeconds: 15,
			}))
			if catErr != nil || catResp.Msg.ExitCode != 0 {
				continue
			}
			if err := ka.BrainStore.SaveRunDiff(ctx, input.AgentRunID, "", name, catResp.Msg.Stdout); err != nil {
				logger.Warn("Failed to save run diff", "file", name, "error", err)
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
		for i, diff := range diffs {
			activity.RecordHeartbeat(ctx, fmt.Sprintf("embedding diff %d/%d", i+1, len(diffs)))
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
		for i, text := range textChunks {
			activity.RecordHeartbeat(ctx, fmt.Sprintf("embedding log chunk %d/%d", i+1, len(textChunks)))
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
	PodIP         string
}

// HydrateContextOutput contains the result of context hydration.
type HydrateContextOutput struct {
	ContextWritten bool
}

// HydrateContext queries pgvector for relevant past work and writes a context file
// to the agent's workspace before the agent starts.
// The context file is written via the sidecar ExecCommand RPC so it lands on the
// agent pod's PVC rather than the Temporal worker's local filesystem.
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

	if input.PodIP == "" {
		logger.Warn("No pod IP provided, skipping context hydration")
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

	// Write context file to agent pod PVC via sidecar ExecCommand.
	// Use tee so the content flows through stdin and lands in the right place.
	contextDir := input.WorkspacePath + "/.aot/context"
	contextPath := contextDir + "/past-work.md"

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	sc := agentv1connect.NewAgentSidecarServiceClient(http.DefaultClient, sidecarURL)

	// First, ensure the directory exists on the pod
	mkdirResp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "mkdir -p " + contextDir,
		TimeoutSeconds: 10,
	}))
	if err != nil || mkdirResp.Msg.ExitCode != 0 {
		logger.Warn("Failed to create context directory via sidecar", "error", err)
		return &HydrateContextOutput{}, nil
	}

	// Write via bash printf to handle arbitrary content safely
	writeCmd := fmt.Sprintf("printf '%%s' %s > %s",
		shellescape(contextContent), contextPath)
	writeResp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        writeCmd,
		TimeoutSeconds: 15,
	}))
	if err != nil || writeResp.Msg.ExitCode != 0 {
		logger.Warn("Failed to write context file via sidecar", "error", err,
			"stderr", writeResp.Msg.Stderr)
		return &HydrateContextOutput{}, nil
	}

	logger.Info("Context hydration complete",
		"agentRunID", input.AgentRunID,
		"codeResults", len(codeResults),
		"traceResults", len(traceResults))

	return &HydrateContextOutput{ContextWritten: true}, nil
}

// shellescape wraps a string in single quotes and escapes any embedded single quotes
// so it can be safely passed as a shell argument.
func shellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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
