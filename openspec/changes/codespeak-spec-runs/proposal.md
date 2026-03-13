## Why

Today the only way to spawn an agent run is to write a free-text prompt. CodeSpeak offers a more structured alternative — markdown specs (`.cs.md` files) that describe *what* code should do, which an LLM compiles to production code. Integrating CodeSpeak specs as a first-class run trigger gives users a higher-fidelity way to describe work, and the specs themselves become durable, version-controlled artifacts. We want to edit these specs in the web UI using Monaco editor (VS Code's editor) and also support syncing them to/from GitHub repos, so the same spec can be authored in the UI, committed to a repo, or pushed via CI — all triggering the same agent run pipeline.

## What Changes

- **Monaco editor in the web UI** — embed Microsoft's Monaco editor for authoring `.cs.md` specs with markdown syntax highlighting, providing a VS Code-like editing experience
- **Spec-driven agent runs** — a new run type where the input is a CodeSpeak spec instead of a free-text prompt; the agent's job is to run `codespeak build` against the spec in the workspace
- **Spec storage** — specs are stored server-side (as part of the AgentRunSpec or as a referenced file) so they can be retrieved, edited, and re-run
- **GitHub sync** — specs can be pushed to a GitHub repo (committing the `.cs.md` file) or pulled from a repo (loading a spec from a file path), creating a bidirectional bridge between the UI and version control
- **Webhook/hook trigger** — a mechanism to trigger spec-driven runs from external sources (GitHub push webhook, CI pipeline, CLI) when a `.cs.md` file changes in a repo
- **Proto/CRD extension** — add spec content and spec source fields to `AgentRunSpec` to support spec-driven runs alongside prompt-driven runs

## Capabilities

### New Capabilities
- `spec-editor`: Monaco editor integration in the web UI for authoring and editing CodeSpeak `.cs.md` specs — editor component, syntax support, file management
- `spec-driven-runs`: New agent run type triggered by CodeSpeak specs — run creation, spec-to-prompt translation, `codespeak build` execution, spec storage on the run
- `spec-sync`: Bidirectional sync of spec files with GitHub repos and external triggers — push specs to repos, pull specs from repos, webhook-triggered runs

### Modified Capabilities
<!-- No existing spec-level requirements change -->

## Impact

- **Proto/CRD** (`api.proto`, `types.go`): New fields on `AgentRunSpec` for spec content, spec file path, and spec source metadata
- **Web UI**: New `SpecEditor` component wrapping Monaco, new "Spec Run" tab/mode in the agent run form, Monaco npm dependency
- **Shared types** (`packages/shared`): Extended `AgentRunSpec` with spec fields
- **gRPC handler** (`internal/server/grpc.go`): Handle spec content in `CreateAgentRun`, translate spec to agent prompt
- **Workflow/Hydrator**: Ensure `codespeak` is available in agent devbox environment, write spec file to workspace before agent starts
- **New API endpoint**: Webhook receiver for external spec-run triggers (GitHub webhooks, CI)
- **Dependencies**: `monaco-editor` npm package, `@monaco-editor/react` wrapper
