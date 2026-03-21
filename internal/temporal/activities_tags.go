package temporal

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/activity"

	agentv1 "github.com/uncworks/aot/gen/go/agent/v1"
	"github.com/uncworks/aot/gen/go/agent/v1/agentv1connect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnrichRunTagsInput contains parameters for post-run tag enrichment.
type EnrichRunTagsInput struct {
	AgentRunName string
	Namespace    string
	PodIP        string
	RepoPath     string
}

// EnrichRunTags derives tags from the git diff stat and merges them into the
// AgentRun CRD labels.
func (a *Activities) EnrichRunTags(ctx context.Context, input EnrichRunTagsInput) error {
	activity.RecordHeartbeat(ctx, "enriching run tags from diff")

	sidecarURL := fmt.Sprintf("http://%s:%d", input.PodIP, sidecarPort)
	httpClient := a.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	sc := agentv1connect.NewAgentSidecarServiceClient(httpClient, sidecarURL)

	// 1. Run "git diff HEAD~1 --stat" via sidecar
	resp, err := sc.ExecCommand(ctx, connect.NewRequest(&agentv1.ExecCommandRequest{
		Command:        "git diff HEAD~1 --stat",
		WorkingDir:     input.RepoPath,
		TimeoutSeconds: 30,
	}))
	if err != nil {
		// Non-fatal: diff may not be available (e.g., first commit)
		activity.GetLogger(ctx).Warn("EnrichRunTags: git diff failed", "error", err)
		return nil
	}

	diffStat := resp.Msg.Stdout
	if resp.Msg.ExitCode != 0 {
		activity.GetLogger(ctx).Warn("EnrichRunTags: git diff exited non-zero",
			"exitCode", resp.Msg.ExitCode, "stderr", resp.Msg.Stderr)
		return nil
	}

	// 2. Derive tags from diff stat
	newTags := deriveTagsFromDiff(diffStat)
	if len(newTags) == 0 {
		return nil
	}

	// 3. Get existing CRD and merge tags
	gvr := schema.GroupVersionResource{
		Group:    "aot.uncworks.io",
		Version:  "v1alpha1",
		Resource: "agentruns",
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    "AgentRun",
	})

	if err := a.K8sClient.Get(ctx, client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.AgentRunName,
	}, obj); err != nil {
		activity.GetLogger(ctx).Warn("EnrichRunTags: failed to get AgentRun CRD", "error", err)
		return nil
	}

	// 4. Merge new tags with existing tags label
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	existingTags := labels["aot.uncworks.io/tags"]
	tagSet := make(map[string]bool)
	if existingTags != "" {
		for _, t := range strings.Split(existingTags, "_") {
			if t != "" {
				tagSet[t] = true
			}
		}
	}
	for _, t := range newTags {
		tagSet[t] = true
	}

	// Build sorted tag list for deterministic label value
	var merged []string
	for t := range tagSet {
		merged = append(merged, t)
	}
	// Simple sort for determinism
	for i := 0; i < len(merged); i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[i] > merged[j] {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}

	// Kubernetes label values are limited to 63 chars
	tagValue := strings.Join(merged, "_")
	if len(tagValue) > 63 {
		tagValue = tagValue[:63]
		// Trim at last underscore to avoid partial tag
		if idx := strings.LastIndex(tagValue, "_"); idx > 0 {
			tagValue = tagValue[:idx]
		}
	}

	labels["aot.uncworks.io/tags"] = tagValue
	obj.SetLabels(labels)

	// 5. Update CRD
	if err := a.K8sClient.Update(ctx, obj); err != nil {
		activity.GetLogger(ctx).Warn("EnrichRunTags: failed to update AgentRun CRD", "error", err)
		return nil
	}

	return nil
}

// deriveTagsFromDiff parses git diff --stat output and returns tags.
// Tags include language tags derived from file extensions and a scope tag
// based on the number of files changed.
func deriveTagsFromDiff(diffStat string) []string {
	lines := strings.Split(strings.TrimSpace(diffStat), "\n")
	if len(lines) == 0 {
		return nil
	}

	extCounts := make(map[string]int)
	fileCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// The summary line looks like: "N files changed, X insertions(+), Y deletions(-)"
		if strings.Contains(line, "files changed") || strings.Contains(line, "file changed") {
			continue
		}

		// Each file line looks like: " path/to/file.ext | N ++--"
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 1 {
			continue
		}

		filePath := strings.TrimSpace(parts[0])
		if filePath == "" {
			continue
		}

		fileCount++
		ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
		if ext != "" {
			extCounts[ext]++
		}
	}

	if fileCount == 0 {
		return nil
	}

	// Map file extensions to language tags
	extToLang := map[string]string{
		"go":    "go",
		"ts":    "typescript",
		"tsx":   "typescript",
		"js":    "javascript",
		"jsx":   "javascript",
		"py":    "python",
		"rs":    "rust",
		"java":  "java",
		"rb":    "ruby",
		"css":   "css",
		"scss":  "css",
		"html":  "html",
		"yaml":  "yaml",
		"yml":   "yaml",
		"json":  "json",
		"md":    "docs",
		"proto": "proto",
		"sql":   "sql",
		"sh":    "shell",
		"bash":  "shell",
		"zsh":   "shell",
	}

	tagSet := make(map[string]bool)
	for ext, count := range extCounts {
		if lang, ok := extToLang[ext]; ok && count > 0 {
			tagSet[lang] = true
		}
	}

	// Add scope tag based on number of files changed
	switch {
	case fileCount <= 3:
		tagSet["scope-small"] = true
	case fileCount <= 10:
		tagSet["scope-medium"] = true
	default:
		tagSet["scope-large"] = true
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}

	return tags
}
