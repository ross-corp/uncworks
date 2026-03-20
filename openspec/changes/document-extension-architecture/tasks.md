## 1. Architecture Documentation

- [ ] 1.1 Create `docs/architecture/extensions.md` with overview of both extension files
- [ ] 1.2 Document that sidecar loads `aot-determinism.ts` via `--extension` flag
- [ ] 1.3 Document what `pi-aot-extension` does (gRPC transport, OTel tracing, HITL harness)
- [ ] 1.4 Add comparison table (transport, purpose, loaded-by, status)
- [ ] 1.5 Clarify if `pi-aot-extension` is currently used or deprecated

## 2. Code Comments

- [ ] 2.1 Add header comment to `extensions/aot-determinism.ts` explaining its role (5-8 lines)

## 3. Verification

- [ ] 3.1 Confirm sidecar code references `--extension extensions/aot-determinism.ts`
- [ ] 3.2 Check if pi-aot-extension is imported/loaded anywhere at runtime
