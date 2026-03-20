## Context

A codebase audit identified three frontend files with zero imports: `SpecEditor.tsx` (a component that was likely part of an earlier spec editing feature), `use-mobile.tsx` (a mobile detection hook, possibly from shadcn scaffolding), and `use-toast.ts` (a toast notification hook, superseded by the current notification approach). None are referenced anywhere.

## Goals / Non-Goals

**Goals:**
- Remove all three dead files
- Confirm the TypeScript build still passes after deletion

**Non-Goals:**
- Auditing for other dead code (covered by separate change)
- Replacing functionality these files provided (they were unused)

## Decisions

### Decision 1: Delete without replacement

All three files are unused. No replacement or migration is needed. If toast or mobile detection is needed in the future, it can be re-added.

### Decision 2: Verify via tsc --noEmit

Run the TypeScript compiler in check-only mode to confirm no transitive imports are broken. This catches any indirect dependencies that a simple grep might miss.

## Risks / Trade-offs

- **Very low risk**: Files have zero imports. The only risk is a transitive dependency not caught by grep, which `tsc --noEmit` will catch.
