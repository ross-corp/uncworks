## 1. Workflow Error Resilience

- [x] 1.1 Add `maxConsecutiveStatusErrors = 5` constant and `consecutiveErrors` counter to the polling loop in `internal/temporal/workflow.go`
- [x] 1.2 On GetAgentStatus error: increment counter, log warning with error and count; on success: reset counter to 0
- [x] 1.3 When counter reaches threshold: transition to Failed phase with "sidecar unreachable after N consecutive errors" message, break the loop
- [x] 1.4 Change cleanup defer block: capture RevokeLLMKey error and log at Error level via `workflow.GetLogger(ctx).Error()`
- [x] 1.5 Change cleanup defer block: capture CleanupPod error and log at Error level via `workflow.GetLogger(ctx).Error()`
- [x] 1.6 Add workflow test: `TestWorkflow_ConsecutiveStatusErrors` — mock GetAgentStatus to fail 5 times, verify workflow transitions to Failed
- [x] 1.7 Add workflow test: `TestWorkflow_TransientStatusError` — mock GetAgentStatus to fail 3 times then succeed, verify workflow continues

## 2. Web UI Routing Setup

- [x] 2.1 Add `@solidjs/router` to `web/package.json` and install
- [x] 2.2 Create `web/src/pages/RunListPage.tsx` — moves list logic from App.tsx, uses store, polls listAgentRuns every 5s
- [x] 2.3 Create `web/src/pages/RunDetailPage.tsx` — fetches run by ID from route params, starts watchAgentRun stream
- [x] 2.4 Rewrite `web/src/App.tsx` to use Router with routes: `/` → RunListPage, `/runs/:id` → RunDetailPage
- [x] 2.5 Wire createAgentStore as the shared store (create once in App, pass via context or props)
- [x] 2.6 Update `web/src/components/AgentRunList.tsx` to use `<A>` links for navigation instead of onClick callback

## 3. Create Agent Run Form

- [x] 3.1 Create `web/src/components/CreateRunForm.tsx` with repos section (url, branch, path), prompt textarea, backend selector
- [x] 3.2 Add "Add Repository" button to support multiple repos in the form
- [x] 3.3 Add collapsible "Advanced" section with devboxConfig, ttlSeconds, envVars (key-value pairs), image fields
- [x] 3.4 Add client-side validation: require at least one repo URL and non-empty prompt
- [x] 3.5 Wire form submission to AOTClient.createAgentRun, navigate to new run's detail page on success
- [x] 3.6 Show error message on submission failure

## 4. Run Actions (Cancel and Human Input)

- [x] 4.1 Add Cancel button to RunDetailPage, visible only for non-terminal phases (Pending, Running, WaitingForInput)
- [x] 4.2 Add confirmation dialog on Cancel click; on confirm, call AOTClient.cancelAgentRun
- [x] 4.3 Create `web/src/components/HumanInputForm.tsx` — text input + submit button, visible only when phase is WaitingForInput
- [x] 4.4 Wire HumanInputForm submission to AOTClient.sendHumanInput; clear input on success

## 5. Event Streaming and Event Log

- [x] 5.1 In RunDetailPage, start `AOTClient.watchAgentRun(id, onEvent)` on mount; abort on cleanup
- [x] 5.2 Feed events into store via `store.addEvent(event)` — phase_changed events auto-update run phase
- [x] 5.3 Create `web/src/components/EventLog.tsx` — scrollable panel showing events with timestamp, type badge, payload
- [x] 5.4 Add auto-scroll behavior: scroll to bottom on new events unless user has scrolled up
- [x] 5.5 Add reconnection: if stream errors, show "Reconnecting..." indicator and retry after 2s delay
- [x] 5.6 Update vite.config.ts proxy if needed for streaming endpoints

## 6. Verification

- [x] 6.1 Run `go test ./test/temporal/...` — workflow tests pass including new error resilience tests
- [x] 6.2 Run `npx tsc --noEmit -p web/tsconfig.json` — web UI compiles
- [x] 6.3 Run `npm run dev` in web/ — verify list page loads, create form works, detail page streams events
- [x] 6.4 Commit and push
