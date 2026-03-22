## 1. Theme Consistency

- [x] 1.1 Update `FilePreview.tsx` to read theme from `useThemeNew()` and pass `"vs"` (light) or `"vs-dark"` (dark) to Monaco
- [x] 1.2 Create xterm theme objects (light + dark) in `ShellTerminalInner.tsx` using CSS custom property values; apply based on current theme
- [x] 1.3 Verify theme toggle in `Layout.tsx` footer is visible and functional; move to header if not prominent enough
- [x] 1.4 Take screenshots of all views in both light and dark mode to verify consistency

## 2. Monaco Editors for Prompt/Spec

- [x] 2.1 Create `MarkdownEditor.tsx` component wrapping `@monaco-editor/react` with markdown language, word wrap, no minimap, theme-aware, controlled value/onChange
- [x] 2.2 Replace prompt `<textarea>` in `NewRunView.tsx` with `MarkdownEditor` component
- [x] 2.3 Replace spec `<textarea>` in `NewRunView.tsx` with `MarkdownEditor` component
- [x] 2.4 Verify Monaco lazy-loads correctly and doesn't block initial render

## 3. shadcn Component Migration

- [x] 3.1 Replace raw `<select>` elements in `NewRunView.tsx` with shadcn `Select` component for model and orchestration pickers
- [x] 3.2 Replace tab bar in `RunDetailView.tsx` with shadcn `Tabs` component (already using shadcn Tabs)
- [x] 3.3 Replace status filter buttons in `RunListView.tsx` header with shadcn `Badge`
- [x] 3.4 Ensure all `<button>` elements use shadcn `Button` component throughout views

## 4. Dual Model Config

- [x] 4.1 Add `manageModelTier` and `implementModelTier` fields to `AgentRunSpec` in `agent-run.ts` types
- [x] 4.2 Add dual model selector UI in `NewRunView.tsx` — show two model dropdowns when orchestration mode is "Progressive"
- [x] 4.3 Add `manage_model_tier` and `implement_model_tier` to CRD spec in `agentrun-crd.yaml`
- [x] 4.4 Add fields to Go types in `types.go` (proto regeneration deferred — fields flow via CRD)
- [x] 4.5 Update `BuildWorkflowInput` in `mapping.go` to pass manage/implement model tiers to workflow
- [x] 4.6 Override per-stage model in `runSpecDrivenPipeline`: manage→plan/verify, implement→execute

## 5. Archive Runs

- [x] 5.1 Add `archived` boolean field to `AgentRunStatus` in CRD YAML and Go types
- [x] 5.2 Add `ArchiveRun` REST endpoint (`POST /api/v1/runs/{id}/archive`)
- [x] 5.3 Add `BulkArchiveRuns` REST endpoint (`POST /api/v1/runs/bulk-archive`)
- [x] 5.4 Add PVC cleanup in controller: when a run is archived, delete its PVC
- [x] 5.5 Update `ListAgentRuns` to exclude archived runs by default; add `X-Include-Archived` header
- [x] 5.6 Add "Show archived" toggle to `RunListView.tsx` header
- [x] 5.7 Add archive button to run detail view (alongside cancel/retry)
- [x] 5.8 Add mass select mode to `RunListView.tsx` — checkboxes, floating action bar, bulk archive

## 6. Run List Metrics

- [x] 6.1 Add `totalCost`, `totalAdditions`, `totalDeletions` fields to CRD status and frontend types
- [x] 6.2 Update run list grid to new layout: `[select] [name] [status] [model] [cost] [+/-] [PR] [age]`
- [x] 6.3 Add cost column rendering (`$X.XX` or `—`)
- [x] 6.4 Add diff stats column rendering (`+N/-M` with green/red)
- [x] 6.5 Add PR badge column — clickable link when `prUrl` exists
- [x] 6.6 Model column shows modelTier (dual display deferred to when backend populates manageModelTier)
- [x] 6.7 Feature group status badges already inline — verified

## 7. Fix Logs Loading

- [x] 7.1 Investigate "Loading activity..." stuck state in `RunDetailView.tsx` — check if the activity feed endpoint returns data for completed runs
- [x] 7.2 Fix the loading state to show "No activity" for runs without log data instead of infinite loading

## 8. Verification

- [x] 8.1 Take screenshots of every view in both themes after all changes
- [ ] 8.2 Submit a test run to neph.nvim with prompt "Add comments to all functions" using Progressive mode with dual models
- [ ] 8.3 Verify the run appears in the list with correct metrics after completion
- [ ] 8.4 Archive the run, verify it disappears, toggle "show archived" to see it
