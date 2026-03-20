## ADDED Requirements

### Requirement: No dead UI files in the codebase

The web frontend SHALL NOT contain source files that are not imported by any other module. Specifically, the following files SHALL be removed:

#### Scenario: SpecEditor.tsx is deleted
- **GIVEN** `web/src/components/SpecEditor.tsx` exists and has zero imports
- **WHEN** the file is deleted
- **THEN** `tsc --noEmit` passes with no errors
- **AND** no grep for `SpecEditor` finds any remaining references

#### Scenario: use-mobile.tsx is deleted
- **GIVEN** `web/src/hooks/use-mobile.tsx` exists and has zero imports
- **WHEN** the file is deleted
- **THEN** `tsc --noEmit` passes with no errors
- **AND** no grep for `use-mobile` finds any remaining references

#### Scenario: use-toast.ts is deleted
- **GIVEN** `web/src/hooks/use-toast.ts` exists and has zero imports
- **WHEN** the file is deleted
- **THEN** `tsc --noEmit` passes with no errors
- **AND** no grep for `use-toast` finds any remaining references

### Requirement: Build integrity after deletion

The TypeScript build SHALL compile without errors after all three files are removed.

#### Scenario: Clean build after deletion
- **WHEN** all three files are deleted
- **AND** `tsc --noEmit` is run
- **THEN** the compiler reports zero errors
