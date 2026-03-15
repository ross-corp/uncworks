## Context

AOT currently models one run = one agent. A user creates an `AgentRun` CRD with a prompt, the controller starts a Temporal workflow, and a single agent executes it. The existing `SpawnJuniorWorkflow` function can create child workflows, but there is no data model, orchestration logic, or UI support for multi-agent decomposition. This design introduces the SpecRun orchestration layer.

## Goals / Non-Goals

**Goals:**
- A spec can decompose into multiple focused agent runs automatically
- Users can explicitly define the orchestration tree when they want control
- Parent-child relationships are tracked and queryable
- The run graph is visualizable in the web UI
- Backward compatible: existing single-agent runs work unchanged

**Non-Goals:**
- Dynamic re-planning (senior adjusting the plan mid-execution based on junior failures) — future work
- Cross-run shared memory or context windows — juniors work independently
- Distributed file locking across junior workspaces — out of scope
- Authentication/authorization for orchestration — separate concern

## Decisions

### 1. SpecRun is a label, not a CRD

SpecRun is implemented as a label (`aot.uncworks.io/spec-run-id`) and annotations on existing AgentRun CRDs. No new CRD.

**Why not a separate CRD?** A SpecRun CRD would duplicate lifecycle management already handled by AgentRun + Temporal. The label approach keeps the resource model flat — you can query all runs in a spec execution with `kubectl get agentruns -l aot.uncworks.io/spec-run-id=<id>`. The senior AgentRun itself serves as the "root" of the graph. Adding a CRD adds reconciliation complexity without clear benefit.

**Labels and annotations:**
- `aot.uncworks.io/spec-run-id` (label): Groups all AgentRuns from one spec execution. Set on both senior and junior runs. Value is the senior's AgentRun name.
- `aot.uncworks.io/run-role` (label): `senior` or `junior`. Enables filtering.
- `aot.uncworks.io/parent-run` (annotation): Name of the parent AgentRun. Empty for the root senior.

### 2. Orchestration mode lives on AgentRunSpec

Three modes, specified via a new `orchestrationMode` field on `AgentRunSpec`:

- **`single`** (default): Current behavior. One agent, one run. No decomposition. This is the zero-config path — existing users notice nothing.
- **`auto`**: Senior agent reads the spec/prompt, produces a decomposition plan as structured JSON output, and the workflow spawns junior AgentRuns. The senior waits for all juniors to complete, then reviews their outputs.
- **`manual`**: User defines the orchestration tree in the spec via an `orchestration` field containing a list of sub-tasks with prompts. The workflow spawns these directly without senior agent involvement.

**Hick's Law applied:** The mode selection is a single enum field with a sensible default. Users don't need to think about orchestration unless they want to.

### 3. How the senior decides to decompose (auto mode)

The senior agent's prompt is augmented with a decomposition preamble:

```
You are a senior engineer. Analyze the following spec and decompose it into
independent sub-tasks that can be executed in parallel by junior agents.

Output a JSON object with this schema:
{
  "tasks": [
    {
      "name": "short-kebab-case-name",
      "prompt": "Detailed task description for the junior agent",
      "repos": ["optional subset of repos relevant to this task"]
    }
  ],
  "integration_prompt": "Instructions for reviewing and integrating junior outputs"
}

Rules:
- Each task should be independently executable
- Each task should produce a clear, verifiable output (code change, test, etc.)
- Maximum 7 tasks (if more are needed, group related work)
- If the spec is simple enough for one agent, return {"tasks": []} and it will
  be executed as a single run
```

The workflow calls `StartAgent` with this augmented prompt, collects the structured output via `StreamOutput`, parses the JSON, then spawns junior workflows.

**Miller's Law applied:** The decomposition is capped at 7 sub-tasks. If the senior identifies more, it must group related work. This keeps the run graph comprehensible.

**Why structured JSON output?** Text is the universal interface (Unix philosophy), and JSON is structured text. The senior's output is machine-parseable, enabling the workflow to spawn juniors programmatically. If the JSON is malformed, the workflow falls back to single-run mode.

### 4. Junior workspaces: separate with result aggregation

Each junior gets its own workspace (PVC + deployment), cloning the same repos. Juniors work independently — no shared filesystem, no coordination. This follows Unix philosophy: each process has its own address space.

**Result aggregation:** When all juniors complete, the senior agent is re-invoked with a review prompt containing:
- The original spec
- Each junior's task description and final status (succeeded/failed)
- Each junior's git diff output (collected from the workspace)

The senior reviews, identifies conflicts or gaps, and produces a final integration commit if needed. The senior's workspace contains the original repo state; it applies junior diffs as patches.

