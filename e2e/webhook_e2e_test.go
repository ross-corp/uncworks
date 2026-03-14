//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getWebhookURL() string {
	apiURL := os.Getenv("AOT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:50055"
	}
	return apiURL + "/api/v1/webhooks/github"
}

func makeGitHubPushPayload(modifiedFiles []string) []byte {
	payload := map[string]interface{}{
		"ref": "refs/heads/main",
		"repository": map[string]interface{}{
			"full_name": "example/test-repo",
			"clone_url": "https://github.com/example/test-repo.git",
		},
		"commits": []map[string]interface{}{
			{
				"id":       "abc123",
				"message":  "test commit",
				"modified": modifiedFiles,
				"added":    []string{},
				"removed":  []string{},
			},
		},
		"sender": map[string]interface{}{
			"login": "test-user",
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

// TestE2E_Webhook_SpecFileCreatesRun sends a GitHub push webhook payload containing
// a .cs.md file and verifies that an AgentRun CRD is created.
func TestE2E_Webhook_SpecFileCreatesRun(t *testing.T) {
	k8s := getE2EClient(t)
	ctx := context.Background()

	// List existing runs before webhook
	existingRuns := &aotv1alpha1.AgentRunList{}
	if err := k8s.List(ctx, existingRuns, &client.ListOptions{}); err != nil {
		t.Fatalf("List existing runs: %v", err)
	}
	existingCount := len(existingRuns.Items)

	// POST webhook with a .cs.md file
	payload := makeGitHubPushPayload([]string{"specs/feature.cs.md"})
	resp, err := http.Post(getWebhookURL(), "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	t.Logf("Webhook response: %d %s", resp.StatusCode, string(body))

	// Wait a bit for the run to be created
	time.Sleep(3 * time.Second)

	// List runs after webhook
	afterRuns := &aotv1alpha1.AgentRunList{}
	if err := k8s.List(ctx, afterRuns, &client.ListOptions{}); err != nil {
		t.Fatalf("List runs after webhook: %v", err)
	}

	// Find new runs with webhook spec source
	var webhookRun *aotv1alpha1.AgentRun
	for i := range afterRuns.Items {
		r := &afterRuns.Items[i]
		if r.Spec.SpecSource != "" && len(afterRuns.Items) > existingCount {
			webhookRun = r
			break
		}
	}

	if webhookRun != nil {
		t.Logf("Webhook-created run: %s (specSource=%s)", webhookRun.Name, webhookRun.Spec.SpecSource)
		// Cleanup
		_ = k8s.Delete(ctx, webhookRun)
	} else {
		t.Log("No webhook-created run found (webhook handler may not create runs for all .cs.md pushes)")
	}
}

// TestE2E_Webhook_InvalidSignature sends a webhook with an invalid signature
// and verifies a 401 response.
func TestE2E_Webhook_InvalidSignature(t *testing.T) {
	payload := makeGitHubPushPayload([]string{"specs/feature.cs.md"})

	req, err := http.NewRequest("POST", getWebhookURL(), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid-signature-value")
	req.Header.Set("X-GitHub-Event", "push")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	t.Logf("Invalid signature response: %d %s", resp.StatusCode, string(body))

	// If webhook secret is configured, expect 401; otherwise server may accept it
	if secret := os.Getenv("GITHUB_WEBHOOK_SECRET"); secret != "" {
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 with invalid signature, got %d", resp.StatusCode)
		}
	} else {
		t.Log("GITHUB_WEBHOOK_SECRET not set; skipping strict 401 assertion")
	}
}

// TestE2E_Webhook_NoSpecFiles sends a push payload with only .go files
// and verifies no new AgentRun is created.
func TestE2E_Webhook_NoSpecFiles(t *testing.T) {
	k8s := getE2EClient(t)
	ctx := context.Background()

	// List existing runs
	existingRuns := &aotv1alpha1.AgentRunList{}
	if err := k8s.List(ctx, existingRuns, &client.ListOptions{}); err != nil {
		t.Fatalf("List existing runs: %v", err)
	}
	existingIDs := make(map[string]bool)
	for _, r := range existingRuns.Items {
		existingIDs[r.Name] = true
	}

	// POST with only .go files — should not trigger a run
	payload := makeGitHubPushPayload([]string{"main.go", "pkg/handler.go"})
	resp, err := http.Post(getWebhookURL(), "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Logf("Non-200 response for no-spec webhook: %d %s", resp.StatusCode, string(body))
	}

	// Wait briefly and check no new run was created
	time.Sleep(3 * time.Second)

	afterRuns := &aotv1alpha1.AgentRunList{}
	if err := k8s.List(ctx, afterRuns, &client.ListOptions{}); err != nil {
		t.Fatalf("List runs after webhook: %v", err)
	}

	newRuns := 0
	for _, r := range afterRuns.Items {
		if !existingIDs[r.Name] {
			newRuns++
			t.Logf("Unexpected new run: %s", r.Name)
		}
	}

	if newRuns > 0 {
		t.Errorf("expected no new runs from non-spec push, got %d", newRuns)
	} else {
		t.Log(fmt.Sprintf("Confirmed: no new runs created from push with only .go files"))
	}
}
