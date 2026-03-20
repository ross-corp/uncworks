## Why

Three UI files are not imported anywhere in the codebase: `SpecEditor.tsx`, `use-mobile.tsx`, and `use-toast.ts`. They add confusion for contributors navigating the code and inflate bundle size. A grep for their imports confirms zero references.

## What Changes

- **Delete `SpecEditor.tsx`**: Unused component with no imports.
- **Delete `use-mobile.tsx`**: Unused hook with no imports.
- **Delete `use-toast.ts`**: Unused hook with no imports.
- **Verify build**: Run `tsc --noEmit` to confirm no imports break.

## Capabilities

### New Capabilities

None — this is dead code removal.

### Modified Capabilities

None — no behavior changes.

## Impact

- `web/src/components/SpecEditor.tsx` — Delete
- `web/src/hooks/use-mobile.tsx` — Delete
- `web/src/hooks/use-toast.ts` — Delete
