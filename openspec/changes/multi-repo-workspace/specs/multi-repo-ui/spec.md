## ADDED Requirements

### Requirement: Web types carry full repos array
The web-side `AgentRunSpec` type SHALL use `repos: Repository[]` instead of `repoURL: string` and `branch: string`. The `Repository` type SHALL have `url: string`, `branch: string`, and optional `path: string`.

#### Scenario: Type definition
- **WHEN** the web types are compiled
- **THEN** `AgentRunSpec` has a `repos` field of type `Repository[]` and a `workspaceName` field of type `string`
- **AND** there is no `repoURL` or `branch` field on `AgentRunSpec`

### Requirement: mapRun preserves full repos array
The `mapRun()` function in `useClient.ts` SHALL pass through the full `repos` array from the shared types instead of extracting only the first repo.

#### Scenario: Multi-repo run mapping
- **WHEN** a shared AgentRun has `spec.repos` with 3 entries
- **THEN** the mapped web AgentRun has `spec.repos` with 3 entries, each with `url` and `branch`

### Requirement: AgentRunForm supports per-repo branches
Each repo entry in the agent run form SHALL have its own branch field, defaulting to "main".

#### Scenario: Different branches per repo
- **WHEN** a user adds 2 repos and sets repo A to branch "main" and repo B to branch "feature/xyz"
- **THEN** the submitted spec contains `repos: [{url: A, branch: "main"}, {url: B, branch: "feature/xyz"}]`

### Requirement: AgentRunForm allows adding repos
The form SHALL allow adding new repos by URL (with auto-complete from the registry/known repos) and removing repos from the current list.

#### Scenario: Add repo to run
- **WHEN** a user clicks "+ Add repo" in the form
- **THEN** a new repo row appears with a URL input (showing known repos as options) and a branch field

#### Scenario: Remove repo from run
- **WHEN** a user clicks the remove button on a repo row
- **THEN** the repo is removed from the form's repo list
- **AND** the workspace preset (if any) is not modified

### Requirement: AgentRunTable multi-repo column
The repo column in the agent run table SHALL display all repos for a run, truncating when there are more than 2.

#### Scenario: Single repo
- **WHEN** a run has 1 repo "api"
- **THEN** the column shows "api"

#### Scenario: Two repos
- **WHEN** a run has repos "api" and "web"
- **THEN** the column shows "api, web"

#### Scenario: Three or more repos
- **WHEN** a run has repos "api", "web", and "protos"
- **THEN** the column shows "api, web +1"
- **AND** a title tooltip shows all repo names

### Requirement: AgentRunDetailPanel repo list
The detail panel SHALL display all repos with their branches instead of a single repo/branch pair.

#### Scenario: Multi-repo detail display
- **WHEN** a run has 3 repos with different branches
- **THEN** the detail panel shows a "Repositories" section with each repo name and branch listed

#### Scenario: Workspace label in detail
- **WHEN** a run has `workspaceName: "payments-platform"`
- **THEN** the detail panel shows "Workspace: payments-platform"

### Requirement: Sidebar multi-repo-aware repo counts
The sidebar repo filter SHALL count a run under every repo it contains, not just the first.

#### Scenario: Multi-repo run counted under all repos
- **WHEN** a run has repos `[api, web]`
- **THEN** the run is counted under both "api" and "web" in the sidebar repo filter

#### Scenario: Filter shows all matching runs
- **WHEN** a user clicks "api" in the sidebar repo filter
- **THEN** all runs that include "api" in their repos array are shown (even if they also include other repos)

### Requirement: Search matches all repos
The search function SHALL match the query against all repo URLs in a run, not just the first.

#### Scenario: Search by non-primary repo
- **WHEN** a run has repos `[api, web]` and the user searches "web"
- **THEN** the run appears in search results

### Requirement: ReposView interactive registry
The ReposView component SHALL allow users to add and remove repos from the registry, not just display them.

#### Scenario: Add repo via ReposView
- **WHEN** a user enters a repo URL and clicks Add
- **THEN** the URL is saved to the repo registry in localStorage
- **AND** it appears in the table and becomes available in the form

#### Scenario: Remove repo via ReposView
- **WHEN** a user clicks Remove on a repo row
- **THEN** the repo is removed from the localStorage registry
