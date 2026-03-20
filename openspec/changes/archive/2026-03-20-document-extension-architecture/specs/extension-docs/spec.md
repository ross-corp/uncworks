## Purpose

Document the two extension implementations so contributors and agents understand the extension architecture.

## ADDED Requirements

### Requirement: Architecture doc explains both extensions
A `docs/architecture/extensions.md` file SHALL exist explaining both `extensions/aot-determinism.ts` and `packages/pi-aot-extension/`.

#### Scenario: Doc covers aot-determinism.ts
- **WHEN** reading the architecture doc
- **THEN** it explains that aot-determinism.ts is loaded by the sidecar via `--extension` flag and enforces determinism policies

#### Scenario: Doc covers pi-aot-extension
- **WHEN** reading the architecture doc
- **THEN** it explains what pi-aot-extension does (gRPC, OTel, HITL) and states its current status (active/deprecated)

#### Scenario: Comparison table
- **WHEN** reading the architecture doc
- **THEN** a table compares the two extensions (transport, purpose, loaded by, status)

### Requirement: aot-determinism.ts has header comment
The file `extensions/aot-determinism.ts` SHALL have a header comment (5-8 lines) explaining its purpose and how it's loaded.

#### Scenario: Header comment present
- **WHEN** opening `extensions/aot-determinism.ts`
- **THEN** the first lines are a block comment explaining the file's role as the sidecar-loaded policy extension
