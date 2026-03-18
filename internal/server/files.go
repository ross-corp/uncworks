package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// FileHandler serves file explorer API endpoints by exec-ing into agent pods
// or reading directly from PVC host paths when the pod is scaled to 0.
type FileHandler struct {
	k8sClient  runtimeclient.Client
	restConfig *rest.Config
	namespace  string
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(k8sClient runtimeclient.Client, restConfig *rest.Config, namespace string) *FileHandler {
	return &FileHandler{
		k8sClient:  k8sClient,
		restConfig: restConfig,
		namespace:  namespace,
	}
}

// RegisterFileHandlers registers the file explorer REST endpoints on the given mux.
func (f *FileHandler) RegisterFileHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/runs/{id}/files", f.handleListFiles)
	mux.HandleFunc("GET /api/v1/runs/{id}/files/content", f.handleFileContent)
	mux.HandleFunc("GET /api/v1/runs/{id}/logs", f.handleLogs)
	mux.HandleFunc("GET /api/v1/runs/{id}/logs/structured", f.handleStructuredLogs)
	mux.HandleFunc("GET /api/v1/runs/{id}/verification", f.handleVerificationResult)
}

// FileEntry represents a single entry in a directory listing.
type FileEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// FileListResponse is the JSON response for directory listings.
type FileListResponse struct {
	Entries []FileEntry `json:"entries"`
}

// getDeploymentReplicas looks up the AgentRun CRD to get the deployment name,
// then reads the Deployment's spec.replicas.
func (f *FileHandler) getDeploymentReplicas(ctx context.Context, runID string) (int32, error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := f.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: f.namespace,
		Name:      runID,
	}, crd); err != nil {
		return 0, fmt.Errorf("get AgentRun: %w", err)
	}

	deployName := crd.Status.DeploymentName
	if deployName == "" {
		return 0, fmt.Errorf("no deployment name on AgentRun %q status", runID)
	}

	deploy := &appsv1.Deployment{}
	if err := f.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: f.namespace,
		Name:      deployName,
	}, deploy); err != nil {
		return 0, fmt.Errorf("get Deployment %q: %w", deployName, err)
	}

	if deploy.Spec.Replicas == nil {
		return 1, nil // default is 1 if not set
	}
	return *deploy.Spec.Replicas, nil
}

// getPVCHostPath looks up the PVC aot-ws-{runID}, finds its bound PV,
// and returns the PV's spec.hostPath.path (used by local-path-provisioner).
func (f *FileHandler) getPVCHostPath(ctx context.Context, runID string) (string, error) {
	pvcName := "aot-ws-" + runID

	pvc := &corev1.PersistentVolumeClaim{}
	if err := f.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: f.namespace,
		Name:      pvcName,
	}, pvc); err != nil {
		return "", fmt.Errorf("PVC %q not found: %w", pvcName, err)
	}

	pvName := pvc.Spec.VolumeName
	if pvName == "" {
		return "", fmt.Errorf("PVC %q has no bound PV", pvcName)
	}

	pv := &corev1.PersistentVolume{}
	if err := f.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Name: pvName,
	}, pv); err != nil {
		return "", fmt.Errorf("PV %q not found: %w", pvName, err)
	}

	if pv.Spec.HostPath == nil {
		return "", fmt.Errorf("PV %q does not use hostPath", pvName)
	}

	return pv.Spec.HostPath.Path, nil
}

