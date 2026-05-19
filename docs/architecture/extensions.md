# Extensions

Two implementations exist; one is wired in.

| | `extensions/aot-determinism.ts` | `packages/pi-aot-extension/` |
|---|---|---|
| Style | Function (default export) into pi `ExtensionAPI` | Class harness `AOTExtension` |
| Transport | File IPC under `/workspace/.aot/input/` | gRPC (`AgentNotificationService`) |
| Tracing | JSONL audit logging | OpenTelemetry spans per tool call |
| HITL | `ask_user` writes `question.json`, polls `response.txt` | `waitForHumanInput()` over stdin |
| Loaded by sidecar | **Yes** — `--extension /opt/aot/extensions/aot-determinism.ts` | No — typechecked, tested, not wired |

## `aot-determinism.ts` (active)

The policy layer. Copied into the sidecar image via `docker/Dockerfile.sidecar`:

```dockerfile
COPY extensions/aot-determinism.ts /opt/aot/extensions/aot-determinism.ts
```

And the sidecar appends the flag in `internal/sidecar/gateway.go`:

```go
const aotExtensionPath = "/opt/aot/extensions/aot-determinism.ts"
args = append(args, "--extension", aotExtensionPath)
```

### Enforces

1. **Loop detection** — blocks after 3 consecutive identical tool calls.
2. **Turn cap** — 50 turns.
3. **Write validation (plan)** — spec files must use `SHALL`/`MUST`; `tasks.md` ≤ 30 checkboxes.
4. **Protected paths** — writes outside `/workspace` blocked.
5. **Role policies** (`PI_ROLE`):
   - `manage` → only `openspec/` + `.aot/`.
   - `implement` → no `ask_user`; must surface questions in output.

### Custom tools

- `ask_user` — writes `/workspace/.aot/input/question.json`, polls `response.txt`. 5-minute timeout. Manage only.
- `delegate_task` — writes a marker to `/workspace/.aot/subagents/<id>.json` for dashboard visibility. Handled inline (no subprocess).

## `packages/pi-aot-extension/` (available, dormant)

Class-based harness offering:

- Connect-RPC notifications to the sidecar (`AgentNotificationService.NotifyEvent` for `STARTED`, `TOOL_CALL`, `WAITING_FOR_INPUT`, `LOG`).
- OpenTelemetry spans per tool execution with `tool.name` and `agent_run_id`.
- Stdin-based HITL with buffered promise resolution.
- Tool registry: `registerTool` / `executeTool` / `getTools`.

CI typechecks it (`tsc --noEmit`) and `task test:extension` runs its tests — but no Dockerfile, Helm chart, or Go code references it at runtime.

## How they relate

`aot-determinism.ts` is the policy layer. `pi-aot-extension` would be the observability / transport layer if wired up. They are complementary in design; only the former is loaded. Activating the latter would layer OTel spans and direct gRPC events on top of the existing policy guardrails.
