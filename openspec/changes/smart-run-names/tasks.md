## 1. Proto and CRD: Add display_name Field

- [ ] 1.1 Add `string display_name = N` to the `AgentRunSpec` message in `proto/aot/api/v1/api.proto`
- [ ] 1.2 Regenerate Go proto types (`buf generate`)
- [ ] 1.3 Regenerate TypeScript proto types (`buf generate`)
- [ ] 1.4 Add `DisplayName string` field to AgentRunSpec in `api/v1alpha1/agentrun_types.go` with `json:"displayName,omitempty"`
- [ ] 1.5 Regenerate CRD manifests (`make manifests` or controller-gen)
- [ ] 1.6 Update shared TypeScript types in `packages/shared/` to include `displayName`
- [ ] 1.7 Update web types if separately maintained

## 2. LLM Name Generation in API Server

- [ ] 2.1 Add name generation function in `internal/server/` — accepts prompt string, calls LiteLLM proxy with system prompt and truncated user prompt, returns generated name
- [ ] 2.2 Add regex validation (`^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$`) — sanitize and validate the LLM response (strip whitespace, lowercase, check regex)
- [ ] 2.3 Add 3-second timeout on the LLM HTTP call using `context.WithTimeout`
- [ ] 2.4 Wire name generation into `CreateAgentRun` handler — call before CRD creation, set `display_name` on the spec
- [ ] 2.5 Implement fallback — if LLM call fails or returns invalid name, log warning and leave `display_name` empty
- [ ] 2.6 Add unit tests: successful generation, invalid LLM response (fails regex), LLM timeout, LLM unavailable

## 3. Web UI: Show display_name

- [ ] 3.1 Add `displayName` helper function: `(run) => run.spec.displayName || run.metadata.name`
- [ ] 3.2 Update RunList component to show `display_name` as primary text, K8s name as secondary/muted
- [ ] 3.3 Update RunDetail header to show `display_name` as title, K8s name as subtitle
- [ ] 3.4 Update CommandPalette to search both `display_name` and K8s name, show `display_name` as primary label

## 4. E2E Test

- [ ] 4.1 Add E2E test: create a run via API with a descriptive prompt, verify `display_name` is set on the returned run and is non-empty
- [ ] 4.2 Add E2E test: create a run when LLM is unavailable (or mock unavailability), verify run still creates successfully with empty `display_name`

## 5. Verification

- [ ] 5.1 Run `go test ./internal/server/...` — unit tests pass including name generation tests
- [ ] 5.2 Run `buf lint` — proto linting passes
- [ ] 5.3 Run `npx tsc --noEmit -p web/tsconfig.json` — web UI compiles
- [ ] 5.4 Verify display name appears in the run list and detail view in the web UI