func (f *FileHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/workspace"
	}

	// Check if pod is running (replicas > 0).
	replicas, replicaErr := f.getDeploymentReplicas(r.Context(), runID)

	if replicaErr == nil && replicas > 0 {
		// Pod is running — use exec (current behavior).
		podName, err := f.lookupPodName(r.Context(), runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("agent run %q not found: %v", runID, err)})
			return
		}
		if podName == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "pod not available for this agent run"})
			return
		}

		stdout, stderr, err := f.execInPod(r.Context(), podName, []string{"ls", "-la", "--time-style=long-iso", dirPath})
		if err != nil {
			log.Printf("exec ls in pod %s failed: %v, stderr: %s", podName, err, stderr)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list files: " + err.Error()})
			return
		}

		if strings.Contains(stderr, "No such file or directory") {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "path not found: " + dirPath})
			return
		}

		entries := parseLsOutput(stdout)
		writeJSON(w, http.StatusOK, FileListResponse{Entries: entries})
		return
	}

	// Pod is scaled to 0 — read from PVC host path.
	hostPath, err := f.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v (run may be archived or deleted)", runID, err)})
		return
	}

	// Map the in-container path to the host path.
	// In the container, workspace is at /workspace. On the host, it's at the PVC host path.
	relativePath := strings.TrimPrefix(dirPath, "/workspace")
	relativePath = strings.TrimPrefix(relativePath, "/")
	diskPath := filepath.Join(hostPath, relativePath)

	// Prevent path traversal attacks: ensure resolved path stays within hostPath.
	resolvedPath, err := filepath.Abs(diskPath)
	if err != nil || !strings.HasPrefix(resolvedPath, hostPath) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid path"})
		return
	}

	dirEntries, err := os.ReadDir(diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "path not found: " + dirPath})
			return
		}
		log.Printf("failed to read directory %s: %v", diskPath, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list files: " + err.Error()})
		return
	}

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		info, infoErr := de.Info()
		entry := FileEntry{
			Name: de.Name(),
		}
		switch {
		case de.IsDir():
			entry.Type = "directory"
		case de.Type()&fs.ModeSymlink != 0:
			entry.Type = "symlink"
		default:
			entry.Type = "file"
		}
		if infoErr == nil {
			entry.Size = info.Size()
			entry.Modified = info.ModTime().Format("2006-01-02 15:04")
		}
		entries = append(entries, entry)
	}

	writeJSON(w, http.StatusOK, FileListResponse{Entries: entries})
}

func (f *FileHandler) handleFileContent(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "path query parameter is required"})
		return
	}

	// Check if pod is running (replicas > 0).
	replicas, replicaErr := f.getDeploymentReplicas(r.Context(), runID)

	if replicaErr == nil && replicas > 0 {
		// Pod is running — use exec (current behavior).
		podName, err := f.lookupPodName(r.Context(), runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("agent run %q not found: %v", runID, err)})
			return
		}
		if podName == "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "pod not available for this agent run"})
			return
		}

		stdout, stderr, err := f.execInPod(r.Context(), podName, []string{"cat", filePath})
		if err != nil {
			if strings.Contains(stderr, "No such file or directory") {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "file not found: " + filePath})
				return
			}
			if strings.Contains(stderr, "Permission denied") {
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "permission denied: " + filePath})
				return
			}
			log.Printf("exec cat in pod %s failed: %v, stderr: %s", podName, err, stderr)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read file: " + err.Error()})
			return
		}

		contentType := detectContentType(filePath)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(stdout))
		return
	}

	// Pod is scaled to 0 — read from PVC host path.
	hostPath, err := f.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v (run may be archived or deleted)", runID, err)})
		return
	}

	relativePath := strings.TrimPrefix(filePath, "/workspace/")
	diskPath := filepath.Join(hostPath, relativePath)

	// Prevent path traversal attacks: ensure resolved path stays within hostPath.
	resolvedPath, err := filepath.Abs(diskPath)
	if err != nil || !strings.HasPrefix(resolvedPath, hostPath) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid path"})
		return
	}

	data, err := os.ReadFile(diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "file not found: " + filePath})
			return
		}
		if os.IsPermission(err) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "permission denied: " + filePath})
			return
		}
		log.Printf("failed to read file %s: %v", diskPath, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read file: " + err.Error()})
		return
	}

	contentType := detectContentType(filePath)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// lookupPodName delegates to the shared lookupRunningPod function.
func (f *FileHandler) lookupPodName(ctx context.Context, runID string) (string, error) {
	return lookupRunningPod(ctx, f.k8sClient, f.namespace, runID)
}

