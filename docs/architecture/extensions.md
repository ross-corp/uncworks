# Extension Architecture

The UNCWORKS platform has two extension implementations that serve different purposes. This document explains what each one does, which one the sidecar loads at runtime, and how they relate.

## Overview

| Aspect | `extensions/aot-determinism.ts` | `packages/pi-aot-extension/` |
|--------|--------------------------------|------------------------------|
| **Architecture** | Function-based (default export) | Class-based (`AOTExtension`) |
| **API surface** | pi-coding-agent `ExtensionAPI` hooks (`on("tool_call", ...)`, `on("turn_start", ...)`) | Standalone harness with its own `registerTool` / `executeTool` methods |
| **Transport** | File-based IPC (writes JSON to `/workspace/.aot/input/`) | gRPC via Connect-RPC (`AgentNotificationService`) |
| **Tracing** | None (audit logging to JSONL) | Full OpenTelemetry spans per tool call (`@opentelemetry/sdk-trace-node`) |
| **HITL mechanism** | `ask_user` tool: writes `question.json`, polls for `response.txt` | `waitForHumanInput()`: stdin reader with buffered promise resolution |
| **Loaded by sidecar** | Yes -- `--extension /opt/aot/extensions/aot-determinism.ts` | No -- not referenced in any Dockerfile or runtime config |
| **Status** | **Active** -- loaded on every agent run | **Available but not loaded by default** -- type-checked in CI, has tests, but not wired into the sidecar |

## `extensions/aot-determinism.ts` (Active)

This is the policy enforcement layer that the sidecar loads into `pi-coding-agent` at runtime. The sidecar constructs the flag in `internal/sidecar/gateway.go`:

```go
const aotExtensionPath = "/opt/aot/extensions/aot-determinism.ts"
args = append(args, "--extension", aotExtensionPath)
```

The file is copied into the sidecar container image by `docker/Dockerfile.sidecar`:

```dockerfile
COPY extensions/aot-determinism.ts /opt/aot/extensions/aot-determinism.ts
```

### What it enforces

1. **Loop detection** -- blocks repeated identical tool calls after 3 consecutive duplicates
2. **Turn limit** -- kills the agent after 50 turns to prevent runaway execution
3. **Write validation** -- ensures OpenSpec files use SHALL/MUST language (plan stage only)
4. **Protected paths** -- blocks writes outside `/workspace`
5. **Role-based policies** -- reads `PI_ROLE` env var to restrict what manage vs implement agents can do:
   - Manage agents cannot write/edit repo files (only `openspec/` and `.aot/`)
   - Implement agents cannot use `ask_user` (must surface questions in output)
6. **Task count guard** -- blocks `tasks.md` files with more than 30 checkboxes (plan stage only)

### Custom tools registered

- **`ask_user`** -- file-based HITL: writes a question to `/workspace/.aot/input/question.json` and polls for a response at `response.txt`. The sidecar's `SendInput` RPC writes that response file.
- **`delegate_task`** -- tracks subtask delegations as marker files under `/workspace/.aot/subagents/` for dashboard visibility. The subtask is handled inline (no actual subprocess).

## `packages/pi-aot-extension/` (Available, Not Loaded)

This is a class-based harness (`AOTExtension`) that provides:

- **gRPC notifications** -- uses Connect-RPC to call `AgentNotificationService.NotifyEvent` on the sidecar, sending events like `WAITING_FOR_INPUT`, `STARTED`, `TOOL_CALL`, and `LOG`
- **OpenTelemetry tracing** -- wraps every tool execution in an OTel span with `tool.name` and `agent_run_id` attributes, reporting success/error status
- **Stdin-based HITL** -- reads human input from `process.stdin` using `readline`, with a buffered promise model (input can arrive before or after the agent asks)
- **Tool registry** -- provides `registerTool()` / `executeTool()` / `getTools()` for managing tool definitions

### Current status

This package is **type-checked in CI** (`ci/main.go` runs `tsc --noEmit`) and has its own test suite (`Taskfile.yml` task `test:extension`), but it is **not loaded by the sidecar at runtime**. It is not referenced in any Dockerfile, Helm chart, or Go code that constructs agent commands.

It represents a more structured approach to the same problems `aot-determinism.ts` solves -- particularly OTel tracing and gRPC transport -- but has not been integrated into the runtime pipeline.

## How they relate

`aot-determinism.ts` is the **policy layer**: it hooks into `pi-coding-agent` events to enforce determinism, register custom tools, and guard against runaway behavior. It communicates with the sidecar indirectly through the filesystem (file-based HITL).

`pi-aot-extension` is the **observability/transport harness**: it wraps tool execution with OTel spans and communicates with the sidecar directly via gRPC. It does not enforce policies.

They are complementary in design but currently only `aot-determinism.ts` is wired into the runtime. If `pi-aot-extension` were activated, it would add structured tracing and direct gRPC event delivery on top of the policy guardrails that `aot-determinism.ts` provides.
