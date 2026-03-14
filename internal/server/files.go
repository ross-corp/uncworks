package server

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
