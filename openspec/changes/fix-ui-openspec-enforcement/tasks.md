## 1. ChainRunDetailView tab cleanup

- [ ] 1.1 Rename "DAG" tab label to "Overview" in ChainRunDetailView.tsx (type Tab, tab switcher render)
- [ ] 1.2 Remove "Timeline" tab from the Tab type union and tab switcher
- [ ] 1.3 Remove the Timeline tab content block (the `{tab === "timeline" && ...}` JSX section)
- [ ] 1.4 Remove `formatDuration` / `elapsedSecs` usage that only served the timeline tab (if no longer used elsewhere)
- [ ] 1.5 Fix Runs sub-tab: remove `max-w-2xl mx-auto` from the table container div so it fills panel width

## 2. RunListView approval-mode badge

- [ ] 2.1 Add approval-mode badge rendering in `UnifiedRunRow`: for agent runs, read `ur.agentRun.spec.approvalMode` and display "hitl" or "llm-judge" label
- [ ] 2.2 Style the badge consistently with the existing kind badge (small, colored, pill shape)

## 3. Trace detail panel tool output

- [ ] 3.1 In SpanDetail (TraceTimeline.tsx), add a "Tool Output" collapsible section after "Tool Input" that reads `meta.toolOutput`
- [ ] 3.2 Cap display at 256 lines with a truncation notice when toolOutput exceeds that limit
- [ ] 3.3 Add `"toolOutput"` to the exclusion list in the "All Metadata" section so it doesn't appear twice

## 4. Hide system-role log entries from UI

- [ ] 4.1 In the run log viewer (structured logs display), filter out entries where `role === "system"` before rendering
- [ ] 4.2 Verify the filter does not hide error or warning entries that happen to use the system role

## 5. OpenSpec change field on AgentRun

- [ ] 5.1 Add `openspecChange string` field to `AgentRunSpec` in `api/v1alpha1/types.go` with json tag `openspecChange,omitempty`
- [ ] 5.2 Add `openspecChange: type: string` to the AgentRun CRD schema in both `deploy/crds/agentrun-crd.yaml` and `deploy/helm/aot/crds/agentrun-crd.yaml`
- [ ] 5.3 Pass `openspecChange` through from run creation to the verification activity input

## 6. LLM judge salvageability and retry

- [ ] 6.1 Add `Salvageable bool`, `SalvageGuidance string`, `ConfidenceScore float64`, and `RetryCount int` fields to the verification result struct
- [ ] 6.2 Update the LLM judge prompt to instruct the judge to assess salvageability and confidence, and return structured JSON including these fields
- [ ] 6.3 Add `MaxRetries int` field to `AgentRunSpec` (default 2) and `RetryCount int` to `AgentRunStatus`
- [ ] 6.4 In the Temporal workflow, after a failed verification: if `salvageable=true` and `retryCount < maxRetries`, schedule an additional implement activity with salvageGuidance prepended to the execute prompt
- [ ] 6.5 Increment `retryCount` on the AgentRun CRD when a retry implement stage is scheduled
- [ ] 6.6 Pass prior run logOutput as context to the retry implement agent prompt

## 7. LLM judge trace and log visibility

- [ ] 7.1 Wrap the LLM judge invocation in a named trace span `verification.llm-judge` with start/end time and verdict metadata
- [ ] 7.2 Write the per-criterion verdict breakdown to the run's log output before the run status transitions
- [ ] 7.3 Write the overall verdict as the final log line

## 8. Validation

- [ ] 8.1 Build the web app and confirm no TypeScript errors (`cd web && npm run build`)
- [ ] 8.2 Build the Go binaries and confirm no compile errors (`go build ./...`)
- [ ] 8.3 Verify ChainRunDetailView shows "Overview" and "Runs" tabs only (no "Timeline")
- [ ] 8.4 Verify UnifiedRunRow shows approval-mode badge for agent runs
- [ ] 8.5 Verify SpanDetail shows "Tool Output" section for tool spans with toolOutput metadata
