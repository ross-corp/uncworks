## Why

Two extension implementations exist with completely different architectures, and their relationship is undocumented:
- `packages/pi-aot-extension/` — class-based TypeScript, gRPC transport, OTel tracing, HITL harness
- `extensions/aot-determinism.ts` — function-based, pi hooks API, policy enforcement (blocked commands, path restrictions)

It's unclear which one the sidecar loads, whether pi-aot-extension is actively used or deprecated, and how they relate to each other. New contributors (and agents) can't reason about the extension system.

## What Changes

- **Architecture doc** — `docs/architecture/extensions.md` explaining both extension files, their purposes, and which is active
- **Code comment** — header comment in `extensions/aot-determinism.ts` explaining its role
- **Clarification** — document whether `pi-aot-extension` is currently loaded or deprecated

## Capabilities

### New Capabilities
- `extension-docs`: Documentation explaining the two extension implementations and their relationship.

### Modified Capabilities

None.

## Impact

- `docs/architecture/extensions.md` — new file
- `extensions/aot-determinism.ts` — header comment added
