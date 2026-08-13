# Extensions

Two extension implementations exist. The sidecar loads one of them.

| | `extensions/aot-determinism.ts` | `packages/pi-aot-extension/` |
|---|---|---|
| Style | Function (default export) into pi `ExtensionAPI` | Class harness `AOTExtension` |
| Transport | File IPC under `/workspace/.aot/input/` | gRPC (`AgentNotificationService`) |
| Tracing | JSONL audit logging | OpenTelemetry spans per tool call |
| HITL | `ask_user` writes `question.json`, polls `response.txt` | `waitForHumanInput()` over stdin |
| Loaded by sidecar | Yes, through `--extension /opt/aot/extensions/aot-determinism.ts` | No. It is typechecked and tested, but nothing loads it |

## `aot-determinism.ts`, the loaded extension

This is the policy layer. `docker/Dockerfile.sidecar` copies it into the sidecar
image.

```dockerfile
COPY extensions/aot-determinism.ts /opt/aot/extensions/aot-determinism.ts
```

The sidecar appends the flag in `internal/sidecar/gateway.go`.

```go
const aotExtensionPath = "/opt/aot/extensions/aot-determinism.ts"
args = append(args, "--extension", aotExtensionPath)
```

### What it enforces

1. Loop detection. It blocks the agent after 3 identical tool calls in a row.
2. A turn cap of 50 turns.
3. Write validation during the Plan stage. Every spec file MUST use `SHALL` or
   `MUST`, and `tasks.md` MUST hold no more than 30 checkboxes.
4. Protected paths. It blocks every write outside `/workspace`.
5. Role policy, read from `PI_ROLE`. The `manage` role may write only to
   `openspec/` and `.aot/`. The `implement` role may not call `ask_user`, so it
   MUST raise a question in its output instead.

### Custom tools

- `ask_user` writes `/workspace/.aot/input/question.json` and polls
  `response.txt`. It times out after 5 minutes. Only the `manage` role may call
  it.
- `delegate_task` writes a marker to `/workspace/.aot/subagents/<id>.json` so the
  dashboard can show the delegation. It runs inline and starts no subprocess.

## `packages/pi-aot-extension/`, the dormant extension

This is a class-based harness. It provides:

- ConnectRPC notifications to the sidecar, through
  `AgentNotificationService.NotifyEvent`, for `STARTED`, `TOOL_CALL`,
  `WAITING_FOR_INPUT`, and `LOG`.
- OpenTelemetry spans per tool execution with `tool.name` and `agent_run_id`.
- Human input over stdin, with buffered promise resolution.
- A tool registry: `registerTool`, `executeTool`, and `getTools`.

CI typechecks it with `tsc --noEmit`, and `task test:extension` runs its tests.
No Dockerfile, Helm chart, or Go source references it at runtime.

## How they relate

`aot-determinism.ts` is the policy layer. `pi-aot-extension` would be the
observability and transport layer once something loads it. The two are
complementary, and only the first one runs today. Loading the second one would
add OpenTelemetry spans and direct gRPC events on top of the existing policy
guardrails.
