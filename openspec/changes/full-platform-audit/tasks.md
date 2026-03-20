## 1. Go Backend Audit

- [x] 1.1 Read every file in cmd/ — verify each binary compiles and has correct main() entrypoint
- [x] 1.2 Check internal/brain and internal/embeddings — are they dead code? Do they require PostgreSQL?
- [x] 1.3 Audit internal/server — verify all REST endpoints match what the web UI calls
- [x] 1.4 Audit internal/sidecar — verify ExecCommand, StartAgent, SendInput, resolveWorkDir are correct
- [x] 1.5 Audit internal/temporal — verify workflow_spec_driven pipeline matches design doc
- [x] 1.6 Audit internal/controller — verify PipelineConfig passthrough, CRD status updates
- [x] 1.7 Audit internal/hydration — verify workspace layout matches current design (no /src/ prefix)
- [x] 1.8 Check for unused Go functions/types across all packages

## 2. Proto/API Audit

- [x] 2.1 Compare proto messages to Go CRD types — find mismatches
- [x] 2.2 Verify generated code (gen/go/) is up to date with proto files
- [x] 2.3 Check for deprecated/unused proto fields

## 3. Web UI Audit

- [x] 3.1 Verify every component is imported somewhere (find dead components)
- [x] 3.2 Check SpecEditor.tsx — is it dead code?
- [x] 3.3 Check use-mobile.tsx and use-toast.ts — are they used?
- [x] 3.4 Verify useTraceSpans SSE endpoint exists on server
- [x] 3.5 Check all apiFetch/useClient calls match server endpoints
- [x] 3.6 Check ShellTerminal WebSocket — does the exec endpoint work?
- [x] 3.7 Review NewRunView — does it support all current orchestration modes and model tiers?
- [x] 3.8 Review RunListView — keyboard nav works, filtering correct?

## 4. Kubernetes/Helm Audit

- [x] 4.1 Compare Helm values.yaml defaults to actual deployment
- [x] 4.2 Verify CRD YAML matches Go types (api/v1alpha1/types.go)
- [x] 4.3 Check all Helm templates for stale env vars or selectors
- [x] 4.4 Verify Docker base images are current

## 5. Extension Audit

- [x] 5.1 Verify aot-determinism.ts compiles with pi-coding-agent types
- [x] 5.2 Test ask_user file-based HITL end-to-end
- [x] 5.3 Test delegate_task tool
- [x] 5.4 Verify role-based policies match design (manage vs implement)

## 6. CI/CD Audit

- [x] 6.1 Verify Dagger ci/main.go compiles and runs locally
- [x] 6.2 Check release-chart.yaml and release-images.yaml — do they work?
- [x] 6.3 Verify wiki-sync workflow logic (flatten paths, sidebar generation)
- [x] 6.4 Run doc-staleness script, assess noise level

## 7. Test Coverage Audit

- [x] 7.1 Run `go test -cover` on each package, report coverage percentages
- [x] 7.2 Identify packages with 0% coverage for recent changes
- [x] 7.3 Check if brain/embeddings tests require PostgreSQL (mark as integration-only)
- [x] 7.4 Verify e2e tests reference current API surface (not stale)
- [x] 7.5 Check if contract tests cover PipelineConfig, manage/implement roles

## 8. OpenSpec Spec Audit

- [x] 8.1 Read each of 18 specs, verify they match current implementation
- [x] 8.2 Identify stale specs (semantic-search-api if brain is dead, etc.)
- [x] 8.3 Identify missing specs (features that exist but have no spec)

## 9. Findings Report + Follow-Up Proposals

- [x] 9.1 Write findings.md with all issues categorized by severity
- [ ] 9.2 Create /opsx:propose for each major finding category