**Why not shared workspaces?** Shared filesystems introduce race conditions, merge conflicts during execution, and coupling between juniors. Separate workspaces with post-hoc integration is simpler, more predictable, and easier to debug. A failed junior doesn't corrupt the workspace for others.

### 5. Run graph data model

The parent-child relationship is tracked via:

```protobuf
message AgentRunSpec {
  // ... existing fields ...
  string parent_run_id = 15;
  OrchestrationMode orchestration_mode = 16;
  Orchestration orchestration = 17;
}

enum OrchestrationMode {
  ORCHESTRATION_MODE_UNSPECIFIED = 0;
  ORCHESTRATION_MODE_SINGLE = 1;
  ORCHESTRATION_MODE_AUTO = 2;
  ORCHESTRATION_MODE_MANUAL = 3;
}

message Orchestration {
  repeated OrchestrationTask tasks = 1;
}

message OrchestrationTask {
  string name = 1;
  string prompt = 2;
  repeated string repo_urls = 3;
}
```

The CRD mirrors this with Go types. The controller sets `spec-run-id` and `run-role` labels when creating AgentRuns that have a `parent_run_id`.

**Querying the graph:** `ListAgentRuns` gains an optional `spec_run_id` filter. The API returns all runs in the spec execution, and the client reconstructs the tree from `parent_run_id` pointers.

### 6. Workflow changes

The `AgentRunWorkflow` gains an orchestration preamble before step 1:

```
Step 0: Orchestration check
  if orchestration_mode == "auto":
    → Run senior decomposition (start agent with decomposition prompt)
    → Parse structured output
    → For each task: spawn junior via SpawnJuniorWorkflow (blocking=true, parallel via goroutines)
    → Wait for all juniors
    → Run senior integration (start agent with review prompt + junior diffs)
    → Return
  if orchestration_mode == "manual":
    → For each task in orchestration.tasks: spawn junior via SpawnJuniorWorkflow
    → Wait for all juniors
    → Return (no senior review — user defined the tree, they review)
  if orchestration_mode == "single" or unspecified:
    → Continue with existing workflow (no change)
```

The existing `SpawnJuniorWorkflow` is already designed for this. The key change is making it set `parent_run_id` and labels on the child's `WorkflowInput`, which propagates to the CRD via the controller.

**Parallel execution:** Juniors run in parallel using Temporal's child workflow fan-out. The senior workflow uses `workflow.Go` goroutines to start all children, then waits on all futures. Temporal handles the concurrency.

### 7. How results aggregate

After all juniors complete, the senior workflow:

1. Queries each junior's workspace for git diff output (new activity: `CollectJuniorResults`)
2. Composes a review prompt with all diffs and the original spec
3. Starts the senior agent with this review prompt
4. The senior agent applies diffs, resolves conflicts, runs tests, produces a final commit

If any junior failed, the senior's review prompt includes the failure message. The senior decides whether to retry, work around, or flag the failure.

**Goal-Gradient (UX):** As juniors complete, the UI shows progress through the tree. Each completed junior is a visible milestone. The integration phase at the end provides a clear finish line.

### 8. UI changes

The run detail view gains a "Run Graph" section when the run has children or a parent:

- **Tree view:** Collapsible tree showing senior at root, juniors as children. Each node shows name, phase badge, and duration.
- **Chunking (UX):** Each junior is a natural chunk — a discrete, named task with its own status. Users understand progress at a glance.
- **Miller's Law:** Max 7 juniors visible without scrolling. If there are more (shouldn't happen due to decomposition cap), pagination kicks in.
- **Navigation:** Clicking a junior node navigates to its detail page. Breadcrumb shows the path back to the senior.

The list view shows spec-run groups: runs with the same `spec-run-id` are visually grouped, with the senior as the primary entry and juniors indented beneath.

## Risks / Trade-offs

- **[Risk] Senior produces bad decomposition** → Fallback: if JSON parsing fails or tasks array is empty, execute as single run. Log a warning. The user can switch to `manual` mode for full control.
- **[Risk] Junior workspace cost** → Each junior gets its own PVC and deployment. For 7 juniors, that is 7x the resource cost. Mitigated by the 7-task cap and TTL-based cleanup. Future optimization: shared read-only base image with copy-on-write overlay.
- **[Risk] Integration conflicts** → Multiple juniors editing the same files will produce conflicting diffs. The senior's integration step handles this, but it may fail. Mitigated by the decomposition prompt instructing the senior to assign non-overlapping file scopes.
- **[Trade-off] No new CRD** → We lose the ability to set spec-run-level policies (e.g., "retry the whole spec if >2 juniors fail"). Acceptable for v1; can add a SpecRun CRD later if needed.
- **[Trade-off] Separate workspaces over shared** → Higher resource cost but dramatically simpler concurrency model. Unix philosophy: processes don't share memory by default.
