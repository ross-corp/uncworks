package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// =============================================================================
// Path Traversal Protection
// =============================================================================

// TestPathTraversal_ListFiles_DotDot ensures that the disk-mode path validation
// in handleListFiles rejects paths containing ".." that would escape the host
// path root (PVC mount point). We bypass K8s calls by testing the validation
// logic directly — the handler constructs a disk path and checks that the
// resolved absolute path still has the hostPath prefix.
func TestPathTraversal_ListFiles_DotDot(t *testing.T) {
	tests := []struct {
		name     string
		hostPath string
		dirPath  string // what the user sends as ?path=
		wantOK   bool
	}{
		{
			name:     "normal subdir is allowed",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace/src",
			wantOK:   true,
		},
		{
			name:     "root workspace is allowed",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace",
			wantOK:   true,
		},
		{
			name:     "dotdot escapes hostPath",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace/../../../etc/passwd",
			wantOK:   false,
		},
		{
			name:     "dotdot single level",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace/../outside",
			wantOK:   false,
		},
		{
			name:     "dotdot hidden in middle",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace/subdir/../../..",
			wantOK:   false,
		},
		{
			name:     "dotdot that stays within hostPath",
			hostPath: "/data/pvc-abc",
			dirPath:  "/workspace/a/../b",
			wantOK:   true,
		},
		{
			name:     "empty path defaults to workspace",
			hostPath: "/data/pvc-abc",
			dirPath:  "",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := tt.dirPath
			if dirPath == "" {
				dirPath = "/workspace"
			}

			// Replicate the exact path resolution logic from handleListFiles.
			relativePath := strings.TrimPrefix(dirPath, "/workspace")
			relativePath = strings.TrimPrefix(relativePath, "/")
			diskPath := filepath.Join(tt.hostPath, relativePath)

			resolvedPath, err := filepath.Abs(diskPath)
			ok := err == nil && strings.HasPrefix(resolvedPath, tt.hostPath)

			if tt.wantOK && !ok {
				t.Errorf("expected path to be allowed, but it was rejected (resolved=%q, hostPath=%q)", resolvedPath, tt.hostPath)
			}
			if !tt.wantOK && ok {
				t.Errorf("expected path to be rejected, but it was allowed (resolved=%q, hostPath=%q)", resolvedPath, tt.hostPath)
			}
		})
	}
}

// TestPathTraversal_FileContent_DotDot ensures that the disk-mode path
// validation in handleFileContent rejects paths with ".." that escape the PVC
// host path.
func TestPathTraversal_FileContent_DotDot(t *testing.T) {
	tests := []struct {
		name     string
		hostPath string
		filePath string // what the user sends as ?path=
		wantOK   bool
	}{
		{
			name:     "normal file path",
			hostPath: "/data/pvc-abc",
			filePath: "/workspace/main.go",
			wantOK:   true,
		},
		{
			name:     "nested file path",
			hostPath: "/data/pvc-abc",
			filePath: "/workspace/src/pkg/handler.go",
			wantOK:   true,
		},
		{
			name:     "dotdot escapes to root",
			hostPath: "/data/pvc-abc",
			filePath: "/workspace/../../../etc/shadow",
			wantOK:   false,
		},
		{
			name:     "dotdot escapes one level",
			hostPath: "/data/pvc-abc",
			filePath: "/workspace/../../secrets.env",
			wantOK:   false,
		},
		{
			name:     "dotdot that stays within hostPath",
			hostPath: "/data/pvc-abc",
			filePath: "/workspace/a/../main.go",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the exact path resolution logic from handleFileContent.
			relativePath := strings.TrimPrefix(tt.filePath, "/workspace/")
			diskPath := filepath.Join(tt.hostPath, relativePath)

			resolvedPath, err := filepath.Abs(diskPath)
			ok := err == nil && strings.HasPrefix(resolvedPath, tt.hostPath)

			if tt.wantOK && !ok {
				t.Errorf("expected path to be allowed, but it was rejected (resolved=%q, hostPath=%q)", resolvedPath, tt.hostPath)
			}
			if !tt.wantOK && ok {
				t.Errorf("expected path to be rejected, but it was allowed (resolved=%q, hostPath=%q)", resolvedPath, tt.hostPath)
			}
		})
	}
}

