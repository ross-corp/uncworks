## Architecture

### Data Model

All organization is label-based on the existing AgentRun CRD — no new resources.

```
AgentRun CRD Labels:
  aot.uncworks.io/project: "neph-nvim"           # user-assigned or LLM-suggested
  aot.uncworks.io/feature: "factory-droid"        # derived from OpenSpec change or LLM
  aot.uncworks.io/tags: "backend,feature,lua"     # comma-separated freeform
  aot.uncworks.io/repo: "neph.nvim"               # auto from repos[0].url

AgentRun CRD Spec (new fields):
  project: string       # project name
  feature: string       # feature name (often = OpenSpec change name)
  tags: []string         # freeform tags
```

### Classification Pipeline

```
Run Created
  │
  ├── Tier 1: Deterministic (instant, no LLM)
  │   ├── repo label ← extract from repos[0].url
  │   ├── display name ← LLM (already exists)
  │   └── lineage ← parentRunID, specRunID (already exists)
  │
  ├── Tier 2: LLM Classification (async, <1s)
  │   ├── feature name ← classify prompt against existing features
  │   ├── project suggestion ← match against existing projects
  │   └── tags ← extract from prompt keywords
  │   │
  │   └── Input: prompt, repos, existing projects[], existing features[]
  │       Output: { feature, featureIsNew, project, tags }
  │       Model: cheapest available (deepseek-v3.1)
  │
  └── Tier 3: User Confirmation
      └── NewRunView pre-fills suggestions, user edits before submit
```

### API Changes

```protobuf
// New fields on AgentRunSpec
string project = 24;
string feature = 25;
repeated string tags = 26;

// New RPC
rpc ClassifyRun(ClassifyRunRequest) returns (ClassifyRunResponse);
  // Input: prompt, repos
  // Output: suggested project, feature, tags

// Enhanced ListAgentRuns filters
string project_filter = 5;
string feature_filter = 6;
string tag_filter = 7;
```

### Frontend Views

**Run List — Feature Mode (default)**
```
  [p] neph-nvim ▾                              [@] unc

  / filter   [1] features  [2] all  [3] running  [4] failed

  FEATURE                         STATUS      RUNS  PR
  ▸ factory-droid-backend          FAILED      2/2   —
    architect-agent                DONE        1/1   #41
    architecture-docs              DONE        1/1   #40

  UNASSIGNED
    ar-kh95jn  rewrite markdown    SUCCEEDED
```

**Project Picker (press `p`)**
```
  ┌─────────────────────────┐
  │  Switch Project         │
  │  > neph-nvim            │
  │    uncworks-platform    │
  │    (all projects)       │
  │    + new project...     │
  └─────────────────────────┘
```

**New Run View — Auto-classified fields**
```
  Prompt:  [Add factory-droid backend...]
  Repo:    [github.com/roshbhatia/neph.nvim]

  Feature: [factory-droid-backend ▾ ]  ← LLM-suggested, editable
  Project: [neph-nvim ▾            ]  ← LLM-suggested, editable
  Tags:    [feature] [backend] [+  ]  ← LLM-suggested, editable
```

### Feature ↔ OpenSpec Change Binding

When a spec-driven run creates an OpenSpec change (e.g., `ar-fmupi3`), the feature label is set to the change name. If the user provided a feature name at creation, the OpenSpec change is named to match. This creates a 1:1 binding:

```
  Feature "factory-droid"
    ↕
  OpenSpec change "factory-droid"
    ↕
  Runs with label aot.uncworks.io/feature=factory-droid
```

Retrying a failed feature reuses the same feature label, linking attempts.

### Post-Run Enrichment (Tier 4, async)

After a run completes, a background Temporal activity analyzes the git diff and enriches tags:
- File types touched → auto-tag: "lua", "go", "typescript"
- Scope → auto-tag: "small-change" (<5 files) or "large-change" (>20 files)
- Change pattern → auto-tag: "feature" (new files), "fix" (modified only), "docs" (*.md only)

### Contextual Prompts

Surfaced in the UI (non-blocking banners) at decision points:

| Trigger | Prompt |
|---------|--------|
| Prompt similar to failed run | "Looks like a retry of feature X. [Link] [Confirm]" |
| Run succeeded, no PR | "Create PR for feature X? [Create PR] [Skip]" |
| PR merged (webhook) | "Feature X delivered. [Archive] [Keep open]" |
| All tasks complete | "Feature X is complete. [Create PR] [Mark done]" |
