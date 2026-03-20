## 1. Server Tests

- [x] 1.1 Add tests for `parseAgentJSONL` dedup logic (duplicate message IDs, malformed lines)
- [x] 1.2 Add tests for `isHiddenDir` (dotfiles, normal dirs, nested paths)
- [x] 1.3 Add tests for `parseLsOutput` (normal output, empty output, permission errors)

## 2. Sidecar Tests

- [x] 2.1 Add tests for `ExecCommand` — verify exact workdir is set on exec.Cmd
- [x] 2.2 Add tests for `extractToolCallSignature` — identical calls, different calls, edge cases

## 3. Temporal Tests

- [x] 3.1 Add tests for `PlanRun` — openspec scaffolding produces valid spec structure, validation rejects bad input
- [x] 3.2 Add tests for `VerifyRun` — 5 gate logic (format, scaffold, lint, test, verify) with mock spec content
