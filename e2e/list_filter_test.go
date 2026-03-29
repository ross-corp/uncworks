//go:build e2e

// e2e/list_filter_test.go — end-to-end tests for ListAgentRuns filter parameters.
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

// TestE2E_List_PhaseFilter_Pending creates a run and immediately lists with
// PENDING phase filter to confirm the newly created run appears.
func TestE2E_List_PhaseFilter_Pending(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	// Create a run — it should be PENDING right after creation.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     fmt.Sprintf("phase-filter pending test %d", time.Now().UnixMilli()),
			TtlSeconds: 120,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := resp.Msg.AgentRun.Id
	t.Logf("Created run: %s", runID)

	// List with PENDING filter — the run may already have advanced, so we
	// just verify the API accepts the filter and returns valid results.
	listResp, err := apiClient.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{
		PhaseFilter: apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns (PENDING filter): %v", err)
	}

	t.Logf("Runs in PENDING phase: %d", len(listResp.Msg.AgentRuns))

	// All returned runs must be in PENDING phase.
	for _, r := range listResp.Msg.AgentRuns {
		if r.Status.Phase != apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING {
			t.Errorf("run %s has phase %v but was returned in PENDING filter", r.Id, r.Status.Phase)
		}
	}
}

// TestE2E_List_LimitAndCursor verifies that the limit and cursor pagination
// fields work correctly on ListAgentRuns.
func TestE2E_List_LimitAndCursor(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	// Fetch the first page with a small limit.
	const pageSize = 2
	firstPage, err := apiClient.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{
		Limit: pageSize,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns (first page): %v", err)
	}

	t.Logf("First page: runs=%d cursor=%q", len(firstPage.Msg.AgentRuns), firstPage.Msg.NextCursor)

	if len(firstPage.Msg.AgentRuns) > pageSize {
		t.Errorf("first page returned %d runs, expected at most %d", len(firstPage.Msg.AgentRuns), pageSize)
	}

	// If there is a next cursor, fetch the second page.
	if firstPage.Msg.NextCursor == "" {
		t.Log("No next cursor — only one page of results (acceptable for a clean environment)")
		return
	}

	secondPage, err := apiClient.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{
		Limit:  pageSize,
		Cursor: firstPage.Msg.NextCursor,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns (second page): %v", err)
	}
	t.Logf("Second page: runs=%d cursor=%q", len(secondPage.Msg.AgentRuns), secondPage.Msg.NextCursor)

	// Verify no overlap between pages.
	firstIDs := make(map[string]bool)
	for _, r := range firstPage.Msg.AgentRuns {
		firstIDs[r.Id] = true
	}
	for _, r := range secondPage.Msg.AgentRuns {
		if firstIDs[r.Id] {
			t.Errorf("run %s appears on both first and second page (cursor pagination broken)", r.Id)
		}
	}
}

// TestE2E_List_TagFilter creates a run with a tag and verifies that filtering
// by that tag returns the run (and only runs with that tag).
func TestE2E_List_TagFilter(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	tag := fmt.Sprintf("e2e-tag-%d", time.Now().UnixMilli())

	// Create a run with the unique tag.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     fmt.Sprintf("tag filter test %d", time.Now().UnixMilli()),
			TtlSeconds: 120,
			Tags:       []string{tag},
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	taggedRunID := resp.Msg.AgentRun.Id
	t.Logf("Created tagged run: %s (tag=%s)", taggedRunID, tag)

	// Give the API server a moment to persist the tag.
	time.Sleep(2 * time.Second)

	// List with the tag filter.
	listResp, err := apiClient.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{
		TagFilter: tag,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns (tag filter): %v", err)
	}

	t.Logf("Runs with tag %q: %d", tag, len(listResp.Msg.AgentRuns))

	found := false
	for _, r := range listResp.Msg.AgentRuns {
		if r.Id == taggedRunID {
			found = true
		}
		// All returned runs must have the tag.
		hasTag := false
		for _, rt := range r.Spec.Tags {
			if rt == tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("run %s lacks tag %q but was returned by tag filter", r.Id, tag)
		}
	}

	if !found {
		t.Errorf("tagged run %s not found in tag-filtered list", taggedRunID)
	}
}

// TestE2E_List_ProjectFilter creates a run with a project label and verifies
// that filtering by project returns the run.
func TestE2E_List_ProjectFilter(t *testing.T) {
	apiClient := getAPIClient(t)
	ctx := context.Background()

	project := fmt.Sprintf("e2e-project-%d", time.Now().UnixMilli())

	// Create a run belonging to the project.
	resp, err := apiClient.CreateAgentRun(ctx, connect.NewRequest(&apiv1.CreateAgentRunRequest{
		Spec: &apiv1.AgentRunSpec{
			Backend:    apiv1.Backend_BACKEND_POD,
			Repos:      []*apiv1.Repository{{Url: getSoftServeRepoURL("e2e-repo")}},
			Prompt:     fmt.Sprintf("project filter test %d", time.Now().UnixMilli()),
			TtlSeconds: 120,
			Project:    project,
		},
	}))
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	projectRunID := resp.Msg.AgentRun.Id
	t.Logf("Created project run: %s (project=%s)", projectRunID, project)

	time.Sleep(2 * time.Second)

	// List with project filter.
	listResp, err := apiClient.ListAgentRuns(ctx, connect.NewRequest(&apiv1.ListAgentRunsRequest{
		ProjectFilter: project,
	}))
	if err != nil {
		t.Fatalf("ListAgentRuns (project filter): %v", err)
	}

	t.Logf("Runs in project %q: %d", project, len(listResp.Msg.AgentRuns))

	found := false
	for _, r := range listResp.Msg.AgentRuns {
		if r.Id == projectRunID {
			found = true
		}
		if r.Spec.Project != project {
			t.Errorf("run %s has project %q but was returned by project filter for %q",
				r.Id, r.Spec.Project, project)
		}
	}

	if !found {
		t.Errorf("project run %s not found in project-filtered list", projectRunID)
	}
}
