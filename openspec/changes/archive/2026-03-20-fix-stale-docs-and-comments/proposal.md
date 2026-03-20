## Why

Proto comments in `types.go:66` and `api.proto:81` reference `/workspace/src/` but the actual workspace layout uses `/workspace/<repo>/`. The sidecar-exec spec also references the old `/workspace/src/` path. Meanwhile, the doc staleness script flags Helm values (dotted notation like `web.port`) as stale refs, producing false positives. These stale references mislead contributors who read comments to understand the system.

## What Changes

- **Fix Go type comment**: Update the path example in `api/v1alpha1/types.go` line 66 from `/workspace/src/` to `/workspace/<repo>/`.
- **Fix proto comment**: Update the path example in `proto/aot/api/v1/api.proto` line 81 similarly.
- **Fix sidecar-exec spec**: Update path references in the sidecar-exec spec document.
- **Tune staleness script**: Exclude Helm value patterns (dotted.notation) from being flagged as stale references.
- **Sweep remaining references**: Search for any other `/workspace/src/` occurrences in docs or code comments.

## Capabilities

### New Capabilities

None — this is a documentation/comment fix.

### Modified Capabilities

None — no behavior changes.

## Impact

- `api/v1alpha1/types.go` — Fix path comment
- `proto/aot/api/v1/api.proto` — Fix path comment
- Sidecar-exec spec — Fix path example
- Doc staleness script — Exclude Helm dotted-notation patterns
