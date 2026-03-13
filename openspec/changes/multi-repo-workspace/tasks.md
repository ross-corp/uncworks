## 1. Hydrator: uncspace.yaml Generation

- [ ] 1.1 Add `generateManifest` method to `Hydrator` that scans cloned worktrees and writes `/workspace/uncspace.yaml` with repos (path, url, branch) and devbox source metadata
- [ ] 1.2 Call `generateManifest` in `Hydrator.Run()` after all repos are cloned, before devbox setup
- [ ] 1.3 Add tests for manifest generation: multi-repo, single-repo, zero-repo, mixed devbox presence

## 2. Hydrator: Devbox Auto-Composition

- [ ] 2.1 Add `composeDevbox` method that scans each worktree for `devbox.json` and generates a root `/workspace/devbox.json` with `include` directives
- [ ] 2.2 Modify `setupDevbox` to auto-compose when `DevboxConfig` is empty — skip composition when `DevboxConfig` is explicitly set (preserve existing behavior)
- [ ] 2.3 Run `devbox install` at `/workspace` (root) instead of primary worktree when using composed config
- [ ] 2.4 Add tests for devbox composition: multiple configs, single config, no configs, explicit override, install failure error messaging

## 3. Proto/CRD: workspace_name Field

- [ ] 3.1 Add `string workspace_name = 12` to proto `AgentRunSpec` message in `api.proto`
- [ ] 3.2 Add `WorkspaceName string` field to CRD `AgentRunSpec` type in `types.go`
- [ ] 3.3 Regenerate proto Go code (`buf generate`)
- [ ] 3.4 Pass `workspace_name` through controller, workflow, and gRPC handler (no logic, just plumbing)

## 4. Workflow: Workspace Root Working Directory

- [ ] 4.1 Change `workspacePath` computation in `AgentRunWorkflow` from first-repo path to `/workspace`
- [ ] 4.2 Update `StartAgentInput.RepoPath` to always be `/workspace`
- [ ] 4.3 Update workflow tests to verify `/workspace` is passed as working directory

## 5. Sidecar: Run from Workspace Root

- [ ] 5.1 Verify `startAgentProcess` correctly uses the `RepoPath` from the request (which will now be `/workspace`) — likely no code change needed, just confirm behavior
- [ ] 5.2 Ensure `devbox run` executes from `/workspace` where the composed `devbox.json` lives

## 6. Web Types: Multi-Repo Data Model

- [ ] 6.1 Replace `repoURL: string` and `branch: string` with `repos: Repository[]` and `workspaceName?: string` on web `AgentRunSpec` type
- [ ] 6.2 Add `Repository` type (`url: string`, `branch: string`, `path?: string`) to web types
- [ ] 6.3 Update `mapRun()` in `useClient.ts` to pass through full `repos` array and `workspaceName` instead of flattening to `repos[0]`
- [ ] 6.4 Update `mapEvent()` if it references repo data

## 7. Web Hooks: Workspace & Repo Persistence

- [ ] 7.1 Create `useWorkspaces` hook — CRUD operations on `uncworks:workspaces` localStorage key, returns `workspaces`, `addWorkspace`, `updateWorkspace`, `deleteWorkspace`
- [ ] 7.2 Create `useRepoRegistry` hook — manages `uncworks:repos` localStorage key, merges with run-derived repos, returns `repos`, `addRepo`, `removeRepo`

## 8. Web UI: AgentRunForm Redesign

- [ ] 8.1 Add workspace selector section — radio/list of saved workspaces + "Custom repos" option
- [ ] 8.2 Replace single repo `<select>` with multi-repo list: each row has URL (select from known repos or type new) + branch input + remove button
- [ ] 8.3 Add "+ Add repo" button that appends a new repo row to the list
- [ ] 8.4 Pre-fill repos from selected workspace, allow per-run modification without changing the preset
- [ ] 8.5 Submit full `repos[]` array and `workspaceName` to the API
- [ ] 8.6 Remove the single global "Branch" field (replaced by per-repo branches)

## 9. Web UI: Workspace Editor Modal

- [ ] 9.1 Create `WorkspaceEditor` component — modal with name, description, repo list (url + branch + remove), "+ Add repo" button
- [ ] 9.2 Support create mode (empty fields) and edit mode (pre-filled from existing workspace)
- [ ] 9.3 Add delete button with confirmation
- [ ] 9.4 Wire save/delete to `useWorkspaces` hook
- [ ] 9.5 Open editor from sidebar "+ New workspace" and from workspace context menu

## 10. Web UI: Sidebar Updates

- [ ] 10.1 Add "Workspaces" collapsible section between "Agent Runs" and "Repositories"
- [ ] 10.2 Show each workspace preset with count of runs tagged with that workspace name
- [ ] 10.3 Click workspace to filter runs by `workspaceName`
- [ ] 10.4 Add "+ New workspace..." button that opens workspace editor
- [ ] 10.5 Update repo count logic to count a run under every repo in its `repos[]` array
- [ ] 10.6 Update repo filter to show runs that include the selected repo (not just first repo match)

## 11. Web UI: AgentRunTable Updates

- [ ] 11.1 Update repo column to display comma-separated repo names from `repos[]`
- [ ] 11.2 Truncate to `name1, name2 +N` when >2 repos, with title tooltip showing all
- [ ] 11.3 Remove branch display from table column (moved to detail panel per-repo)

## 12. Web UI: AgentRunDetailPanel Updates

- [ ] 12.1 Replace single Repository/Branch MetaRow with "Repositories" section listing all repos with their branches
- [ ] 12.2 Show "Workspace: <name>" if `workspaceName` is set on the run
- [ ] 12.3 Update `repoName()` utility to work with repos array

## 13. Web UI: Search & Filtering

- [ ] 13.1 Update search in `App.tsx` to match against all repo URLs in `repos[]`, not just a single `repoURL`
- [ ] 13.2 Update `selectedRepo` filtering to check if the repo appears anywhere in a run's `repos[]`

## 14. Web UI: ReposView Interactive Registry

- [ ] 14.1 Add "Add Repository" input (URL field + Add button) to ReposView
- [ ] 14.2 Add Remove button to each repo row
- [ ] 14.3 Wire add/remove to `useRepoRegistry` hook
- [ ] 14.4 Merge registry repos with run-derived repos for display (deduplicated)

## 15. App.tsx: State & Wiring

- [ ] 15.1 Add workspace state: `selectedWorkspace`, `workspaces` from `useWorkspaces` hook
- [ ] 15.2 Add repo registry state from `useRepoRegistry` hook
- [ ] 15.3 Update `handleCreate` to pass full `repos[]` and `workspaceName` to API
- [ ] 15.4 Wire workspace filtering into the filter chain (selectedWorkspace → selectedRepo → phaseFilter → search)
- [ ] 15.5 Pass workspace/repo data down to all child components
