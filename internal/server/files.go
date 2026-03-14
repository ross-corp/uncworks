package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// FileHandler serves file explorer API endpoints by exec-ing into agent pods.
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

func (f *FileHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/workspace"
	}

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
}

func (f *FileHandler) handleFileContent(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "path query parameter is required"})
		return
	}

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
}

// lookupPodName retrieves the pod name from the AgentRun CRD status.
func (f *FileHandler) lookupPodName(ctx context.Context, runID string) (string, error) {
	crd := &aotv1alpha1.AgentRun{}
	if err := f.k8sClient.Get(ctx, runtimeclient.ObjectKey{
		Namespace: f.namespace,
		Name:      runID,
	}, crd); err != nil {
		return "", err
	}
	return crd.Status.PodName, nil
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

// handleLogs returns the rpc-gateway container logs for an agent run's pod.
func (f *FileHandler) handleLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		http.Error(w, `{"error":"missing run id"}`, http.StatusBadRequest)
		return
	}

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
}
