## ADDED Requirements

### Requirement: data-testid on all interactive components
All interactive web components SHALL have `data-testid` attributes following the naming convention `[component]-[element]-[qualifier]`.

#### Scenario: Sidebar elements have test IDs
- **WHEN** the sidebar renders
- **THEN** phase filter buttons have `data-testid="sidebar-phase-{phase}"`
- **AND** workspace buttons have `data-testid="sidebar-workspace-{name}"`
- **AND** repo buttons have `data-testid="sidebar-repo-{index}"`

#### Scenario: Table rows have test IDs
- **WHEN** the agent run table renders
- **THEN** each row has `data-testid="table-row-{id}"`
- **AND** phase badges within rows have `data-testid="table-row-{id}-phase"`

#### Scenario: Detail panel elements have test IDs
- **WHEN** the detail panel renders for a selected run
- **THEN** the panel has `data-testid="detail-panel"`
- **AND** key fields have test IDs: `detail-name`, `detail-phase`, `detail-hitl-input`, `detail-hitl-send`

#### Scenario: Form elements have test IDs
- **WHEN** the agent run form renders
- **THEN** the form has `data-testid="form-modal"`
- **AND** input fields have test IDs: `form-name-input`, `form-repo-row-{index}-url`, `form-tab-prompt`, `form-tab-spec`, `form-submit`

### Requirement: Stale Playwright tests removed
The existing stale `web/e2e/app.spec.ts` SHALL be deleted and replaced with the new spec files.

#### Scenario: Old tests removed
- **WHEN** the new Playwright suite is created
- **THEN** `web/e2e/app.spec.ts` no longer exists
- **AND** all new spec files target current component structure with valid data-testid selectors
