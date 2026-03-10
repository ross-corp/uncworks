# AOT Roadmap

Improvement areas discovered during implementation. Items are grouped by priority.

## High Priority

### Web Dashboard: Replace Mock Data with ConnectRPC Client
The web dashboard (`web/src/App.tsx`) currently renders hardcoded mock data. It should use the `AOTClient` from `@aot/shared/grpc` to fetch real agent runs and stream events via `WatchAgentRun`. This is the last step to complete the full ConnectRPC stack end-to-end.

### Contract Testing Between Components
No contract tests exist between the API server, sidecar, and TypeScript client. Add tests that verify proto compatibility across Go and TypeScript, e.g. by serializing/deserializing the same proto messages in both languages.

### E2E Test Pipeline
The `e2e/system_test.go` exists but requires a running k0s cluster and real API keys. Create a CI-friendly e2e test that uses mocked LLM responses (e.g. via LiteLLM proxy with mock backend) and a lightweight cluster (kind or k3d).

### Temporal Workflow Orchestration
Replace the ad-hoc controller reconciliation loop with Temporal workflows for agent lifecycle management. This provides durable execution, automatic retries, visibility, and a cleaner separation of orchestration logic.

## Medium Priority

### LiteLLM In-Cluster LLM Gateway
Deploy LiteLLM as an in-cluster service to provide a unified LLM API. This enables:
- Using Ollama or other local models for development/testing
- OpenRouter free tier for CI
- Model routing and fallback strategies
- Token usage tracking

### pi-aot-extension: Migrate to ConnectRPC
The `pi-aot-extension` uses `@grpc/grpc-js` for OpenTelemetry tracing but doesn't use the AOT shared client. If/when it needs to call AOT APIs directly, it should use the ConnectRPC client from `@aot/shared/grpc` instead of raw gRPC.

### Proto Breaking Change CI Gate
`buf breaking --against '.git#branch=main'` is configured but doesn't yet run in CI. Add it to the GitHub Actions workflow to prevent accidental proto breaking changes.

### Helm Chart: Temporal as Optional Dependency
The Helm chart should declare Temporal as an explicit optional dependency (not installed by default). Document the setup in `deploy/` with values for connecting to an external Temporal cluster or deploying one in-cluster.

## Low Priority

### Generated Code Packaging
The `gen/ts/` directory requires `@bufbuild/protobuf` to be installed at the repo root for module resolution. Consider publishing `gen/ts/` as a workspace package (`@aot/proto`) with its own `package.json` and dependencies for cleaner module boundaries.

### Devbox Shell Detection in Git Hooks
Git hooks use `command -v buf >/dev/null && ... || echo "skipping"` to gracefully degrade outside devbox shell. This works but is fragile. Consider using lefthook's `skip` conditions or a wrapper script that checks for devbox.

### Web Dashboard: Playwright Tests with Real API
Current Playwright tests only verify static rendering. Add tests that mock the ConnectRPC transport to verify streaming behavior, error handling, and HITL interactions.
