## ADDED Requirements

### Requirement: Playwright harness connects to Wails app via CDP
The e2e test suite SHALL use Playwright's `connectOverCDP` to attach to the running UNCWORKS desktop app's WKWebView when the app is built with `--devtools` flag. A `web/playwright.config.ts` SHALL configure the base URL and CDP endpoint.

#### Scenario: CDP connection established
- **WHEN** UNCWORKS.app is running with devtools enabled
- **THEN** Playwright can connect and evaluate JavaScript in the webview

#### Scenario: App is discoverable
- **WHEN** `task test:e2e:app` runs
- **THEN** the task checks that UNCWORKS.app is running and fails with a clear message if not

### Requirement: Smoke e2e test navigates core pages
A smoke e2e test SHALL verify that the app loads, the main navigation is visible, and the three primary views (Projects, Runs, Settings) can be navigated to without crashing.

#### Scenario: App loads to projects page
- **WHEN** the app window opens
- **THEN** the projects list or empty state is visible within 5 seconds

#### Scenario: Navigate to runs page
- **WHEN** user clicks the Runs navigation item
- **THEN** the runs list view renders without an error toast

#### Scenario: Navigate to settings page
- **WHEN** user clicks the Settings navigation item
- **THEN** the settings form is visible with at least the namespace field

### Requirement: CustomSelect works in WKWebView
E2e tests SHALL verify that CustomSelect dropdowns open and select values in the real WKWebView environment, confirming the native `<select>` replacement works in production.

#### Scenario: CustomSelect opens and selects in WKWebView
- **WHEN** user clicks a CustomSelect in the real app
- **THEN** the options appear and clicking one updates the displayed value

### Requirement: Copilot sends message in real app
E2e tests SHALL verify the copilot panel can send a message and receive a response in the real app environment (requires live cluster + OpenRouter).

#### Scenario: Copilot message round-trip
- **WHEN** user types a message in the copilot panel and presses Enter
- **THEN** a non-empty assistant response appears within 30 seconds

### Requirement: Settings persist across app restart
E2e tests SHALL verify that saving a setting in the Settings view persists to `~/.config/uncworks/app.yaml` and is still present after the app window is hidden and shown again.

#### Scenario: Namespace change survives re-show
- **WHEN** user saves a namespace change and hides then shows the window
- **THEN** the namespace field still shows the saved value