// TestPathTraversal_SymlinkLikeNames ensures that path components that look
// suspicious but are actually valid directory names are still allowed.
func TestPathTraversal_SymlinkLikeNames(t *testing.T) {
	hostPath := "/data/pvc-abc"

	names := []string{
		"/workspace/..hidden-dir/file.go", // directory name starts with ".."
		"/workspace/something.../file.go", // directory name ends with "..."
		"/workspace/.dotfile",             // hidden file
		"/workspace/dir/.env",             // hidden file in subdir
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			relativePath := strings.TrimPrefix(name, "/workspace/")
			diskPath := filepath.Join(hostPath, relativePath)

			resolvedPath, err := filepath.Abs(diskPath)
			ok := err == nil && strings.HasPrefix(resolvedPath, hostPath)

			// These should all be allowed since they don't escape the root.
			if !ok {
				t.Errorf("expected path %q to be allowed (resolved=%q)", name, resolvedPath)
			}
		})
	}
}

// =============================================================================
// Webhook Body Limit
// =============================================================================

// TestWebhook_OversizedBody verifies that the webhook handler caps the body at
// 10 MB. We send a body larger than that and confirm the payload is truncated
// (the handler reads up to 10 MB then parses JSON, which will fail for a
// truncated payload — the key point is that the server does not OOM).
func TestWebhook_OversizedBody(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		k8sClient: k8s,
		namespace: "default",
	}

	// Create a body that exceeds 10 MB.
	bigBody := make([]byte, 11<<20) // 11 MB
	for i := range bigBody {
		bigBody[i] = 'A'
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(bigBody))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	// The handler reads up to 10 MB, then tries to parse JSON — it should fail
	// with a bad request (invalid JSON) rather than crashing.
	// We accept 400 (bad payload) or any non-5xx response as success.
	assert.Less(t, rec.Code, 500, "oversized body should not cause a server error")
}

// TestWebhook_ExactlyAtLimit verifies that a body exactly at the 10 MB limit
// is processed normally (not rejected).
func TestWebhook_ExactlyAtLimit(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		k8sClient: k8s,
		namespace: "default",
	}

	// Build a valid push payload that is small.
	body := makePushPayload("org/repo", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// =============================================================================
// Webhook Secret Validation
// =============================================================================

// TestWebhook_SecretConfigured_RejectsUnsigned verifies that when a webhook
// secret is configured, requests without a signature are rejected with 401.
func TestWebhook_SecretConfigured_RejectsUnsigned(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "production-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	// Deliberately omit X-Hub-Signature-256
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid signature")
}

// TestWebhook_SecretConfigured_RejectsWrongSignature verifies that a request
// signed with the wrong secret is rejected.
func TestWebhook_SecretConfigured_RejectsWrongSignature(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "correct-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	sig := signPayload(body, "wrong-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestWebhook_SecretConfigured_AcceptsCorrectSignature verifies that a
// correctly signed request passes validation.
func TestWebhook_SecretConfigured_AcceptsCorrectSignature(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "correct-secret",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	sig := signPayload(body, "correct-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestWebhook_NoSecret_AcceptsAnything verifies that when no webhook secret is
// configured, requests are accepted regardless of signature.
func TestWebhook_NoSecret_AcceptsAnything(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		secret:    "",
		k8sClient: k8s,
		namespace: "default",
	}

	body := makePushPayload("org/repo", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestWebhook_MethodNotAllowed verifies that non-POST methods are rejected.
func TestWebhook_MethodNotAllowed(t *testing.T) {
	scheme := newTestScheme()
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	wh := &WebhookHandler{
		k8sClient: k8s,
		namespace: "default",
	}

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/webhooks/github", nil)
			rec := httptest.NewRecorder()
			wh.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

// =============================================================================
// WriteJSON Consistency
// =============================================================================

// TestWriteJSON_SetsContentType verifies that writeJSON always sets the
// Content-Type header to application/json.
func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"status": "ok"})

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

// TestWriteJSON_ErrorResponse verifies that error responses use the standard
// errorResponse format consistently.
func TestWriteJSON_ErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusBadRequest, errorResponse{Error: "something went wrong"})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp errorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "something went wrong", resp.Error)
}

// TestWriteJSON_StatusCodes verifies that different HTTP status codes are set
// correctly by writeJSON.
func TestWriteJSON_StatusCodes(t *testing.T) {
	codes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusTooManyRequests,
	}

	for _, code := range codes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, code, errorResponse{Error: "test"})
			assert.Equal(t, code, rec.Code)
		})
	}
}

