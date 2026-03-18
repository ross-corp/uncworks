## Why

The system cannot be used by customers today. Two critical gaps: (1) the Temporal workflow silently swallows errors — if the sidecar crashes mid-run, the workflow polls forever showing "Running" while the agent is dead, and cleanup failures leak pods and LLM keys with no visibility; (2) the web UI is a read-only skeleton — users can't create runs, cancel them, send human input, or see real-time events, despite the API client and store being fully implemented.

## What Changes

- **Workflow error handling**: Add consecutive error counting on GetAgentStatus polls; fail the workflow after 5 consecutive failures instead of looping forever. Log cleanup errors (RevokeLLMKey, CleanupPod) via Temporal workflow logger instead of silently discarding them.
- **Web UI create form**: New CreateRunForm component with repos (url/branch/path), prompt textarea, backend selector, and optional fields (devboxConfig, ttlSeconds, envVars, image).
- **Web UI cancel**: Cancel button on run detail view with confirmation dialog, wired to AOTClient.cancelAgentRun().
- **Web UI event streaming**: Replace 5-second polling with AOTClient.watchAgentRun() server-streaming. Add event log panel displaying log, tool_call, and phase_changed events in real-time.
- **Web UI human input**: Input form that appears when run phase is WaitingForInput, wired to AOTClient.sendHumanInput().
- **Web UI routing**: Add @solidjs/router with / for list view and /:id for detail view with shareable URLs.
- **Web UI store integration**: Replace local signals in App.tsx with the existing createAgentStore from packages/shared.

## Capabilities

### New Capabilities
- `workflow-error-resilience`: Consecutive error counting on status polls, cleanup error logging, and graceful failure after sustained errors
- `web-create-run`: Form to create agent runs from the web UI
- `web-run-actions`: Cancel runs and send human input from the web UI
- `web-event-streaming`: Real-time event streaming via server-sent events replacing polling
- `web-routing`: Client-side routing with shareable run URLs

### Modified Capabilities

## Impact

- `internal/temporal/workflow.go` — polling loop error handling, cleanup defer block
- `web/src/App.tsx` — rewrite to use store and router
- `web/src/components/` — new CreateRunForm, EventLog, HumanInputForm, CancelButton components; update AgentRunDetail
- `web/package.json` — add @solidjs/router dependency
- `test/temporal/workflow_test.go` — new tests for error counting and cleanup logging
