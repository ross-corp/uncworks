## Why

Run names are currently `ar-{random6chars}` (e.g., `ar-a3gfp3`). These are meaningless — users must read the prompt to understand what a run does. In a list of 20+ runs, this makes the UI unusable without clicking into each one.

## What Changes

- **Name generation**: At run creation time, the API server calls the in-cluster Ollama (qwen2.5:0.5b) via the existing LiteLLM proxy to generate a short descriptive kebab-case name from the prompt. Example: "Fix the auth middleware to handle expired tokens" becomes `fix-auth-token-expiry`.
- **New field**: Add `display_name` field to the AgentRunSpec proto message and CRD. This holds the human-readable name. The K8s resource name (`ar-{random}`) remains the internal ID.
- **UI updates**: The web dashboard shows `display_name` everywhere — run list, run detail header, command palette search. Falls back to the K8s name if `display_name` is empty.
- **Fallback**: If the LLM is unavailable or returns an invalid name, fall back to the current `ar-{random}` naming scheme.
- **Validation**: Generated names must match `^[a-z0-9][a-z0-9-]{2,48}[a-z0-9]$` (lowercase alphanumeric with hyphens, 4-50 chars).

## Capabilities

### New Capabilities
- `llm-name-generation`: LLM generates descriptive kebab-case names for agent runs at creation time via the LiteLLM proxy.
- `display-name-field`: New `display_name` field on proto AgentRunSpec and CRD, displayed throughout the web UI.

### Modified Capabilities

## Impact

- `proto/aot/api/v1/api.proto` — add `display_name` field to `AgentRunSpec`
- `gen/` — regenerate Go and TypeScript proto types
- `api/v1alpha1/agentrun_types.go` — add `DisplayName` to CRD spec
- `internal/server/grpc.go` — add LLM name generation call in `CreateAgentRun`
- `internal/server/grpc_test.go` — test name generation, fallback, validation
- `packages/shared/src/types/` — update shared TypeScript types
- `web/src/components/AgentRunList.tsx` — show `display_name` with fallback
- `web/src/pages/RunDetailPage.tsx` — show `display_name` in header
- `web/src/components/CommandPalette.tsx` — search includes `display_name`
- `deploy/helm/aot/templates/` — CRD manifest updated for new field
