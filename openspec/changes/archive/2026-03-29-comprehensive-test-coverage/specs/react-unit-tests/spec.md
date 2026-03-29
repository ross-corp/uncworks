## ADDED Requirements

### Requirement: Vitest is configured for the web package
`web/` SHALL have a `vitest.config.ts` using the `jsdom` environment and `@testing-library/react`, with tests discovered in `src/**/__tests__/**/*.test.{ts,tsx}`.

#### Scenario: Tests discovered and run
- **WHEN** `npm run test:unit` is executed in `web/`
- **THEN** all `*.test.ts` and `*.test.tsx` files in `src/` SHALL be discovered and executed

#### Scenario: jsdom environment active
- **WHEN** a test imports a React component
- **THEN** DOM APIs (document, window, HTMLElement) SHALL be available

### Requirement: Core hooks are unit tested
The hooks `useSettings`, `usePoll`, `apiFetch`, and `useThemeNew` SHALL have unit tests covering their primary behaviors.

#### Scenario: useSettings loads and caches settings
- **WHEN** `useSettings` is rendered
- **THEN** it SHALL call the settings API once and return the parsed settings object

#### Scenario: usePoll calls callback on interval
- **WHEN** `usePoll(fn, 1000)` is rendered with fake timers
- **THEN** `fn` SHALL be called immediately and again after each 1000ms tick

#### Scenario: apiFetch prepends base URL
- **WHEN** `apiFetch('/api/v1/runs')` is called with `VITE_API_URL` set to `http://localhost:50055`
- **THEN** the resulting fetch SHALL target `http://localhost:50055/api/v1/runs`

#### Scenario: apiFetch uses relative URL when base not set
- **WHEN** `apiFetch('/api/v1/runs')` is called with no `VITE_API_URL`
- **THEN** the fetch SHALL use `/api/v1/runs` as the URL

### Requirement: Key view components are behavior-tested
`RunListView`, `ProjectListView`, and `SettingsView` SHALL have tests covering state logic and user interactions, not visual rendering.

#### Scenario: RunListView renders run rows
- **WHEN** `RunListView` is rendered with mocked API returning 3 runs
- **THEN** 3 run rows SHALL appear in the document

#### Scenario: RunListView shows empty state
- **WHEN** `RunListView` is rendered with mocked API returning empty array
- **THEN** an empty state message SHALL be visible

#### Scenario: ProjectListView navigates on row click
- **WHEN** a project row is clicked in `ProjectListView`
- **THEN** the router SHALL navigate to `/projects/<name>`

#### Scenario: SettingsView saves on button click
- **WHEN** the user changes a settings field and clicks Save
- **THEN** `SaveSettings` SHALL be called with the updated value

### Requirement: Tests run in CI as part of the web check step
Web unit tests SHALL run in CI alongside TypeScript type checking.

#### Scenario: Unit tests in CI
- **WHEN** the Dagger `Check()` function runs
- **THEN** `npm run test:unit` SHALL be executed for the `web/` package
