## 1. Test Fixture Repository

- [x] 1.1 Create `test/fixtures/e2e-repo/` with `devbox.json` (`{"packages": []}`), `main.go` (simple Go program), and `README.md`
- [x] 1.2 Create `test/fixtures/e2e-repo-frontend/` with `package.json` (`{"name": "e2e-frontend"}`) and `index.ts` for multi-repo tests
- [x] 1.3 Create `test/fixtures/push-fixtures.sh` script that initializes git repos from fixture dirs and pushes them to a Soft-Serve instance at `$SOFT_SERVE_ADDR`

## 2. Soft-Serve Integration & Taskfile

- [x] 2.1 Add `soft-serve` to `devbox.json` packages (or document `go install github.com/charmbracelet/soft-serve/cmd/soft@latest`)
- [x] 2.2 Add `test:e2e:setup` task to Taskfile: starts Soft-Serve process (background, temp data dir, git daemon on `:9418`, bind `0.0.0.0`), waits for ready, runs `push-fixtures.sh`
- [x] 2.3 Add `test:e2e:teardown` task to Taskfile: stops Soft-Serve process, cleans temp data dir
- [x] 2.4 Add `test:e2e:go` task: runs `go test ./e2e/... -tags e2e -v -timeout 15m`
- [x] 2.5 Add `test:e2e:playwright` task: runs `cd web && npx playwright test`
- [x] 2.6 Add `test:e2e:full` task: runs setup → go → playwright → teardown (with trap for cleanup on failure)

## 3. Go E2E Test Harness

- [x] 3.1 Create `e2e/harness_test.go` with helper `getSoftServeRepoURL(repoName string) string` that returns `git://{SOFT_SERVE_ADDR}/{repoName}` using env var with default
- [x] 3.2 Update `e2e/api_test.go` — replace all `https://github.com/example/repo.git` URLs with `getSoftServeRepoURL("e2e-repo")`
- [x] 3.3 Update `e2e/temporal_test.go` — replace all fake repo URLs with Soft-Serve URLs
- [x] 3.4 Update `e2e/llm_test.go` — replace all fake repo URLs with Soft-Serve URLs
- [x] 3.5 Update `e2e/system_test.go` — replace all fake repo URLs with Soft-Serve URLs

## 4. Go E2E: Full Lifecycle Tests

- [x] 4.1 Add `TestE2E_FullLifecycle_SimplePrompt` — create run with Soft-Serve repo + simple prompt ("create DONE.txt with PASS"), wait for Succeeded phase, verify pod existed
- [x] 4.2 Add `TestE2E_FullLifecycle_TTLExpiry` — create run with 10s TTL and a slow prompt, verify it reaches Failed with TTL message
- [x] 4.3 Add `TestE2E_FullLifecycle_CancelRunning` — create run, wait for Running, cancel, verify Cancelled phase and pod cleanup

## 5. Go E2E: Spec-Driven and Multi-Repo Tests

- [x] 5.1 Add `e2e/spec_test.go` with `TestE2E_SpecDrivenRun` — create run with `spec_content` set and empty prompt, verify auto-prompt and workflow completion
- [x] 5.2 Add `e2e/multirepo_test.go` with `TestE2E_MultiRepo_TwoRepos` — create run with two Soft-Serve repos, verify workflow starts and agent gets `/workspace` as working dir
- [x] 5.3 Add `TestE2E_MultiRepo_WorkspaceName` — create run with `workspace_name` set, verify it's preserved on the CRD

## 6. Go E2E: Webhook and API Tests

- [x] 6.1 Add `e2e/webhook_e2e_test.go` with `TestE2E_Webhook_SpecFileCreatesRun` — POST push payload to `/api/v1/webhooks/github` with `.cs.md` file, verify AgentRun CRD created
- [x] 6.2 Add `TestE2E_Webhook_InvalidSignature` — POST with bad signature, verify 401
- [x] 6.3 Add `TestE2E_Webhook_NoSpecFiles` — POST push with only `.go` files, verify no run created
- [x] 6.4 Add `TestE2E_ConcurrentRuns` — create 3 runs simultaneously, verify all reach terminal phase with unique pod names

