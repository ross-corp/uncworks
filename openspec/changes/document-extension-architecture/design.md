## Context

The sidecar launches the pi-coding-agent with `--extension extensions/aot-determinism.ts`. This file uses the pi hooks API (`onToolCall`, `onMessage`) to enforce determinism policies: blocking dangerous shell commands, restricting file paths, and enforcing workspace boundaries. Separately, `packages/pi-aot-extension/` is a full npm package with gRPC client code, OTel tracing setup, and a HITL (human-in-the-loop) harness. Their relationship is unclear from code alone.

## Goals / Non-Goals

**Goals:**
- Document what each extension file does and why it exists
- Clarify which extension the sidecar actually loads at runtime
- State whether pi-aot-extension is active, deprecated, or planned
- Add a header comment to aot-determinism.ts for quick orientation

**Non-Goals:**
- Merging the two extensions into one
- Changing any runtime behavior
- Documenting the pi hooks API itself (that's upstream)

## Decisions

### Decision 1: Single architecture doc covers both extensions

One `docs/architecture/extensions.md` file covers both extensions side-by-side with a comparison table. This makes the relationship immediately clear.

**Rationale:** Separate docs for each extension would miss the point — the confusion is about how they relate, not what each one does individually.

### Decision 2: Header comment, not inline comments

Add a block comment at the top of `aot-determinism.ts` (5-8 lines) explaining its role, rather than scattering inline comments. The file is short enough that a header is sufficient.

### Decision 3: Explicitly state pi-aot-extension status

The doc must have a clear "Status" field for pi-aot-extension: active, deprecated, or experimental. No ambiguity.