// execInPod runs a command in the rpc-gateway container and returns stdout/stderr.
func (f *FileHandler) execInPod(ctx context.Context, podName string, command []string) (string, string, error) {
	clientset, err := kubernetes.NewForConfig(f.restConfig)
	if err != nil {
		return "", "", fmt.Errorf("create clientset: %w", err)
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(f.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "rpc-gateway",
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(f.restConfig, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

// parseLsOutput parses the output of `ls -la --time-style=long-iso` into FileEntry structs.
func parseLsOutput(output string) []FileEntry {
	var entries []FileEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		// Skip the "total" line and empty lines
		if strings.HasPrefix(line, "total ") || line == "" {
			continue
		}

		// ls -la --time-style=long-iso format:
		// drwxr-xr-x 2 root root 4096 2024-01-01 12:00 dirname
		// -rw-r--r-- 1 root root 1234 2024-01-01 12:00 filename
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		perms := fields[0]
		name := strings.Join(fields[7:], " ") // file name may contain spaces

		// Skip . and .. entries
		if name == "." || name == ".." {
			continue
		}

		entryType := "file"
		if len(perms) > 0 && perms[0] == 'd' {
			entryType = "directory"
		} else if len(perms) > 0 && perms[0] == 'l' {
			entryType = "symlink"
		}

		size, _ := strconv.ParseInt(fields[4], 10, 64)
		modified := fields[5] + " " + fields[6]

		entries = append(entries, FileEntry{
			Name:     name,
			Type:     entryType,
			Size:     size,
			Modified: modified,
		})
	}

	return entries
}

// detectContentType returns a Content-Type based on file extension.
func detectContentType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}

	// Common code/text types that mime.TypeByExtension might not cover well
	textTypes := map[string]string{
		".go":   "text/plain; charset=utf-8",
		".py":   "text/plain; charset=utf-8",
		".rs":   "text/plain; charset=utf-8",
		".ts":   "text/plain; charset=utf-8",
		".tsx":  "text/plain; charset=utf-8",
		".js":   "text/plain; charset=utf-8",
		".jsx":  "text/plain; charset=utf-8",
		".md":   "text/markdown; charset=utf-8",
		".yaml": "text/yaml; charset=utf-8",
		".yml":  "text/yaml; charset=utf-8",
		".toml": "text/plain; charset=utf-8",
		".sh":   "text/plain; charset=utf-8",
		".bash": "text/plain; charset=utf-8",
		".zsh":  "text/plain; charset=utf-8",
		".txt":  "text/plain; charset=utf-8",
		".log":  "text/plain; charset=utf-8",
		".csv":  "text/csv; charset=utf-8",
		".json": "application/json",
		".xml":  "application/xml",
	}

	if ct, ok := textTypes[ext]; ok {
		return ct
	}

	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}

	return "application/octet-stream"
}

// handleLogs returns the rpc-gateway container logs for an agent run's pod,
// or reads from the persisted log file on disk when the pod is scaled to 0.
func (f *FileHandler) handleLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, `{"error":"missing run id"}`, http.StatusBadRequest)
		return
	}

	// Check if pod is running (replicas > 0).
	replicas, replicaErr := f.getDeploymentReplicas(r.Context(), runID)

	if replicaErr == nil && replicas > 0 {
		// Pod is running — stream container logs (current behavior).
		podName, err := f.lookupPodName(r.Context(), runID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}

		clientset, err := kubernetes.NewForConfig(f.restConfig)
		if err != nil {
			http.Error(w, `{"error":"k8s client error"}`, http.StatusInternalServerError)
			return
		}

		tailLines := int64(1000)
		logReq := clientset.CoreV1().Pods(f.namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container: "rpc-gateway",
			TailLines: &tailLines,
		})

		stream, err := logReq.Stream(r.Context())
		if err != nil {
			log.Printf("Failed to stream logs for pod %s: %v", podName, err)
			http.Error(w, fmt.Sprintf(`{"error":"failed to get logs: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer func() { _ = stream.Close() }()

		w.Header().Set("Content-Type", "text/plain")
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(stream); err != nil {
			log.Printf("Failed to read logs for pod %s: %v", podName, err)
		}
		_, _ = w.Write(buf.Bytes())
		return
	}

	// Pod is scaled to 0 — read log file from PVC host path.
	hostPath, err := f.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v (run may be archived or deleted)", runID, err)})
		return
	}

	logPath := filepath.Join(hostPath, ".aot", "logs", "agent.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No log file yet — return empty response.
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("failed to read log file %s: %v", logPath, err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to read logs: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// AgentLogEntry is a structured log entry parsed from the agent's JSONL output.
type AgentLogEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`    // user, assistant, tool_call, tool_result, system
	Content   string `json:"content"` // text content
	ToolName  string `json:"toolName,omitempty"`
	ToolInput string `json:"toolInput,omitempty"`
	Model     string `json:"model,omitempty"`
}

// handleStructuredLogs reads the agent's JSONL log file and returns parsed
// conversation entries as a JSON array. This is the "what did the agent say and do"
// view, not raw container stderr.
func (f *FileHandler) handleStructuredLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing run id"})
		return
	}

	// Always read from PVC host path (JSONL is written to disk, not container logs).
	hostPath, err := f.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v", runID, err)})
		return
	}

	jsonlPath := filepath.Join(hostPath, ".aot", "logs", "agent.jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []AgentLogEntry{})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to read agent log"})
		return
	}

	entries := parseAgentJSONL(string(data))
	writeJSON(w, http.StatusOK, entries)
}

