## Why

AOT has unit tests and basic E2E tests, but lacks contract testing between components and a full-stack E2E workflow that exercises the real agent lifecycle with a real LLM. As the system grows (ConnectRPC, Temporal, LiteLLM, multi-agent), the surface area for integration bugs grows quadratically. Without schema-level contract enforcement (`buf breaking`), semantic validation (`protovalidate`), service-level contract tests (GripMock), and E2E tests that actually run agents against a model, regressions will slip through silently. The testing pipeline must be as composable as the system itself -- each layer testable in isolation, each boundary verified by contract.

## What Changes

- **`buf breaking` in CI**: Every PR is checked for protobuf schema breaking changes against `main`. Blocks merges on incompatible changes.
- **`buf lint` enforcement**: Proto style consistency enforced in CI.
- **GripMock service contract tests**: Mock gRPC servers/clients verify that each component correctly implements and consumes the proto contract. Tests run without any real infrastructure.
- **Temporal workflow unit tests**: Using `go.temporal.io/sdk/testsuite` to test workflow logic with mocked activities and fast-forwarded timers.
- **Temporal integration tests**: Using `temporal-cli server start-dev` for real workflow execution in CI.
- **Enhanced envtest coverage**: Controller tests verify the CRD→Temporal-workflow bridge with mocked Temporal client.
- **testcontainers for PostgreSQL**: Brain store tests use real PostgreSQL via testcontainers instead of mocks.
- **Full E2E with LiteLLM + Ollama**: E2E tests deploy to k0s with LiteLLM proxying to Ollama (tiny model). Agents execute real prompts and complete real tasks.
- **HITL E2E flow**: E2E test that creates an agent, waits for `WaitingForInput` phase, sends human input, and verifies agent resumes and completes.
- **Multi-agent E2E flow**: E2E test where a senior agent spawns a junior, junior completes, and senior observes completion.
- **Playwright E2E via Connect**: Web dashboard E2E tests use ConnectRPC streaming (replacing WebSocket tests).

## Capabilities

### New Capabilities
- `schema-contract-tests`: `buf lint` + `buf breaking` CI gates that enforce proto schema compatibility and style.
- `service-contract-tests`: GripMock-based tests verifying each gRPC service implementation matches its proto contract, and each client correctly consumes the contract.
- `temporal-workflow-tests`: Temporal SDK test suite for workflow unit tests with mocked activities, plus integration tests with the dev server.
- `e2e-llm-workflow`: Full-stack E2E tests that exercise the complete agent lifecycle with a real LLM backend (Ollama via LiteLLM).

### Modified Capabilities
- `testing-infra`: Existing test infrastructure expanded with new test stages (contract, temporal, LLM-backed E2E) and new Taskfile targets.

## Impact

- **`proto/`**: No changes to proto files (contract tests verify existing protos).
- **`e2e/`**: Expanded with HITL, multi-agent, and LLM-backed test scenarios.
- **New test directories**: `test/contract/` for GripMock service tests, `test/temporal/` for workflow tests.
- **`Taskfile.yml`**: New targets: `proto:lint`, `proto:breaking`, `test:contract`, `test:temporal`, `test:e2e:hitl`, `test:e2e:multi-agent`, `test:e2e:llm`.
- **CI configuration**: Pipeline stages: schema gate → unit → contract → integration → E2E.
- **`devbox.json`**: Add `gripmock` (or run via Docker).
- **Dependencies**: `go.temporal.io/sdk/testsuite` (test only), GripMock (Docker image for CI).
- **Infrastructure (CI)**: E2E stage requires k0s cluster + PostgreSQL + Temporal dev server + LiteLLM + Ollama. Can run in Docker-in-Docker or a dedicated CI runner.
