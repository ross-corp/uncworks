## ADDED Requirements

### Requirement: MSW test infrastructure exists
The test suite SHALL include a Mock Service Worker (MSW v2) setup using `setupServer` (Node mode) in `web/src/mocks/server.ts`, with default happy-path handlers in `web/src/mocks/handlers.ts` and typed fixture factories in `web/src/mocks/fixtures.ts`. Vitest setup SHALL start the MSW server before all tests and reset handlers after each test.

#### Scenario: MSW server starts
- **WHEN** Vitest runs any test file that imports from `src/mocks/server`
- **THEN** the MSW server intercepts fetch calls without requiring a real API

#### Scenario: Fixtures are type-safe
- **WHEN** an API response shape changes in TypeScript
- **THEN** the fixture factory produces a type error at compile time

### Requirement: ProjectListView renders and navigates
Tests SHALL verify that the project list fetches projects via the real `apiFetch` pipeline (intercepted by MSW) and renders them, and that clicking a project row navigates to the detail view.

#### Scenario: Projects load and display
- **WHEN** MSW returns two projects from `GET /api/v1/projects`
- **THEN** both project names appear in the rendered list

#### Scenario: Empty state shown when no projects
- **WHEN** MSW returns an empty array from `GET /api/v1/projects`
- **THEN** an empty-state message is visible

#### Scenario: Click navigates to detail
- **WHEN** user clicks a project row
- **THEN** the router navigates to `/projects/:name`

### Requirement: ProjectDetailView loads and displays runs
Tests SHALL verify that navigating to a project detail page fetches runs via `listAgentRuns`, applies `mapRun` correctly, and renders the runs tab content.

#### Scenario: Runs tab shows runs for the project
- **WHEN** MSW returns a list of runs and user switches to the Runs tab
- **THEN** only runs whose `spec.projectRef` matches the current project are shown

#### Scenario: mapRun phase mapping is correct
- **WHEN** MSW returns a run with `status.phase: "Running"`
- **THEN** the rendered badge shows "running" (lowercase web type)

#### Scenario: No crash on empty run list
- **WHEN** MSW returns an empty array from `listAgentRuns`
- **THEN** the runs tab renders without error

### Requirement: NewRunView submits run creation
Tests SHALL verify the new run form collects required fields and submits a POST request via the real fetch pipeline.

#### Scenario: Form submit fires correct API call
- **WHEN** user fills in prompt, selects a project, and clicks Submit
- **THEN** `POST /api/v1/runs` is called with the prompt and projectRef in the body

#### Scenario: Model select uses CustomSelect
- **WHEN** user opens the model dropdown
- **THEN** all available model options are visible and selectable via CustomSelect

### Requirement: RunListView filters and displays runs
Tests SHALL verify the run list fetches all runs, applies filter state, and re-fetches on interval.

#### Scenario: Filter by status
- **WHEN** user selects "failed" in the status filter CustomSelect
- **THEN** only failed runs are visible in the list

#### Scenario: Null API response does not crash
- **WHEN** MSW returns `null` instead of an array
- **THEN** the component renders an empty list without throwing

### Requirement: SettingsView persists changes via real API
Tests SHALL verify that saving settings calls `SaveSettings` (the Wails binding) with the correct payload, exercising the full settings data pipeline.

#### Scenario: Namespace change is saved
- **WHEN** user changes the namespace field and clicks Save
- **THEN** the Wails `SaveSettings` binding is called with the new namespace value

#### Scenario: LLM URL change updates isClusterLiteLLM
- **WHEN** user sets litellmURL to `https://openrouter.ai/api/v1`
- **THEN** the LiteLLM service row is hidden from the services section

### Requirement: CopilotBottomPanel sends and displays messages
Tests SHALL verify the copilot panel POSTs to `/api/v1/chat/stream`, streams SSE tokens, and renders the assistant reply.

#### Scenario: Message sent and response streamed
- **WHEN** user types a message and presses Enter
- **THEN** `POST /api/v1/chat/stream` is called with the message content and model

#### Scenario: Streamed tokens accumulate in the reply
- **WHEN** MSW sends three SSE `data:` chunks
- **THEN** all three tokens are appended to the assistant message

#### Scenario: Error response shown as toast
- **WHEN** MSW returns a 502 from the chat endpoint
- **THEN** a toast error is displayed and no blank message is appended

### Requirement: CustomSelect works in all views
Tests SHALL verify that the `CustomSelect` component opens on click, selects a value, and closes on outside click.

#### Scenario: Option selected via click
- **WHEN** user clicks a CustomSelect and clicks an option
- **THEN** the `onChange` callback is called with the option's value

#### Scenario: Closes on outside click
- **WHEN** user clicks outside the open CustomSelect
- **THEN** the dropdown closes without calling onChange

### Requirement: ScheduleNewView creates a schedule
Tests SHALL verify the schedule creation form collects fields and submits.

#### Scenario: Schedule form submission
- **WHEN** user fills cron expression, selects a chain, and clicks Create
- **THEN** `POST /api/v1/schedules` is called with cron and chainRef

### Requirement: ChainNewView creates a chain
Tests SHALL verify the chain creation form submits correctly.

#### Scenario: Chain form submission
- **WHEN** user fills the chain name and adds a step
- **THEN** `POST /api/v1/chains` is called with name and steps array

### Requirement: RunDetailView renders run phases
Tests SHALL verify that the run detail page renders phase badges and activity feed from the runs API.

#### Scenario: Phase badges render
- **WHEN** MSW returns a run with plan/implement phases
- **THEN** phase badges for each phase are visible on the page

### Requirement: Layout renders navigation
Tests SHALL verify the main Layout renders the navigation sidebar and responds to route changes.

#### Scenario: Active route is highlighted
- **WHEN** the current route is `/projects`
- **THEN** the Projects nav item has the active style applied
