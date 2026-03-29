//go:build e2e

// e2e/chain_e2e_test.go — end-to-end tests for Chain execution ordering.
// Verifies that templates and chains can be created, triggered, and that
// step ordering is preserved by the controller.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// chainAPIBase returns the base URL for chain/template REST endpoints.
func chainAPIBase() string {
	return apiBaseURL()
}

// createTemplate creates a RunTemplate via the REST API and returns its name.
// It registers cleanup to delete the template after the test.
func createTemplate(t *testing.T, name, prompt string) {
	t.Helper()

	body := map[string]interface{}{
		"name":        name,
		"displayName": name,
		"prompt":      prompt,
		"ttlSeconds":  120,
		"repos": []map[string]interface{}{
			{"url": getSoftServeRepoURL("e2e-repo"), "branch": "main"},
		},
	}
	payload, _ := json.Marshal(body)

	resp, err := http.Post(
		chainAPIBase()+"/api/v1/templates",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/templates: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201 from POST /api/v1/templates, got %d: %s", resp.StatusCode, string(respBody))
	}
	t.Logf("Created template %q: HTTP %d", name, resp.StatusCode)

	t.Cleanup(func() {
		req, _ := http.NewRequest(http.MethodDelete, chainAPIBase()+"/api/v1/templates/"+name, nil)
		r, err := http.DefaultClient.Do(req)
		if err == nil {
			r.Body.Close()
		}
	})
}

