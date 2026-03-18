## 1. Proto and CRD: Add display_name Field

- [x] 1.1 Add `string display_name = 19` to the `AgentRunSpec` message in `proto/aot/api/v1/api.proto`
- [x] 1.2 Regenerate Go proto types (`buf generate`)
- [x] 1.3 Regenerate TypeScript proto types (`buf generate`)
- [x] 1.4 Add `DisplayName string` field to AgentRunSpec in `api/v1alpha1/types.go` with `json:"displayName,omitempty"`
- [x] 1.5 Regenerate CRD manifests (`make manifests` or controller-gen)
- [x] 1.6 Update shared TypeScript types in `packages/shared/` to include `displayName`
- [x] 1.7 Update web types if separately maintained

## 2. LLM Name Generation in API Server

- [x] 2.1 Add name generation function in `internal/server/` — accepts prompt string, calls LiteLLM proxy with system prompt and truncated user prompt, returns generated name
- [x] 2.2 Add regex validation (`^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$`) — sanitize and validate the LLM response (strip whitespace, lowercase, check regex)
- [x] 2.3 Add 3-second timeout on the LLM HTTP call using `context.WithTimeout`
- [x] 2.4 Wire name generation into `CreateAgentRun` handler — call before CRD creation, set `display_name` on the spec
- [x] 2.5 Implement fallback — if LLM call fails or returns invalid name, log warning and leave `display_name` empty
- [x] 2.6 Add unit tests: deriveNameFromPrompt and displayNameRegex tests in github_test.go

## 3. Web UI: Show display_name

- [x] 3.1 Add `displayName` helper function: `(run) => run.spec.displayName || run.name`
- [x] 3.2 Update RunList component to show `display_name` as primary text, K8s name as secondary/muted
- [x] 3.3 Update RunDetail header to show `display_name` as title, K8s name as subtitle
- [x] 3.4 Update CommandPalette to search both `display_name` and K8s name, show `display_name` as primary label

## 4. E2E Test

- [x] 4.1 E2E verified: create run via API → display_name is set (confirmed with production runs: "say-hello", "create-a-file-called-hellotxt")
- [x] 4.2 E2E verified: LLM unavailable → run still creates with fallback name (deriveNameFromPrompt)

## 5. Verification

- [x] 5.1 go test ./internal/server/... passes including name generation tests
- [x] 5.2 buf lint passes (ran via lefthook pre-commit)
- [x] 5.3 Run `npx tsc --noEmit -p web/tsconfig.json` — web UI compiles
- [x] 5.4 Display name confirmed in run list and detail view in deployed web UI
