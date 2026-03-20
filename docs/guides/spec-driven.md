# Spec-Driven Pipeline

The spec-driven orchestration mode uses OpenSpec to impose structure on agent work. Instead of giving an agent a free-form prompt, the pipeline decomposes work into three stages: Plan, Execute, and Verify.

## How It Works

### Stage 1: Plan

A **manage** agent generates formal specifications from the user's prompt:

1. Scaffolds an OpenSpec change directory at `/workspace/openspec/changes/<name>/`
2. Reads the codebase to understand what needs to change
3. Writes `proposal.md`, spec files (with SHALL/MUST requirements and WHEN/THEN scenarios), `tasks.md`, and `design.md`
4. Validates the spec via `openspec validate` and `openspec status`

The plan stage fails if the spec is structurally invalid or incomplete.

### Stage 2: Execute

An **implement** agent receives the spec and implements the changes:

1. Reads the specs at `/workspace/openspec/changes/<name>/`
2. Implements code changes according to the requirements
3. Marks tasks complete in `tasks.md` as it progresses

### Stage 3: Verify

The system runs automated verification against the spec:

1. **Task completion** -- Checks that all tasks in `tasks.md` are marked done
2. **Structural validation** -- Runs `openspec validate` to ensure spec integrity
3. **File existence** -- Verifies files referenced in WHEN/THEN scenarios exist
4. **Test commands** -- Extracts and runs commands from spec scenarios (e.g., backtick-wrapped commands in WHEN/THEN lines)
5. **LLM judge** -- A verification agent evaluates each WHEN/THEN scenario against the implementation

If verification fails, the pipeline retries the Execute stage with the failure report appended to the prompt. The default retry limit is 3 attempts.

## Agent Roles

### manage (Plan + Verify)

- Can only write to `/workspace/openspec/` and `/workspace/.aot/`
- Cannot modify repository source code
- Can use `ask_user` to request human clarification
- Responsible for spec generation and verification oversight

### implement (Execute)

- Can read and write repository source code
- Cannot use `ask_user` (must surface questions in output)
- Follows the spec created by the manage agent

## Pipeline Configuration

Per-stage configuration is available via `PipelineConfig` in the AgentRun spec:

```yaml
pipelineConfig:
  plan:
    model: "default-cloud"
    timeoutSeconds: 300
    maxRetries: 2
    onFailure: "fail"
  execute:
    model: "default-cloud"
    timeoutSeconds: 900
    maxRetries: 3
    onFailure: "retry"
  verify:
    model: "default-cloud"
    timeoutSeconds: 180
    maxRetries: 1
    onFailure: "fail"
```

## Verification Result

The verification stage produces a structured JSON result written to the change directory:

```json
{
  "pass": true,
  "tasksCompleted": 5,
  "tasksTotal": 5,
  "validationValid": true,
  "automatedChecks": [...],
  "llmVerdict": { "pass": true, "criteria": [...] },
  "executionTimeMs": 45000
}
```