// handleVerificationResult returns the verification-result.json from the workspace.
func (f *FileHandler) handleVerificationResult(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing run id"})
		return
	}

	hostPath, err := f.getPVCHostPath(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: fmt.Sprintf("workspace not found for run %q: %v", runID, err)})
		return
	}

	// Check multiple possible locations for the verification result.
	candidates := []string{
		filepath.Join(hostPath, ".openspec", "changes", runID, "verification-result.json"),
		filepath.Join(hostPath, "openspec", "changes", runID, "verification-result.json"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
	}

	// Also check the CRD status field as a fallback.
	crd := &aotv1alpha1.AgentRun{}
	if err := f.k8sClient.Get(r.Context(), runtimeclient.ObjectKey{
		Namespace: f.namespace,
		Name:      runID,
	}, crd); err == nil && crd.Status.VerificationResult != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(crd.Status.VerificationResult))
		return
	}

	writeJSON(w, http.StatusNotFound, errorResponse{Error: "no verification result available for this run"})
}

// parseAgentJSONL parses the pi-coding-agent JSONL format into structured log entries.
// It handles the full event lifecycle: message_start/end for complete messages,
// turn_end for tool results, and agent_end for the final conversation summary.
func parseAgentJSONL(raw string) []AgentLogEntry {
	var entries []AgentLogEntry
	var sessionTimestamp string
	seenToolCalls := make(map[string]bool)   // deduplicate tool calls
	seenTexts := make(map[string]bool)       // deduplicate text messages
	seenToolResults := make(map[string]bool) // deduplicate tool results

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "session":
			if ts, ok := event["timestamp"].(string); ok {
				sessionTimestamp = ts
			}

		case "tool_execution_start":
			toolName, _ := event["toolName"].(string)
			toolCallID, _ := event["toolCallId"].(string)
			args, _ := event["args"].(map[string]interface{})
			argsJSON, _ := json.Marshal(args)
			dedupeKey := "tc:" + toolCallID
			if toolCallID != "" && seenToolCalls[dedupeKey] {
				continue
			}
			if toolCallID != "" {
				seenToolCalls[dedupeKey] = true
			}
			entries = append(entries, AgentLogEntry{
				Timestamp: sessionTimestamp,
				Type:      "tool_call",
				ToolName:  toolName,
				ToolInput: string(argsJSON),
			})

		case "tool_execution_end":
			toolName, _ := event["toolName"].(string)
			toolCallID, _ := event["toolCallId"].(string)
			isError, _ := event["isError"].(bool)
			resultText := ""
			if result, ok := event["result"].(map[string]interface{}); ok {
				resultText = extractResultContent(result["content"])
			}
			// Mark this tool call's result as seen (prevents dups from message_end/turn_end)
			resultDedup := "tool_result:" + toolName + ":" + resultText
			if len(resultDedup) > 200 {
				resultDedup = resultDedup[:200]
			}
			seenTexts[resultDedup] = true
			if toolCallID != "" {
				seenToolResults["tr:"+toolCallID] = true
			}
			if isError {
				resultText = "[error] " + resultText
			}
			if resultText != "" {
				entries = append(entries, AgentLogEntry{
					Timestamp: sessionTimestamp,
					Type:      "tool_result",
					Content:   resultText,
					ToolName:  toolName,
				})
			}

		case "message_end":
			msg, ok := event["message"].(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			contents, _ := msg["content"].([]interface{})
			ts := formatTimestamp(msg["timestamp"], sessionTimestamp)
			model, _ := msg["model"].(string)

			extractContentEntries(&entries, contents, role, ts, model, seenToolCalls, seenTexts)

		case "turn_end":
			// turn_end contains toolResults that aren't in message_end
			ts := sessionTimestamp
			if msg, ok := event["message"].(map[string]interface{}); ok {
				ts = formatTimestamp(msg["timestamp"], sessionTimestamp)
				// Also extract assistant content from turn_end message
				model, _ := msg["model"].(string)
				if contents, ok := msg["content"].([]interface{}); ok {
					extractContentEntries(&entries, contents, "assistant", ts, model, seenToolCalls, seenTexts)
				}
			}
			if results, ok := event["toolResults"].([]interface{}); ok {
				for _, r := range results {
					rm, ok := r.(map[string]interface{})
					if !ok {
						continue
					}
					toolName, _ := rm["toolName"].(string)
					resultText := extractResultContent(rm["content"])
					if resultText == "" {
						continue
					}
					resultKey := "tool_result:" + toolName + ":" + resultText
					if len(resultKey) > 200 {
						resultKey = resultKey[:200]
					}
					if seenTexts[resultKey] {
						continue
					}
					seenTexts[resultKey] = true
					entries = append(entries, AgentLogEntry{
						Timestamp: ts,
						Type:      "tool_result",
						Content:   resultText,
						ToolName:  toolName,
					})
				}
			}

		case "agent_end":
			// agent_end contains the full final conversation — use it to fill
			// any gaps from auto-retry or compaction.
			if messages, ok := event["messages"].([]interface{}); ok {
				for _, m := range messages {
					msg, ok := m.(map[string]interface{})
					if !ok {
						continue
					}
					role, _ := msg["role"].(string)
					ts := formatTimestamp(msg["timestamp"], sessionTimestamp)
					model, _ := msg["model"].(string)

					if role == "toolResult" {
						toolName, _ := msg["toolName"].(string)
						resultText := extractResultContent(msg["content"])
						if resultText == "" {
							continue
						}
						resultKey := "tool_result:" + toolName + ":" + resultText
						if len(resultKey) > 200 {
							resultKey = resultKey[:200]
						}
						if seenTexts[resultKey] {
							continue
						}
						seenTexts[resultKey] = true
						entries = append(entries, AgentLogEntry{
							Timestamp: ts,
							Type:      "tool_result",
							Content:   resultText,
							ToolName:  toolName,
						})
						continue
					}

					if contents, ok := msg["content"].([]interface{}); ok {
						extractContentEntries(&entries, contents, role, ts, model, seenToolCalls, seenTexts)
					}
				}
			}
			entries = append(entries, AgentLogEntry{
				Timestamp: sessionTimestamp,
				Type:      "system",
				Content:   "Agent finished",
			})

		case "agent_start":
			entries = append(entries, AgentLogEntry{
				Timestamp: sessionTimestamp,
				Type:      "system",
				Content:   "Agent started",
			})
		}
	}

	return entries
}

