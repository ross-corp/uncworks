## ADDED Requirements

### Requirement: ExtendedRunStatus typed interface replaces unknown casts
The system SHALL define an `ExtendedRunStatus` interface in `web/src/hooks/useClient.ts` that extends the shared `RunStatus` type with all fields currently accessed via `as unknown as { ... }` casts. The fields SHALL include at minimum: `archived?: boolean`, `totalCost?: string`, `totalAdditions?: number`, `totalDeletions?: number`, `ciFixAttempts?: number`, `lastCIStatus?: string`, `parentPRUrl?: string`.

#### Scenario: No unsafe casts in useClient.ts
- **WHEN** `useClient.ts` is compiled with strict TypeScript
- **THEN** there are zero occurrences of `as unknown as` in the file

#### Scenario: Field access is type-safe
- **WHEN** code accesses `status.archived` via the `ExtendedRunStatus` interface
- **THEN** TypeScript infers the type as `boolean | undefined` without requiring a cast

### Requirement: All 36 unknown-cast files are audited and resolved
The system SHALL eliminate all `as unknown as { ... }` patterns across the 36 affected files in `web/src/`, either by introducing typed interfaces or, where a cast is genuinely unavoidable, replacing it with a typed assertion function with an explanatory comment.

#### Scenario: Codebase-wide cast audit passes
- **WHEN** a search for `as unknown as` is run against `web/src/`
- **THEN** zero results are returned
