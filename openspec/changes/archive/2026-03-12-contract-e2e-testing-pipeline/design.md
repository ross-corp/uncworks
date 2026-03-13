## Context

AOT has unit tests (bufconn-based gRPC tests in `internal/server/grpc_test.go`) and basic E2E tests (CRD lifecycle in `e2e/system_test.go`), but lacks contract testing between components and full-stack E2E workflows that exercise the real agent lifecycle with a real LLM. The system has multiple integration boundaries -- ConnectRPC clients, gRPC servers, Temporal workflows, Kubernetes controllers, PostgreSQL brain store, and LiteLLM/Ollama -- each of which can drift independently. A layered testing pipeline is needed where each boundary is verified by the appropriate level of test, and each layer can run in isolation for fast developer feedback.

## Goals / Non-Goals

### Goals
- Establish a five-stage testing pipeline that catches regressions at the earliest possible stage
- Enable each test stage to run independently for local development (`task test:unit`, `task test:contract`, etc.)
- Verify proto schema compatibility on every PR via `buf lint` and `buf breaking`
- Verify that Go gRPC server implementations and TypeScript Connect clients match their proto contracts
- Test Temporal workflow logic with mocked activities and fast-forwarded timers
- Run full E2E tests with a real LLM backend (Ollama via LiteLLM) exercising HITL and multi-agent flows
- Use real infrastructure (PostgreSQL via testcontainers, Temporal dev server) for integration tests

### Non-Goals
- Prescribing a specific CI platform (GitHub Actions, GitLab CI, etc.)
- Testing LLM output quality or prompt engineering effectiveness
- Load testing or performance benchmarking
- Testing third-party services (Ollama, LiteLLM) themselves

## Decisions

### Five-stage pipeline

The testing pipeline has five stages, each gating the next:

1. **Schema Gate** -- `buf lint` + `buf breaking` verify proto schema style and backward compatibility. Fastest to run, catches the most fundamental contract violations.
2. **Unit Tests** -- bufconn gRPC tests, gomock-based Go tests, vitest for TypeScript. No external infrastructure.
3. **Contract Tests** -- GripMock-based service contract tests verify that each gRPC server implementation matches its proto contract and each client correctly consumes it. Runs via Docker container, no real infrastructure.
4. **Integration Tests** -- envtest for K8s controllers, testcontainers-go for PostgreSQL brain store, temporal-cli dev server for workflow integration. Real components, isolated environments.
5. **E2E Tests** -- Full deployment to k0s with PostgreSQL, Temporal, LiteLLM, and Ollama. Agents execute real prompts against a real (tiny) model.

### Each stage is independently runnable

Each stage has its own Taskfile target: `task test:unit`, `task test:contract`, `task test:temporal`, `task test:integration`, `task test:e2e`. In CI, stages run sequentially with each gating the next. For local development, any stage can be run in isolation.

### GripMock for service contracts

GripMock reads `.proto` files and creates mock gRPC servers with stubs defined in YAML. This enables testing that clients send correctly-shaped requests and handle responses properly, without standing up real server infrastructure. The GripMock container runs during `task test:contract`. Server-side contract tests verify that the Go implementations return correct response shapes and error codes for all RPCs.

### Test directory structure

- `test/contract/` -- GripMock service contract tests and stub YAML files
- `test/temporal/` -- Temporal workflow unit and integration tests
- `e2e/` -- Expanded with HITL, multi-agent, and LLM-backed scenarios (existing directory)

### Temporal test suite

Workflow unit tests use `go.temporal.io/sdk/testsuite` with mocked activities and fast-forwarded timers to test workflow logic (happy path, TTL expiry, HITL signals, cancel signals, spawn_junior child workflows, compensation on failure). Integration tests use `temporal-cli server start-dev` for real workflow execution.

### testcontainers for PostgreSQL

`github.com/testcontainers/testcontainers-go` spins up a real PostgreSQL container for brain store integration tests. This replaces any mock or in-memory DB approaches, ensuring SQL queries and migrations are tested against the real database engine.

### E2E with real LLM

Full E2E deploys to k0s with LiteLLM proxying to Ollama running a minimal model (e.g., `qwen2.5:0.5b`). Test prompts are designed to produce deterministic, verifiable outcomes (e.g., "create a file called hello.txt with 'hello world'"). Tests verify the file was created, not the quality of the LLM output.

### HITL E2E flow

CreateAgentRun with a prompt designed to trigger `ask_human` -> wait for `WaitingForInput` phase -> SendHumanInput -> wait for `Succeeded`. The prompt is crafted to deterministically trigger the human-in-the-loop path.

### Multi-agent E2E flow

CreateAgentRun with a prompt designed to trigger `spawn_junior` -> verify child AgentRun created -> wait for child `Succeeded` -> wait for parent `Succeeded`. Senior agent prompt is designed to deterministically delegate to a junior.

### CI configuration

The pipeline stages and their infrastructure requirements are documented, but the specific CI platform configuration (workflow YAML, pipeline config) is left to the implementer. Requirements: schema gate needs only buf, unit tests need only Go/Node, contract tests need Docker, integration tests need Docker, E2E needs k0s + Docker.

### Playwright update

Web E2E tests migrate from WebSocket assertions to Connect streaming assertions, matching the ConnectRPC transport used by the web dashboard.

## Risks / Trade-offs

- **GripMock maintenance burden**: Stub YAML files must be kept in sync with proto changes. Mitigated by the schema gate stage catching proto changes first, and by keeping stubs minimal.
- **E2E flakiness with real LLM**: Even tiny models can produce non-deterministic output. Mitigated by designing prompts for deterministic outcomes and verifying side effects (file creation) rather than output text.
- **E2E infrastructure cost**: Full E2E requires k0s + PostgreSQL + Temporal + LiteLLM + Ollama. This is heavyweight for CI. Mitigated by making E2E the final stage that only runs after all cheaper stages pass.
- **Ollama model size**: Even `qwen2.5:0.5b` requires GPU or significant CPU time. CI runners may need specific hardware. Alternative: use a mock LLM endpoint for CI, real LLM for nightly/manual runs.
- **testcontainers Docker dependency**: Integration tests require Docker, which may not be available in all CI environments. This is a standard requirement for modern CI.
- **Test tag discipline**: Developers must correctly tag tests (`-tags integration`, `-tags e2e`) to prevent slow tests from running in the unit test stage. Mitigated by directory-based separation (`test/`, `e2e/`) and CI stage enforcement.
