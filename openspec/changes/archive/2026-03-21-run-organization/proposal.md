## Why

Runs accumulate fast — after two days of development there are 40+ runs with no way to group, filter, or understand them as a body of work. Engineers think in features and projects, not individual run IDs. Without organization, the run list becomes a flat wall of noise where finding "that factory-droid attempt from yesterday" requires scrolling and reading prompts.

## What Changes

- Add **project** as a long-lived grouping concept (cross-repo, user-created)
- Add **feature** as a unit-of-value concept that groups run attempts (maps to OpenSpec changes)
- Add **tags** as ephemeral cross-cutting labels
- LLM auto-classifies runs at creation time (feature name, project suggestion, tags) with user-editable defaults
- Deterministic auto-assignment for repo, lineage, and display name
- Feature-centric view in the run list (group runs by feature, show attempt count and status)
- Contextual prompts: "looks like a retry," "create PR?", "archive feature?"

## Capabilities

### New Capabilities
- `project-management`: Create, list, switch, and delete projects. Runs can be assigned to a project. The run list filters by active project.
- `feature-grouping`: Runs are grouped into features (units of value). A feature tracks multiple attempts, links to its OpenSpec change, and shows aggregate status (tasks complete, PR status).
- `auto-classification`: At run creation, an LLM call suggests feature name, project, and tags based on the prompt and existing metadata. Suggestions are pre-filled but user-editable.
- `tag-system`: Freeform tags on runs for cross-cutting filtering. Post-run enrichment auto-tags based on diff analysis (scope, file types, change type).
- `run-list-hierarchy`: The run list view supports switching between flat (all runs), feature-grouped, and project-filtered views.

### Modified Capabilities
- None

## Impact

- **CRD**: Add labels for `aot.uncworks.io/project`, `aot.uncworks.io/feature`, `aot.uncworks.io/tags`
- **Proto/API**: Add project, feature, tags fields to AgentRunSpec and list filters
- **Frontend**: Run list gets project selector, feature grouping, tag chips
- **NewRunView**: Auto-classification pre-fills project/feature/tags
- **LiteLLM**: One cheap classification call per run creation (~100 tokens)
- **No new CRDs** — projects and features are label-based, not separate resources
