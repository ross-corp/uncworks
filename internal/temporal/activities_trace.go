package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"
)

// WriteTraceSpanInput contains parameters for writing a trace span to the workspace.
type WriteTraceSpanInput struct {
	AgentRunName string
	PodIP        string
	Span         TraceSpanData
}

// TraceSpanData is a JSON-serializable trace span written by the workflow.
type TraceSpanData struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"traceId,omitempty"`
	ParentID  string                 `json:"parentId,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	StartTime string                 `json:"startTime"`
	EndTime   string                 `json:"endTime,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	HasDiff   bool                   `json:"hasDiff"`
}

// WriteTraceSpan writes a trace span to the workspace's spans.jsonl file
// via the sidecar ExecCommand RPC.
func (a *Activities) WriteTraceSpan(ctx context.Context, input WriteTraceSpanInput) error {
	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	httpClient := a.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	sc := agentv1connect.NewAgentSidecarServiceClient(httpClient, sidecarURL)

	data, err := json.Marshal(input.Span)
	if err != nil {
		return fmt.Errorf("marshal trace span: %w", err)
	}

	// Escape single quotes in the JSON for safe shell embedding.
	escaped := strings.ReplaceAll(string(data), "'", "'\\''")
	cmd := fmt.Sprintf("mkdir -p /workspace/.aot/traces && echo '%s' >> /workspace/.aot/traces/spans.jsonl", escaped)

	resp, rpcErr := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        cmd,
		WorkingDir:     "/workspace",
		TimeoutSeconds: 10,
	}))
	if rpcErr != nil {
		return fmt.Errorf("write trace span RPC: %w", rpcErr)
	}
	if resp.Msg.ExitCode != 0 {
		return fmt.Errorf("write trace span exited %d: %s", resp.Msg.ExitCode, resp.Msg.Stderr)
	}

	return nil
}
