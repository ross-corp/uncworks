## 1. Proto/CRD: Spec Fields

- [ ] 1.1 Add `string spec_content = 13` and `string spec_source = 14` to proto `AgentRunSpec` in `api.proto`
- [ ] 1.2 Add `SpecContent string` and `SpecSource string` fields to CRD `AgentRunSpec` in `types.go`
- [ ] 1.3 Regenerate proto Go code (`buf generate`)
- [ ] 1.4 Update `specProtoToCRD` and `crdToProto` in `grpc.go` to pass through spec fields
- [ ] 1.5 Update shared TypeScript types to include `specContent` and `specSource` on `AgentRunSpec`
- [ ] 1.6 Update `toAgentRun` mapping in shared gRPC client to include spec fields

## 2. Hydrator: Spec File Placement

- [ ] 2.1 Add spec content to hydrator config (new `SpecContent` field on `Config`, read from `AOT_SPEC_CONTENT` env var)
- [ ] 2.2 Add `writeSpec` method that writes `spec/main.cs.md` and `codespeak.json` to `/workspace` when spec content is present
- [ ] 2.3 Call `writeSpec` in `Hydrator.Run()` after repo cloning, before devbox setup
- [ ] 2.4 Update `BuildAgentPod` in `activities.go` to set `AOT_SPEC_CONTENT` env var on the pod when spec content is present
- [ ] 2.5 Add tests for spec file writing: spec present, spec absent, spec with repos

## 3. Workflow: Spec-Driven Prompt

- [ ] 3.1 In `AgentRunWorkflow`, auto-generate prompt for spec runs: if `spec_content` is non-empty and `prompt` is empty, set prompt to `codespeak build` instruction
- [ ] 3.2 Pass spec content through `WorkflowInput` → `CreateAgentPodInput` for env var injection
- [ ] 3.3 Add tests for prompt auto-generation logic

## 4. Web UI: Monaco Editor Component

- [ ] 4.1 Install `monaco-editor` and `@monaco-editor/react` npm dependencies
- [ ] 4.2 Create `SpecEditor` component: wraps Monaco with markdown language, dark theme, word wrap, line numbers, minimap off
- [ ] 4.3 Support props: `value`, `onChange`, `readOnly`, `height`
- [ ] 4.4 Implement lazy loading: dynamic `import()` with loading spinner fallback
- [ ] 4.5 Style Monaco container to match design system (surface-1 background, edge borders, rounded corners)

## 5. Web UI: Agent Run Form — Spec Tab

- [ ] 5.1 Add Prompt/Spec tab selector to the agent run form (styled tabs matching design system)
- [ ] 5.2 Show textarea when Prompt tab active, SpecEditor when Spec tab active
- [ ] 5.3 Track `specContent` state alongside existing `prompt` state, preserve both on tab switch
- [ ] 5.4 On form submit: if spec mode, send `specContent` and `specSource: "editor"` in the API call; if prompt mode, send `prompt` as before
- [ ] 5.5 Update `handleCreate` in `App.tsx` to pass spec fields through to `createAgentRun`

## 6. Web UI: Spec Display

- [ ] 6.1 In `AgentRunDetailPanel`, show "Spec" section with read-only SpecEditor when `specContent` is present
- [ ] 6.2 Show `specSource` as metadata row in detail panel (formatted: "editor", "github:org/repo/path")
- [ ] 6.3 In `AgentRunTable`, add a small spec indicator (badge/icon) on runs that have `specContent`
- [ ] 6.4 Update web `AgentRunSpec` type to include `specContent?: string` and `specSource?: string`
- [ ] 6.5 Update `mapRun()` in `useClient.ts` to pass through spec fields

## 7. Web UI: GitHub Push/Pull

- [ ] 7.1 Add "Load from GitHub" button to spec editor — opens a small modal with repo + path inputs
- [ ] 7.2 Add "Push to GitHub" button to spec editor — opens a modal with repo + path + commit message inputs
- [ ] 7.3 Create `useGitHub` hook (or utility) that calls the API endpoints for push/pull operations
- [ ] 7.4 On load: populate Monaco editor with fetched content, set `specSource` to `"github:..."`
- [ ] 7.5 On push: send spec content to API, show success/error toast

## 8. Backend: GitHub Integration API

- [ ] 8.1 Add `POST /api/v1/specs/push` endpoint — accepts repo, path, content, commit message; commits file via GitHub API
- [ ] 8.2 Add `GET /api/v1/specs/pull` endpoint — accepts repo + path query params; returns file content from GitHub API
- [ ] 8.3 Configure GitHub authentication (PAT from environment variable initially, GitHub App later)
- [ ] 8.4 Add error handling: repo not found, file not found, auth failure, rate limiting

## 9. Backend: Webhook Receiver

- [ ] 9.1 Add `POST /api/v1/webhooks/github` HTTP endpoint on the gRPC server mux
- [ ] 9.2 Implement GitHub webhook signature validation (HMAC-SHA256 with `X-Hub-Signature-256`)
- [ ] 9.3 Parse push event payload: extract modified files, filter for `.cs.md` extensions
- [ ] 9.4 For each modified `.cs.md` file: fetch content from GitHub, create AgentRun with `spec_content` and `spec_source: "webhook:github:..."`
- [ ] 9.5 Add webhook secret configuration (env var or k8s Secret)
- [ ] 9.6 Add repo allowlist configuration — only process webhooks from configured repos
- [ ] 9.7 Add tests for webhook validation, payload parsing, and run creation

## 10. Agent Environment: CodeSpeak CLI

- [ ] 10.1 Add `codespeak` to the agent container image's devbox config (or document how to add it via `devbox add`)
- [ ] 10.2 Verify `codespeak build` executes correctly in the agent pod environment
- [ ] 10.3 Document environment variable requirements for CodeSpeak (Anthropic API key passthrough)