// TestE2E_Chain_TemplateCreateAndGet verifies that a RunTemplate can be created
// via REST and immediately retrieved.
func TestE2E_Chain_TemplateCreateAndGet(t *testing.T) {
	name := fmt.Sprintf("e2e-tmpl-%d", time.Now().UnixMilli())
	createTemplate(t, name, "Create a file called TEMPLATE_TEST.txt containing 'ok'")

	// GET the template back.
	resp, err := http.Get(chainAPIBase() + "/api/v1/templates/" + name)
	if err != nil {
		t.Fatalf("GET /api/v1/templates/%s: %v", name, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from GET template, got %d: %s", resp.StatusCode, string(body))
	}

	var tmpl map[string]interface{}
	if err := json.Unmarshal(body, &tmpl); err != nil {
		t.Fatalf("Parse template JSON: %v", err)
	}

	specRaw, ok := tmpl["spec"]
	if !ok {
		t.Fatal("template response missing 'spec' field")
	}
	spec, ok := specRaw.(map[string]interface{})
	if !ok {
		t.Fatal("spec is not an object")
	}

	if spec["displayName"] != name {
		t.Errorf("expected displayName %q, got %v", name, spec["displayName"])
	}
	t.Logf("Template %q retrieved successfully", name)
}

// TestE2E_Chain_TemplateList verifies that ListTemplates includes a newly
// created template.
func TestE2E_Chain_TemplateList(t *testing.T) {
	name := fmt.Sprintf("e2e-tmpl-list-%d", time.Now().UnixMilli())
	createTemplate(t, name, "list test prompt")

	resp, err := http.Get(chainAPIBase() + "/api/v1/templates")
	if err != nil {
		t.Fatalf("GET /api/v1/templates: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var templates []map[string]interface{}
	if err := json.Unmarshal(body, &templates); err != nil {
		t.Fatalf("Parse templates list: %v", err)
	}

	found := false
	for _, tmpl := range templates {
		meta, _ := tmpl["metadata"].(map[string]interface{})
		if meta != nil && meta["name"] == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("newly created template %q not found in list", name)
	} else {
		t.Logf("Template %q found in list (%d total)", name, len(templates))
	}
}

// TestE2E_Chain_CreateAndTrigger creates a two-step Chain and triggers it,
// verifying that a ChainRun is created with the expected step structure.
func TestE2E_Chain_CreateAndTrigger(t *testing.T) {
	base := time.Now().UnixMilli()
	tmplA := fmt.Sprintf("e2e-tmpl-a-%d", base)
	tmplB := fmt.Sprintf("e2e-tmpl-b-%d", base)
	chainName := fmt.Sprintf("e2e-chain-%d", base)

	// Create two templates for the steps.
	createTemplate(t, tmplA, "Create a file called CHAIN_A.txt containing 'step-a'")
	createTemplate(t, tmplB, "Create a file called CHAIN_B.txt containing 'step-b'")

	// Create the chain with step-b depending on step-a.
	chainBody := map[string]interface{}{
		"name":        chainName,
		"displayName": chainName,
		"steps": []map[string]interface{}{
			{"name": "step-a", "templateRef": tmplA},
			{"name": "step-b", "templateRef": tmplB, "dependsOn": []string{"step-a"}},
		},
	}
	payload, _ := json.Marshal(chainBody)
	resp, err := http.Post(
		chainAPIBase()+"/api/v1/chains",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/chains: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201 from POST /api/v1/chains, got %d: %s", resp.StatusCode, string(respBody))
	}
	t.Logf("Created chain %q: HTTP %d", chainName, resp.StatusCode)

	// Register cleanup for the chain.
	t.Cleanup(func() {
		req, _ := http.NewRequest(http.MethodDelete, chainAPIBase()+"/api/v1/chains/"+chainName, nil)
		r, _ := http.DefaultClient.Do(req)
		if r != nil {
			r.Body.Close()
		}
	})

	// Trigger the chain.
	triggerResp, err := http.Post(
		chainAPIBase()+"/api/v1/chains/"+chainName+"/trigger",
		"application/json",
		bytes.NewReader([]byte("{}")),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/chains/%s/trigger: %v", chainName, err)
	}
	defer triggerResp.Body.Close()
	triggerBody, _ := io.ReadAll(triggerResp.Body)

	if triggerResp.StatusCode != http.StatusCreated && triggerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 2xx from trigger, got %d: %s", triggerResp.StatusCode, string(triggerBody))
	}
	t.Logf("Triggered chain %q: HTTP %d body=%s", chainName, triggerResp.StatusCode, string(triggerBody))

	// Parse out the ChainRun name from the trigger response.
	var triggerResult map[string]interface{}
	if err := json.Unmarshal(triggerBody, &triggerResult); err != nil {
		t.Logf("Could not parse trigger response (non-fatal): %v", err)
		return
	}

	chainRunName := ""
	if meta, ok := triggerResult["metadata"].(map[string]interface{}); ok {
		chainRunName, _ = meta["name"].(string)
	}
	if chainRunName == "" {
		t.Log("Could not extract chainRun name from trigger response; skipping step-order check")
		return
	}
	t.Logf("ChainRun created: %s", chainRunName)

	// Give the controller a moment to start processing.
	time.Sleep(5 * time.Second)

	// GET /api/v1/chainruns/{name} and verify step structure.
	crResp, err := http.Get(chainAPIBase() + "/api/v1/chainruns/" + chainRunName)
	if err != nil {
		t.Fatalf("GET /api/v1/chainruns/%s: %v", chainRunName, err)
	}
	defer crResp.Body.Close()
	crBody, _ := io.ReadAll(crResp.Body)

	if crResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from GET chainrun, got %d: %s", crResp.StatusCode, string(crBody))
	}

	var chainRun map[string]interface{}
	if err := json.Unmarshal(crBody, &chainRun); err != nil {
		t.Fatalf("Parse ChainRun JSON: %v", err)
	}

	status, _ := chainRun["status"].(map[string]interface{})
	if status == nil {
		t.Log("ChainRun has no status yet (controller may not have reconciled)")
		return
	}

	steps, _ := status["steps"].([]interface{})
	t.Logf("ChainRun steps count: %d", len(steps))

	// Verify that step-b appears after step-a in the status list (ordering
	// reflects dependency: step-a must be first since step-b depends on it).
	stepAIdx, stepBIdx := -1, -1
	for i, s := range steps {
		step, _ := s.(map[string]interface{})
		if step == nil {
			continue
		}
		switch step["name"] {
		case "step-a":
			stepAIdx = i
		case "step-b":
			stepBIdx = i
		}
	}

	if stepAIdx == -1 || stepBIdx == -1 {
		t.Logf("Steps not yet visible in status (stepAIdx=%d stepBIdx=%d); controller may still be starting",
			stepAIdx, stepBIdx)
		return
	}

	// The dependency model does not guarantee index ordering in the status
	// slice, but step-b must not be in Running or Succeeded phase while
	// step-a is still Pending.
	stepAPhase := ""
	stepBPhase := ""
	if sA, _ := steps[stepAIdx].(map[string]interface{}); sA != nil {
		stepAPhase, _ = sA["phase"].(string)
	}
	if sB, _ := steps[stepBIdx].(map[string]interface{}); sB != nil {
		stepBPhase, _ = sB["phase"].(string)
	}
	t.Logf("step-a phase=%s, step-b phase=%s", stepAPhase, stepBPhase)

	if stepAPhase == "pending" && (stepBPhase == "running" || stepBPhase == "succeeded") {
		t.Error("step-b is running/succeeded while step-a is still pending — dependency ordering violated")
	}
}

// TestE2E_Chain_GetNonExistent verifies that fetching a chain that does not
// exist returns a 404.
func TestE2E_Chain_GetNonExistent(t *testing.T) {
	resp, err := http.Get(chainAPIBase() + "/api/v1/chains/non-existent-chain-xyz")
	if err != nil {
		t.Skipf("Cannot reach chains endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 404 for non-existent chain, got %d: %s", resp.StatusCode, string(body))
	} else {
		t.Log("Got expected 404 for non-existent chain")
	}
}

// TestE2E_Chain_DeleteTemplateReferencedByChain verifies that deleting a
// template that is referenced by an existing chain returns a 409 Conflict.
func TestE2E_Chain_DeleteTemplateReferencedByChain(t *testing.T) {
	base := time.Now().UnixMilli()
	tmplName := fmt.Sprintf("e2e-tmpl-ref-%d", base)
	chainName := fmt.Sprintf("e2e-chain-ref-%d", base)

	createTemplate(t, tmplName, "referenced template test")

	// Create a chain referencing the template.
	chainBody := map[string]interface{}{
		"name": chainName,
		"steps": []map[string]interface{}{
			{"name": "only-step", "templateRef": tmplName},
		},
	}
	payload, _ := json.Marshal(chainBody)
	resp, err := http.Post(chainAPIBase()+"/api/v1/chains", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /api/v1/chains: %v", err)
	}
	resp.Body.Close()

	t.Cleanup(func() {
		req, _ := http.NewRequest(http.MethodDelete, chainAPIBase()+"/api/v1/chains/"+chainName, nil)
		r, _ := http.DefaultClient.Do(req)
		if r != nil {
			r.Body.Close()
		}
	})

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("chain creation failed: %d", resp.StatusCode)
	}

	// Attempt to delete the template — should be rejected.
	req, _ := http.NewRequest(http.MethodDelete, chainAPIBase()+"/api/v1/templates/"+tmplName, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/v1/templates/%s: %v", tmplName, err)
	}
	defer delResp.Body.Close()
	delBody, _ := io.ReadAll(delResp.Body)

	t.Logf("DELETE referenced template: HTTP %d body=%s", delResp.StatusCode, string(delBody))

	if delResp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 Conflict when deleting a template referenced by a chain, got %d", delResp.StatusCode)
	} else {
		t.Log("Got expected 409 — template deletion blocked by chain reference")
	}
}