## 7. UI Test Instrumentation: data-testid Attributes

- [x] 7.1 Add `data-testid` to `Sidebar.tsx`: phase filter buttons, workspace buttons, repo buttons, "New workspace" button
- [x] 7.2 Add `data-testid` to `AgentRunTable.tsx`: table rows (`table-row-{id}`), phase badges, spec badge
- [x] 7.3 Add `data-testid` to `AgentRunDetailPanel.tsx`: panel container, name, phase, repos section, HITL input/send, cancel button
- [x] 7.4 Add `data-testid` to `AgentRunForm.tsx`: modal container, name input, repo rows (url/branch), add-repo button, tabs (prompt/spec), submit button
- [x] 7.5 Add `data-testid` to `SpecEditor.tsx`: editor container
- [x] 7.6 Add `data-testid` to `GitHubModal.tsx`: modal container, repo input, path input, submit button
- [x] 7.7 Add `data-testid` to `WorkspaceEditor.tsx`: modal container, name input, repo rows, save/delete buttons
- [x] 7.8 Add `data-testid` to `ReposView.tsx`: add input, add button, repo rows with remove buttons
- [x] 7.9 Add `data-testid` to `Layout.tsx`: search input, new-run button
- [x] 7.10 Add `data-testid` to `Toast.tsx`: toast container

## 8. Playwright: Setup and Smoke Tests

- [x] 8.1 Delete stale `web/e2e/app.spec.ts`
- [x] 8.2 Update `web/playwright.config.ts`: increase default timeout, configure retries, ensure webServer starts dev server
- [x] 8.3 Create `web/e2e/smoke.spec.ts`: dashboard renders (sidebar, table, header visible), phase filters show counts

## 9. Playwright: Run Creation Tests

- [x] 9.1 Create `web/e2e/create-run.spec.ts` with test: open form → fill name, repo, prompt → submit → toast appears → run in table
- [x] 9.2 Add test: create spec-based run → switch to Spec tab → type in Monaco → submit → spec badge on row
- [x] 9.3 Add test: form validation — submit empty form, verify it doesn't close
- [x] 9.4 Add test: workspace preset selection → repos pre-fill → submit with workspace name

## 10. Playwright: Lifecycle and HITL Tests

- [x] 10.1 Create `web/e2e/lifecycle.spec.ts` with test: create run → watch phase transition in table (Pending → Running → terminal)
- [x] 10.2 Add test: select run → detail panel opens with correct data (name, phase, repos, prompt)
- [x] 10.3 Add test: cancel running run → confirm cancel → phase becomes "cancelled"
- [x] 10.4 Add test: HITL — wait for "waiting_for_input" phase → type input → send → phase transitions

## 11. Playwright: Filtering, Search, and Workspace Tests

- [x] 11.1 Create `web/e2e/filter-search.spec.ts` with test: click phase filter → table shows only matching runs
- [x] 11.2 Add test: type in search bar → table filters by name/prompt/repo
- [x] 11.3 Create `web/e2e/workspace.spec.ts` with test: create workspace → appears in sidebar → select → filters table
- [x] 11.4 Add test: edit workspace → change name → sidebar updates
- [x] 11.5 Add test: delete workspace (2-step confirm) → removed from sidebar

## 12. Playwright: Spec Editor and Repo Registry Tests

- [x] 12.1 Create `web/e2e/spec-editor.spec.ts` with test: switch to Spec tab → Monaco loads (lazy) → type content → switch to Prompt tab and back → content preserved
- [x] 12.2 Add test: click "Load from GitHub" → modal opens → mock API response via `page.route()` → editor populated
- [x] 12.3 Create `web/e2e/repos.spec.ts` with test: navigate to Repos view → add repo URL → appears in list
- [x] 12.4 Add test: remove repo → disappears from list

## 13. Verification

- [x] 13.1 Run `go build ./e2e/...` — all E2E tests compile
- [x] 13.2 Run `npx tsc --noEmit -p web/tsconfig.json` — web compiles with data-testid attributes
- [x] 13.3 Run `task test:e2e:full` — full E2E suite passes against aot-local cluster
