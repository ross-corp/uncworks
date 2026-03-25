## ADDED Requirements

### Requirement: useRunForm hook encapsulates NewRunView form state
The system SHALL provide a `useRunForm()` hook in `web/src/hooks/useRunForm.ts` that manages all 32 form fields currently implemented as individual `useState` calls in `NewRunView.tsx`. The hook SHALL return `{ form, set, reset }` where `form` is a typed object of all field values, `set` is a record of per-field setter functions, and `reset` restores all fields to their initial values.

#### Scenario: Form state initialized with defaults
- **WHEN** `useRunForm()` is called on component mount
- **THEN** `form` contains the same default values that the 32 `useState` calls previously initialized

#### Scenario: Per-field setter updates only that field
- **WHEN** `set.modelName("gpt-4o")` is called
- **THEN** `form.modelName` updates to `"gpt-4o"` and all other fields remain unchanged

#### Scenario: Reset restores defaults
- **WHEN** `reset()` is called after fields have been modified
- **THEN** all `form` fields return to their initial default values

### Requirement: NewRunView uses useRunForm exclusively
The system SHALL update `NewRunView.tsx` to call `useRunForm()` and remove all 32 individual `useState` declarations for form fields.

#### Scenario: No individual form useState calls remain
- **WHEN** `NewRunView.tsx` is inspected
- **THEN** there are no `useState` calls for individual form fields — all form state comes from `useRunForm()`