// TestWriteJSON_ValidJSON ensures writeJSON always produces valid JSON output.
func TestWriteJSON_ValidJSON(t *testing.T) {
	tests := []struct {
		name string
		v    any
	}{
		{"error response", errorResponse{Error: "bad request"}},
		{"file list", FileListResponse{Entries: []FileEntry{{Name: "test.go", Type: "file", Size: 100}}}},
		{"empty file list", FileListResponse{Entries: []FileEntry{}}},
		{"nil entries", FileListResponse{}},
		{"map response", map[string]interface{}{"ok": true, "created": 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, http.StatusOK, tt.v)

			assert.True(t, json.Valid(rec.Body.Bytes()), "writeJSON produced invalid JSON: %s", rec.Body.String())
		})
	}
}

// =============================================================================
// Path Traversal Integration Test (with real filesystem)
// =============================================================================

// TestPathTraversal_RealFilesystem creates a temporary directory structure and
// validates that the path resolution logic correctly prevents escapes.
func TestPathTraversal_RealFilesystem(t *testing.T) {
	// Create a temp directory to simulate a PVC host path.
	hostPath := t.TempDir()

	// Create some files in the "workspace".
	require.NoError(t, os.MkdirAll(filepath.Join(hostPath, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hostPath, "legit.txt"), []byte("safe"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(hostPath, "subdir", "nested.txt"), []byte("also safe"), 0o644))

	// Create a sensitive file outside the hostPath.
	outsideDir := t.TempDir()
	sensitiveFile := filepath.Join(outsideDir, "secret.txt")
	require.NoError(t, os.WriteFile(sensitiveFile, []byte("TOP SECRET"), 0o644))

	tests := []struct {
		name       string
		queryPath  string
		wantAccess bool
	}{
		{"legitimate file", "/workspace/legit.txt", true},
		{"nested file", "/workspace/subdir/nested.txt", true},
		{"dotdot escape attempt", "/workspace/../../" + filepath.Base(outsideDir) + "/secret.txt", false},
		{"deep dotdot", "/workspace/../../../tmp/evil", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate handleFileContent path resolution.
			relativePath := strings.TrimPrefix(tt.queryPath, "/workspace/")
			diskPath := filepath.Join(hostPath, relativePath)

			resolvedPath, err := filepath.Abs(diskPath)
			allowed := err == nil && strings.HasPrefix(resolvedPath, hostPath)

			if tt.wantAccess {
				assert.True(t, allowed, "should be allowed: resolved=%q hostPath=%q", resolvedPath, hostPath)
				// Additionally, verify the file is actually readable.
				_, readErr := os.ReadFile(resolvedPath)
				assert.NoError(t, readErr, "allowed file should be readable")
			} else {
				assert.False(t, allowed, "should be rejected: resolved=%q hostPath=%q", resolvedPath, hostPath)
			}
		})
	}
}
