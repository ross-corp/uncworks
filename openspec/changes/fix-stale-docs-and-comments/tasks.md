## 1. Fix Stale Path References

- [x] 1.1 Fix comment in `api/v1alpha1/types.go` line 66: replace `/workspace/src/` with `/workspace/<repo>/`
- [x] 1.2 Fix comment in `proto/aot/api/v1/api.proto` line 81: replace `/workspace/src/` with `/workspace/<repo>/`
- [x] 1.3 Fix sidecar-exec spec path example to use `/workspace/<repo>/`
- [x] 1.4 Search entire codebase for remaining `/workspace/src/` references in comments and docs; fix any found

## 2. Tune Doc Staleness Script

- [x] 2.1 Update staleness script to exclude Helm dotted-notation patterns (e.g., `web.port`, `worker.image`)
- [x] 2.2 Verify script no longer flags Helm values as stale refs
- [x] 2.3 Verify script still catches actual stale file path references
