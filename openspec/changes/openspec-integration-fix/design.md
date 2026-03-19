## Context

A line-by-line audit found 9 issues in the spec-driven pipeline. The verify stage always passes because: Gate 2 hardcodes `ValidationValid = true` (overwriting the check), Gate 3 is a stub (returns nil), Gate 4 discards the LLM output, and Gate 5 swallows errors. PlanRun skips all validation. No openspec init happens. No pre-execute check exists.

## Goals / Non-Goals

**Goals:**
- Every OpenSpec CLI call has proper error handling (no `2>/dev/null`, no `|| true`)
- JSON parsing in Go (no python3 pipes)
- PlanRun actually validates via openspec CLI
- All 5 verify gates can actually fail
- LLM judge verdict is parsed and included in results
- Test commands extracted from spec scenarios and executed
- Workspace initialized with `openspec init` before planning
- Pre-execute artifact existence check

**Non-Goals:**
- Changing the pipeline architecture
- Adding new stages
- Modifying agent system prompts (they're already correct)

## Decisions

### Decision 1: Go-native OpenSpec JSON parsing

Create a `parseOpenSpecJSON` helper:
```go
func parseOpenSpecJSON(raw string) ([]byte, error) {
    // OpenSpec CLI outputs "- Loading..." prefix before JSON
    // Find first { and parse from there
    idx := strings.Index(raw, "{")
    if idx < 0 { return nil, fmt.Errorf("no JSON found in output") }
    return []byte(raw[idx:]), nil
}
```

This replaces all python3 inline parsing and `| tail -1` patterns.

### Decision 2: Gate 2 validation fix

Remove line 165 (`result.ValidationValid = true`). The actual validation result from the `if` block above should stand. If the ExecCommand fails, treat it as a gate failure, not a pass.

### Decision 3: Gate 4 LLM verdict parsing

After `pollUntilAgentDone` for the verify agent, read the agent's JSONL log file from the workspace PVC. Find the last `agent_end` event's messages. Extract the assistant's final message. Parse it as the JSON verdict structure. Include each criterion in the VerificationResult.

### Decision 4: Gate 3 test command extraction

Extend `detectTestCommands` to parse spec scenarios for backtick-wrapped commands on lines containing "run", "execute", "pass", "exit", "build", "test" keywords. Execute found commands via ExecCommand.

### Decision 5: Pre-execute check via ExecCommand

After PlanRun, use ExecCommand to verify artifact existence:
```bash
test -f openspec/changes/<id>/proposal.md && \
test -d openspec/changes/<id>/specs && \
ls openspec/changes/<id>/specs/*/spec.md >/dev/null 2>&1 && \
test -f openspec/changes/<id>/tasks.md
```

### Decision 6: File existence checks must run in pod, not on worker

Gate 2b currently uses `os.Stat` on the worker's filesystem via hostPath mount. This only works on single-node clusters. Switch to ExecCommand (`test -f <path>`) so it works on multi-node clusters.

## Risks / Trade-offs

- **Stricter validation will surface real issues** — pipelines that silently passed will now fail. This is intentional.
- **LLM verdict parsing depends on agent output format** — if the verify agent doesn't produce valid JSON, the verdict is treated as "no verdict" (non-fatal).
- **Test command extraction is heuristic** — may miss some commands or pick up false positives. Mitigated by only running commands that look like real commands (contain spaces, start with known tools).
