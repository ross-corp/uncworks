## Why

Two OpenSpec specs don't match the current implementation. The `ui-theming` spec requires 12 shadcn themes but only light/dark mode is actually implemented. The `sidecar-exec` spec references `/workspace/src/` as the default path, but the real layout is `/workspace/<repo>/`. Stale specs cause agent runs to attempt work that doesn't align with reality, wasting tokens and producing wrong verification results.

## What Changes

- **ui-theming spec** — change the 12-theme requirement to light/dark/system mode toggle (what's actually shipped)
- **sidecar-exec spec** — change path examples from `/workspace/src/` to `/workspace/<repo>/`
- **Audit all specs** — grep for `/workspace/src/` across all spec files and fix any other stale references

## Capabilities

### New Capabilities
- `spec-accuracy`: Specs accurately reflect the current implementation state.

### Modified Capabilities

None.

## Impact

- `openspec/specs/ui-theming/spec.md` — theme count reduced from 12 to 2+system
- `openspec/specs/sidecar-exec/spec.md` — path examples corrected
- Any other specs referencing `/workspace/src/`
