## ADDED Requirements

### Requirement: Storybook configured with Vite
The project SHALL have Storybook 8 configured with `@storybook/react-vite`, sharing the existing Vite and Tailwind configuration from the web app.

#### Scenario: Storybook starts
- **WHEN** `npm run storybook` is run in the `web/` directory
- **THEN** Storybook launches and renders all stories with Tailwind styles applied

### Requirement: Every UI component has stories
Each component in `web/src/components/` SHALL have a corresponding `.stories.tsx` file demonstrating the component in isolation with multiple states.

#### Scenario: StatusBadge stories
- **WHEN** the StatusBadge story is viewed
- **THEN** it shows all 6 phase variants (pending, running, waiting_for_input, succeeded, failed, cancelled)

#### Scenario: AgentRunTable stories
- **WHEN** the AgentRunTable story is viewed
- **THEN** it shows: populated table with mock runs, empty state, loading state

#### Scenario: AgentRunDetailPanel stories
- **WHEN** the AgentRunDetailPanel story is viewed
- **THEN** it shows: running state with events, waiting_for_input state with input form, completed state, failed state

#### Scenario: AgentRunForm stories
- **WHEN** the AgentRunForm story is viewed
- **THEN** it shows the create form with default values and validation states

#### Scenario: Layout and Sidebar stories
- **WHEN** the Layout/Sidebar stories are viewed
- **THEN** they show the shell with sidebar navigation in collapsed and expanded states

#### Scenario: ConfirmDialog stories
- **WHEN** the ConfirmDialog story is viewed
- **THEN** it shows the dialog in open state with cancel/confirm actions

### Requirement: Stories use mock data fixtures
Stories SHALL import mock data from a shared fixtures file (`web/src/data/mock.ts`) rather than duplicating test data inline.

#### Scenario: Mock data consistency
- **WHEN** a story renders with mock data
- **THEN** the mock data matches the `AgentRun` TypeScript type exactly