// extractContentEntries parses content blocks and appends structured entries.
func extractContentEntries(entries *[]AgentLogEntry, contents []interface{}, role, ts, model string, seenToolCalls, seenTexts map[string]bool) {
	for _, c := range contents {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		ctype, _ := cm["type"].(string)

		switch ctype {
		case "text":
			text, _ := cm["text"].(string)
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			dedupeKey := role + ":" + text
			if seenTexts[dedupeKey] {
				continue
			}
			seenTexts[dedupeKey] = true
			*entries = append(*entries, AgentLogEntry{
				Timestamp: ts,
				Type:      role,
				Content:   text,
				Model:     model,
			})

		case "toolCall":
			name, _ := cm["name"].(string)
			id, _ := cm["id"].(string)
			dedupeKey := "tc:" + id
			if id != "" && seenToolCalls[dedupeKey] {
				continue // deduplicate
			}
			if id != "" {
				seenToolCalls[dedupeKey] = true
			}
			args, _ := cm["arguments"].(map[string]interface{})
			argsJSON, _ := json.Marshal(args)
			*entries = append(*entries, AgentLogEntry{
				Timestamp: ts,
				Type:      "tool_call",
				ToolName:  name,
				ToolInput: string(argsJSON),
				Model:     model,
			})

		case "toolResult":
			toolName, _ := cm["toolName"].(string)
			resultText := extractResultContent(cm["content"])
			if resultText == "" {
				continue
			}
			resultKey := "tool_result:" + toolName + ":" + resultText
			if len(resultKey) > 200 {
				resultKey = resultKey[:200]
			}
			if seenTexts[resultKey] {
				continue
			}
			seenTexts[resultKey] = true
			*entries = append(*entries, AgentLogEntry{
				Timestamp: ts,
				Type:      "tool_result",
				Content:   resultText,
				ToolName:  toolName,
			})
		}
	}
}

// extractResultContent extracts text from a toolResult content field,
// which can be a string or an array of content blocks.
func extractResultContent(content interface{}) string {
	switch cv := content.(type) {
	case string:
		return cv
	case []interface{}:
		var parts []string
		for _, block := range cv {
			if bm, ok := block.(map[string]interface{}); ok {
				if text, ok := bm["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// formatTimestamp converts a pi-agent timestamp (Unix ms number or ISO string) to ISO format.
func formatTimestamp(ts interface{}, fallback string) string {
	switch v := ts.(type) {
	case float64:
		return time.Unix(int64(v)/1000, (int64(v)%1000)*1e6).UTC().Format(time.RFC3339)
	case string:
		return v
	default:
		return fallback
	}
}
