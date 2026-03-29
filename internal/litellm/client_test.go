package litellm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-master-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}

		var req GenerateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.KeyAlias != "aot-default-test-run" {
			t.Errorf("unexpected key_alias: %s", req.KeyAlias)
		}
		if req.MaxBudget == nil || *req.MaxBudget != 5.0 {
			t.Errorf("unexpected max_budget: %v", req.MaxBudget)
		}
		if len(req.Models) != 2 || req.Models[0] != "default" {
			t.Errorf("unexpected models: %v", req.Models)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GenerateKeyResponse{
			Key:     "sk-generated-key-123",
			KeyName: "aot-default-test-run",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-master-key")
	budget := 5.0
	resp, err := client.GenerateKey(context.Background(), GenerateKeyRequest{
		KeyAlias:  "aot-default-test-run",
		MaxBudget: &budget,
		Models:    []string{"default", "default-cloud"},
	})
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if resp.Key != "sk-generated-key-123" {
		t.Errorf("unexpected key: %s", resp.Key)
	}
}

func TestGenerateKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid master key"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "wrong-key")
	_, err := client.GenerateKey(context.Background(), GenerateKeyRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}

func TestDeleteKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req DeleteKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(req.Keys) != 1 || req.Keys[0] != "sk-to-delete" {
			t.Errorf("unexpected keys: %v", req.Keys)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DeleteKeyResponse{
			DeletedKeys: []string{"sk-to-delete"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-master-key")
	resp, err := client.DeleteKey(context.Background(), []string{"sk-to-delete"})
	if err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if len(resp.DeletedKeys) != 1 || resp.DeletedKeys[0] != "sk-to-delete" {
		t.Errorf("unexpected deleted keys: %v", resp.DeletedKeys)
	}
}

func TestDeleteKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.DeleteKey(context.Background(), []string{"sk-x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMasterKey(t *testing.T) {
	c := NewClient("http://localhost:4000", "my-master-key")
	if got := c.MasterKey(); got != "my-master-key" {
		t.Errorf("MasterKey() = %q, want %q", got, "my-master-key")
	}
}

func TestSafeBody_RedactsSKToken(t *testing.T) {
	input := `{"key":"sk-abc123","other":"value"}`
	got := safeBody([]byte(input))
	if strings.Contains(got, "sk-abc123") {
		t.Errorf("safeBody did not redact sk- token; got %q", got)
	}
	if !strings.Contains(got, "sk-[REDACTED]") {
		t.Errorf("safeBody missing redaction marker; got %q", got)
	}
}

func TestSafeBody_TruncatesLongBody(t *testing.T) {
	// Build a body longer than 512 bytes with an sk- token after byte 512.
	prefix := strings.Repeat("x", 520)
	input := prefix + "sk-shouldnotappear"
	got := safeBody([]byte(input))
	if len(got) > 512+len("sk-[REDACTED]") {
		// Allow for redaction expansion, but body should be truncated
		t.Errorf("safeBody output too long (%d bytes), expected truncation", len(got))
	}
	if strings.Contains(got, "sk-shouldnotappear") {
		t.Error("safeBody leaked sk- token that was beyond truncation boundary")
	}
}

func TestSafeBody_NoTokenPassthrough(t *testing.T) {
	input := `{"error": "not found"}`
	got := safeBody([]byte(input))
	if got != input {
		t.Errorf("safeBody altered body with no sk- token: got %q, want %q", got, input)
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4","object":"model"},{"id":"claude-3","object":"model"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	resp, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "gpt-4" {
		t.Errorf("Data[0].ID = %q, want %q", resp.Data[0].ID, "gpt-4")
	}
}

func TestListModels_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid key"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-key")
	_, err := c.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestModelIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-a"},{"id":"model-b"},{"id":"model-c"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	ids, err := c.ModelIDs(context.Background())
	if err != nil {
		t.Fatalf("ModelIDs: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	if ids[0] != "model-a" || ids[1] != "model-b" || ids[2] != "model-c" {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestModelIDs_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	_, err := c.ModelIDs(context.Background())
	if err == nil {
		t.Fatal("expected error when ListModels fails")
	}
}
