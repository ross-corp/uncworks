## ADDED Requirements

### Requirement: NewRunView form state managed by useRunForm
The NewRunView component SHALL manage all form fields via the `useRunForm()` hook and SHALL NOT contain individual `useState` calls for form fields.

#### Scenario: Form renders with correct initial values
- **WHEN** the user navigates to `/new`
- **THEN** all form fields display the same default values as before the `useRunForm` migration

#### Scenario: Field changes update form state
- **WHEN** the user changes any form field in NewRunView
- **THEN** the field value updates correctly via the `useRunForm` setter

### Requirement: NewRunView catch blocks log error detail
The NewRunView component SHALL log all caught errors to `console.error` with the component name prefix before any existing toast or no-op, so that errors are visible during debugging.

#### Scenario: Error in form submission is logged
- **WHEN** NewRunView's run submission throws an error
- **THEN** the error is logged via `console.error('[NewRunView]', err)` in addition to any existing user-facing toast
